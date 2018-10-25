package goamiddleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/fabric8-services/fabric8-common/test/auth"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"

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
	schema := &goa.JWTSecurity{}
	errUnauthorized := goa.NewErrorClass("token_validation_failed", 401)

	rw := httptest.NewRecorder()
	rq := &http.Request{Header: make(map[string][]string)}
	h := handler(auth.TokenManager, schema, dummyHandler, errUnauthorized)

	err := h(context.Background(), rw, rq)
	require.Error(s.T(), err)
	assert.Equal(s.T(), "security scheme with location (in) \"\" not supported", err.Error())

	// OK if no Authorization header
	schema.In = "header"
	err = h(context.Background(), rw, rq)
	require.Error(s.T(), err)
	assert.Equal(s.T(), "next-handler-error", err.Error())

	// OK if not bearer
	schema.Name = "Authorization"
	rq.Header.Set("Authorization", "something")
	err = h(context.Background(), rw, rq)
	require.Error(s.T(), err)
	assert.Equal(s.T(), "next-handler-error", err.Error())

	// Get 401 if token is invalid
	rq.Header.Set("Authorization", "bearer token")
	err = h(context.Background(), rw, rq)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "401 token_validation_failed: token is invalid", err.Error())
	assert.Equal(s.T(), "LOGIN url=https://auth.openshift.io/api/login, description=\"re-login is required\"", rw.Header().Get("WWW-Authenticate"))
	assert.Contains(s.T(), rw.Header().Get("Access-Control-Expose-Headers"), "WWW-Authenticate")

	// OK if token is valid
	rw = httptest.NewRecorder()
	t := auth.GenerateServiceAccountToken("sa-name")
	rq.Header.Set("Authorization", "bearer "+t)
	err = h(context.Background(), rw, rq)
	require.Error(s.T(), err)
	assert.Equal(s.T(), "next-handler-error", err.Error())
	header := textproto.MIMEHeader(rw.Header())
	assert.NotContains(s.T(), header, "WWW-Authenticate")
	assert.NotContains(s.T(), header, "Access-Control-Expose-Headers")
}

func dummyHandler(ctx context.Context, rw http.ResponseWriter, r *http.Request) error {
	return errors.New("next-handler-error")
}
