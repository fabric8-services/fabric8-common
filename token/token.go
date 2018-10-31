package token

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"sync"

	errs "github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/log"
	"github.com/fabric8-services/fabric8-common/token/jwk"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"github.com/fabric8-services/fabric8-auth-client/auth"
	"net/url"
)

const (
	// Service Account Names

	Auth         = "fabric8-auth"
	WIT          = "fabric8-wit"
	OsoProxy     = "fabric8-oso-proxy"
	Tenant       = "fabric8-tenant"
	Notification = "fabric8-notification"
	JenkinsIdler = "fabric8-jenkins-idler"
	JenkinsProxy = "fabric8-jenkins-proxy"

	devModeKeyID = "test-key"
)

var defaultManager Manager
var defaultOnce sync.Once
var defaultErr error

// DefaultManager creates the default manager if it has not created yet.
// This function must be called in main to make sure the default manager is created during service startup.
// It will try to create the default manager only once even if called multiple times.
func DefaultManager(config ManagerConfiguration) (Manager, error) {
	defaultOnce.Do(func() {
		defaultManager, defaultErr = NewManager(config)
	})
	return defaultManager, defaultErr
}

type Configuration interface {
	GetAuthServiceURL() string
}

// ManagerConfiguration represents configuration needed to construct a token manager
type ManagerConfiguration interface {
	Configuration
	GetDevModePrivateKey() []byte
}

// TokenClaims represents access token claims
type TokenClaims struct {
	Name          string         `json:"name"`
	Username      string         `json:"preferred_username"`
	GivenName     string         `json:"given_name"`
	FamilyName    string         `json:"family_name"`
	Email         string         `json:"email"`
	EmailVerified bool           `json:"email_verified"`
	Company       string         `json:"company"`
	SessionState  string         `json:"session_state"`
	Approved      bool           `json:"approved"`
	Permissions   *[]Permissions `json:"permissions"`
	jwt.StandardClaims
}

// Permissions represents a "permissions" claim in the AuthorizationPayload
type Permissions struct {
	ResourceSetName *string  `json:"resource_set_name"`
	ResourceSetID   *string  `json:"resource_set_id"`
	Scopes          []string `json:"scopes"`
	Expiry          int64    `json:"exp"`
}

// Parser parses a token and exposes the public keys for the Goa JWT middleware.
type Parser interface {
	Parse(ctx context.Context, tokenString string) (*jwt.Token, error)
	PublicKeys() []*rsa.PublicKey
}

// Manager generate and find auth token information
type Manager interface {
	Parser
	Locate(ctx context.Context) (uuid.UUID, error)
	ParseToken(ctx context.Context, tokenString string) (*TokenClaims, error)
	ParseTokenWithMapClaims(ctx context.Context, tokenString string) (jwt.MapClaims, error)
	PublicKey(keyID string) *rsa.PublicKey
	AddLoginRequiredHeader(rw http.ResponseWriter)
}

type tokenManager struct {
	publicKeysMap map[string]*rsa.PublicKey
	publicKeys    []*jwk.PublicKey
	config        ManagerConfiguration
}

