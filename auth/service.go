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

	return &serviceImpl{u}, nil
}

type serviceImpl struct {
	authURL *url.URL
}

// RequireScope does a permission check for the identity represented by the token in the given context,
// to determine whether the user has a particular scope for the specified resource.
// It will return a ForbiddenError if the identity does not have the specified scope for the resource.
func (a *serviceImpl) RequireScope(ctx context.Context, resourceID, requiredScope string) error {
	client := a.createClient(ctx)
	resp, err := client.ScopesResource(goasupport.ForwardContextRequestID(ctx), authclient.ScopesResourcePath(resourceID))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.NewInternalErrorFromString(fmt.Sprintf("get resource's scope failed with error '%s'", resp.Status))
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

// createClient creates a new client to be used to call Auth service
func (a *serviceImpl) createClient(ctx context.Context) *authclient.Client {
	c := authclient.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = a.authURL.Host
	c.Scheme = a.authURL.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return c
}
