package jwk_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/fabric8-services/fabric8-common/test/recorder"
	testtoken "github.com/fabric8-services/fabric8-common/test/token"
	"github.com/fabric8-services/fabric8-common/token/jwk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchKeys(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	r, err := recorder.New("fetch_keys_blackbox_test")
	require.NoError(t, err)
	defer r.Stop()

	t.Run("ok", func(t *testing.T) {
		// when
		loadedKeys, err := jwk.FetchKeys("https://auth-ok/api/keys", httpsupport.WithRoundTripper(r))
		// then all three keys are loaded
		require.NoError(t, err)
		require.Len(t, loadedKeys, 3)
		for _, key := range loadedKeys {
			pk := testtoken.TokenManager.PublicKey(key.KeyID)
			require.NotNil(t, pk)
			require.Equal(t, pk, key.Key)
		}

	})

	t.Run("failure", func(t *testing.T) {
		t.Run("server error", func(t *testing.T) {
			// when
			loadedKeys, err := jwk.FetchKeys("https://auth-error/api/keys", httpsupport.WithRoundTripper(r))
			// then
			require.Error(t, err)
			assert.Empty(t, loadedKeys)
		})

		t.Run("invalid JSON response", func(t *testing.T) {
			// when
			loadedKeys, err := jwk.FetchKeys("https://auth-json/api/keys", httpsupport.WithRoundTripper(r))
			// then
			require.Error(t, err)
			assert.Empty(t, loadedKeys)
		})
	})

}