// NewManager returns a new token Manager for handling tokens
func NewManager(config ManagerConfiguration, options ...httpsupport.HTTPClientOption) (Manager, error) {

	// Load public keys from Auth service and add them to the manager
	tm := &tokenManager{
		publicKeysMap: map[string]*rsa.PublicKey{},
	}
	tm.config = config

	authURL := httpsupport.RemoveTrailingSlashToURL(config.GetAuthServiceURL())
	keysEndpoint := fmt.Sprintf("%s%s", authURL, auth.KeysTokenPath())
	remoteKeys, err := jwk.FetchKeys(keysEndpoint, options...)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":      err,
			"keys_url": keysEndpoint,
		}, "unable to load public keys from auth service")
		return nil, errors.New("unable to load public keys from auth service")
	}
	for _, remoteKey := range remoteKeys {
		tm.publicKeysMap[remoteKey.KeyID] = remoteKey.Key
		tm.publicKeys = append(tm.publicKeys, &jwk.PublicKey{KeyID: remoteKey.KeyID, Key: remoteKey.Key})
		log.Info(nil, map[string]interface{}{
			"kid": remoteKey.KeyID,
		}, "Public key added")
	}

	devModePrivateKey := config.GetDevModePrivateKey()
	if devModePrivateKey != nil {
		log.Info(nil, map[string]interface{}{}, "adding dev-mode private key, too...")
		// Add the public key which will be used to verify tokens generated in Dev Mode
		rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM(devModePrivateKey)
		if err != nil {
			return nil, err
		}
		tm.publicKeysMap["test-key"] = &rsaKey.PublicKey
		tm.publicKeys = append(tm.publicKeys, &jwk.PublicKey{KeyID: "test-key", Key: &rsaKey.PublicKey})
		log.Info(nil, map[string]interface{}{
			"kid": devModeKeyID,
		}, "Public key added")
	}
	return tm, nil
}

// NewManagerWithPublicKey returns a new token Manager for handling tokens with the only public key
func NewManagerWithPublicKey(id string, key *rsa.PublicKey, config ManagerConfiguration) Manager {
	return &tokenManager{
		publicKeysMap: map[string]*rsa.PublicKey{id: key},
		publicKeys:    []*jwk.PublicKey{{KeyID: id, Key: key}},
		config:        config,
	}
}

// ParseToken parses token claims
func (mgm *tokenManager) ParseToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, mgm.keyFunction(ctx))
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(*TokenClaims)
	if token.Valid {
		return claims, nil
	}
	return nil, errors.WithStack(errors.New("token is not valid"))
}

// ParseTokenWithMapClaims parses token claims
func (mgm *tokenManager) ParseTokenWithMapClaims(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, mgm.keyFunction(ctx))
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(jwt.MapClaims)
	if token.Valid {
		return claims, nil
	}
	return nil, errors.WithStack(errors.New("token is not valid"))
}

func (mgm *tokenManager) keyFunction(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		kid := token.Header["kid"]
		if kid == nil {
			log.Error(ctx, map[string]interface{}{}, "There is no 'kid' header in the token")
			return nil, errors.New("There is no 'kid' header in the token")
		}
		key := mgm.PublicKey(fmt.Sprintf("%s", kid))
		if key == nil {
			log.Error(ctx, map[string]interface{}{
				"kid": kid,
			}, "There is no public key with such ID")
			return nil, errors.New(fmt.Sprintf("There is no public key with such ID: %s", kid))
		}
		return key, nil
	}
}

func (mgm *tokenManager) Locate(ctx context.Context) (uuid.UUID, error) {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return uuid.UUID{}, errors.New("Missing token") // TODO, make specific tokenErrors
	}
	id := token.Claims.(jwt.MapClaims)["sub"]
	if id == nil {
		return uuid.UUID{}, errors.New("Missing sub")
	}
	idTyped, err := uuid.FromString(id.(string))
	if err != nil {
		return uuid.UUID{}, errors.New("uuid not of type string")
	}
	return idTyped, nil
}

// PublicKey returns the public key by the ID
func (mgm *tokenManager) PublicKey(keyID string) *rsa.PublicKey {
	return mgm.publicKeysMap[keyID]
}

// PublicKeys returns all the public keys
func (mgm *tokenManager) PublicKeys() []*rsa.PublicKey {
	keys := make([]*rsa.PublicKey, 0, len(mgm.publicKeysMap))
	for _, key := range mgm.publicKeys {
		keys = append(keys, key.Key)
	}
	return keys
}

func (mgm *tokenManager) Parse(ctx context.Context, tokenString string) (*jwt.Token, error) {
	keyFunc := mgm.keyFunction(ctx)
	jwtToken, err := jwt.Parse(tokenString, keyFunc)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to parse token")
		return nil, errs.NewUnauthorizedError(err.Error())
	}
	return jwtToken, nil
}

