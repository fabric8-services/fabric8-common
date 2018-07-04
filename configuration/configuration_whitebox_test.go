package configuration

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var reqLong *http.Request
var reqShort *http.Request
var config *Registry

func init() {
	// ensure that the content here is executed only once.
	reqLong = &http.Request{Host: "api.service.domain.org"}
	reqShort = &http.Request{Host: "api.domain.org"}
	resetConfiguration()
}

func resetConfiguration() {
	var err error
	config, err = Get()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestGetLogLevelOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_LOG_LEVEL"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, defaultLogLevel, config.GetLogLevel())

	os.Setenv(key, "warning")
	resetConfiguration()

	assert.Equal(t, "warning", config.GetLogLevel())
}

func TestGetTransactionTimeoutOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_POSTGRES_TRANSACTION_TIMEOUT"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, time.Duration(5*time.Minute), config.GetPostgresTransactionTimeout())

	os.Setenv(key, "6m")
	resetConfiguration()

	assert.Equal(t, time.Duration(6*time.Minute), config.GetPostgresTransactionTimeout())
}

func TestValidRedirectURLsInDevModeCanBeOverridden(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_REDIRECT_VALID"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	whitelist, err := config.GetValidRedirectURLs(nil)
	require.NoError(t, err)
	assert.Equal(t, devModeValidRedirectURLs, whitelist)

	os.Setenv(key, "https://someDomain.org/redirect")
	resetConfiguration()
}

func TestRedirectURLsForLocalhostRequestAreExcepted(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	// Valid if requesting prod-preview to redirect to localhost or to openshift.io
	// OR if requesting openshift to redirect to openshift.io
	// Invalid otherwise
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api", "http://localhost:3000/home"))
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api", "https://127.0.0.1"))
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io:8080/api", "https://127.0.0.1"))
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api", "https://prod-preview.openshift.io/home"))
	assert.True(t, validateRedirectURL(t, "https://api.openshift.io/api", "https://openshift.io/home"))
	assert.True(t, validateRedirectURL(t, "https://api.openshift.io:8080/api", "https://openshift.io/home"))
	assert.False(t, validateRedirectURL(t, "https://api.openshift.io/api", "http://localhost:3000/api"))
	assert.False(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api", "http://domain.com"))
	assert.False(t, validateRedirectURL(t, "https://api.openshift.io/api", "http://domain.com"))
}

func validateRedirectURL(t *testing.T, request string, redirect string) bool {
	r, err := http.NewRequest("", request, nil)
	require.NoError(t, err)
	whitelist, err := config.checkLocalhostRedirectException(r)
	require.NoError(t, err)

	matched, err := regexp.MatchString(whitelist, redirect)
	require.NoError(t, err)
	return matched
}

func TestOSOProxyURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	assert.Equal(t, "", config.GetOpenshiftProxyURL())
}
