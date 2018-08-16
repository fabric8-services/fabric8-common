package test

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/fabric8-services/fabric8-common/configuration"
	"github.com/fabric8-services/fabric8-common/token"
	"github.com/fabric8-services/fabric8-common/token/tokencontext"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	jwtgoa "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/satori/go.uuid"
)

var config = configurationData()
var TokenManager = newManager()

type Identity struct {
	ID       uuid.UUID
	Username string
}

func NewIdentity() *Identity {
	return &Identity{
		ID:       uuid.NewV4(),
		Username: uuid.NewV4().String(),
	}
}

// EmbedUserTokenInContext generates a token for the given identity and embed it into the context along with token manager
func EmbedUserTokenInContext(ctx context.Context, identity *Identity) context.Context {
	if identity == nil {
		identity = NewIdentity()
	}
	_, token := GenerateSignedUserToken(identity)
	return embedTokenInContext(ctx, token)
}

// GenerateSignedServiceAccountToken generates a token for the given identity and embed it into the context along with token manager
func EmbedServiceAccountTokenInContext(ctx context.Context, identity *Identity) context.Context {
	_, token := GenerateSignedServiceAccountToken(identity)
	return embedTokenInContext(ctx, token)
}

func embedTokenInContext(ctx context.Context, token *jwt.Token) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	jwtCtx := jwtgoa.WithJWT(ctx, token)
	jwtCtx = ContextWithRequest(jwtCtx)
	return tokencontext.ContextWithTokenManager(jwtCtx, TokenManager)
}

// GenerateSignedUserToken generates a JWT token and signs it using the default private key
func GenerateSignedUserToken(identity *Identity) (string, *jwt.Token) {
	token := generateUserToken(identity)
	tokenStr := signToken(token)

	return tokenStr, token
}

func GenerateSignedServiceAccountToken(identity *Identity) (string, *jwt.Token) {
	token := generateServiceAccountToken(identity)
	tokenStr := signToken(token)

	return tokenStr, token
}

func signToken(token *jwt.Token) string {
	key, _ := privateKey()
	tokenStr, err := token.SignedString(key)
	if err != nil {
		panic(err.Error())
	}
	token.Raw = tokenStr

	return tokenStr
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
	claims["sub"] = identity.Username
	claims["email"] = identity.Username

	token.Header["kid"] = "test-key"

	return token
}

// generateServiceAccountToken generates a JWT token
func generateServiceAccountToken(identity *Identity) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["service_accountname"] = identity.Username
	claims["sub"] = identity.ID
	claims["jti"] = uuid.NewV4().String()
	claims["iat"] = time.Now().Unix()

	token.Header["kid"] = "test-key"

	return token
}

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

func ContextWithTokenAndRequestID() (context.Context, *Identity, string) {
	identity := NewIdentity()
	ctx := EmbedUserTokenInContext(nil, identity)
	reqID := uuid.NewV4().String()
	ctx = client.SetContextRequestID(ctx, reqID)

	return ctx, identity, reqID
}

func ContextWithTokenManager() context.Context {
	return tokencontext.ContextWithTokenManager(context.Background(), TokenManager)
}

func configurationData() *configuration.ConfigurationData {
	config, err := configuration.GetConfigurationData()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}
	return config
}

func newManager() token.Manager {
	tm, err := token.NewManager(config)
	if err != nil {
		panic("failed to create token manager: " + err.Error())
	}
	return tm
}

func privateKey() (*rsa.PrivateKey, string) {
	key := config.GetDevModePrivateKey()
	pk, err := jwt.ParseRSAPrivateKeyFromPEM(key)
	if err != nil {
		panic(err.Error())
	}
	return pk, "test-key"
}

// ServiceAsUser creates a new service and fill the context with input Identity
func ServiceAsUser(serviceName string, identity *Identity) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = EmbedUserTokenInContext(nil, identity)
	return svc
}

// UnsecuredService creates a new service with token manager injected by without any identity in context
func UnsecuredService(serviceName string) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = tokencontext.ContextWithTokenManager(svc.Context, TokenManager)
	return svc
}

// ServiceAsServiceAccountUser generates the minimal service needed to satisfy the condition of being a service account.
func ServiceAsServiceAccountUser(serviceName string, identity *Identity) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = EmbedServiceAccountTokenInContext(nil, identity)
	return svc
}
