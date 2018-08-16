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

// TODO Test that there is no test key if not run in Dev Mode!!!

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
	assert.Equal(s.T(), "LOGIN url=http://localhost/api/login, description=\"re-login is required\"", rw.Header().Get("WWW-Authenticate"))
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
	s.checkInvalidToken("7423742yuuiy-INVALID-73842342389h")

	// Missing kid
	s.checkInvalidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjE1MTk2MDc5NTIsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.n_dbSp05SWx8L6D4wA8es5IKaFDCyYMeepPDgVztYIMk5WOzNTbqdsXDn73WekLN0LM65o6bGYR7AtQr0h2ad_m_1amgB3PDj9-a2bE_TznZK_PNmicom42kZQ6l_ihhjBNF8g-6nrSJ9JwMHIdVCiFT1ewdRJgylfVPKa-KO9KDCMOwFqLazAdOv6LqK9H610xR5y9HKAgdH4AaTSXF5MLoooKP9dLPwDYH5haJMA86rwSei6RMtrVazcnriod8tLCRbirwBHgF-KSwOK2OwYNYzg7P-qe0o34oM_RO7B7ZIWuzSzi4FJyHAIpvcL3Au1Pfh5NMwC8xpmImyiT8-g")

	// Unknown kid
	s.checkInvalidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InVua25vd25raWQifQ.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjE1MTk2MDc5NTIsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.U9ViRKA047HZZViS0YtaT9a52QMQpoMQcEPM6Uwwz2mBWdWUWDjkE3N9oC4Dm45DYCN-rRdRa95WqeHEHl7F-3s8p1Fqx0YpDZIoD3u_wqE2MkKamrUN1na2R1oKZI_wG5BsjwxdcujRwL2Y7Lk288-1FD0naqrc4J9Ofi8ZELBWXPGUcatkk1unSHAUhxx51X8BqI5Nrmu2p__9EQNtN83esuqvGzCrL83XG1LBjgfypikd3_H1YaB6Vi6yUoCU56pcNTUItMeDr4oan4pBB5V4M5O4EsQt7cmdNikLvfCRChMjilxCuZmtpRzsH2U5yWKDJVZv2QUeT3WLHzTdMQ")

	// Invalid signature
	s.checkInvalidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjE1MTk2MDc5NTIsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC5jb20ifQ.cp1upW_7cyRLGLP_bo2XUj2qqpkf77T5ZN7S1YlHgaV9g0dDNkmXqAActup2jUOVZlfpktsTQLmnaBUJkmsCRSABZ0_brqelWgsrPlEGlDHuDmbkVyjwgv5lW_2fvvqoxWlLmRB-EEoFl18cvkTX7rUUzKM9ItkMbIHwudV01CHXeTbON6F5cv4fcXyvqzxAr1JjfXQljtbqKZ9l4zm3JK7TmA_K1fjwu5oSDg6CzFD2cAhSfHFoa0-Sbrine1TCNsyyca70aNbpJOBpt68OFrKsSNPByD_MYaKYYeNn2yD6mmZgmp704GndeKfW-XAkI5CVAVG6rIT3UXLpe8y5wg")

	// Expired
	s.checkInvalidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjExMTk2MDc5NTIsIm5iZiI6MCwiaWF0IjoxMTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.g90TCdckT5YFctQSwQke7jmeGDZQotYiCa8AE_x0o8M8ncgb6m07glGXgcGJXftkzUL-uZn1U9JzixOYaI8B__jtB9BbMqMnrXyz-_gTYHAlj06l-9axVyKV7cpO8IIt_cFVt5lv4pPEcjEMzDLbjxxo6qH9lihry_KL3zESt8hxaosSnY5b8XvN7WCL-5NYTDF_i7QBI5x8XBljQpTJSwLY6-X7TDgAThET8OgWDV3H40UsSSsJUfpdEJZuiDsqoCsEpb0E7AfiYD-y0iZ5ULSxTiNf0EYf26irmy-jyQlWujOSb9kV2utsywZn-zDmHX3W_hS2wRD5eVgePFTBKA")

	// OK
	s.checkValidToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6InRlc3Qta2V5In0.eyJqdGkiOiIwMjgyYjI5Yy01MTczLTQyZDgtODE0NS1iNDVmYTFlMzUzOGIiLCJleHAiOjE1MTk2MDc5NTIsIm5iZiI6MCwiaWF0IjoxNTE3MDE1OTUyLCJpc3MiOiJ0ZXN0IiwiYXVkIjoiZmFicmljOC1vbmxpbmUtcGxhdGZvcm0iLCJzdWIiOiIyMzk4NDM5OC04NTVhLTQyZDYtYTdmZS05MzZiYjRlOTJhMGMiLCJ0eXAiOiJCZWFyZXIiLCJzZXNzaW9uX3N0YXRlIjoiZWFkYzA2NmMtMTIzNC00YTU2LTlmMzUtY2U3MDdiNTdhNGU5IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sImFwcHJvdmVkIjp0cnVlLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IlRlc3QiLCJjb21wYW55IjoiIiwicHJlZmVycmVkX3VzZXJuYW1lIjoidGVzdHVzZXIiLCJnaXZlbl9uYW1lIjoiIiwiZmFtaWx5X25hbWUiOiIiLCJlbWFpbCI6InRAdGVzdC50In0.XKvMmiluFOh7ndMWxtvjbhLFFr5CVQXR_sY-5UyS_7lysEJ_DkP3NvnSCOsJHUzJfk4cxFEFZHVa6njhro2NY8CaxDCrnDOf1c7KnM6yU74OAv9I-_dvjuMeiFahkMdb2NQpDNJhlYV-lYs8qiYGeehj44gSrOlmOl4XjPx1irc_YuGS98VKxTb9TdNWQX3ciPDfA3bbk9RCy175mdyFX9GxLUpHX6ruinm1-_qVWMlImNDrWhppzFf7ixq8BW6Fo9qZ4-UOuYpEHgfEAcvxUIRg_zZC1MUyPKdpgu_gvKUMgsOW67ssL9Zk5pPPRkNmPKCtp0ovBfwItkLvVAPC7Q")
}

func (s *TestTokenSuite) checkInvalidToken(token string) {
	_, err := testsupport.TokenManager.ParseToken(context.Background(), token)
	assert.Error(s.T(), err)
	_, err = testsupport.TokenManager.ParseTokenWithMapClaims(context.Background(), token)
	assert.Error(s.T(), err)
	_, err = testsupport.TokenManager.Parse(context.Background(), token)
	assert.Error(s.T(), err)
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
