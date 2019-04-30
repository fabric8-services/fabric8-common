package internal

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DBCleanupTestSuite struct {
	DBTestSuite
}

func TestDBCleanup(t *testing.T) {
	suite.Run(t, &DBCleanupTestSuite{NewDBTestSuiteSuite()})
}

func (s *DBCleanupTestSuite) TestCleanupOK() {
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

	require.NoError(s.T(), s.CleanTest())

	s.T().Run("cleanup composite PKs OK", func(t *testing.T) {
		r, err = s.DB.DB().Exec(`SELECT * FROM identity_clusters WHERE cluster_id = '1'`)
		assertRows(t, 0, r, err)
	})

	s.T().Run("cleanup single PK OK", func(t *testing.T) {
		r, err = s.DB.DB().Exec(`SELECT * FROM clusters WHERE id = '1'`)
		assertRows(t, 0, r, err)
	})
}

func (s *DBCleanupTestSuite) TestCleanupFailIfConstrainsViolated() {
	// Bootstrap tables
	_, err := s.DB.DB().Exec(`
			CREATE TABLE users (id int NOT NULL PRIMARY KEY);
			CREATE TABLE identities (id int NOT NULL PRIMARY KEY, user_id int NOT NULL references users(id));
		`)
	require.NoError(s.T(), err)

	// Create a record with recording it in the DB cleaner
	err = s.DB.Create(&user{ID: 1}).Error
	require.NoError(s.T(), err)
	// Create a record without recording
	err = s.DB.Exec(`INSERT INTO identities (id, user_id) VALUES ('1', '1')`).Error
	require.NoError(s.T(), err)
	defer func() {
		s.DB.Delete(&identity{ID: 1})
	}()
	// Check if they are available then clean the test data
	r, err := s.DB.DB().Exec(`SELECT * FROM identities WHERE id = '1'`)
	assertRows(s.T(), 1, r, err)
	r, err = s.DB.DB().Exec(`SELECT * FROM users WHERE id = '1'`)
	assertRows(s.T(), 1, r, err)

	err = s.CleanTest()
	assert.EqualError(s.T(), err, "failed to delete entities for 'users' table: pq: update or delete on table \"users\" violates foreign key constraint \"identities_user_id_fkey\" on table \"identities\"")
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

type identity struct {
	ID     int `sql:"type:int" gorm:"primary_key;column:id"`
	UserID int `sql:"type:int" gorm:"column:user_id"`
}

type user struct {
	ID int `sql:"type:int" gorm:"primary_key;column:id"`
}
