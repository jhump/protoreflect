package internal

import (
	"testing"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestCreatePrefixList(t *testing.T) {
	list := CreatePrefixList("")
	testutil.Eq(t, []string{""}, list)

	list = CreatePrefixList("pkg")
	testutil.Eq(t, []string{"pkg", ""}, list)

	list = CreatePrefixList("fully.qualified.pkg.name")
	testutil.Eq(t, []string{"fully.qualified.pkg.name", "fully.qualified.pkg", "fully.qualified", "fully", ""}, list)
}
