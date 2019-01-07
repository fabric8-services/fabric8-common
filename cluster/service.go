package cluster

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	goaclient "github.com/goadesign/goa/client"

	"github.com/fabric8-services/fabric8-common/errors"
	"github.com/fabric8-services/fabric8-common/goasupport"

	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
)

type Service interface {
	ClustersUser(ctx context.Context) (*clusterclient.ClusterList, error)
}

func NewClusterService(clusterURL string) (Service, error) {
	url, err := url.Parse(clusterURL)
	if err != nil {
		return nil, err
	}
	return &service{url}, nil
}

type service struct {
	clusterURL *url.URL
}

func (s *service) createClient(ctx context.Context) *clusterclient.Client {
	c := clusterclient.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = s.clusterURL.Host
	c.Scheme = s.clusterURL.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return c
}

func (s *service) ClustersUser(ctx context.Context) (*clusterclient.ClusterList, error) {
	client := s.createClient(ctx)
	resp, err := client.ClustersUser(goasupport.ForwardContextRequestID(ctx), clusterclient.ClustersUserPath())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewInternalErrorFromString(fmt.Sprintf("get clusters for user failed with status '%s'", resp.Status))
	}
	defer resp.Body.Close()
	return client.DecodeClusterList(resp)
}
