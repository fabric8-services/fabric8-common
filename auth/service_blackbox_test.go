package auth_test

import (
	"testing"

	authclient "github.com/fabric8-services/fabric8-auth-client/auth"
	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/errors"
	testsupport "github.com/fabric8-services/fabric8-common/test"
	testauth "github.com/fabric8-services/fabric8-common/test/auth"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	gock "gopkg.in/h2non/gock.v1"
)

var url = "https://auth.prod-preview.openshift.io"

type AuthServiceTestSuite struct {
	testsuite.UnitTestSuite
	authService auth.AuthService
}

func TestAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, &AuthServiceTestSuite{UnitTestSuite: testsuite.NewUnitTestSuite()})
}

func (s *AuthServiceTestSuite) SetupSuite() {
	s.UnitTestSuite.SetupSuite()

	var err error
	s.authService, err = auth.NewAuthService(url)
	require.NoError(s.T(), err)
}

func (s *AuthServiceTestSuite) TearDownSuite() {
	gock.Disable()
}

func (s *AuthServiceTestSuite) TestCheckResourceScope() {
	ctx, _, token, requestID, err := testauth.ContextWithTokenAndRequestID()
	require.NoError(s.T(), err)

	s.T().Run("scope_found_ok", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(authclient.ScopesResourcePath(resID.String())).
			MatchHeader("Authorization", "Bearer "+token).
			MatchHeader("X-Request-Id", requestID).
			Reply(200).
			BodyString(`{"data":[{"id":"view","type":"user_resource_scope"},{"id":"contribute","type":"user_resource_scope"},{"id":"manage","type":"user_resource_scope"}]}`)

		err := s.authService.RequireScope(ctx, resID.String(), "manage")
		assert.NoError(t, err)
	})

	s.T().Run("scope_forbidden", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(authclient.ScopesResourcePath(resID.String())).
			MatchHeader("Authorization", "Bearer "+token).
			MatchHeader("X-Request-Id", requestID).
			Reply(200).
			BodyString(`{"data":[{"id":"view","type":"user_resource_scope"},{"id":"contribute","type":"user_resource_scope"}]}`)

		err := s.authService.RequireScope(ctx, resID.String(), "manage")
		testsupport.AssertError(t, err, errors.ForbiddenError{}, "missing required scope 'manage' on '%s' resource", resID)
	})

	s.T().Run("error_unauthorized", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(authclient.ScopesResourcePath(resID.String())).
			MatchHeader("Authorization", "Bearer "+token).
			MatchHeader("X-Request-Id", requestID).
			Reply(401)

		err := s.authService.RequireScope(ctx, resID.String(), "manage")
		testsupport.AssertError(t, err, errors.InternalError{}, "get resource's scope failed with error '401 Unauthorized'")
	})

	s.T().Run("error_internal_server", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(authclient.ScopesResourcePath(resID.String())).
			MatchHeader("Authorization", "Bearer "+token).
			MatchHeader("X-Request-Id", requestID).
			Reply(500)

		err := s.authService.RequireScope(ctx, resID.String(), "manage")
		testsupport.AssertError(t, err, errors.InternalError{}, "get resource's scope failed with error '500 Internal Server Error'")
	})

	s.T().Run("error_not_found", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(authclient.ScopesResourcePath(resID.String())).
			MatchHeader("Authorization", "Bearer "+token).
			MatchHeader("X-Request-Id", requestID).
			Reply(404)

		err := s.authService.RequireScope(ctx, resID.String(), "manage")
		testsupport.AssertError(t, err, errors.InternalError{}, "get resource's scope failed with error '404 Not Found'")
	})
}
