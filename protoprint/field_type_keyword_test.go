package protoprint

import (
	"bytes"
	"fmt"
	"maps"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/require"
)

// TestPrintTypeNameStartingWithFieldKeyword exercises the case where the
// relative form of a printed type reference would begin with a proto keyword
// that takes a different grammar branch at the start of a field declaration.
// Each subtest covers a keyword the spec excludes from field-type
// identifiers in regular, oneof, or extension field declarations, and
// asserts the printed type uses the leading-dot fully-qualified form.
func TestPrintTypeNameStartingWithFieldKeyword(t *testing.T) {
	t.Parallel()

	keywords := []string{
		"group",
		"optional", "required", "repeated",
		"map",
		"message", "enum", "oneof",
		"reserved", "extensions", "extend",
		"option",
	}

	for _, kw := range keywords {
		t.Run("message_field_type_in_subpackage_"+kw, func(t *testing.T) {
			t.Parallel()
			files := map[string]string{
				fmt.Sprintf("pkg/%s/dep.proto", kw): fmt.Sprintf(`syntax = "proto3";
package pkg.%s;
message X {
  string name = 1;
}`, kw),
				"pkg/root.proto": fmt.Sprintf(`syntax = "proto3";
package pkg;
import "pkg/%[1]s/dep.proto";
message Foo {
  pkg.%[1]s.X v = 1;
}`, kw),
			}
			runPrintTest(t, files, "pkg/root.proto", fmt.Sprintf(".pkg.%s.X", kw))
		})
	}

	t.Run("extendee_in_subpackage_group", func(t *testing.T) {
		t.Parallel()
		files := map[string]string{
			"pkg/group/dep.proto": `syntax = "proto2";
package pkg.group;
message Opts {
  extensions 50000 to 60000;
}`,
			"pkg/root.proto": `syntax = "proto2";
package pkg;
import "pkg/group/dep.proto";
extend pkg.group.Opts {
  optional string label = 50001;
}`,
		}
		runPrintTest(t, files, "pkg/root.proto", ".pkg.group.Opts")
	})
}

// runPrintTest compiles files, prints entry with protoprint, asserts the
// printed output contains wantTypeRef, and re-parses the printed output to
// catch other regressions.
func runPrintTest(t *testing.T, files map[string]string, entry, wantTypeRef string) {
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(files),
		}),
	}
	fds, err := compiler.Compile(t.Context(), entry)
	require.NoError(t, err, "input proto did not compile")

	var buf bytes.Buffer
	err = (&Printer{}).PrintProtoFile(fds[0], &buf)
	require.NoError(t, err, "PrintProtoFile returned error")
	printed := buf.String()

	require.Contains(t, printed, wantTypeRef,
		"printed output uses the relative form instead of leading-dot FQN")

	roundTripFiles := maps.Clone(files)
	roundTripFiles[entry] = printed

	roundTrip := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(roundTripFiles),
		}),
	}
	_, err = roundTrip.Compile(t.Context(), entry)
	require.NoError(t, err, "round-trip compile failed; printed output is not valid proto:\n%s", printed)
}
