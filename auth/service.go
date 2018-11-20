package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	authclient "github.com/fabric8-services/fabric8-auth-client/auth"
	"github.com/fabric8-services/fabric8-common/errors"

	goaclient "github.com/goadesign/goa/client"
	"github.com/goadesign/goa/middleware/security/jwt"
)

type AuthService interface {
	RequireScope(ctx context.Context, resourceID, requiredScope string) error
}

func NewAuthService(authURL string) (AuthService, error) {
	u, err := url.Parse(authURL)
	if err != nil {
		return nil, err
	}

	client := http.Client{}
	c := authclient.New(&doer{
		target: goaclient.HTTPClientDoer(&client),
	})
	c.Host = u.Host
	c.Scheme = u.Scheme
	return &auth{c}, nil
}

type doer struct {
	target goaclient.Doer
}

func (d *doer) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	token := jwt.ContextJWT(ctx)
	if token != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Raw))
	}
	return d.target.Do(ctx, req)
}

type auth struct {
	*authclient.Client
}

func (a *auth) RequireScope(ctx context.Context, resourceID, requiredScope string) error {
	resp, err := a.Client.ScopesResource(ctx, authclient.ScopesResourcePath(resourceID))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.NewInternalErrorFromString(fmt.Sprintf("get space's scope failed with error '%s'", resp.Status))
	}

	defer resp.Body.Close()
	scopes, _ := a.Client.DecodeResourceScopesData(resp)
	for _, scope := range scopes.Data {
		if requiredScope == scope.ID {
			return nil
		}
	}
	return errors.NewForbiddenError(fmt.Sprintf("missing required scope '%s' on '%s' resource", requiredScope, resourceID))
}
