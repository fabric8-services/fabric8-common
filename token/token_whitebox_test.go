package token

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/resource"

	testconfiguration "github.com/fabric8-services/fabric8-common/test/configuration"
	testkeys "github.com/fabric8-services/fabric8-common/test/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyLoaded(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	config := testconfiguration.NewDefaultMockTokenManagerConfiguration(t)
	config.GetAuthServiceURLFunc = func() string {
		return "https://auth.prod-preview.openshift.io"
	}

	t.Run("dev mode enabled", func(t *testing.T) {
		// given
		config.GetDevModePrivateKeyFunc = func() []byte {
			return []byte(testkeys.DevModePrivateKey)
		}
		tm, err := NewManager(config)
		require.NoError(t, err)
		// when
		key := tm.PublicKey(devModeKeyID)
		// then
		assert.NotNil(t, key)
	})

	t.Run("dev mode not enabled", func(t *testing.T) {
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
