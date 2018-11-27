package auth_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-common/auth"
	testsupport "github.com/fabric8-services/fabric8-common/test"
	testauth "github.com/fabric8-services/fabric8-common/test/auth"
	testsuite "github.com/fabric8-services/fabric8-common/test/suite"
	"github.com/satori/go.uuid"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TokenContextTestSuite struct {
	testsuite.UnitTestSuite
}

func TestTokenContext(t *testing.T) {
	suite.Run(t, &TokenContextTestSuite{UnitTestSuite: testsuite.NewUnitTestSuite()})
}

func (s *TokenContextTestSuite) TestEmptyContext() {
	s.T().Run("Reading manager from empty context fails", func(t *testing.T) {
		_, err := auth.ReadManagerFromContext(context.Background())
		testsupport.AssertError(t, err, errors.New(""), "missing token manager")
	})
	s.T().Run("Locating identity in empty context fails", func(t *testing.T) {
		_, err := auth.LocateIdentity(context.Background())
		testsupport.AssertError(t, err, errors.New(""), "missing token manager")
	})
}

func (s *TokenContextTestSuite) TestServiceAccount() {
	ctx, identity, err := testauth.EmbedUserTokenInContext(nil, nil)
	require.NoError(s.T(), err)

	s.T().Run("Reading manager from context and creating context is OK", func(t *testing.T) {
		// Reading the context
		tm, err := auth.ReadManagerFromContext(ctx)
		require.NoError(t, err)
		assert.NotNil(t, tm)

		id, err := tm.Locate(ctx)
		require.NoError(t, err)
		assert.Equal(t, identity.ID, id)
		assert.NotEqual(t, id, uuid.UUID{})

		// Creating new context
		ctx := auth.ContextWithTokenManager(context.Background(), tm)
		require.NotNil(t, ctx)
		rTm, err := auth.ReadManagerFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, tm, rTm)
	})
	s.T().Run("Locating identity in context OK", func(t *testing.T) {
		id, err := auth.LocateIdentity(ctx)
		require.NoError(t, err)
		assert.Equal(t, identity.ID, id)
		assert.NotEqual(t, id, uuid.UUID{})
	})
}
