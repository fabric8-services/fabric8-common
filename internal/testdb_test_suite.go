package internal

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/fabric8-services/fabric8-common/test/suite"

	"github.com/stretchr/testify/require"
)

const (
	dbName      = "test"
	defaultHost = "localhost"
	defaultPort = "5435"
)

// NewDBTestSuiteSuite instantiates a new DBTestSuite
func NewDBTestSuiteSuite() DBTestSuite {
	cfg := &config{}
	return DBTestSuite{
		DBTestSuite: suite.NewDBTestSuite(cfg),
		config:      cfg,
		DBName:      dbName,
	}
}

// DBTestSuite is a base for tests using a gorm db
type DBTestSuite struct {
	suite.DBTestSuite
	config suite.DBTestSuiteConfiguration
	db     *sql.DB
	DBName string
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	// Create test DB
	dbConfig := postgresConfigString(false)
	s.T().Logf("DB config string: %s", dbConfig)
	var err error
	s.db, err = sql.Open("postgres", dbConfig)
	require.NoError(s.T(), err, "cannot connect to database: %s", dbName)
	_, err = s.db.Exec("DROP DATABASE IF EXISTS " + dbName)
	require.NoError(s.T(), err, "failed to drop database '%s'", dbName)
	_, err = s.db.Exec("CREATE DATABASE " + dbName)
	require.NoError(s.T(), err, "failed to create database '%s'", dbName)

	s.DBTestSuite.SetupSuite()
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *DBTestSuite) TearDownSuite() {
	s.DBTestSuite.TearDownSuite()

	// Drop test DB
	_, err := s.db.Exec("DROP DATABASE IF EXISTS " + dbName)
	require.NoError(s.T(), err, "failed to drop database '%s'", dbName)
	s.db.Close()
}

type config struct {
}

func (c *config) GetPostgresConfigString() string {
	return postgresConfigString(true)
}

func postgresConfigString(includeDBName bool) string {
	host := os.Getenv("F8_POSTGRES_HOST")
	if host == "" {
		host = defaultHost
	}
	port := os.Getenv("F8_POSTGRES_PORT")
	if port == "" {
		port = defaultPort
	}

	var dbname string
	if includeDBName {
		dbname = "dbname=" + dbName
	}
	return fmt.Sprintf("host=%s port=%s user=postgres password=mysecretpassword %s sslmode=disable connect_timeout=5", host, port, dbname)
}

func (c *config) IsDBLogsEnabled() bool {
	return true
}

func (c *config) IsCleanTestDataEnabled() bool {
	return true
}
