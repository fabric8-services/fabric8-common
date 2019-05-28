package auth_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/fabric8-services/fabric8-common/test/auth"

	"github.com/dgrijalva/jwt-go"
	jwtgoa "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	username := "foo"
	identityID := "bcdc2643-d810-4d22-9ef8-7022165edd7f"

	t.Run("default email", func(t *testing.T) {
		// when
		ctx, token, err := auth.EmbedTokenInContext(identityID, username)
		// then
		require.NoError(t, err)
		// verify that the token is in the context
		tk := jwtgoa.ContextJWT(ctx)
		require.NotNil(t, tk)
		assert.Equal(t, token, tk.Raw)
		// check the token claims
		claims := tk.Claims.(jwt.MapClaims)
		assert.Equal(t, identityID, claims["uuid"])
		assert.Equal(t, identityID, claims["sub"])
		assert.Equal(t, username, claims["preferred_username"])
		assert.Equal(t, username+"@email.com", claims["email"])
	})

	t.Run("custom email", func(t *testing.T) {
		// when
		ctx, token, err := auth.EmbedTokenInContext(identityID, username, auth.WithEmailClaim("foo@bar.com"))
		// then
		require.NoError(t, err)
		// verify that the token is in the context
		tk := jwtgoa.ContextJWT(ctx)
		require.NotNil(t, tk)
		assert.Equal(t, token, tk.Raw)
		// check the token claims
		claims := tk.Claims.(jwt.MapClaims)
		assert.Equal(t, identityID, claims["uuid"])
		assert.Equal(t, identityID, claims["sub"])
		assert.Equal(t, username, claims["preferred_username"])
		assert.Equal(t, "foo@bar.com", claims["email"])
	})

}
