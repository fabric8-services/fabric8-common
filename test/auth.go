package test

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"time"

	testtoken "github.com/fabric8-services/fabric8-common/test/token"
	"github.com/fabric8-services/fabric8-common/token"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	client "github.com/goadesign/goa/client"
	jwtgoa "github.com/goadesign/goa/middleware/security/jwt"

	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

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
func EmbedUserTokenInContext(ctx context.Context, identity *Identity, tm token.Manager, config token.ManagerConfiguration) (context.Context, error) {
	if identity == nil {
		identity = NewIdentity()
	}
	_, token, err := GenerateSignedUserToken(identity, config)
	if err != nil {
		return nil, err
	}
	return embedTokenInContext(ctx, token, tm)
}

// EmbedServiceAccountTokenInContext generates a token for the given identity and embed it into the context along with token manager
func EmbedServiceAccountTokenInContext(ctx context.Context, identity *Identity, tm token.Manager, config token.ManagerConfiguration) (context.Context, error) {
	_, token, err := GenerateSignedServiceAccountToken(identity, config)
	if err != nil {
		return nil, err
	}
	return embedTokenInContext(ctx, token, tm)
}

func embedTokenInContext(ctx context.Context, tk *jwt.Token, tm token.Manager) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = token.ContextWithTokenManager(ctx, tm)
	ctx = jwtgoa.WithJWT(ctx, tk)
	return ContextWithRequest(ctx)
}

// GenerateSignedUserToken generates a JWT token and signs it using the default private key
func GenerateSignedUserToken(identity *Identity, config token.ManagerConfiguration) (string, *jwt.Token, error) {
	token := generateUserToken(identity)
	tokenStr, err := signToken(token, config)
	if err != nil {
		return "", nil, errs.Wrapf(err, "unable to generate user token")
	}
	return tokenStr, token, nil
}

// GenerateSignedServiceAccountToken generates a JWT SA token and signs it using the default private key
func GenerateSignedServiceAccountToken(identity *Identity, config token.ManagerConfiguration) (string, *jwt.Token, error) {
	token := generateServiceAccountToken(identity)
	tokenStr, err := signToken(token, config)
	if err != nil {
		return "", nil, errs.Wrapf(err, "unable to generate SA token")
	}
	return tokenStr, token, nil
}

func signToken(token *jwt.Token, config token.ManagerConfiguration) (string, error) {
	key, _, err := privateKey(config)
	if err != nil {
		return "", err
	}
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
	ctx = token.ContextWithTokenManager(ctx, testtoken.TokenManager)
	reqID := uuid.NewV4().String()
	ctx = client.SetContextRequestID(ctx, reqID)
	return ctx, identityID, ctxToken, reqID, nil
}

// EmbedTokenInContext generates a token and embeds it into the context along with token manager
func EmbedTokenInContext(sub, username string) (context.Context, string, error) {
	tokenString := testtoken.GenerateToken(sub, username)
	extracted, err := testtoken.TokenManager.Parse(context.Background(), tokenString)
	if err != nil {
		return nil, "", err
	}
	// Embed Token in the context
	ctx := jwtgoa.WithJWT(context.Background(), extracted)
	ctx, err = ContextWithRequest(ctx)
	if err != nil {
		return nil, "", err
	}
	return token.ContextWithTokenManager(ctx, testtoken.TokenManager), tokenString, nil
}

func privateKey(config token.ManagerConfiguration) (*rsa.PrivateKey, string, error) {
	key := config.GetDevModePrivateKey()
	pk, err := jwt.ParseRSAPrivateKeyFromPEM(key)
	if err != nil {
		return nil, "", errs.Wrapf(err, "unable to get 'dev mode' private key")
	}
	return pk, "test-key", nil
}

// ServiceAsUser creates a new service and fill the context with input Identity
func ServiceAsUser(serviceName string, identity *Identity, tm token.Manager, config token.ManagerConfiguration) (*goa.Service, error) {
	svc := goa.New(serviceName)
	ctx, err := EmbedUserTokenInContext(context.Background(), identity, tm, config)
	if err != nil {
		return nil, err
	}
	svc.Context = ctx
	return svc, nil
}

// UnsecuredService creates a new service with token manager injected by without any identity in context
func UnsecuredService(serviceName string, tm token.Manager) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = token.ContextWithTokenManager(svc.Context, tm)
	return svc
}

// ServiceAsServiceAccountUser generates the minimal service needed to satisfy the condition of being a service account.
func ServiceAsServiceAccountUser(serviceName string, identity *Identity, tm token.Manager, config token.ManagerConfiguration) (*goa.Service, error) {
	svc := goa.New(serviceName)
	ctx, err := EmbedServiceAccountTokenInContext(context.Background(), identity, tm, config)
	if err != nil {
		return nil, err
	}
	svc.Context = ctx
	return svc, nil
}
