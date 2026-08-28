package grpcreflect

import (
	"testing"

	reflectv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/internal"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestFileByFileNameForWellKnownProtos(t *testing.T) {
	wellKnownProtos := map[string][]string{
		"google/protobuf/any.proto":             {"google.protobuf.Any"},
		"google/protobuf/api.proto":             {"google.protobuf.Api", "google.protobuf.Method", "google.protobuf.Mixin"},
		"google/protobuf/descriptor.proto":      {"google.protobuf.FileDescriptorSet", "google.protobuf.DescriptorProto"},
		"google/protobuf/duration.proto":        {"google.protobuf.Duration"},
		"google/protobuf/empty.proto":           {"google.protobuf.Empty"},
		"google/protobuf/field_mask.proto":      {"google.protobuf.FieldMask"},
		"google/protobuf/source_context.proto":  {"google.protobuf.SourceContext"},
		"google/protobuf/struct.proto":          {"google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.NullValue"},
		"google/protobuf/timestamp.proto":       {"google.protobuf.Timestamp"},
		"google/protobuf/type.proto":            {"google.protobuf.Type", "google.protobuf.Field", "google.protobuf.Syntax"},
		"google/protobuf/wrappers.proto":        {"google.protobuf.DoubleValue", "google.protobuf.Int32Value", "google.protobuf.StringValue"},
		"google/protobuf/compiler/plugin.proto": {"google.protobuf.compiler.CodeGeneratorRequest"},
	}

	for file, types := range wellKnownProtos {
		fd, err := client.FileByFilename(file)
		testutil.Ok(t, err)
		testutil.Eq(t, file, fd.GetName())
		for _, typ := range types {
			d := fd.FindSymbol(typ)
			testutil.Require(t, d != nil)
		}

		// also try loading via alternate name
		file = internal.StdFileAliases[file]
		if file == "" {
			// not a file that has a known alternate, so nothing else to check...
			continue
		}
		fd, err = client.FileByFilename(file)
		testutil.Ok(t, err)
		testutil.Eq(t, file, fd.GetName())
		for _, typ := range types {
			d := fd.FindSymbol(typ)
			testutil.Require(t, d != nil)
		}
	}
}

// TestFileByFilenameAlias makes sure that asking for a file by an alias returns
// a descriptor named with that alias, even when the server only knows the file
// by its canonical name. The name matters: it is the name that importing files
// use to refer to this one, so a mismatch means the importer cannot be linked.
func TestFileByFilenameAlias(t *testing.T) {
	const canonical = "google/protobuf/any.proto"
	alias := internal.StdFileAliases[canonical]
	testutil.Require(t, alias != "", "no known alias for %s", canonical)

	// The server knows the file only by its canonical name. The contents don't
	// matter here; only the names do.
	newServer := func() fakeReflectionServer {
		return fakeReflectionServer{
			fileFor: func(filename string) *descriptorpb.FileDescriptorProto {
				switch filename {
				case canonical:
					return newFileProto(canonical)
				case "importer.proto":
					return newFileProto("importer.proto", alias)
				default:
					// Notably, this includes the alias itself: the server has
					// never heard of it, so the client must fall back to
					// requesting the canonical name.
					return nil
				}
			},
		}
	}

	t.Run("requested directly", func(t *testing.T) {
		cc := startFakeReflectionServer(t, newServer())
		cr := NewClientV1(t.Context(), reflectv1.NewServerReflectionClient(cc))
		t.Cleanup(cr.Reset)

		fd, err := cr.FileByFilename(alias)
		testutil.Ok(t, err)
		testutil.Eq(t, alias, fd.GetName())
	})

	t.Run("requested as a dependency", func(t *testing.T) {
		cc := startFakeReflectionServer(t, newServer())
		cr := NewClientV1(t.Context(), reflectv1.NewServerReflectionClient(cc))
		t.Cleanup(cr.Reset)

		// importer.proto imports the alias, so resolving it requires the
		// downloaded dependency to come back named as the alias, not as the
		// canonical name the server was actually asked for.
		fd, err := cr.FileByFilename("importer.proto")
		testutil.Ok(t, err)
		deps := fd.GetDependencies()
		testutil.Eq(t, 1, len(deps))
		testutil.Eq(t, alias, deps[0].GetName())
	})
}
