package ptr_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/convert/ptr"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	var strVal string
	strVal = "mystring"

	var strPtr *string
	strPtr = ptr.String(strVal)

	assert.Equal(t, strVal, *strPtr)
}
