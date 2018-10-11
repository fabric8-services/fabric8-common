package token_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	testsupport "github.com/fabric8-services/fabric8-common/test"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"
	"github.com/fabric8-services/fabric8-common/token"

	"strings"

	"github.com/dgrijalva/jwt-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestToken(t *testing.T) {
	suite.Run(t, &TestTokenSuite{})
}

type TestTokenSuite struct {
	testsuite.UnitTestSuite
}

func (s *TestTokenSuite) TestNotAServiceAccountFails() {
	ctx := createInvalidSAContext()
	assert.False(s.T(), token.IsSpecificServiceAccount(ctx, "someName"))
}

func (s *TestTokenSuite) TestIsServiceAccountFails() {
	ctx := createInvalidSAContext()
	assert.False(s.T(), token.IsServiceAccount(ctx))
}

func createInvalidSAContext() context.Context {
	claims := jwt.MapClaims{}
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	return goajwt.WithJWT(context.Background(), token)
}

func (s *TestTokenSuite) TestAddLoginRequiredHeader() {
	rw := httptest.NewRecorder()
	testsupport.TokenManager.AddLoginRequiredHeader(rw)

	s.checkLoginRequiredHeader(rw)

	rw = httptest.NewRecorder()
	rw.Header().Set("Access-Control-Expose-Headers", "somecustomvalue")
	testsupport.TokenManager.AddLoginRequiredHeader(rw)
	s.checkLoginRequiredHeader(rw)
}

func (s *TestTokenSuite) checkLoginRequiredHeader(rw http.ResponseWriter) {
	assert.Equal(s.T(), "LOGIN url=https://auth.prod-preview.openshift.io/api/login, description=\"re-login is required\"", rw.Header().Get("WWW-Authenticate"))
	header := textproto.MIMEHeader(rw.Header())
	assert.Contains(s.T(), header["Access-Control-Expose-Headers"], "WWW-Authenticate")
}

func (s *TestTokenSuite) assertHeaders(tokenString string) {
	jwtToken, err := testsupport.TokenManager.Parse(context.Background(), tokenString)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "aUGv8mQA85jg4V1DU8Uk1W0uKsxn187KQONAGl6AMtc", jwtToken.Header["kid"])
	assert.Equal(s.T(), "RS256", jwtToken.Header["alg"])
	assert.Equal(s.T(), "JWT", jwtToken.Header["typ"])
}

func (s *TestTokenSuite) TestParseValidTokenOK() {
	identity := testsupport.NewIdentity()
	generatedToken, _ := testsupport.GenerateSignedUserToken(identity)

	claims, err := testsupport.TokenManager.ParseToken(context.Background(), generatedToken)
	require.Nil(s.T(), err)
	assert.Equal(s.T(), identity.ID.String(), claims.Subject)
	assert.Equal(s.T(), identity.Username, claims.Username)

	jwtToken, err := testsupport.TokenManager.Parse(context.Background(), generatedToken)
	require.Nil(s.T(), err)

	s.checkClaim(jwtToken, "sub", identity.ID.String())
	s.checkClaim(jwtToken, "preferred_username", identity.Username)
}

func (s *TestTokenSuite) checkClaim(token *jwt.Token, claimName string, expectedValue string) {
	jwtClaims := token.Claims.(jwt.MapClaims)
	claim, ok := jwtClaims[claimName]
	require.True(s.T(), ok)
	assert.Equal(s.T(), expectedValue, claim)
}

