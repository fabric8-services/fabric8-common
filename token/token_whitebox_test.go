package token

import (
	"sync"
	"testing"

	"github.com/fabric8-services/fabric8-common/configuration"
	tokensupport "github.com/fabric8-services/fabric8-common/test/generated/token"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestToken(t *testing.T) {
	suite.Run(t, &TestWhiteboxTokenSuite{})
}

type TestWhiteboxTokenSuite struct {
	testsuite.UnitTestSuite
}

func (s *TestWhiteboxTokenSuite) TestDefaultManager() {
	config := defaultMockTokenManagerConfiguration(s.T())
	config.GetAuthServiceURLFunc = func() string {
		return "https://auth.prod-preview.openshift.io"
	}
	config.GetDevModePrivateKeyFunc = func() []byte {
		return []byte(configuration.DevModeRsaPrivateKey)
	}

	s.T().Run("init default manager OK", func(t *testing.T) {
		assertDefaultManager(t, config)
		resetDefaultManager()
		assertDefaultManager(t, config)
	})

	s.T().Run("default manager is not initialized second time", func(t *testing.T) {
		config.GetDevModePrivateKeyFunc = func() []byte {
			return []byte("broken-key")
		}
		resetDefaultManager()
		_, err1 := DefaultManager(config) // Use broken configuration
		require.Error(t, err1)

		config.GetDevModePrivateKeyFunc = func() []byte {
			return []byte(configuration.DevModeRsaPrivateKey)
		}
		_, err2 := DefaultManager(config)
		require.Error(t, err2)
		assert.Equal(t, err1, err2)

		resetDefaultManager()
		manager1, err := DefaultManager(config)
		require.NoError(t, err)
		manager2, err := DefaultManager(config)
		require.NoError(t, err)
		assert.Equal(t, manager1, manager2)
		assert.Equal(t, manager1, defaultManager)
		assert.NotNil(t, defaultManager)
	})
}

func resetDefaultManager() {
	defaultManager = nil
	defaultErr = nil
	defaultOnce = sync.Once{}
}

func assertDefaultManager(t *testing.T, config ManagerConfiguration) {
	manager, err := DefaultManager(config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, defaultManager, manager)
}

func (s *TestWhiteboxTokenSuite) TestKeyLoaded() {
	// given
	config := defaultMockTokenManagerConfiguration(s.T())
	config.GetAuthServiceURLFunc = func() string {
		return "https://auth.prod-preview.openshift.io"
	}

	s.T().Run("dev mode enabled", func(t *testing.T) {
		// given
		config.GetDevModePrivateKeyFunc = func() []byte {
			return []byte(configuration.DevModeRsaPrivateKey)
		}
		tm, err := NewManager(config)
		require.NoError(t, err)
		// when
		key := tm.PublicKey(devModeKeyID)
		// then
		assert.NotNil(t, key)
	})

	s.T().Run("dev mode not enabled", func(t *testing.T) {
		// given
		config.GetDevModePrivateKeyFunc = func() []byte {
			return nil
		}
		tm, err := NewManager(config)
		require.NoError(t, err)
		// when
		key := tm.PublicKey(devModeKeyID)
		// then
		assert.Nil(t, key)
	})
}

func defaultMockTokenManagerConfiguration(t *testing.T) *tokensupport.ManagerConfigurationMock {
	config := tokensupport.NewManagerConfigurationMock(t)
	config.GetAuthServiceURLFunc = func() string {
		return "https://auth.prod-preview.openshift.io"
	}

	config.GetDevModePrivateKeyFunc = func() []byte {
		return []byte(configuration.DevModeRsaPrivateKey)
	}
	return config
}
