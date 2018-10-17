package httpsupport_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/httpsupport"
	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/stretchr/testify/require"
)

func TestValidateEmail(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		isValid, err := httpsupport.ValidateEmail("a@a.com")
		require.NoError(t, err)
		require.True(t, isValid)
	})

	t.Run("fail", func(t *testing.T) {
		isValid, err := httpsupport.ValidateEmail("a.a@com")
		require.NoError(t, err)
		require.False(t, isValid)
	})

}
