package convert_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/convert"
	"github.com/fabric8-services/fabric8-common/resource"
	"github.com/stretchr/testify/assert"
)

// foo implements the Equaler interface
type foo struct{}

// Ensure foo implements the Equaler interface
var _ convert.Equaler = foo{}
var _ convert.Equaler = (*foo)(nil)

func (f foo) Equal(u convert.Equaler) bool {
	_, ok := u.(foo)
	if !ok {
		return false
	}
	return true
}

func TestDummyEqualerEqual(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := convert.DummyEqualer{}
	b := convert.DummyEqualer{}

	// Test for type difference
	assert.False(t, a.Equal(foo{}))

	// Test for equality
	assert.True(t, a.Equal(b))
}
