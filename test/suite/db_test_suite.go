package suite

import (
	"context"
	"fmt"
	"os"

	"github.com/fabric8-services/fabric8-common/log"
	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/stretchr/testify/assert"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // need to import postgres driver
	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = &DBTestSuite{}
var _ suite.TearDownAllSuite = &DBTestSuite{}

// DBTestSuiteConfiguration the interface for the DBTestSuite configuration
type DBTestSuiteConfiguration interface {
	GetPostgresConfigString() string
	IsDBLogsEnabled() bool
	IsCleanTestDataEnabled() bool
	IsCleanTestDataErrorReportingRequired() bool
}

// NewDBTestSuite instantiates a new DBTestSuite
func NewDBTestSuite(config DBTestSuiteConfiguration) DBTestSuite {
	return DBTestSuite{
		config: config,
	}
}

// DBTestSuite is a base for tests using a gorm db
type DBTestSuite struct {
	suite.Suite
	config          DBTestSuiteConfiguration
	Ctx             context.Context
	DB              *gorm.DB
	SetupSubtest    func()
	TearDownSubtest func()
	CleanTest       func() error
	CleanSuite      func() error
}

// Run overrides the default behaviour of the Suite.Run method, in order
// to run the SetupSubtest and TearDownSubtest methods for each subtest
func (s *DBTestSuite) Run(name string, subtest func()) bool {
	fmt.Printf("==== RUN Subtest '%s'\n", name)
	if s.SetupSubtest != nil {
		s.SetupSubtest()
	}
	defer func() {
		fmt.Printf("==== END Subtest '%s'\n", name)
		if s.TearDownSubtest != nil {
			s.TearDownSubtest()
		}
	}()
	return s.Suite.Run(name, subtest)
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Database)
	if _, c := os.LookupEnv(resource.Database); c != false {
		var err error
		s.DB, err = gorm.Open("postgres", s.config.GetPostgresConfigString())
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err":             err,
				"postgres_config": s.config.GetPostgresConfigString(),
			}, "failed to connect to the database")
		}
	}
	// configures the log mode for the SQL queries (by default, disabled)
	s.DB.LogMode(s.config.IsDBLogsEnabled())
	s.CleanSuite = DeleteCreatedEntities(s.DB, s.config)
}

// SetupTest implements suite.SetupTest
func (s *DBTestSuite) SetupTest() {
	s.CleanTest = DeleteCreatedEntities(s.DB, s.config)
}

// TearDownTest implements suite.TearDownTest
func (s *DBTestSuite) TearDownTest() {
	// in some cases, we might need to keep the test data in the DB for inspecting/reproducing
	// the SQL queries. In that case, the `AUTH_CLEAN_TEST_DATA` env variable should be set to `false`.
	// By default, test data will be removed from the DB after each test
	if s.config.IsCleanTestDataEnabled() {
		assert.NoError(s.T(), s.CleanTest())
	}
}

// PopulateDBTestSuite populates the DB with common values
func (s *DBTestSuite) PopulateDBTestSuite(ctx context.Context) {
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *DBTestSuite) TearDownSuite() {
	// in some cases, we might need to keep the test data in the DB for inspecting/reproducing
	// the SQL queries. In that case, the `AUTH_CLEAN_TEST_DATA` env variable should be set to `false`.
	// By default, test data will be removed from the DB after each test
	defer s.DB.Close()
	if s.config.IsCleanTestDataEnabled() {
		assert.NoError(s.T(), s.CleanSuite())
	}
}

// DisableGormCallbacks will turn off gorm's automatic setting of `created_at`
// and `updated_at` columns. Call this function and make sure to `defer` the
// returned function.
//
//    resetFn := DisableGormCallbacks()
//    defer resetFn()
func (s *DBTestSuite) DisableGormCallbacks() func() {
	gormCallbackName := "gorm:update_time_stamp"
	// remember old callbacks
	oldCreateCallback := s.DB.Callback().Create().Get(gormCallbackName)
	oldUpdateCallback := s.DB.Callback().Update().Get(gormCallbackName)
	// remove current callbacks
	s.DB.Callback().Create().Remove(gormCallbackName)
	s.DB.Callback().Update().Remove(gormCallbackName)
	// return a function to restore old callbacks
	return func() {
		s.DB.Callback().Create().Register(gormCallbackName, oldCreateCallback)
		s.DB.Callback().Update().Register(gormCallbackName, oldUpdateCallback)
	}
}
