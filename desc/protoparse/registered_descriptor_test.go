package protoparse

import "testing"

func TestGetRegisteredDescriptors(t *testing.T) {
	// We simply test that the descriptor retrieval succeeds, without doing
	// too much in the way of verifying what is in them. It seems vanishingly
	// unlikely that this would succeed but with the wrong descriptor or similar.
	tests := []struct {
		testName  string
		filenames []string
	}{
		{
			"WellKnownFiles",
			[]string{"google/protobuf/timestamp.proto", "google/protobuf/duration.proto"},
		},
		{
			"ImportsWellKnownFiles",
			[]string{"desc_test_wellknowntypes.proto"},
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			descriptors, err := GetRegisteredDescriptors(test.filenames...)
			if err != nil {
				t.Error(err)
			}
			for _, filename := range test.filenames {
				if _, ok := descriptors[filename]; !ok {
					t.Errorf("Expected file %q.", filename)
				}
			}
		})
	}
}

func TestGetRegisteredDescriptorsMissing(t *testing.T) {
	_, err := GetRegisteredDescriptors("bogus.proto")
	if err == nil {
		t.Errorf("GetRegisteredDescriptors returned a descriptor for a missing file.")
	}
}
