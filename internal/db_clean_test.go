package internal_test

import (
	"database/sql"
	"testing"

	"github.com/fabric8-services/fabric8-common/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DBCleanupTestSuite struct {
	internal.DBTestSuite
}

func TestDBCleanup(t *testing.T) {
	suite.Run(t, &DBCleanupTestSuite{internal.NewDBTestSuiteSuite()})
}

func (s *DBCleanupTestSuite) TestCleanup() {
	// Bootstrap tables
	_, err := s.DB.DB().Exec(`
			CREATE TABLE clusters (id int NOT NULL primary key, name text);
			CREATE TABLE identity_clusters (identity_id int NOT NULL, cluster_id int NOT NULL references clusters(id), PRIMARY KEY(identity_id, cluster_id));
		`)
	require.NoError(s.T(), err)

	// Create records
	err = s.DB.Create(&cluster{ID: 1, Name: "test-name"}).Error
	require.NoError(s.T(), err)
	err = s.DB.Create(&identityCluster{ClusterID: 1, IdentityID: 1}).Error
	require.NoError(s.T(), err)

	// Check if they are available then clean the test data
	r, err := s.DB.DB().Exec(`SELECT * FROM identity_clusters WHERE cluster_id = '1'`)
	assertRows(s.T(), 1, r, err)
	r, err = s.DB.DB().Exec(`SELECT * FROM clusters WHERE id = '1'`)
	assertRows(s.T(), 1, r, err)

	s.CleanTest()

	s.T().Run("cleanup composite PKs OK", func(t *testing.T) {
		r, err = s.DB.DB().Exec(`SELECT * FROM identity_clusters WHERE cluster_id = '1'`)
		assertRows(t, 0, r, err)
	})

	s.T().Run("cleanup single PK OK", func(t *testing.T) {
		r, err = s.DB.DB().Exec(`SELECT * FROM clusters WHERE id = '1'`)
		assertRows(t, 0, r, err)
	})
}

func assertRows(t *testing.T, expected int, result sql.Result, err error) {
	require.NoError(t, err)
	rows, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(expected), rows)
}

type identityCluster struct {
	IdentityID int `sql:"type:int" gorm:"primary_key;column:identity_id"`
	ClusterID  int `sql:"type:int" gorm:"primary_key;column:cluster_id"`
}

type cluster struct {
	ID   int `sql:"type:int" gorm:"primary_key;column:id"`
	Name string
}
