package httpsupport_test

import (
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyConfig struct {
	devMode bool
}

func (c *dummyConfig) IsPostgresDeveloperModeEnabled() bool {
	return c.devMode
}

func TestAbsolute(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	req := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}

	t.Run("HTTP", func(t *testing.T) {
		urlStr := httpsupport.AbsoluteURL(req, "/testpath", nil)
		assert.Equal(t, "http://api.service.domain.org/testpath", urlStr)

	})

	t.Run("HTTPS", func(t *testing.T) {
		// HTTPS
		r, err := http.NewRequest("", "https://api.service.domain.org", nil)
		require.Nil(t, err)
		req = &goa.RequestData{
			Request: r,
		}
		urlStr := httpsupport.AbsoluteURL(req, "/testpath2", nil)
		assert.Equal(t, "https://api.service.domain.org/testpath2", urlStr)
	})

	t.Run("Prod mode", func(t *testing.T) {
		urlStr := httpsupport.AbsoluteURL(req, "/testpath2", &dummyConfig{devMode: false})
		assert.Equal(t, "https://api.service.domain.org/testpath2", urlStr)
	})

	t.Run("Dev mode", func(t *testing.T) {
		urlStr := httpsupport.AbsoluteURL(req, "/testpath2", &dummyConfig{devMode: true})
		assert.Equal(t, "https://auth.openshift.io/testpath2", urlStr)
	})

	t.Run("Proxy forward to HTTPS", func(t *testing.T) {
		// HTTPS
		r, err := http.NewRequest("", "http://api.service.domain.org", nil)
		require.Nil(t, err)
		r.Header.Set("X-Forwarded-Proto", "https")
		req := &goa.RequestData{
			Request: r,
		}
		urlStr := httpsupport.AbsoluteURL(req, "/testpath2", nil)
		assert.Equal(t, "https://api.service.domain.org/testpath2", urlStr)
	})

}

func TestReplaceDomainPrefix(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	t.Run("non-empty replaceby", func(t *testing.T) {
		host, err := httpsupport.ReplaceDomainPrefix("api.service.domain.org", "sso")
		require.NoError(t, err)
		assert.Equal(t, "sso.service.domain.org", host)
	})

	t.Run("empty replaceby", func(t *testing.T) {
		host, err := httpsupport.ReplaceDomainPrefix("api.service.domain.org", "")
		require.NoError(t, err)
		assert.Equal(t, "service.domain.org", host)
	})

	t.Run("fail - too short", func(t *testing.T) {
		_, err := httpsupport.ReplaceDomainPrefix("org", "sso")
		assert.Error(t, err)
	})
}

func TestReplaceDomainPrefixInAbsoluteURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	req := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}

	t.Run("HTTP", func(t *testing.T) {
		urlStr, err := httpsupport.ReplaceDomainPrefixInAbsoluteURL(req, "auth", "/testpath", nil)
		require.NoError(t, err)
		assert.Equal(t, "http://auth.service.domain.org/testpath", urlStr)
	})

	t.Run("HTTPS", func(t *testing.T) {
		r, err := http.NewRequest("", "https://api.service.domain.org", nil)
		require.Nil(t, err)
		req = &goa.RequestData{
			Request: r,
		}
		urlStr, err := httpsupport.ReplaceDomainPrefixInAbsoluteURL(req, "auth", "/testpath2", nil)
		require.NoError(t, err)
		assert.Equal(t, "https://auth.service.domain.org/testpath2", urlStr)
	})

	t.Run("without replaceby", func(t *testing.T) {
		urlStr, err := httpsupport.ReplaceDomainPrefixInAbsoluteURL(req, "", "/testpath3", nil)
		require.NoError(t, err)
		assert.Equal(t, "https://service.domain.org/testpath3", urlStr)
	})

	t.Run("prod mode", func(t *testing.T) {
		urlStr, err := httpsupport.ReplaceDomainPrefixInAbsoluteURL(req, "", "/testpath4", &dummyConfig{devMode: false})
		require.NoError(t, err)
		assert.Equal(t, "https://service.domain.org/testpath4", urlStr)
	})

	t.Run("dev mode", func(t *testing.T) {
		t.Run("without replaceby", func(t *testing.T) {
			urlStr, err := httpsupport.ReplaceDomainPrefixInAbsoluteURL(req, "", "/testpath5", &dummyConfig{devMode: true})
			require.NoError(t, err)
			assert.Equal(t, "https://openshift.io/testpath5", urlStr)
		})
		t.Run("with replaceby", func(t *testing.T) {
			urlStr, err := httpsupport.ReplaceDomainPrefixInAbsoluteURL(req, "core", "/testpath6", &dummyConfig{devMode: true})
			require.NoError(t, err)
			assert.Equal(t, "https://core.openshift.io/testpath6", urlStr)
		})
	})
}

func TestAddTrailingSlashToURLSuccess(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	t.Run("add slash", func(t *testing.T) {
		assert.Equal(t, "https://openshift.io/", httpsupport.AddTrailingSlashToURL("https://openshift.io"))
	})
	t.Run("slash already exists", func(t *testing.T) {
		assert.Equal(t, "https://openshift.io/", httpsupport.AddTrailingSlashToURL("https://openshift.io/"))
	})
	t.Run("empty URL", func(t *testing.T) {
		assert.Equal(t, "", httpsupport.AddTrailingSlashToURL(""))
	})
}