// AddLoginRequiredHeader adds "WWW-Authenticate: LOGIN" header to the response
func (mgm *tokenManager) AddLoginRequiredHeader(rw http.ResponseWriter) {
	rw.Header().Add("Access-Control-Expose-Headers", "WWW-Authenticate")
	loginURL := httpsupport.AddTrailingSlashToURL(mgm.config.GetAuthServiceURL()) + "api/login"
	rw.Header().Set("WWW-Authenticate", fmt.Sprintf("LOGIN url=%s, description=\"re-login is required\"", loginURL))
}

// IsSpecificServiceAccount checks if the request is done by a service account listed in the names param
// based on the JWT Token provided in context
func IsSpecificServiceAccount(ctx context.Context, names ...string) bool {
	accountName, ok := extractServiceAccountName(ctx)
	if !ok {
		return false
	}
	for _, name := range names {
		if accountName == name {
			return true
		}
	}
	return false
}

// IsServiceAccount checks if the request is done by a
// Service account based on the JWT Token provided in context
func IsServiceAccount(ctx context.Context) bool {
	_, ok := extractServiceAccountName(ctx)
	return ok
}

func extractServiceAccountName(ctx context.Context) (string, bool) {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return "", false
	}
	accountName := token.Claims.(jwt.MapClaims)["service_accountname"]
	if accountName == nil {
		return "", false
	}
	accountNameTyped, isString := accountName.(string)
	return accountNameTyped, isString
}

// CheckClaims checks if all the required claims are present in the access token
func CheckClaims(claims *TokenClaims) error {
	if claims.Subject == "" {
		return errors.New("subject claim not found in token")
	}
	_, err := uuid.FromString(claims.Subject)
	if err != nil {
		return errors.New("subject claim from token is not UUID " + err.Error())
	}
	if claims.Username == "" {
		return errors.New("username claim not found in token")
	}
	if claims.Email == "" {
		return errors.New("email claim not found in token")
	}
	return nil
}

func ServiceAccountToken(ctx context.Context, config Configuration, clientID, clientSecret string, options ...httpsupport.HTTPClientOption) (token string, err error) {
	authURL := config.GetAuthServiceURL()
	u, err := url.Parse(authURL)
	if err != nil {
		return "", err
	}

	httpClient := http.DefaultClient
	for _, opt := range options {
		opt(httpClient)
	}

	client := auth.New(&httpsupport.HTTPClientDoer{
		HTTPClient: httpClient})
	client.Host = u.Host
	client.Scheme = u.Scheme

	path := auth.ExchangeTokenPath()
	payload := &auth.TokenExchange{
		ClientID:     clientID,
		ClientSecret: &clientSecret,
		GrantType:    "client_credentials",
	}
	contentType := "application/x-www-form-urlencoded"

	res, err := client.ExchangeToken(ctx, path, payload, contentType)
	if err != nil {
		return "", errors.Wrapf(err, "error while doing the request")
	}
	defer func() {
		httpsupport.CloseResponse(res)
	}()

	if res.StatusCode >= 400 {
		log.Error(ctx, map[string]interface{}{
			"response_status": res.Status,
			"response_body":   res.Body,
			"url":             authURL,
		}, "failed to obtain token from auth server")
		return "", fmt.Errorf("failed to obtain token from auth server %q", authURL)
	}

	oauthToken, err := client.DecodeOauthToken(res)
	if err != nil {
		return "", errors.Wrapf(err, "error from server %q", authURL)
	}

	if oauthToken.AccessToken == nil || *oauthToken.AccessToken == "" {
		return "", fmt.Errorf("received empty token from server %q", authURL)
	}

	return *oauthToken.AccessToken, nil
}

// InjectTokenManager is a middleware responsible for setting up tokenManager in the context for every request.
func InjectTokenManager(tokenManager Manager) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			ctxWithTM := ContextWithTokenManager(ctx, tokenManager)
			return h(ctxWithTM, rw, req)
		}
	}
}
