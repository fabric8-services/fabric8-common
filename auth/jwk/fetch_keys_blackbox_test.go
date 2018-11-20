package jwk_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/auth/jwk"
	"github.com/fabric8-services/fabric8-common/configuration"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/resource"
	testtoken "github.com/fabric8-services/fabric8-common/test/auth"
	tokensupport "github.com/fabric8-services/fabric8-common/test/generated/token"
	"github.com/fabric8-services/fabric8-common/test/recorder"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchKeys(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	r, err := recorder.New("fetch_keys_blackbox_test")
	require.NoError(t, err)
	defer func() {
		err := r.Stop()
		require.NoError(t, err)
	}()
	config := defaultMockTokenManagerConfiguration(t)
	config.GetAuthServiceURLFunc = func() string {
		return "https://auth-ok"
	}
	tm, err := auth.NewManager(config, httpsupport.WithRoundTripper(r))
	require.NoError(t, err)

	t.Run("ok", func(t *testing.T) {
		// when
		loadedKeys, err := jwk.FetchKeys("https://auth-ok/api/token/keys", httpsupport.WithRoundTripper(r))
		// then all three keys are loaded
		require.NoError(t, err)
		require.Len(t, loadedKeys, 3)
		for _, key := range loadedKeys {
			t.Logf("checking key '%s' in %d keys...", key.KeyID, len(testtoken.TokenManager.PublicKeys()))
			pk := tm.PublicKey(key.KeyID)
			require.NotNil(t, pk)
			require.Equal(t, pk, key.Key)
		}

	})

	t.Run("failure", func(t *testing.T) {
		t.Run("server error", func(t *testing.T) {
			// when
			loadedKeys, err := jwk.FetchKeys("https://auth-error/api/token/keys", httpsupport.WithRoundTripper(r))
			// then
			require.Error(t, err)
			assert.Empty(t, loadedKeys)
		})

		t.Run("invalid JSON response", func(t *testing.T) {
			// when
			loadedKeys, err := jwk.FetchKeys("https://auth-json/api/token/keys", httpsupport.WithRoundTripper(r))
			// then
			require.Error(t, err)
			assert.Empty(t, loadedKeys)
		})
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
