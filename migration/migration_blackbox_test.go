package migration_test

import (
	"database/sql"
	"testing"

	"github.com/fabric8-services/fabric8-common/internal"
	"github.com/fabric8-services/fabric8-common/migration"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MigrationTestSuite struct {
	internal.DBTestSuite
}

func TestMigration(t *testing.T) {
	suite.Run(t, &MigrationTestSuite{internal.NewDBTestSuiteSuite()})
}

func (s *MigrationTestSuite) TestMigrate() {
	s.T().Run("migration OK", func(t *testing.T) {
		err := Migrate(s.DB.DB(), s.DBName)
		require.NoError(t, err)

		checkMigrate(t, s.DB, s.DB.Dialect())
	})

	s.T().Run("migration rollback if failed", func(t *testing.T) {
		err := migration.Migrate(s.DB.DB(), s.DBName, rollbackData{})
		require.Error(t, err)
		assert.Equal(t, "failed to execute migration of step 0 of version 7: pq: smallint out of range", err.Error())

		checkRollback(t, s.DB, s.DB.Dialect())
	})
}

func checkMigrate(t *testing.T, gormDB *gorm.DB, dialect gorm.Dialect) {
	assert.True(t, dialect.HasTable("environments"))
	assert.True(t, dialect.HasColumn("environments", "name"))
	assert.True(t, dialect.HasColumn("environments", "namespace"))
	assert.True(t, dialect.HasColumn("environments", "cluster"))

	count := -1
	gormDB.Table("environments").Count(&count)
	assert.Equal(t, 2, count)

	rows, err := gormDB.Table("environments").Select("name,cluster,namespace").Rows()
	require.NoError(t, err)
	assert.NotNil(t, rows)
	defer rows.Close()

	var name, namespace, cluster string
	require.True(t, rows.Next())
	rows.Scan(&name, &cluster, &namespace)
	assert.Equal(t, "osio-stage", name)
	assert.Equal(t, "cluster1.com", cluster)
	assert.Equal(t, "", namespace)

	require.True(t, rows.Next())
	rows.Scan(&name, &cluster, &namespace)
	assert.Equal(t, "osio-prod", name)
	assert.Equal(t, "cluster1.com", cluster)
	assert.Equal(t, "dsaas-prod", namespace)
}

func checkRollback(t *testing.T, gormDB *gorm.DB, dialect gorm.Dialect) {
	assert.True(t, dialect.HasTable("users"))
	assert.True(t, dialect.HasColumn("users", "name"))
	assert.True(t, dialect.HasColumn("users", "allocated_storage"))

	count := -1
	gormDB.Table("users").Count(&count)
	assert.Equal(t, 0, count) // as rollback, found zero records
}

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

type rollbackData struct {
}

func (d rollbackData) Asset(name string) ([]byte, error) {
	return Asset(name)
}

// AssetNameWithArgs impl example
func (d rollbackData) AssetNameWithArgs() [][]string {
	names := [][]string{
		{"000-bootstrap.sql"},                    // add version table
		{"001-create-tables.sql"},                // add environments table with id, name, type
		{"002-insert-test-data.sql"},             // insert record
		{"003-alter-tables.sql"},                 // add 'namesapce' col
		{"004-insert-test-data.sql"},             // add record with namesapce col
		{"005-alter-tables.sql", "cluster1.com"}, // add cluster col; "cluster1.com" accessed with '{{ index . 0}}'
		{"006-create-users-table.sql"},           // create users table
		{"007-insert-invalid-data.sql"},          // insert record with wrong data (out of range for smallint)
	}
	return names
}
