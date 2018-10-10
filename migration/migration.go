package migration

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"text/template"

	"github.com/fabric8-services/fabric8-common/log"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	errs "github.com/pkg/errors"
)

// MigrateData should be implemented by caller of migration.Migrate() to provide different data for migration.
type MigrateData interface {
	// AssetNameWithArgs returns list of file names (sql filenames) with args.
	// First entry is considered as filename and rest are considered as args.
	AssetNameWithArgs() [][]string
	// Asset returns content of given name (filename)
	Asset(name string) ([]byte, error)
}

// advisoryLockID is a random number that should be used within the application
// by anybody who wants to modify the "version" table.
const advisoryLockID = 42

// fn defines the type of function that can be part of a migration steps
type fn func(tx *sql.Tx) error

// steps defines a collection of all the functions that make up a version
type steps []fn

// migrations defines all a collection of all the steps
type migrations []steps

// Migrate executes the required migration of the database on startup.
// For each successful migration, an entry will be written into the "version"
// table, that states when a certain version was reached.
func Migrate(db *sql.DB, catalog string, migrateData MigrateData) error {
	var err error

	if db == nil {
		return errs.Errorf("database handle is nil")
	}

	m := getMigrations(migrateData)

	var tx *sql.Tx
	for nextVersion := int64(0); nextVersion < int64(len(m)) && err == nil; nextVersion++ {

		tx, err = db.Begin()
		if err != nil {
			return errs.Errorf("failed to start transaction: %s", err)
		}

		err = migrateToNextVersion(tx, &nextVersion, m, catalog)

		if err != nil {
			oldErr := err
			log.Info(nil, map[string]interface{}{
				"next_version": nextVersion,
				"migrations":   m,
				"err":          err,
			}, "Rolling back transaction due to: %v", err)

			if err = tx.Rollback(); err != nil {
				log.Error(nil, map[string]interface{}{
					"next_version": nextVersion,
					"migrations":   m,
					"err":          err,
				}, "error while rolling back transaction: ", err)
				return errs.Errorf("error while rolling back transaction: %s", err)
			}
			return oldErr
		}

		if err = tx.Commit(); err != nil {
			log.Error(nil, map[string]interface{}{
				"migrations": m,
				"err":        err,
			}, "error during transaction commit: %v", err)
			return errs.Errorf("error during transaction commit: %s", err)
		}

	}

	if err != nil {
		log.Error(nil, map[string]interface{}{
			"migrations": m,
			"err":        err,
		}, "migration failed with error: %v", err)
		return errs.Errorf("migration failed with error: %s", err)
	}

	return nil
}

// NewMigrationContext aims to create a new goa context where to initialize the
// request and req_id context keys.
// NOTE: We need this function to initialize the goa.ContextRequest
func NewMigrationContext(ctx context.Context) context.Context {
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx = goa.NewContext(ctx, nil, req, params)
	// set a random request ID for the context
	var reqID string
	ctx, reqID = client.ContextWithRequestID(ctx)

	log.Debug(ctx, nil, "Initialized the migration context with Request ID: %v", reqID)

	return ctx
}

// getMigrations returns the migrations all the migrations we have.
func getMigrations(migrateData MigrateData) migrations {
	m := migrations{}
	for _, nameWithArgs := range migrateData.AssetNameWithArgs() {
		m = append(m, steps{executeSQLFile(migrateData.Asset, nameWithArgs[0], nameWithArgs[1:]...)})
	}
	return m
}

