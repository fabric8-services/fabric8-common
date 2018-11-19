package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-auth-client/auth"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/service"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	gock "gopkg.in/h2non/gock.v1"
)

var url = "https://auth.prod-preview.openshift.io"

type AuthServiceTestSuite struct {
	testsuite.UnitTestSuite
	authService service.Auth
}

func TestAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, &AuthServiceTestSuite{UnitTestSuite: testsuite.NewUnitTestSuite()})
}

func (s *AuthServiceTestSuite) SetupSuite() {
	s.UnitTestSuite.SetupSuite()

	var err error
	s.authService, err = service.NewAuthService(url)
	require.NoError(s.T(), err)
}

func (s *AuthServiceTestSuite) TestCheckResourceScope() {
	s.T().Run("scope_found_ok", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(auth.ScopesResourcePath(resID.String())).
			Reply(200).
			BodyString(`{"data":[{"id":"view","type":"user_resource_scope"},{"id":"contribute","type":"user_resource_scope"},{"id":"manage","type":"user_resource_scope"}]}`)

		err := s.authService.RequireScope(context.Background(), resID.String(), "manage")
		assert.NoError(t, err)
	})

	s.T().Run("scope_forbidden", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(auth.ScopesResourcePath(resID.String())).
			Reply(200).
			BodyString(`{"data":[{"id":"view","type":"user_resource_scope"},{"id":"contribute","type":"user_resource_scope"}]}`)

		err := s.authService.RequireScope(context.Background(), resID.String(), "manage")
		assert.Error(t, err)
		_, ok := err.(errors.ForbiddenError)
		assert.True(t, ok, "error is not forbidden error")
	})

	s.T().Run("error_unauthorized", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(auth.ScopesResourcePath(resID.String())).
			Reply(401)

		err := s.authService.RequireScope(context.Background(), resID.String(), "manage")
		assert.Error(t, err)
	})

	s.T().Run("error_internal_server", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(auth.ScopesResourcePath(resID.String())).
			Reply(500)

		err := s.authService.RequireScope(context.Background(), resID.String(), "manage")
		assert.Error(t, err)
	})

	s.T().Run("error_not_found", func(t *testing.T) {
		resID := uuid.NewV4()
		gock.New(url).
			Get(auth.ScopesResourcePath(resID.String())).
			Reply(404)

		err := s.authService.RequireScope(context.Background(), resID.String(), "manage")
		assert.Error(t, err)
	})
}
