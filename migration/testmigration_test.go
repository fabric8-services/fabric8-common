package migration_test

import "database/sql"
import "github.com/fabric8-services/fabric8-common/migration"

type migrateData struct {
}

func Migrate(db *sql.DB, catalog string) error {
	return migration.Migrate(db, catalog, migrateData{})
}

func (d migrateData) Asset(name string) ([]byte, error) {
	return Asset(name)
}

// AssetNameWithArgs impl example
func (d migrateData) AssetNameWithArgs() [][]string {
	names := [][]string{
		{"000-bootstrap.sql"},                    // add version table
		{"001-create-tables.sql"},                // add environments table with id, name, type
		{"002-insert-test-data.sql"},             // insert record
		{"003-alter-tables.sql"},                 // add 'namesapce' col
		{"004-insert-test-data.sql"},             // add record with namesapce col
		{"005-alter-tables.sql", "cluster1.com"}, // add cluster col; "cluster1.com" accessed with '{{ index . 0}}'
	}
	return names
}
