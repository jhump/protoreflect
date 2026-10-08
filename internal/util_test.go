package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitCap(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", InitCap(""))
	assert.Equal(t, "Foo", InitCap("foo"))
	assert.Equal(t, "Foo", InitCap("Foo"))
	assert.Equal(t, "Éclair", InitCap("éclair"))
}