// migrateToNextVersion migrates the database to the nextVersion.
// If the database is already at nextVersion or higher, the nextVersion
// will be set to the actual next version.
func migrateToNextVersion(tx *sql.Tx, nextVersion *int64, m migrations, catalog string) error {
	// Obtain exclusive transaction level advisory that doesn't depend on any table.
	// Once obtained, the lock is held for the remainder of the current transaction.
	// (There is no UNLOCK TABLE command; locks are always released at transaction end.)
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", advisoryLockID); err != nil {
		return errs.Wrapf(err, "failed to acquire lock: %s", advisoryLockID)
	}

	// Determine current version and adjust the outmost loop
	// iterator variable "version"
	currentVersion, err := getCurrentVersion(tx, catalog)
	if err != nil {
		return errs.WithStack(err)
	}
	*nextVersion = currentVersion + 1
	if *nextVersion >= int64(len(m)) {
		// No further updates to apply (this is NOT an error)
		log.Info(nil, map[string]interface{}{
			"next_version":    *nextVersion,
			"current_version": currentVersion,
		}, "Current version %d. Nothing to update.", currentVersion)
		return nil
	}

	log.Info(nil, map[string]interface{}{
		"next_version":    *nextVersion,
		"current_version": currentVersion,
	}, "Attempt to update DB to version %v", *nextVersion)

	// Apply all the updates of the next version
	for j := range m[*nextVersion] {
		if err := m[*nextVersion][j](tx); err != nil {
			return errs.Errorf("failed to execute migration of step %d of version %d: %s", j, *nextVersion, err)
		}
	}

	if _, err := tx.Exec("INSERT INTO version(version) VALUES($1)", *nextVersion); err != nil {
		return errs.Errorf("failed to update DB to version %d: %s", *nextVersion, err)
	}

	log.Info(nil, map[string]interface{}{
		"next_version":    *nextVersion,
		"current_version": currentVersion,
	}, "Successfully updated DB to version %v", *nextVersion)

	return nil
}

// executeSQLFile loads the given filename from the packaged SQL files and
// executes it on the given database. Golang text/template module is used
// to handle all the optional arguments passed to the sql files
func executeSQLFile(Asset func(string) ([]byte, error), filename string, args ...string) fn {
	return func(db *sql.Tx) error {
		data, err := Asset(filename)
		if err != nil {
			return errs.Wrapf(err, "failed to find filename: %s", filename)
		}

		if len(args) > 0 {
			tmpl, err := template.New("sql").Parse(string(data))
			if err != nil {
				return errs.Wrapf(err, "failed to parse SQL template in file %s", filename)
			}
			var sqlScript bytes.Buffer
			writer := bufio.NewWriter(&sqlScript)
			err = tmpl.Execute(writer, args)
			if err != nil {
				return errs.Wrapf(err, "failed to execute SQL template in file %s", filename)
			}
			// We need to flush the content of the writer
			writer.Flush()
			_, err = db.Exec(sqlScript.String())
			if err != nil {
				log.Error(context.Background(), map[string]interface{}{"err": err}, "failed to execute this query in file %s: \n\n%s\n\n", filename, sqlScript.String())
			}
		} else {
			_, err = db.Exec(string(data))
			if err != nil {
				log.Error(context.Background(), map[string]interface{}{"err": err}, "failed to execute this query in file: %s \n\n%s\n\n", filename, string(data))
			}
		}

		return errs.WithStack(err)
	}
}

// getCurrentVersion returns the highest version from the version
// table or -1 if that table does not exist.
//
// Returning -1 simplifies the logic of the migration process because
// the next version is always the current version + 1 which results
// in -1 + 1 = 0 which is exactly what we want as the first version.
func getCurrentVersion(db *sql.Tx, catalog string) (int64, error) {
	query := `SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_catalog=$1
				AND table_name='version')`
	row := db.QueryRow(query, catalog)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return -1, errs.Errorf(`failed to scan if table "version" exists: %s`, err)
	}

	if !exists {
		// table doesn't exist
		return -1, nil
	}

	row = db.QueryRow("SELECT max(version) as current FROM version")

	var current int64 = -1
	if err := row.Scan(&current); err != nil {
		return -1, errs.Errorf(`failed to scan max version in table "version": %s`, err)
	}

	return current, nil
}
