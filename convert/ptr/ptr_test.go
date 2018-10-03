package ptr_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-common/convert/ptr"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	var strVal string = "mystring"
	var strPtr *string = ptr.String(strVal)
	assert.Equal(t, strVal, *strPtr)
}
