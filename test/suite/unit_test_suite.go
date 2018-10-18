package suite

import (
	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/stretchr/testify/suite"
)

// NewRemoteTestSuite instantiates a new UnitTestSuite
func NewUnitTestSuite() UnitTestSuite {
	return UnitTestSuite{}
}

// RemoteTestSuite is a base for unit tests
type UnitTestSuite struct {
	suite.Suite
}

// SetupSuite implements suite.SetupAllSuite
func (s *UnitTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.UnitTest)
}
