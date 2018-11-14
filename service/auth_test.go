package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-auth-client/auth"
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

func (s *AuthServiceTestSuite) TestCheckSpaceScope() {
	s.T().Run("scope_found_true", func(t *testing.T) {
		spaceID := uuid.NewV4()
		gock.New(url).
			Get(auth.ScopesResourcePath(spaceID.String())).
			Reply(200).
			BodyString(`{"data":[{"id":"view","type":"user_resource_scope"},{"id":"contribute","type":"user_resource_scope"},{"id":"manage","type":"user_resource_scope"}]}`)

		authZ, err := s.authService.CheckSpaceScope(context.Background(), spaceID.String(), "manage")
		assert.NoError(t, err)
		assert.True(t, authZ)
	})

	s.T().Run("scope_found_false", func(t *testing.T) {
		spaceID := uuid.NewV4()
		gock.New(url).
			Get(auth.ScopesResourcePath(spaceID.String())).
			Reply(200).
			BodyString(`{"data":[{"id":"view","type":"user_resource_scope"},{"id":"contribute","type":"user_resource_scope"}]}`)

		authZ, err := s.authService.CheckSpaceScope(context.Background(), spaceID.String(), "manage")
		assert.NoError(t, err)
		assert.False(t, authZ)
	})

	s.T().Run("call_unauthorized", func(t *testing.T) {
		spaceID := uuid.NewV4()
		gock.New(url).
			Get(auth.ScopesResourcePath(spaceID.String())).
			Reply(401)

		authZ, err := s.authService.CheckSpaceScope(context.Background(), spaceID.String(), "manage")
		assert.Error(t, err)
		assert.False(t, authZ)
	})
}
