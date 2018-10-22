package httpsupport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSingleJoiningSlash(t *testing.T) {
	assert.Equal(t, "abc/xyz", singleJoiningSlash("abc", "xyz"))
	assert.Equal(t, "abc/xyz", singleJoiningSlash("abc", "/xyz"))
}
