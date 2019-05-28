package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/fabric8-services/fabric8-common/auth"
	"github.com/fabric8-services/fabric8-common/configuration"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	jwtgoa "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var TokenManager = newManager()

func newManager() auth.Manager {
	return auth.NewManagerWithPublicKey("test-key", &defaultPrivateKey().PublicKey, &defaultCfg{})
}

type defaultCfg struct{}

func (c *defaultCfg) GetAuthServiceURL() string    { return "https://auth.openshift.io" }
func (c *defaultCfg) GetDevModePrivateKey() []byte { return []byte(configuration.DevModeRsaPrivateKey) }

// ExtraClaim a function to set claims in the token to generate
type ExtraClaim func(token *jwt.Token)

// WithEmailClaim sets the `email` claim in the token to generate
func WithEmailClaim(email string) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(jwt.MapClaims)["email"] = email
	}
}

// WithEmailVerifiedClaim sets the `email_verified` claim in the token to generate
func WithEmailVerifiedClaim(verified bool) ExtraClaim {
	return func(token *jwt.Token) {
		token.Claims.(jwt.MapClaims)["email_verified"] = verified
	}
}

// GenerateToken generates a JWT user token and signs it using the default private key
func GenerateToken(identityID string, username string, extraClaims ...ExtraClaim) string {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["uuid"] = identityID
	token.Claims.(jwt.MapClaims)["preferred_username"] = username
	token.Claims.(jwt.MapClaims)["sub"] = identityID
	token.Claims.(jwt.MapClaims)["email"] = username + "@email.com"
	for _, extra := range extraClaims {
		extra(token)
	}
	key := defaultPrivateKey()
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

	key := defaultPrivateKey()
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
	key := defaultPrivateKey()
	token.Header["kid"] = "test-key"
	tokenStr, err := token.SignedString(key)
	if err != nil {
		panic(errors.WithStack(err))
	}
	return tokenStr
}

// Identity a user identity
type Identity struct {
	ID       uuid.UUID
	Username string
	Email    string
}

// NewIdentity returns a new, random identity
func NewIdentity() *Identity {
	return &Identity{
		ID:       uuid.NewV4(),
		Username: "testuser-" + uuid.NewV4().String(),
		Email:    uuid.NewV4().String() + "@email.com",
	}
}

// EmbedUserTokenInContext generates a token for the given identity and embed it into the context along with token manager
func EmbedUserTokenInContext(ctx context.Context, identity *Identity) (context.Context, *Identity, error) {
	if identity == nil {
		identity = NewIdentity()
	}
	_, token, err := GenerateSignedUserToken(identity)
	if err != nil {
		return nil, nil, err
	}
	emdCtx, err := embedTokenInContext(ctx, token)
	return emdCtx, identity, err
}

// EmbedServiceAccountTokenInContext generates a token for the given identity and embed it into the context along with token manager
func EmbedServiceAccountTokenInContext(ctx context.Context, identity *Identity) (context.Context, error) {
	_, token, err := GenerateSignedServiceAccountToken(identity)
	if err != nil {
		return nil, err
	}
	return embedTokenInContext(ctx, token)
}

func embedTokenInContext(ctx context.Context, tk *jwt.Token) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = auth.ContextWithTokenManager(ctx, TokenManager)
	ctx = jwtgoa.WithJWT(ctx, tk)
	return ContextWithRequest(ctx)
}

// GenerateSignedUserToken generates a JWT token and signs it using the default private key
func GenerateSignedUserToken(identity *Identity) (string, *jwt.Token, error) {
	token := generateUserToken(identity)
	tokenStr, err := signToken(token)
	if err != nil {
		return "", nil, errs.Wrapf(err, "unable to generate user token")
	}
	return tokenStr, token, nil
}

// GenerateSignedServiceAccountToken generates a JWT SA token and signs it using the default private key
func GenerateSignedServiceAccountToken(identity *Identity) (string, *jwt.Token, error) {
	token := generateServiceAccountToken(identity)
	tokenStr, err := signToken(token)
	if err != nil {
		return "", nil, errs.Wrapf(err, "unable to generate SA token")
	}
	return tokenStr, token, nil
}

