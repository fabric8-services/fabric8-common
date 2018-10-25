package token

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/fabric8-services/fabric8-common/test"

	"github.com/fabric8-services/fabric8-common/configuration"
	"github.com/fabric8-services/fabric8-common/token"

	"github.com/dgrijalva/jwt-go"
	jwtgoa "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

var TokenManager = newManager()

func newManager() token.Manager {
	return token.NewManagerWithPublicKey("test-key", &privateKey().PublicKey, &dummyConfig{})
}

func privateKey() *rsa.PrivateKey {
	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(configuration.DevModeRsaPrivateKey))
	if err != nil {
		panic("Failed: " + err.Error())
	}
	return rsaKey
}

type dummyConfig struct{}

func (c *dummyConfig) GetAuthServiceURL() string    { return "https://auth.openshift.io" }
func (c *dummyConfig) GetDevModePrivateKey() []byte { return nil }

// EmbedTokenInContext generates a token and embeds it into the context along with token manager
func EmbedTokenInContext(sub, username string) (context.Context, string, error) {
	tokenString := GenerateToken(sub, username)
	extracted, err := TokenManager.Parse(context.Background(), tokenString)
	if err != nil {
		return nil, "", err
	}
	// Embed Token in the context
	ctx := jwtgoa.WithJWT(context.Background(), extracted)
	ctx, err = test.ContextWithRequest(ctx)
	if err != nil {
		return nil, "", err
	}
	return token.ContextWithTokenManager(ctx, TokenManager), tokenString, nil
}

// GenerateToken generates a JWT user token and signs it using the default private key
func GenerateToken(identityID string, identityUsername string) string {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["uuid"] = identityID
	token.Claims.(jwt.MapClaims)["preferred_username"] = identityUsername
	token.Claims.(jwt.MapClaims)["sub"] = identityID
	token.Claims.(jwt.MapClaims)["email"] = identityUsername + "@email.com"

	key := privateKey()
	token.Header["kid"] = "test-key"
	tokenStr, err := token.SignedString(key)

	if err != nil {
		panic(errors.WithStack(err))
	}
	return tokenStr
}

// GenerateServiceAccountToken generates a JWT service account token and signs it using the default private key
func GenerateServiceAccountToken(saName string) string {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["service_accountname"] = saName

	key := privateKey()
	token.Header["kid"] = "test-key"
	tokenStr, err := token.SignedString(key)

	if err != nil {
		panic(errors.WithStack(err))
	}
	return tokenStr
}

// GenerateTokenWithClaims generates a JWT token with additional claims
func GenerateTokenWithClaims(claims map[string]interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)

	// default claims
	token.Claims.(jwt.MapClaims)["uuid"] = uuid.NewV4().String()
	token.Claims.(jwt.MapClaims)["preferred_username"] = fmt.Sprintf("testUser-%s", uuid.NewV4().String())
	token.Claims.(jwt.MapClaims)["sub"] = uuid.NewV4().String()
	token.Claims.(jwt.MapClaims)["jti"] = uuid.NewV4().String()
	token.Claims.(jwt.MapClaims)["session_state"] = uuid.NewV4().String()
	token.Claims.(jwt.MapClaims)["iat"] = time.Now().Unix()
	token.Claims.(jwt.MapClaims)["exp"] = time.Now().Unix() + 60*60*24*30
	token.Claims.(jwt.MapClaims)["nbf"] = 0
	token.Claims.(jwt.MapClaims)["iss"] = "fabric8-auth"
	token.Claims.(jwt.MapClaims)["typ"] = "Bearer"
	token.Claims.(jwt.MapClaims)["approved"] = true
	token.Claims.(jwt.MapClaims)["name"] = "Test User"
	token.Claims.(jwt.MapClaims)["company"] = "Company Inc."
	token.Claims.(jwt.MapClaims)["given_name"] = "Test"
	token.Claims.(jwt.MapClaims)["family_name"] = "User"
	token.Claims.(jwt.MapClaims)["email"] = fmt.Sprintf("testuser+%s@email.com", uuid.NewV4().String())
	token.Claims.(jwt.MapClaims)["email_verified"] = true
	// explicit values passed by the caller
	for key, value := range claims {
		token.Claims.(jwt.MapClaims)[key] = value
	}
	key := privateKey()
	token.Header["kid"] = "test-key"
	tokenStr, err := token.SignedString(key)
	if err != nil {
		panic(errors.WithStack(err))
	}
	return tokenStr
}
