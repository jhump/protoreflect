// Package internal contains some code that should not be exported but needs to
// be shared across more than one of the protoreflect sub-packages.
package internal

// These are standard protos included with protoc, but older versions of their
// respective packages registered them using incorrect paths.
var StdFileAliases = map[string]string{
	// Files for the github.com/golang/protobuf/ptypes package at one point were
	// registered using the path where the proto files are mirrored in GOPATH,
	// inside the golang/protobuf repo.
	// (Fixed as of https://github.com/golang/protobuf/pull/412)
	"google/protobuf/any.proto":       "github.com/golang/protobuf/ptypes/any/any.proto",
	"google/protobuf/duration.proto":  "github.com/golang/protobuf/ptypes/duration/duration.proto",
	"google/protobuf/empty.proto":     "github.com/golang/protobuf/ptypes/empty/empty.proto",
	"google/protobuf/struct.proto":    "github.com/golang/protobuf/ptypes/struct/struct.proto",
	"google/protobuf/timestamp.proto": "github.com/golang/protobuf/ptypes/timestamp/timestamp.proto",
	"google/protobuf/wrappers.proto":  "github.com/golang/protobuf/ptypes/wrappers/wrappers.proto",
	// Files for the google.golang.org/genproto/protobuf package at one point
	// were registered with an anomalous "src/" prefix.
	// (Fixed as of https://github.com/google/go-genproto/pull/31)
	"google/protobuf/api.proto":            "src/google/protobuf/api.proto",
	"google/protobuf/field_mask.proto":     "src/google/protobuf/field_mask.proto",
	"google/protobuf/source_context.proto": "src/google/protobuf/source_context.proto",
	"google/protobuf/type.proto":           "src/google/protobuf/type.proto",

	// Other standard files (descriptor.proto and compiler/plugin.proto) are
	// registered correctly, so we don't need rules for them here.
}

func init() {
	// We provide aliasing in both directions, to support files with the
	// proper import path linked against older versions of the generated
	// files AND files that used the aliased import path but linked against
	// newer versions of the generated files (which register with the
	// correct path).

	// Get all files defined above
	keys := make([]string, 0, len(StdFileAliases))
	for k := range StdFileAliases {
		keys = append(keys, k)
	}
	// And add inverse mappings
	for _, k := range keys {
		alias := StdFileAliases[k]
		StdFileAliases[alias] = k
	}
}
