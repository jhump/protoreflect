package grpcreflect

import "testing"

func eq(t *testing.T, expected, actual interface{}) bool {
	if expected != actual {
		t.Errorf("Expecting %v, got %v", expected, actual)
		return false
	}
	return true
}

func ok(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
}