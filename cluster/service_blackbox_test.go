package cluster_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-common/errors"

	testsupport "github.com/fabric8-services/fabric8-common/test"

	clusterclient "github.com/fabric8-services/fabric8-cluster-client/cluster"
	"github.com/fabric8-services/fabric8-common/cluster"
	testauth "github.com/fabric8-services/fabric8-common/test/auth"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"
	"github.com/goadesign/goa/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	gock "gopkg.in/h2non/gock.v1"
)

var clusterURL = "https://cluster.prod-preview.openshift.io"

type ClusterServiceTestSuite struct {
	testsuite.UnitTestSuite
	clusterSvc cluster.Service
}

func (s *ClusterServiceTestSuite) SetupSuite() {
	s.UnitTestSuite.SetupSuite()
	var err error
	s.clusterSvc, err = cluster.NewClusterService(clusterURL)
	require.NoError(s.T(), err)
}

func (s *ClusterServiceTestSuite) TearDownSuite() {
	gock.Disable()
}

func TestClusterServiceTestSuite(t *testing.T) {
	suite.Run(t, &ClusterServiceTestSuite{UnitTestSuite: testsuite.NewUnitTestSuite()})
}

func (s *ClusterServiceTestSuite) TestClustersUser() {
	ctx, _, token, requestID, err := testauth.ContextWithTokenAndRequestID()
	require.NoError(s.T(), err)

	s.T().Run("ok", func(t *testing.T) {
		wantResp := &clusterclient.ClusterList{
			Data: []*clusterclient.ClusterData{
				&clusterclient.ClusterData{
					Name:   "cluster1",
					APIURL: "http://cluster1.com",
				},
				&clusterclient.ClusterData{
					Name:   "cluster2",
					APIURL: "http://cluster2.com",
				},
			},
		}
		gock.New(clusterURL).Get(clusterclient.ClustersUserPath()).
			MatchHeader("Authorization", "Bearer "+token).
			MatchHeader("X-Request-Id", requestID).
			Reply(200).JSON(wantResp)

		gotResp, err := s.clusterSvc.ClustersUser(ctx)
		assert.NoError(t, err)
		checkClusterList(t, wantResp, gotResp, len(wantResp.Data))
	})

	s.T().Run("unauthorized", func(t *testing.T) {
		gock.New(clusterURL).Get(clusterclient.ClustersUserPath()).
			MatchHeader("X-Request-Id", requestID).
			Reply(401)

		ctx := context.Background()
		ctx = client.SetContextRequestID(ctx, requestID)
		_, err := s.clusterSvc.ClustersUser(ctx)
		testsupport.AssertError(t, err, errors.InternalError{}, "get clusters for user failed with status '401 Unauthorized'")
	})

	s.T().Run("internal_error", func(t *testing.T) {
		gock.New(clusterURL).Get(clusterclient.ClustersUserPath()).
			MatchHeader("Authorization", "Bearer "+token).
			MatchHeader("X-Request-Id", requestID).
			Reply(500)

		_, err := s.clusterSvc.ClustersUser(ctx)
		testsupport.AssertError(t, err, errors.InternalError{}, "get clusters for user failed with status '500 Internal Server Error'")
	})
}

func checkClusterList(t *testing.T, want *clusterclient.ClusterList, got *clusterclient.ClusterList, wantLen int) {
	t.Helper()
	require.NotNil(t, want)
	assert.NotNil(t, got)
	assert.Equal(t, wantLen, len(got.Data))
	for i := 0; i < wantLen; i++ {
		checkClusterData(t, want.Data[i], got.Data[i])
	}
}

func checkClusterData(t *testing.T, want *clusterclient.ClusterData, got *clusterclient.ClusterData) {
	t.Helper()
	require.NotNil(t, want)
	assert.NotNil(t, got)
	assert.Equal(t, want.Name, got.Name)
	assert.Equal(t, want.APIURL, got.APIURL)
}
