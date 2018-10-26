package configuration

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/configuration"
	tokensupport "github.com/fabric8-services/fabric8-common/test/generated/token"
)

// NewDefaultMockTokenManagerConfiguration initializes a new mock configuration for a token manager
// functions can be overridden afterwards if needed
func NewDefaultMockTokenManagerConfiguration(t *testing.T) *tokensupport.ManagerConfigurationMock {
	config := tokensupport.NewManagerConfigurationMock(t)
	config.GetAuthServiceURLFunc = func() string {
		return "https://auth.prod-preview.openshift.io"
	}

	config.GetDevModePrivateKeyFunc = func() []byte {
		return []byte(configuration.DevModeRsaPrivateKey)
	}
	return config
}
