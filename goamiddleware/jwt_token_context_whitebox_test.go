package goamiddleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	testsupport "github.com/fabric8-services/fabric8-common/test"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"
	testtoken "github.com/fabric8-services/fabric8-common/test/token"
	testtokenconfig "github.com/fabric8-services/fabric8-common/test/token/configuration"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestJWTokenContext(t *testing.T) {
	suite.Run(t, &TestJWTokenContextSuite{})
}

type TestJWTokenContextSuite struct {
	testsuite.UnitTestSuite
}

func (s *TestJWTokenContextSuite) TestHandler() {
	// given
	rq := &http.Request{Header: make(map[string][]string)}
	schema := &goa.JWTSecurity{}
	errUnauthorized := goa.NewErrorClass("token_validation_failed", 401)
	config := testtokenconfig.NewManagerConfigurationMock(s.T())
	config.GetAuthServiceURLFunc = func() string {
		return "https://auth-test"
	}
	tm := testtoken.NewManager(config)
	h := handler(tm, schema, dummyHandler, errUnauthorized)

	s.T().Run("Error if missing scheme location", func(t *testing.T) {
		// when
		rw := httptest.NewRecorder()
		err := h(context.Background(), rw, rq)
		// then
		require.Error(t, err)
		assert.Equal(t, "whoops, security scheme with location (in) \"\" not supported", err.Error())
	})

	s.T().Run("OK if no Authorization header", func(t *testing.T) {
		// given
		rw := httptest.NewRecorder()
		schema.In = "header"
		// when
		err := h(context.Background(), rw, rq)
		// then
		require.Error(t, err)
		assert.Equal(t, "next-handler-error", err.Error())
	})

	s.T().Run("OK if not bearer", func(t *testing.T) {
		// given
		rw := httptest.NewRecorder()
		schema.Name = "Authorization"
		rq.Header.Set("Authorization", "something")
		// when
		err := h(context.Background(), rw, rq)
		// then
		require.Error(t, err)
		assert.Equal(t, "next-handler-error", err.Error())

	})

	s.T().Run("401 if token is invalid", func(t *testing.T) {
		// given
		rw := httptest.NewRecorder()
		schema.In = "header"
		schema.Name = "Authorization"
		rq.Header.Set("Authorization", "bearer foobartoken")
		err := h(context.Background(), rw, rq)
		// when
		require.Error(t, err)
		// then
		assert.Contains(t, err.Error(), "401 token_validation_failed: token is invalid", err.Error())
		assert.Equal(t, "LOGIN url=https://auth-test/api/login, description=\"re-login is required\"", rw.Header().Get("WWW-Authenticate"))
		assert.Contains(t, rw.Header().Get("Access-Control-Expose-Headers"), "WWW-Authenticate")
	})

	s.T().Run("OK if token is valid", func(t *testing.T) {
		// given
		rw := httptest.NewRecorder()
		tk, _, err := testsupport.GenerateSignedServiceAccountToken(&testsupport.Identity{Username: "sa-name"}, config)
		require.NoError(t, err)
		schema.In = "header"
		schema.Name = "Authorization"
		rq.Header.Set("Authorization", fmt.Sprintf("bearer %s", tk))
		// when
		err = h(context.Background(), rw, rq)
		// then
		require.Error(t, err)
		assert.Equal(t, "next-handler-error", err.Error())
		header := textproto.MIMEHeader(rw.Header())
		assert.NotContains(t, header, "WWW-Authenticate")
		assert.NotContains(t, header, "Access-Control-Expose-Headers")
	})
}

func dummyHandler(ctx context.Context, rw http.ResponseWriter, r *http.Request) error {
	return errors.New("next-handler-error")
}
