package token_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-common/resource"
	testsupport "github.com/fabric8-services/fabric8-common/test"
	testconfiguration "github.com/fabric8-services/fabric8-common/test/configuration"
	"github.com/fabric8-services/fabric8-common/token"

	"github.com/dgrijalva/jwt-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceAccount(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	serviceName := "test-service"

	t.Run("Valid", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = serviceName
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.True(t, token.IsServiceAccount(ctx))
	})
	t.Run("Missing name", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsServiceAccount(ctx))
	})
	t.Run("Missing token", func(t *testing.T) {
		// given
		ctx := context.Background()
		// then
		assert.False(t, token.IsServiceAccount(ctx))
	})
	t.Run("Nil token", func(t *testing.T) {
		// given
		ctx := goajwt.WithJWT(context.Background(), nil)
		// then
		assert.False(t, token.IsServiceAccount(ctx))
	})
	t.Run("Wrong data type", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = 100
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsServiceAccount(ctx))
	})
}

func TestSpecificServiceAccount(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	serviceName := "test-service"
	t.Run("Valid", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = serviceName
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.True(t, token.IsSpecificServiceAccount(ctx, "dummy-service", serviceName))
	})
	t.Run("Missing name", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
	t.Run("Nil token", func(t *testing.T) {
		// given
		ctx := goajwt.WithJWT(context.Background(), nil)
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
	t.Run("Wrong data type", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = 100
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
	t.Run("Missing token", func(t *testing.T) {
		// given
		ctx := context.Background()
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
	t.Run("Wrong name", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = serviceName + "_asdsa"
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
}

func createInvalidSAContext() context.Context {
	claims := jwt.MapClaims{}
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	return goajwt.WithJWT(context.Background(), token)
}

func TestAddLoginRequiredHeader(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	config := testconfiguration.NewDefaultMockTokenManagerConfiguration(t)
	tm, err := token.NewManager(config)
	require.NoError(t, err)
	t.Run("without header set", func(t *testing.T) {
		// given
		rw := httptest.NewRecorder()
		// when
		tm.AddLoginRequiredHeader(rw)
		// then
		checkLoginRequiredHeader(t, rw)
	})
	t.Run("with header set", func(t *testing.T) {
		// given
		rw := httptest.NewRecorder()
		rw.Header().Set("Access-Control-Expose-Headers", "somecustomvalue")
		// when
		tm.AddLoginRequiredHeader(rw)
		// then
		checkLoginRequiredHeader(t, rw)
	})
}

func checkLoginRequiredHeader(t *testing.T, rw http.ResponseWriter) {
	assert.Equal(t, "LOGIN url=https://auth.prod-preview.openshift.io/api/login, description=\"re-login is required\"", rw.Header().Get("WWW-Authenticate"))
	header := textproto.MIMEHeader(rw.Header())
	assert.Contains(t, header["Access-Control-Expose-Headers"], "WWW-Authenticate")
}

func assertHeaders(t *testing.T, tm token.Manager, tokenString string) {
	jwtToken, err := tm.Parse(context.Background(), tokenString)
	assert.NoError(t, err)
	assert.Equal(t, "aUGv8mQA85jg4V1DU8Uk1W0uKsxn187KQONAGl6AMtc", jwtToken.Header["kid"])
	assert.Equal(t, "RS256", jwtToken.Header["alg"])
	assert.Equal(t, "JWT", jwtToken.Header["typ"])
}

func TestParseValidTokenOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	config := testconfiguration.NewDefaultMockTokenManagerConfiguration(t)
	tm, err := token.NewManager(config)
	require.NoError(t, err)
	identity := testsupport.NewIdentity()
	generatedToken, _, err := testsupport.GenerateSignedUserToken(identity, config)
	require.NoError(t, err)

	t.Run("parse token", func(t *testing.T) {
		// when
		claims, err := tm.ParseToken(context.Background(), generatedToken)
		// then
		require.Nil(t, err)
		assert.Equal(t, identity.ID.String(), claims.Subject)
		assert.Equal(t, identity.Username, claims.Username)
	})

	t.Run("parse", func(t *testing.T) {
		// when
		jwtToken, err := tm.Parse(context.Background(), generatedToken)
		// then
		require.Nil(t, err)
		checkClaim(t, jwtToken, "sub", identity.ID.String())
		checkClaim(t, jwtToken, "preferred_username", identity.Username)
	})
}

func checkClaim(t *testing.T, token *jwt.Token, claimName string, expectedValue string) {
	jwtClaims := token.Claims.(jwt.MapClaims)
	claim, ok := jwtClaims[claimName]
	require.True(t, ok)
	assert.Equal(t, expectedValue, claim)
}

func TestParseToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	config := testconfiguration.NewDefaultMockTokenManagerConfiguration(t)
	tm, err := token.NewManager(config)
	require.NoError(t, err)

	t.Run("Invalid token format", func(t *testing.T) {
		checkInvalidToken(t, tm, "7423742yuuiy-INVALID-73842342389h", "token contains an invalid number of segments")
	})

	t.Run("Missing kid", func(t *testing.T) {
		checkInvalidToken(t, tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.mx6DrW3kUQEvsz5Bmaea3nXD9_DB5fBDyNIfanr3u_RFWQyV0romzrAjBP3dKz_dgTS4S5WX2Q1XZiPLjc13PMTCQTNUXp6bFJ5RlDEqX6rJP0Ps9X7bke1pcqS7RhV9cnR1KNH8428bYoKCV57eQnhWtQoCQC1Db6YWJoQNJJLt0IHKCOx7c06r01VF1zcIk1dHnzzz9Qv5aACGXAi8iEJsQ1vURSh7fMETfSJl0UrLJsxGo60fHX9p74cu7bcgD-Zj86axRfgbaHuxn1MMJblltcPsG_TnsMOtmqQr4wlszWTQzwbLnemn8XfwPU8XYc49rVnkiZoB9-BV-oYIxg", "There is no 'kid' header in the token")
	})

	t.Run("Unknown kid", func(t *testing.T) {
		checkInvalidToken(t, tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InVua25vd25raWQifQ.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.ewwDOye1c5mA91YjgMQvk0x4RCXeosKVgag-UeG5lHQASWPv3_cpw0BMahG6-eJw1FTvnMFHlcH1Smw-Nfh43qKUpdgWL7QpVbLHiOgvfcsk1g3-G0QFBZ-N-xh35L53Cp5_seXionSAGjsNWLqStQaHPXJ9jLAc_JYLds1VO2CJ0tPSyJ0kiC8fiyRP17pJ19hiHonnGUAZlfZGPJZhrBCfAx3NBbejE0ZAUoeIAw1wPQJDCfp93vO5jvn0kUubpHlnAFz0YtLKqUfaiw6PfZDpu_HTpxAMVvyY_4PxzP56lWdnqQh6JhiMuVNehJfnKcAKPu4WboNCVVIBW3Gppg", "There is no public key with such ID: unknownkid")
	})

	t.Run("Invalid signature", func(t *testing.T) {
		checkInvalidToken(t, tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC5jb20ifQ.Y66kbPnxdfWyuawsEABDTsPFAylC9UFj934pF7SG-Fwrfqs3iF0gVHAQ56WLwY7E-D4QX_3uUkYuSrjzd4JT1p0bfxt3uu0wzFQlnzB4Uu2ttS287XPkBss4mUlc5uvAj0FRdy1IrQBFnfFpW5s6PWrHqod9PF4R2BTCBO1JqKgRtGzSqFwuWHowW__Sgw3B2NVgplL-6rb762M1OeT0GFWt0QE_uG8k_LPGPTyxDR5AILGfRgz5p-d16SYCAsjbsGSiQh3OGArt3Gzfi3CsKIGsQnhfuVXiorFbUn-nVaDuxRwU7JDzhde5nAj38U7exrkgxhEkybGMe4xZme49vA", "crypto/rsa: verification error")
	})

	t.Run("Expired", func(t *testing.T) {
		checkInvalidToken(t, tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjExMTk2MDc5NTIsIm5iZiI6MCwiaWF0IjoxMTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.g90TCdckT5YFctQSwQke7jmeGDZQotYiCa8AE_x0o8M8ncgb6m07glGXgcGJXftkzUL-uZn1U9JzixOYaI8B__jtB9BbMqMnrXyz-_gTYHAlj06l-9axVyKV7cpO8IIt_cFVt5lv4pPEcjEMzDLbjxxo6qH9lihry_KL3zESt8hxaosSnY5b8XvN7WCL-5NYTDF_i7QBI5x8XBljQpTJSwLY6-X7TDgAThET8OgWDV3H40UsSSsJUfpdEJZuiDsqoCsEpb0E7AfiYD-y0iZ5ULSxTiNf0EYf26irmy-jyQlWujOSb9kV2utsywZn-zDmHX3W_hS2wRD5eVgePFTBKA", "token is expired")
	})

	t.Run("OK", func(t *testing.T) {
		checkValidToken(t, tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.gyoMIWuXnIMMRHewef-__Wkd66qjqSSJxusWcFVtNWaYOXWu7iFV9DhtPVGsbTllXG_lDozPV9BaDmmYRotnn3ZBg7khFDykv9WnoYAjE9vW1d8szNjuoG3tfgQI4Dr9jqopSLndldxq97LGqpxqZFbIDlYd8vN47kv4EePOZDsII6egkTraCMc35eMMilJ4Udd6CMqyV_zaYiGhgAGgeL2ovMFhg_jnc7WhePv7FZkUmtfhCuLUL2TSXS6CyWZYoUDEIcfca6cMzuKOzJoONkDJShNo4u_cQ53duXX_bizdwYNlzBHfIPhSR1LDgV9BXoM6YQnw3It8ReCfF8BEMQ")
	})

}

func checkInvalidToken(t *testing.T, tm token.Manager, token, expectedError string) {
	_, err := tm.ParseToken(context.Background(), token)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(expectedError))
	_, err = tm.ParseTokenWithMapClaims(context.Background(), token)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(expectedError))
	_, err = tm.Parse(context.Background(), token)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(expectedError))
}

func checkValidToken(t *testing.T, tm token.Manager, token string) {
	_, err := tm.ParseToken(context.Background(), token)
	assert.NoError(t, err)
	_, err = tm.ParseTokenWithMapClaims(context.Background(), token)
	assert.NoError(t, err)
	_, err = tm.Parse(context.Background(), token)
	assert.NoError(t, err)
}

func TestCheckClaimsOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	claims := &token.TokenClaims{
		Email:    "somemail@domain.com",
		Username: "testuser",
	}
	claims.Subject = uuid.NewV4().String()
	// when
	err := token.CheckClaims(claims)
	// then
	assert.NoError(t, err)
}

func TestCheckClaimsFails(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	t.Run("no email", func(t *testing.T) {
		// given
		claimsNoEmail := &token.TokenClaims{
			Username: "testuser",
		}
		claimsNoEmail.Subject = uuid.NewV4().String()
		// then
		assert.NotNil(t, token.CheckClaims(claimsNoEmail))
	})

	t.Run("no username", func(t *testing.T) {
		// given
		claimsNoUsername := &token.TokenClaims{
			Email: "somemail@domain.com",
		}
		claimsNoUsername.Subject = uuid.NewV4().String()
		// then
		assert.NotNil(t, token.CheckClaims(claimsNoUsername))
	})

	t.Run("no subject", func(t *testing.T) {
		// given
		claimsNoSubject := &token.TokenClaims{
			Email:    "somemail@domain.com",
			Username: "testuser",
		}
		// then
		assert.NotNil(t, token.CheckClaims(claimsNoSubject))
	})
}

func TestLocateTokenInContext(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	// given
	config := testconfiguration.NewDefaultMockTokenManagerConfiguration(t)
	tm, err := token.NewManager(config)
	require.NoError(t, err)

	t.Run("ok", func(t *testing.T) {
		// given
		id := uuid.NewV4()
		tk := jwt.New(jwt.SigningMethodRS256)
		tk.Claims.(jwt.MapClaims)["sub"] = id.String()
		ctx := goajwt.WithJWT(context.Background(), tk)
		// when
		foundId, err := tm.Locate(ctx)
		// then
		require.NoError(t, err)
		assert.Equal(t, id, foundId, "ID in created context not equal")
	})

	t.Run("missing token in context", func(t *testing.T) {
		// given
		ctx := context.Background()
		// when
		_, err = tm.Locate(ctx)
		// then
		require.Error(t, err)
	})

	t.Run("missing UUID in token", func(t *testing.T) {
		// given
		tk := jwt.New(jwt.SigningMethodRS256)
		ctx := goajwt.WithJWT(context.Background(), tk)
		// when
		_, err = tm.Locate(ctx)
		// then
		require.Error(t, err)
	})

	t.Run("invalid UUID in token", func(t *testing.T) {
		// given
		tk := jwt.New(jwt.SigningMethodRS256)
		tk.Claims.(jwt.MapClaims)["sub"] = "131"
		ctx := goajwt.WithJWT(context.Background(), tk)
		// when
		_, err := tm.Locate(ctx)
		// then
		require.Error(t, err)

	})
}
