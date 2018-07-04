package configuration_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"net/http"

	"time"

	"github.com/fabric8-services/fabric8-common/configuration"
	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultConfigFilePath       = "../config.yaml"
	defaultValuesConfigFilePath = "" // when the code defaults are to be used, the path to config file is ""
)

var reqLong *http.Request
var reqShort *http.Request
var config *configuration.Registry

func TestMain(m *testing.M) {
	resetConfiguration(defaultConfigFilePath)

	reqLong = &http.Request{Host: "api.service.domain.org"}
	reqShort = &http.Request{Host: "api.domain.org"}
	os.Exit(m.Run())
}

func resetConfiguration(configPath string) {
	var err error
	config, err = configuration.New(configPath)
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestGetAuthURLSetByEnvVaribaleOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	env := os.Getenv("F8_AUTH_URL")
	defer func() {
		os.Setenv("F8_AUTH_URL", env)
		resetConfiguration(defaultValuesConfigFilePath)
	}()

	os.Setenv("F8_AUTH_URL", "https://auth.xyz.io")
	resetConfiguration(defaultValuesConfigFilePath)

	url := config.GetAuthServiceURL()
	require.Equal(t, "https://auth.xyz.io", url)
}

func checkGetServiceEndpointOK(t *testing.T, expectedEndpoint string, getEndpoint func(req *http.Request) (string, error)) {
	url, err := getEndpoint(reqLong)
	assert.NoError(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, expectedEndpoint, url)

	url, err = getEndpoint(reqShort)
	assert.NoError(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, expectedEndpoint, url)
}

func TestGetMaxHeaderSizeUsingDefaults(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	viperValue := config.GetHeaderMaxLength()
	require.NotNil(t, viperValue)
	assert.Equal(t, int64(5000), viperValue)
}

func TestGetMaxHeaderSizeSetByEnvVaribaleOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	envName := "F8_HEADER_MAXLENGTH"
	envValue := time.Now().Unix()
	env := os.Getenv(envName)
	defer func() {
		os.Setenv(envName, env)
		resetConfiguration(defaultValuesConfigFilePath)
	}()

	os.Setenv(envName, strconv.FormatInt(envValue, 10))
	resetConfiguration(defaultValuesConfigFilePath)

	viperValue := config.GetHeaderMaxLength()
	require.NotNil(t, viperValue)
	assert.Equal(t, envValue, viperValue)
}

func generateEnvKey(yamlKey string) string {
	return "F8_" + strings.ToUpper(strings.Replace(yamlKey, ".", "_", -1))
}
