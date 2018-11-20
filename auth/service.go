package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	authclient "github.com/fabric8-services/fabric8-auth-client/auth"
	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/goasupport"

	goaclient "github.com/goadesign/goa/client"
)

type AuthService interface {
	RequireScope(ctx context.Context, resourceID, requiredScope string) error
}

func NewAuthService(authURL string) (AuthService, error) {
	u, err := url.Parse(authURL)
	if err != nil {
		return nil, err
	}

	return &auth{u}, nil
}

type auth struct {
	authURL *url.URL
}

func (a *auth) RequireScope(ctx context.Context, resourceID, requiredScope string) error {
	client := a.createClient(ctx)
	resp, err := client.ScopesResource(goasupport.ForwardContextRequestID(ctx), authclient.ScopesResourcePath(resourceID))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.NewInternalErrorFromString(fmt.Sprintf("get space's scope failed with error '%s'", resp.Status))
	}

	defer resp.Body.Close()
	scopes, _ := client.DecodeResourceScopesData(resp)
	for _, scope := range scopes.Data {
		if requiredScope == scope.ID {
			return nil
		}
	}
	return errors.NewForbiddenError(fmt.Sprintf("missing required scope '%s' on '%s' resource", requiredScope, resourceID))
}

func (a *auth) createClient(ctx context.Context) *authclient.Client {
	c := authclient.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = a.authURL.Host
	c.Scheme = a.authURL.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return c
}
