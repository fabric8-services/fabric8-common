package token_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-common/test/auth"
	testconfiguration "github.com/fabric8-services/fabric8-common/test/configuration"
	tokensupport "github.com/fabric8-services/fabric8-common/test/generated/token"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"
	"github.com/fabric8-services/fabric8-common/token"
	"github.com/stretchr/testify/suite"

	"github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/test/recorder"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TokenManagerTestSuite struct {
	testsuite.UnitTestSuite
	tm     token.Manager
	config *tokensupport.ManagerConfigurationMock
}

func TestRunTokenManagerTestSuite(t *testing.T) {
	suite.Run(t, &TokenManagerTestSuite{UnitTestSuite: testsuite.NewUnitTestSuite()})
}

func (s *TokenManagerTestSuite) SetupSuite() {
	s.UnitTestSuite.SetupSuite()

	s.config = testconfiguration.NewDefaultMockTokenManagerConfiguration(s.T())
	var err error
	s.tm, err = token.NewManager(s.config)
	require.NoError(s.T(), err)
}

func (s *TokenManagerTestSuite) TestServiceAccount() {
	serviceName := "test-service"

	s.T().Run("Valid", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = serviceName
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.True(t, token.IsServiceAccount(ctx))
	})
	s.T().Run("Missing name", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsServiceAccount(ctx))
	})
	s.T().Run("Missing token", func(t *testing.T) {
		// given
		ctx := context.Background()
		// then
		assert.False(t, token.IsServiceAccount(ctx))
	})
	s.T().Run("Nil token", func(t *testing.T) {
		// given
		ctx := goajwt.WithJWT(context.Background(), nil)
		// then
		assert.False(t, token.IsServiceAccount(ctx))
	})
	s.T().Run("Wrong data type", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = 100
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsServiceAccount(ctx))
	})
}

