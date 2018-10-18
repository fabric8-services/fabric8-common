package sentry

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/fabric8-services/fabric8-common/resource"
	testtoken "github.com/fabric8-services/fabric8-common/test/token"
	testtokenconfig "github.com/fabric8-services/fabric8-common/test/token/configuration"
	"github.com/fabric8-services/fabric8-common/token"
	"github.com/fabric8-services/fabric8-common/token/tokencontext"

	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/raven-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTokenManager(tm token.Manager) context.Context {
	// this is just normal context object with no, token
	// so this should fail saying no token available
	return tokencontext.ContextWithTokenManager(context.Background(), tm)
}

func withIncompleteToken(tm token.Manager) context.Context {
	ctx := withTokenManager(tm)
	// Here we add a token which is incomplete
	token := jwt.New(jwt.GetSigningMethod("RS256"))
	return goajwt.WithJWT(ctx, token)
}

func withValidToken(t *testing.T, tm token.Manager, identityID string, identityUsername string) context.Context {
	ctx := withTokenManager(tm)
	// Here we add a token that is perfectly valid
	token, err := testtoken.GenerateTokenObject(identityID, identityUsername, testtoken.PrivateKey())
	require.NoErrorf(t, err, "could not generate token: %v", errors.WithStack(err))
	return goajwt.WithJWT(ctx, token)
}

func TestExtractUserInfo(t *testing.T) {
	config := testtokenconfig.NewManagerConfigurationMock(t)
	tm := testtoken.NewManager(config)
	resource.Require(t, resource.UnitTest)
	close, err := InitializeSentryClient(nil,
		WithUser(func(ctx context.Context) (*raven.User, error) {
			m, err := token.ReadManagerFromContext(ctx)
			if err != nil {
				return nil, err
			}
			q := *m
			token := goajwt.ContextJWT(ctx)
			if token == nil {
				return nil, fmt.Errorf("no token found in context")
			}
			t, err := q.ParseToken(ctx, token.Raw)
			if err != nil {
				return nil, err
			}

			return &raven.User{
				Username: t.Username,
				Email:    t.Email,
				ID:       t.Subject,
			}, nil
		}))
	require.NoError(t, err)
	defer close()

	t.Run("random context", func(t *testing.T) {
		// when
		userInfo, err := Sentry().userInfo(context.Background())
		// then
		require.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("missing token", func(t *testing.T) {
		// when
		userInfo, err := Sentry().userInfo(withTokenManager(tm))
		// then
		require.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("incomplete token", func(t *testing.T) {
		// when
		userInfo, err := Sentry().userInfo(withIncompleteToken(tm))
		// then
		require.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("valid token", func(t *testing.T) {
		// when
		userID := uuid.NewV4()
		username := "testuser"
		userInfo, err := Sentry().userInfo(withValidToken(t, tm, userID.String(), username))
		// then
		require.NoError(t, err)
		require.NotNil(t, userInfo)
		assert.Equal(t, raven.User{
			Username: username,
			ID:       userID.String(),
			Email:    username + "@email.com",
		}, *userInfo)
	})

}

func TestDSN(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// Set default DSN via env var
	defaultProject := uuid.NewV4()
	dsn := fmt.Sprintf("https://%s:%s@test.io/%s", uuid.NewV4(), uuid.NewV4(), defaultProject)
	old := os.Getenv("SENTRY_DSN")
	os.Setenv("SENTRY_DSN", dsn)
	defer os.Setenv("SENTRY_DSN", old)

	// Init DSN explicitly
	project := uuid.NewV4()
	dsn = fmt.Sprintf("https://%s:%s@test.io/%s", uuid.NewV4(), uuid.NewV4(), project)
	_, err := InitializeSentryClient(&dsn)
	require.NoError(t, err)

	// The env var is not used. Explicitly set DSN is used instead.
	assert.Equal(t, fmt.Sprintf("https://test.io/api/%s/store/", project), Sentry().c.URL())

	// Init the default DSN
	_, err = InitializeSentryClient(nil)
	require.NoError(t, err)

	// The DSN from the env var is used
	assert.Equal(t, fmt.Sprintf("https://test.io/api/%s/store/", defaultProject), Sentry().c.URL())
}
