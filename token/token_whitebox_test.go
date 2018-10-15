package token

import (
	"testing"

	testconfiguration "github.com/fabric8-services/fabric8-common/test/configuration"
	testkeys "github.com/fabric8-services/fabric8-common/test/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyLoaded(t *testing.T) {

	// given
	config := testconfiguration.NewDefaultMockTokenManagerConfiguration(t)
	config.GetAuthServiceURLFunc = func() string {
		return "https://auth.prod-preview.openshift.io"
	}

	config.GetAuthKeysPathFunc = func() string {
		return "/api/token/keys"
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