func (s *TestTokenSuite) TestParseInvalidTokenFails() {
	// Invalid token format
	s.checkInvalidToken("7423742yuuiy-INVALID-73842342389h", "token contains an invalid number of segments")

	// Missing kid
	s.checkInvalidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.mx6DrW3kUQEvsz5Bmaea3nXD9_DB5fBDyNIfanr3u_RFWQyV0romzrAjBP3dKz_dgTS4S5WX2Q1XZiPLjc13PMTCQTNUXp6bFJ5RlDEqX6rJP0Ps9X7bke1pcqS7RhV9cnR1KNH8428bYoKCV57eQnhWtQoCQC1Db6YWJoQNJJLt0IHKCOx7c06r01VF1zcIk1dHnzzz9Qv5aACGXAi8iEJsQ1vURSh7fMETfSJl0UrLJsxGo60fHX9p74cu7bcgD-Zj86axRfgbaHuxn1MMJblltcPsG_TnsMOtmqQr4wlszWTQzwbLnemn8XfwPU8XYc49rVnkiZoB9-BV-oYIxg", "There is no 'kid' header in the token")

	// Unknown kid
	s.checkInvalidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InVua25vd25raWQifQ.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.ewwDOye1c5mA91YjgMQvk0x4RCXeosKVgag-UeG5lHQASWPv3_cpw0BMahG6-eJw1FTvnMFHlcH1Smw-Nfh43qKUpdgWL7QpVbLHiOgvfcsk1g3-G0QFBZ-N-xh35L53Cp5_seXionSAGjsNWLqStQaHPXJ9jLAc_JYLds1VO2CJ0tPSyJ0kiC8fiyRP17pJ19hiHonnGUAZlfZGPJZhrBCfAx3NBbejE0ZAUoeIAw1wPQJDCfp93vO5jvn0kUubpHlnAFz0YtLKqUfaiw6PfZDpu_HTpxAMVvyY_4PxzP56lWdnqQh6JhiMuVNehJfnKcAKPu4WboNCVVIBW3Gppg", "There is no public key with such ID: unknownkid")

	// Invalid signature
	s.checkInvalidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC5jb20ifQ.Y66kbPnxdfWyuawsEABDTsPFAylC9UFj934pF7SG-Fwrfqs3iF0gVHAQ56WLwY7E-D4QX_3uUkYuSrjzd4JT1p0bfxt3uu0wzFQlnzB4Uu2ttS287XPkBss4mUlc5uvAj0FRdy1IrQBFnfFpW5s6PWrHqod9PF4R2BTCBO1JqKgRtGzSqFwuWHowW__Sgw3B2NVgplL-6rb762M1OeT0GFWt0QE_uG8k_LPGPTyxDR5AILGfRgz5p-d16SYCAsjbsGSiQh3OGArt3Gzfi3CsKIGsQnhfuVXiorFbUn-nVaDuxRwU7JDzhde5nAj38U7exrkgxhEkybGMe4xZme49vA", "crypto/rsa: verification error")

	// Expired
	s.checkInvalidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjExMTk2MDc5NTIsIm5iZiI6MCwiaWF0IjoxMTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.g90TCdckT5YFctQSwQke7jmeGDZQotYiCa8AE_x0o8M8ncgb6m07glGXgcGJXftkzUL-uZn1U9JzixOYaI8B__jtB9BbMqMnrXyz-_gTYHAlj06l-9axVyKV7cpO8IIt_cFVt5lv4pPEcjEMzDLbjxxo6qH9lihry_KL3zESt8hxaosSnY5b8XvN7WCL-5NYTDF_i7QBI5x8XBljQpTJSwLY6-X7TDgAThET8OgWDV3H40UsSSsJUfpdEJZuiDsqoCsEpb0E7AfiYD-y0iZ5ULSxTiNf0EYf26irmy-jyQlWujOSb9kV2utsywZn-zDmHX3W_hS2wRD5eVgePFTBKA", "token is expired")

	// OK
	s.checkValidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjAsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.gyoMIWuXnIMMRHewef-__Wkd66qjqSSJxusWcFVtNWaYOXWu7iFV9DhtPVGsbTllXG_lDozPV9BaDmmYRotnn3ZBg7khFDykv9WnoYAjE9vW1d8szNjuoG3tfgQI4Dr9jqopSLndldxq97LGqpxqZFbIDlYd8vN47kv4EePOZDsII6egkTraCMc35eMMilJ4Udd6CMqyV_zaYiGhgAGgeL2ovMFhg_jnc7WhePv7FZkUmtfhCuLUL2TSXS6CyWZYoUDEIcfca6cMzuKOzJoONkDJShNo4u_cQ53duXX_bizdwYNlzBHfIPhSR1LDgV9BXoM6YQnw3It8ReCfF8BEMQ")
}

