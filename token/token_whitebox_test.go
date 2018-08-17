package token

import (
	"os"
	"testing"

	config "github.com/fabric8-services/fabric8-common/configuration"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestWhiteBoxToken(t *testing.T) {
	suite.Run(t, &WhiteBoxTestTokenSuite{})
}

type WhiteBoxTestTokenSuite struct {
	testsuite.UnitTestSuite
}

func (s *WhiteBoxTestTokenSuite) TestNoTestKeyLoadedIfRunInNotDevMode() {
	// Disable Dev Mode
	existingEnvironment := os.Getenv("F8CLUSTER_DEVELOPER_MODE_ENABLED")
	defer func() {
		os.Setenv("F8CLUSTER_DEVELOPER_MODE_ENABLED", existingEnvironment)
	}()
	os.Unsetenv("F8CLUSTER_DEVELOPER_MODE_ENABLED")

	c, err := config.GetConfigurationData()
	require.NoError(s.T(), err)
	dummyConfig := &dummyConfiguration{ConfigurationData: c}

	// The dev mode key should not be loaded
	tm, err := NewManager(dummyConfig)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), tm.PublicKey(devModeKeyID))
}

func (s *WhiteBoxTestTokenSuite) TestKeyLoadedIfRunInDevMode() {
	tm, err := NewManager(s.Config)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), tm.PublicKey(devModeKeyID))
}

type dummyConfiguration struct {
	*config.ConfigurationData
}

func (c *dummyConfiguration) GetAuthServiceURL() string {
	return "https://auth.prod-preview.openshift.io"
}
