package test

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/fabric8-services/fabric8-common/token"
	"github.com/fabric8-services/fabric8-common/token/tokencontext"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
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
	return embedTokenInContext(ctx, token, tm), nil
}

// EmbedServiceAccountTokenInContext generates a token for the given identity and embed it into the context along with token manager
func EmbedServiceAccountTokenInContext(ctx context.Context, identity *Identity, tm token.Manager, config token.ManagerConfiguration) (context.Context, error) {
	_, token, err := GenerateSignedServiceAccountToken(identity, config)
	if err != nil {
		return nil, err
	}
	return embedTokenInContext(ctx, token, tm), nil
}

func embedTokenInContext(ctx context.Context, token *jwt.Token, tm token.Manager) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	jwtCtx := jwtgoa.WithJWT(ctx, token)
	jwtCtx = ContextWithRequest(jwtCtx)
	return tokencontext.ContextWithTokenManager(jwtCtx, tm)
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
func ContextWithRequest(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	u := &url.URL{
		Scheme: "https",
		Host:   "cluster.openshift.io",
	}
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	return goa.NewContext(goa.WithAction(ctx, "Test"), rw, req, url.Values{})
}

// ContextWithTokenAndRequestID returns a context with an embedded user token
func ContextWithTokenAndRequestID(tm token.Manager, config token.ManagerConfiguration) (context.Context, *Identity, string, error) {
	identity := NewIdentity()
	ctx, err := EmbedUserTokenInContext(nil, identity, tm, config)
	if err != nil {
		return nil, nil, "", err
	}
	reqID := uuid.NewV4().String()
	ctx = client.SetContextRequestID(ctx, reqID)

	return ctx, identity, reqID, nil
}

// ContextWithTokenManager returns a context with the given token manager
func ContextWithTokenManager(tm token.Manager) context.Context {
	return tokencontext.ContextWithTokenManager(context.Background(), tm)
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
	ctx, err := EmbedUserTokenInContext(nil, identity, tm, config)
	if err != nil {
		return nil, err
	}
	svc.Context = ctx
	return svc, nil
}

// UnsecuredService creates a new service with token manager injected by without any identity in context
func UnsecuredService(serviceName string, tm token.Manager) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = tokencontext.ContextWithTokenManager(svc.Context, tm)
	return svc
}

// ServiceAsServiceAccountUser generates the minimal service needed to satisfy the condition of being a service account.
func ServiceAsServiceAccountUser(serviceName string, identity *Identity, tm token.Manager, config token.ManagerConfiguration) (*goa.Service, error) {
	svc := goa.New(serviceName)
	ctx, err := EmbedServiceAccountTokenInContext(nil, identity, tm, config)
	if err != nil {
		return nil, err
	}
	svc.Context = ctx
	return svc, nil
}