func (s *TestTokenSuite) checkInvalidToken(token, expectedError string) {
	_, err := testsupport.TokenManager.ParseToken(context.Background(), token)
	require.Error(s.T(), err)
	assert.Contains(s.T(), strings.ToLower(err.Error()), strings.ToLower(expectedError))
	_, err = testsupport.TokenManager.ParseTokenWithMapClaims(context.Background(), token)
	require.Error(s.T(), err)
	assert.Contains(s.T(), strings.ToLower(err.Error()), strings.ToLower(expectedError))
	_, err = testsupport.TokenManager.Parse(context.Background(), token)
	require.Error(s.T(), err)
	assert.Contains(s.T(), strings.ToLower(err.Error()), strings.ToLower(expectedError))
}

func (s *TestTokenSuite) checkValidToken(token string) {
	_, err := testsupport.TokenManager.ParseToken(context.Background(), token)
	assert.NoError(s.T(), err)
	_, err = testsupport.TokenManager.ParseTokenWithMapClaims(context.Background(), token)
	assert.NoError(s.T(), err)
	_, err = testsupport.TokenManager.Parse(context.Background(), token)
	assert.NoError(s.T(), err)
}

func (s *TestTokenSuite) TestCheckClaimsOK() {
	claims := &token.TokenClaims{
		Email:    "somemail@domain.com",
		Username: "testuser",
	}
	claims.Subject = uuid.NewV4().String()

	assert.Nil(s.T(), token.CheckClaims(claims))
}

func (s *TestTokenSuite) TestCheckClaimsFails() {
	claimsNoEmail := &token.TokenClaims{
		Username: "testuser",
	}
	claimsNoEmail.Subject = uuid.NewV4().String()
	assert.NotNil(s.T(), token.CheckClaims(claimsNoEmail))

	claimsNoUsername := &token.TokenClaims{
		Email: "somemail@domain.com",
	}
	claimsNoUsername.Subject = uuid.NewV4().String()
	assert.NotNil(s.T(), token.CheckClaims(claimsNoUsername))

	claimsNoSubject := &token.TokenClaims{
		Email:    "somemail@domain.com",
		Username: "testuser",
	}
	assert.NotNil(s.T(), token.CheckClaims(claimsNoSubject))
}

func (s *TestTokenSuite) TestLocateTokenInContex() {
	id := uuid.NewV4()

	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["sub"] = id.String()
	ctx := goajwt.WithJWT(context.Background(), tk)

	foundId, err := testsupport.TokenManager.Locate(ctx)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), id, foundId, "ID in created context not equal")
}

func (s *TestTokenSuite) TestLocateMissingTokenInContext() {
	ctx := context.Background()

	_, err := testsupport.TokenManager.Locate(ctx)
	if err == nil {
		s.T().Error("Should have returned error on missing token in contex", err)
	}
}

func (s *TestTokenSuite) TestLocateMissingUUIDInTokenInContext() {
	tk := jwt.New(jwt.SigningMethodRS256)
	ctx := goajwt.WithJWT(context.Background(), tk)

	_, err := testsupport.TokenManager.Locate(ctx)
	require.Error(s.T(), err)
}

func (s *TestTokenSuite) TestLocateInvalidUUIDInTokenInContext() {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["sub"] = "131"
	ctx := goajwt.WithJWT(context.Background(), tk)

	_, err := testsupport.TokenManager.Locate(ctx)
	require.Error(s.T(), err)
}
