package httpsupport_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/resource"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHost(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	req := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}

	t.Run("prod mode", func(t *testing.T) {
		host := httpsupport.Host(req, &dummyConfig{})
		assert.Equal(t, "api.service.domain.org", host)
	})

	t.Run("nil config", func(t *testing.T) {
		host := httpsupport.Host(req, nil)
		assert.Equal(t, "api.service.domain.org", host)
	})

	t.Run("dev mode", func(t *testing.T) {
		host := httpsupport.Host(req, &dummyConfig{true})
		assert.Equal(t, "auth.openshift.io", host)
	})
}

func TestAddParams(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		// given
		testMap := map[string]string{
			"param1": "a",
			"param2": "b",
			"param3": "https://www.redhat.com",
		}
		testHost := "openshift.io"
		// when
		generatedURLString, err := httpsupport.AddParams("https://"+testHost, testMap)
		// then
		require.NoError(t, err)
		// verify that the generated URL can be parsed that all params were set as expected
		generateURL, err := url.Parse(generatedURLString)
		require.NoError(t, err)
		assert.Equal(t, testHost, generateURL.Host)
		assert.Equal(t, "https", generateURL.Scheme)
		m, err := url.ParseQuery(generateURL.RawQuery)
		require.NoError(t, err)
		for k, v := range testMap {
			assert.Equal(t, v, m[k][0])
		}
	})
}

func TestAddParamSuccess(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		generatedURL, err := httpsupport.AddParam("https://openshift.io", "param1", "a")
		require.NoError(t, err)
		assert.Equal(t, "https://openshift.io?param1=a", generatedURL)

		generatedURL, err = httpsupport.AddParam(generatedURL, "param2", "abc")
		require.NoError(t, err)
		equalURLs(t, "https://openshift.io?param1=a&param2=abc", generatedURL)
	})
}

// Can't use test.EqualURLs() because of cycle dependency
func equalURLs(t *testing.T, expected string, actual string) {
	expectedURL, err := url.Parse(expected)
	require.Nil(t, err)
	actualURL, err := url.Parse(actual)
	require.Nil(t, err)
	assert.Equal(t, expectedURL.Scheme, actualURL.Scheme)
	assert.Equal(t, expectedURL.Host, actualURL.Host)
	assert.Equal(t, len(expectedURL.Query()), len(actualURL.Query()))
	for name, value := range expectedURL.Query() {
		assert.Equal(t, value, actualURL.Query()[name])
	}
}
