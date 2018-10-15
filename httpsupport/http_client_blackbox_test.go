package httpsupport_test

import (
	"fmt"
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

	t.Run("with one plus one query params", func(t *testing.T) {
		// when
		generatedURLString, err := httpsupport.AddParam("https://openshift.io", "param1", "a")
		// then
		require.NoError(t, err)
		assert.Equal(t, "https://openshift.io?param1=a", generatedURLString)
		// when adding an extra query param
		generatedURLString, err = httpsupport.AddParam(generatedURLString, "param2", "abc")
		// then
		require.NoError(t, err)
		equalURLs(t, &url.URL{
			Scheme:   "https",
			Host:     "openshift.io",
			Path:     "",
			RawQuery: "param1=a&param2=abc",
		}, generatedURLString)
	})

	t.Run("with three query params at once", func(t *testing.T) {
		// given
		scheme := "https"
		host := "openshift.io"
		path := "/api/status"
		queryParams := map[string]string{
			"param1": "a",
			"param2": "b",
			"param3": "https://www.redhat.com",
		}
		// when
		generatedURLString, err := httpsupport.AddParams(fmt.Sprintf("%s://%s%s", scheme, host, path), queryParams)
		// then verify that the generated URL can be parsed that all params were set as expected
		require.NoError(t, err)
		equalURLs(t, &url.URL{
			Scheme:   "https",
			Host:     host,
			Path:     path,
			RawQuery: "param1=a&param2=b&param3=https%3A%2F%2Fwww.redhat.com",
		}, generatedURLString)
	})

}

// Can't use test.EqualURLs() because of cycle dependency
func equalURLs(t *testing.T, expectedURL *url.URL, actual string) {
	require.Equal(t, expectedURL.String(), actual)
	actualURL, err := url.Parse(actual)
	require.Nil(t, err)
	assert.Equal(t, expectedURL.Scheme, actualURL.Scheme)
	assert.Equal(t, expectedURL.Host, actualURL.Host)
	assert.Equal(t, expectedURL.Path, actualURL.Path)
	assert.Equal(t, len(expectedURL.Query()), len(actualURL.Query()))
	for name, value := range expectedURL.Query() {
		assert.Equal(t, value, actualURL.Query()[name])
	}
}
