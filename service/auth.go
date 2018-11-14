package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	authclient "github.com/fabric8-services/fabric8-auth-client/auth"
	goaclient "github.com/goadesign/goa/client"
	"github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

type Auth interface {
	CheckSpaceScope(ctx context.Context, spaceID, requiredScope string) (bool, error)
}

func NewAuthService(hostURL string) (Auth, error) {
	u, err := url.Parse(hostURL)
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
	fmt.Println("vn: in do, &ctx", &ctx)
	fmt.Println("vn: in do, ctx", ctx)
	token := jwt.ContextJWT(ctx)
	fmt.Println("vn: in do, token:", token)
	if token != nil {
		fmt.Printf("vn: token.Raw:%s\n", token.Raw)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Raw))
	}
	return d.target.Do(ctx, req)
}

type auth struct {
	*authclient.Client
}

func (c *auth) CheckSpaceScope(ctx context.Context, spaceID, requiredScope string) (bool, error) {
	fmt.Println("vn: in check, &ctx", &ctx)
	fmt.Println("vn: in check, ctx", ctx)
	token := jwt.ContextJWT(ctx)
	fmt.Println("vn: in check, token:", token)
	resp, err := c.Client.ScopesResource(ctx, authclient.ScopesResourcePath(spaceID))
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, errs.Errorf("get space's scope failed with error '%s'", resp.Status)
	}

	defer resp.Body.Close()
	scopes, err := c.Client.DecodeResourceScopesData(resp)
	for _, scope := range scopes.Data {
		if requiredScope == scope.ID {
			return true, nil
		}
	}
	return false, nil
}