func (s *TokenManagerTestSuite) TestSpecificServiceAccount() {
	serviceName := "test-service"
	s.T().Run("Valid", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = serviceName
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.True(t, token.IsSpecificServiceAccount(ctx, "dummy-service", serviceName))
	})
	s.T().Run("Missing name", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
	s.T().Run("Nil token", func(t *testing.T) {
		// given
		ctx := goajwt.WithJWT(context.Background(), nil)
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
	s.T().Run("Wrong data type", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = 100
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
	s.T().Run("Missing token", func(t *testing.T) {
		// given
		ctx := context.Background()
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
	s.T().Run("Wrong name", func(t *testing.T) {
		// given
		claims := jwt.MapClaims{}
		claims["service_accountname"] = serviceName + "_asdsa"
		ctx := goajwt.WithJWT(context.Background(), jwt.NewWithClaims(jwt.SigningMethodRS512, claims))
		// then
		assert.False(t, token.IsSpecificServiceAccount(ctx, serviceName))
	})
}

func (s *TokenManagerTestSuite) TestAddLoginRequiredHeader() {
	s.T().Run("without header set", func(t *testing.T) {
		// given
		rw := httptest.NewRecorder()
		// when
		s.tm.AddLoginRequiredHeader(rw)
		// then
		checkLoginRequiredHeader(t, rw)
	})
	s.T().Run("with header set", func(t *testing.T) {
		// given
		rw := httptest.NewRecorder()
		rw.Header().Set("Access-Control-Expose-Headers", "somecustomvalue")
		// when
		s.tm.AddLoginRequiredHeader(rw)
		// then
		checkLoginRequiredHeader(t, rw)
	})
}

func checkLoginRequiredHeader(t *testing.T, rw http.ResponseWriter) {
	assert.Equal(t, "LOGIN url=https://auth.prod-preview.openshift.io/api/login, description=\"re-login is required\"", rw.Header().Get("WWW-Authenticate"))
	header := textproto.MIMEHeader(rw.Header())
	assert.Contains(t, header["Access-Control-Expose-Headers"], "WWW-Authenticate")
}

func (s *TokenManagerTestSuite) TestParseValidTokenOK() {
	// given
	identity := auth.NewIdentity()
	generatedToken, _, err := auth.GenerateSignedUserToken(identity)
	require.NoError(s.T(), err)

	s.T().Run("parse token", func(t *testing.T) {
		// when
		claims, err := s.tm.ParseToken(context.Background(), generatedToken)
		// then
		require.Nil(t, err)
		assert.Equal(t, identity.ID.String(), claims.Subject)
		assert.Equal(t, identity.Username, claims.Username)
	})

	s.T().Run("parse", func(t *testing.T) {
		// when
		jwtToken, err := s.tm.Parse(context.Background(), generatedToken)
		// then
		require.Nil(t, err)
		checkClaim(t, jwtToken, "sub", identity.ID.String())
		checkClaim(t, jwtToken, "preferred_username", identity.Username)
	})
}

func (s *TokenManagerTestSuite) TestAuthServiceURL() {
	config := testconfiguration.NewDefaultMockTokenManagerConfiguration(s.T())

	s.T().Run("OK if auth URL does not have trailing slash", func(t *testing.T) {
		config.GetAuthServiceURLFunc = func() string {
			return "https://auth.prod-preview.openshift.io"
		}
		tm, err := token.NewManager(config)
		require.NoError(t, err)
		assert.Len(t, tm.PublicKeys(), 3)
	})

	s.T().Run("OK if auth URL has trailing slash", func(t *testing.T) {
		config.GetAuthServiceURLFunc = func() string {
			return "https://auth.prod-preview.openshift.io/"
		}
		tm, err := token.NewManager(config)
		require.NoError(t, err)
		assert.Len(t, tm.PublicKeys(), 3)
	})

	s.T().Run("OK if auth URL has trailing slash", func(t *testing.T) {
		config.GetAuthServiceURLFunc = func() string {
			return "https://somedomain.com/"
		}
		_, err := token.NewManager(config)
		require.Error(t, err)
	})
}

func checkClaim(t *testing.T, token *jwt.Token, claimName string, expectedValue string) {
	jwtClaims := token.Claims.(jwt.MapClaims)
	claim, ok := jwtClaims[claimName]
	require.True(t, ok)
	assert.Equal(t, expectedValue, claim)
}

func (s *TokenManagerTestSuite) TestParseToken() {
	s.T().Run("Invalid token format", func(t *testing.T) {
		checkInvalidToken(t, s.tm, "7423742yuuiy-INVALID-73842342389h", "token contains an invalid number of segments")
	})

	s.T().Run("Missing kid", func(t *testing.T) {
		checkInvalidToken(t, s.tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.mx6DrW3kUQEvsz5Bmaea3nXD9_DB5fBDyNIfanr3u_RFWQyV0romzrAjBP3dKz_dgTS4S5WX2Q1XZiPLjc13PMTCQTNUXp6bFJ5RlDEqX6rJP0Ps9X7bke1pcqS7RhV9cnR1KNH8428bYoKCV57eQnhWtQoCQC1Db6YWJoQNJJLt0IHKCOx7c06r01VF1zcIk1dHnzzz9Qv5aACGXAi8iEJsQ1vURSh7fMETfSJl0UrLJsxGo60fHX9p74cu7bcgD-Zj86axRfgbaHuxn1MMJblltcPsG_TnsMOtmqQr4wlszWTQzwbLnemn8XfwPU8XYc49rVnkiZoB9-BV-oYIxg", "There is no 'kid' header in the token")
	})

	s.T().Run("Unknown kid", func(t *testing.T) {
		checkInvalidToken(t, s.tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InVua25vd25raWQifQ.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.ewwDOye1c5mA91YjgMQvk0x4RCXeosKVgag-UeG5lHQASWPv3_cpw0BMahG6-eJw1FTvnMFHlcH1Smw-Nfh43qKUpdgWL7QpVbLHiOgvfcsk1g3-G0QFBZ-N-xh35L53Cp5_seXionSAGjsNWLqStQaHPXJ9jLAc_JYLds1VO2CJ0tPSyJ0kiC8fiyRP17pJ19hiHonnGUAZlfZGPJZhrBCfAx3NBbejE0ZAUoeIAw1wPQJDCfp93vO5jvn0kUubpHlnAFz0YtLKqUfaiw6PfZDpu_HTpxAMVvyY_4PxzP56lWdnqQh6JhiMuVNehJfnKcAKPu4WboNCVVIBW3Gppg", "There is no public key with such ID: unknownkid")
	})

	s.T().Run("Invalid signature", func(t *testing.T) {
		checkInvalidToken(t, s.tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC5jb20ifQ.Y66kbPnxdfWyuawsEABDTsPFAylC9UFj934pF7SG-Fwrfqs3iF0gVHAQ56WLwY7E-D4QX_3uUkYuSrjzd4JT1p0bfxt3uu0wzFQlnzB4Uu2ttS287XPkBss4mUlc5uvAj0FRdy1IrQBFnfFpW5s6PWrHqod9PF4R2BTCBO1JqKgRtGzSqFwuWHowW__Sgw3B2NVgplL-6rb762M1OeT0GFWt0QE_uG8k_LPGPTyxDR5AILGfRgz5p-d16SYCAsjbsGSiQh3OGArt3Gzfi3CsKIGsQnhfuVXiorFbUn-nVaDuxRwU7JDzhde5nAj38U7exrkgxhEkybGMe4xZme49vA", "crypto/rsa: verification error")
	})

	s.T().Run("Expired", func(t *testing.T) {
		checkInvalidToken(t, s.tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjExMTk2MDc5NTIsIm5iZiI6MCwiaWF0IjoxMTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.g90TCdckT5YFctQSwQke7jmeGDZQotYiCa8AE_x0o8M8ncgb6m07glGXgcGJXftkzUL-uZn1U9JzixOYaI8B__jtB9BbMqMnrXyz-_gTYHAlj06l-9axVyKV7cpO8IIt_cFVt5lv4pPEcjEMzDLbjxxo6qH9lihry_KL3zESt8hxaosSnY5b8XvN7WCL-5NYTDF_i7QBI5x8XBljQpTJSwLY6-X7TDgAThET8OgWDV3H40UsSSsJUfpdEJZuiDsqoCsEpb0E7AfiYD-y0iZ5ULSxTiNf0EYf26irmy-jyQlWujOSb9kV2utsywZn-zDmHX3W_hS2wRD5eVgePFTBKA", "token is expired")
	})

	s.T().Run("OK", func(t *testing.T) {
		checkValidToken(t, s.tm, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.gyoMIWuXnIMMRHewef-__Wkd66qjqSSJxusWcFVtNWaYOXWu7iFV9DhtPVGsbTllXG_lDozPV9BaDmmYRotnn3ZBg7khFDykv9WnoYAjE9vW1d8szNjuoG3tfgQI4Dr9jqopSLndldxq97LGqpxqZFbIDlYd8vN47kv4EePOZDsII6egkTraCMc35eMMilJ4Udd6CMqyV_zaYiGhgAGgeL2ovMFhg_jnc7WhePv7FZkUmtfhCuLUL2TSXS6CyWZYoUDEIcfca6cMzuKOzJoONkDJShNo4u_cQ53duXX_bizdwYNlzBHfIPhSR1LDgV9BXoM6YQnw3It8ReCfF8BEMQ")
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

func (s *TokenManagerTestSuite) TestCheckClaimsOK() {
	// given
	claims := &token.TokenClaims{
		Email:    "somemail@domain.com",
		Username: "testuser",
	}
	claims.Subject = uuid.NewV4().String()
	// when
	err := token.CheckClaims(claims)
	// then
	assert.NoError(s.T(), err)
}

func (s *TokenManagerTestSuite) TestCheckClaimsFails() {
	s.T().Run("no email", func(t *testing.T) {
		// given
		claimsNoEmail := &token.TokenClaims{
			Username: "testuser",
		}
		claimsNoEmail.Subject = uuid.NewV4().String()
		// then
		assert.NotNil(t, token.CheckClaims(claimsNoEmail))
	})

	s.T().Run("no username", func(t *testing.T) {
		// given
		claimsNoUsername := &token.TokenClaims{
			Email: "somemail@domain.com",
		}
		claimsNoUsername.Subject = uuid.NewV4().String()
		// then
		assert.NotNil(t, token.CheckClaims(claimsNoUsername))
	})

	s.T().Run("no subject", func(t *testing.T) {
		// given
		claimsNoSubject := &token.TokenClaims{
			Email:    "somemail@domain.com",
			Username: "testuser",
		}
		// then
		assert.NotNil(t, token.CheckClaims(claimsNoSubject))
	})
}

func (s *TokenManagerTestSuite) TestLocateTokenInContext() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		id := uuid.NewV4()
		tk := jwt.New(jwt.SigningMethodRS256)
		tk.Claims.(jwt.MapClaims)["sub"] = id.String()
		ctx := goajwt.WithJWT(context.Background(), tk)
		// when
		foundId, err := s.tm.Locate(ctx)
		// then
		require.NoError(t, err)
		assert.Equal(t, id, foundId, "ID in created context not equal")
	})

	s.T().Run("missing token in context", func(t *testing.T) {
		// given
		ctx := context.Background()
		// when
		_, err := s.tm.Locate(ctx)
		// then
		require.Error(t, err)
	})

	s.T().Run("missing UUID in token", func(t *testing.T) {
		// given
		tk := jwt.New(jwt.SigningMethodRS256)
		ctx := goajwt.WithJWT(context.Background(), tk)
		// when
		_, err := s.tm.Locate(ctx)
		// then
		require.Error(t, err)
	})

	s.T().Run("invalid UUID in token", func(t *testing.T) {
		// given
		tk := jwt.New(jwt.SigningMethodRS256)
		tk.Claims.(jwt.MapClaims)["sub"] = "131"
		ctx := goajwt.WithJWT(context.Background(), tk)
		// when
		_, err := s.tm.Locate(ctx)
		// then
		require.Error(t, err)

	})
}

type DummyAuthConfig struct {
	url string
}

func (c *DummyAuthConfig) GetAuthServiceURL() string {
	return c.url
}

func (s *TokenManagerTestSuite) TestServiceAccountToken() {
	record, err := recorder.New("../test/data/exchange_token")
	require.NoError(s.T(), err)
	defer func() {
		err := record.Stop()
		require.NoError(s.T(), err)
	}()

	s.T().Run("ok", func(t *testing.T) {
		config := &DummyAuthConfig{"http://authservice"}

		saToken, err := token.ServiceAccountToken(context.Background(), config, "c211f1bd-17a7-4f8c-9f80-0917d167889d", "dummy_service", httpsupport.WithRoundTripper(record))

		require.NoError(t, err)
		assert.NotEmpty(t, saToken)
		assert.Equal(t, "jA0ECQMC5AvXo6Jyrj5g0kcBv6Qp8ZTWCgYD6TESuc2OxSDZ1lic1tmV6g4IcQUBlohjT3gyQX2oTa1bWfNkk8xY6wyPq8CUK3ReOnnDK/yo661f6LXgvA==", saToken)
	})

	s.T().Run("ok empty token", func(t *testing.T) {
		config := &DummyAuthConfig{"http://authservice.tokenempty"}

		saToken, err := token.ServiceAccountToken(context.Background(), config, "c211f1bd-17a7-4f8c-9f80-0917d167889d", "dummy_service", httpsupport.WithRoundTripper(record))

		require.Error(t, err)
		assert.Equal(t, "received empty token from server \"http://authservice.tokenempty\"", err.Error())
		assert.Empty(t, saToken)
	})

	s.T().Run("error", func(t *testing.T) {
		config := &DummyAuthConfig{"http://authservice.error"}
		saToken, err := token.ServiceAccountToken(context.Background(), config, "c211f1bd-17a7-4f8c-9f80-0917d167889d", "dummy_service", httpsupport.WithRoundTripper(record))

		require.Error(t, err)
		assert.Equal(t, "failed to obtain token from auth server \"http://authservice.error\": something went wrong", err.Error())
		assert.Empty(t, saToken)
	})

	s.T().Run("baq request", func(t *testing.T) {
		config := &DummyAuthConfig{"http://authservice.bad"}
		saToken, err := token.ServiceAccountToken(context.Background(), config, "c211f1bd-17a7-4f8c-9f80-0917d167889d", "dummy_service", httpsupport.WithRoundTripper(record))

		require.Error(t, err)
		assert.Equal(t, "failed to obtain token from auth server \"http://authservice.bad\": [8sZ5BugD] 400 invalid_request: attribute \"grant_type\" of request is missing and required, attribute: grant_type, parent: request", err.Error())
		assert.Empty(t, saToken)
	})

	s.T().Run("unauthorized", func(t *testing.T) {
		config := &DummyAuthConfig{"http://authservice.unauthorized"}
		saToken, err := token.ServiceAccountToken(context.Background(), config, "c211f1bd-17a7-4f8c-9f80-0917d167889d", "dummy_service", httpsupport.WithRoundTripper(record))

		require.Error(t, err)
		assert.Equal(t, "failed to obtain token from auth server \"http://authservice.unauthorized\": invalid Service Account ID or secret", err.Error())
		assert.Empty(t, saToken)
	})
}
