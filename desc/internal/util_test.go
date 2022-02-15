package internal_test

import (
	"github.com/jhump/protoreflect/desc/internal"
	"testing"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestCreatePrefixList(t *testing.T) {
	list := internal.CreatePrefixList("")
	testutil.Eq(t, []string{""}, list)

	list = internal.CreatePrefixList("pkg")
	testutil.Eq(t, []string{"pkg", ""}, list)

	list = internal.CreatePrefixList("fully.qualified.pkg.name")
	testutil.Eq(t, []string{"fully.qualified.pkg.name", "fully.qualified.pkg", "fully.qualified", "fully", ""}, list)
}
