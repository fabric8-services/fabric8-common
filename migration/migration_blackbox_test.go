package migration_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/fabric8-services/fabric8-common/gormsupport"
	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dbName = "test"
)

var host, port string

func setupTest(t *testing.T) {
	dbConfig := fmt.Sprintf("host=%s port=%s user=postgres password=mysecretpassword sslmode=disable connect_timeout=5", host, port)

	db, err := sql.Open("postgres", dbConfig)
	require.NoError(t, err, "cannot connect to database: %s", dbName)
	defer db.Close()

	_, err = db.Exec("DROP DATABASE " + dbName)
	if err != nil && !gormsupport.IsInvalidCatalogName(err) {
		require.NoError(t, err, "failed to drop database '%s'", dbName)
	}

	_, err = db.Exec("CREATE DATABASE " + dbName)
	require.NoError(t, err, "failed to create database '%s'", dbName)
}

func TestMigrate(t *testing.T) {
	resource.Require(t, resource.Database)

	host = os.Getenv("F8_POSTGRES_HOST")
	require.NotEmpty(t, host, "F8_POSTGRES_HOST is not set")
	port = os.Getenv("F8_POSTGRES_PORT")
	require.NotEmpty(t, port, "F8_POSTGRES_PORT is not set")

	setupTest(t)

	dbConfig := fmt.Sprintf("host=%s port=%s user=postgres password=mysecretpassword dbname=%s sslmode=disable connect_timeout=5",
		host, port, dbName)

	sqlDB, err := sql.Open("postgres", dbConfig)
	require.NoError(t, err, "cannot connect to DB '%s'", dbName)
	defer sqlDB.Close()

	gormDB, err := gorm.Open("postgres", dbConfig)
	require.NoError(t, err, "cannot connect to DB '%s'", dbName)
	defer gormDB.Close()

	dialect := gormDB.Dialect()
	dialect.SetDB(sqlDB)

	err = Migrate(gormDB.DB(), dbName)

	assert.Nil(t, err)
	checkDb(t, gormDB, dialect)
}

func checkDb(t *testing.T, gormDB *gorm.DB, dialect gorm.Dialect) {
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

	var name, namespace, cluster string
	assert.True(t, rows.Next())
	rows.Scan(&name, &cluster, &namespace)
	assert.Equal(t, "osio-stage", name)
	assert.Equal(t, "cluster1.com", cluster)
	assert.Equal(t, "", namespace)

	assert.True(t, rows.Next())
	rows.Scan(&name, &cluster, &namespace)
	assert.Equal(t, "osio-prod", name)
	assert.Equal(t, "cluster1.com", cluster)
	assert.Equal(t, "dsaas-prod", namespace)
}