func signToken(token *jwt.Token) (string, error) {
	key := defaultPrivateKey()
	tokenStr, err := token.SignedString(key)
	if err != nil {
		return "", errs.Wrapf(err, "unable to sign token")
	}
	token.Raw = tokenStr

	return tokenStr, nil
}

// generateUserToken generates a JWT token
func generateUserToken(identity *Identity) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["jti"] = uuid.NewV4().String()
	iat := time.Now().Unix()
	claims["exp"] = 0
	claims["iat"] = iat
	claims["typ"] = "Bearer"
	claims["preferred_username"] = identity.Username
	claims["sub"] = identity.ID.String()
	claims["email"] = identity.Email

	token.Header["kid"] = "test-key"

	return token
}

// generateServiceAccountToken generates a JWT token
func generateServiceAccountToken(identity *Identity) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["service_accountname"] = identity.Username
	claims["sub"] = identity.ID.String()
	claims["jti"] = uuid.NewV4().String()
	claims["iat"] = time.Now().Unix()

	token.Header["kid"] = "test-key"

	return token
}

// ContextWithRequest return a Goa context with a request
func ContextWithRequest(ctx context.Context) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rw := httptest.NewRecorder()
	url := "https://common-test"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errs.Wrapf(err, "unable to initialize a new request")
	}
	return goa.NewContext(goa.WithAction(ctx, "Test"), rw, req, nil), nil
}

func ContextWithTokenAndRequestID() (context.Context, string, string, string, error) {
	identityID := uuid.NewV4().String()
	ctx, ctxToken, err := EmbedTokenInContext(identityID, uuid.NewV4().String())
	if err != nil {
		return nil, "", "", "", err
	}
	ctx = auth.ContextWithTokenManager(ctx, TokenManager)
	reqID := uuid.NewV4().String()
	ctx = client.SetContextRequestID(ctx, reqID)
	return ctx, identityID, ctxToken, reqID, nil
}

// EmbedTokenInContext generates a token and embeds it into the context along with token manager
func EmbedTokenInContext(sub, username string, extraClaims ...ExtraClaim) (context.Context, string, error) {
	tokenString := GenerateToken(sub, username, extraClaims...)
	extracted, err := TokenManager.Parse(context.Background(), tokenString)
	if err != nil {
		return nil, "", err
	}
	// Embed Token in the context
	ctx := jwtgoa.WithJWT(context.Background(), extracted)
	ctx, err = ContextWithRequest(ctx)
	if err != nil {
		return nil, "", err
	}
	return auth.ContextWithTokenManager(ctx, TokenManager), tokenString, nil
}

func defaultPrivateKey() *rsa.PrivateKey {
	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(configuration.DevModeRsaPrivateKey))
	if err != nil {
		panic("Failed: " + err.Error())
	}
	return rsaKey
}

// ServiceAsUser creates a new service and fill the context with input Identity
func ServiceAsUser(serviceName string, identity *Identity) (*goa.Service, error) {
	svc := goa.New(serviceName)
	ctx, _, err := EmbedUserTokenInContext(context.Background(), identity)
	if err != nil {
		return nil, err
	}
	svc.Context = ctx
	return svc, nil
}

// UnsecuredService creates a new service with token manager injected by without any identity in context
func UnsecuredService(serviceName string) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = auth.ContextWithTokenManager(svc.Context, TokenManager)
	return svc
}

// ServiceAsServiceAccountUser generates the minimal service needed to satisfy the condition of being a service account.
func ServiceAsServiceAccountUser(serviceName string, identity *Identity) (*goa.Service, error) {
	svc := goa.New(serviceName)
	ctx, err := EmbedServiceAccountTokenInContext(context.Background(), identity)
	if err != nil {
		return nil, err
	}
	svc.Context = ctx
	return svc, nil
}
