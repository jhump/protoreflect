package protoresolve_test

import (
	"testing"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

func testResolver(t *testing.T, res protoresolve.Resolver) {
	// Verify that the resolver can find the known file
	path := "desc_test1.proto"

	fd, err := res.FindFileByPath(path)
	if err != nil {
		t.Errorf("unexpected error finding %s: %v", path, err)
	} else if fd == nil {
		t.Errorf("expected to find %s, but got nil", path)
	} else if fd.Path() != path {
		t.Errorf("expected found descriptor to have path %s, got %q", path, fd.Path())
	}

	// Verify that the resolver returns an error for a missing file
	_, err = res.FindFileByPath("does_not_exist.proto")
	if err == nil {
		t.Error("expected error finding does_not_exist.proto, but got nil")
	}
}
