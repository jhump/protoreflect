package protoparse

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	_ "github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestSimpleLink(t *testing.T) {
	fds, err := Parser{ImportPaths: []string{"../../internal/testprotos"}}.ParseFiles("desc_test_complex.proto")
	testutil.Ok(t, err)

	b, err := ioutil.ReadFile("../../internal/testprotos/desc_test_complex.protoset")
	testutil.Ok(t, err)
	var files dpb.FileDescriptorSet
	err = proto.Unmarshal(b, &files)
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(files.File[0], fds[0].AsProto()), "linked descriptor did not match output from protoc:\nwanted: %s\ngot: %s", toString(files.File[0]), toString(fds[0].AsProto()))
}

func TestMultiFileLink(t *testing.T) {
	for _, name := range []string{"desc_test2.proto", "desc_test_defaults.proto", "desc_test_field_types.proto", "desc_test_options.proto", "desc_test_proto3.proto", "desc_test_wellknowntypes.proto"} {
		fds, err := Parser{ImportPaths: []string{"../../internal/testprotos"}}.ParseFiles(name)
		testutil.Ok(t, err)

		exp, err := desc.LoadFileDescriptor(name)
		testutil.Ok(t, err)

		checkFiles(t, fds[0], exp, map[string]struct{}{})
	}
}

func checkFiles(t *testing.T, act, exp *desc.FileDescriptor, checked map[string]struct{}) {
	if _, ok := checked[act.GetName()]; ok {
		// already checked
		return
	}
	checked[act.GetName()] = struct{}{}

	testutil.Require(t, proto.Equal(exp.AsFileDescriptorProto(), act.AsProto()), "linked descriptor did not match output from protoc:\nwanted: %s\ngot: %s", toString(exp.AsProto()), toString(act.AsProto()))

	for i, dep := range act.GetDependencies() {
		checkFiles(t, dep, exp.GetDependencies()[i], checked)
	}
}

func toString(m proto.Message) string {
	msh := jsonpb.Marshaler{Indent: "  "}
	s, err := msh.MarshalToString(m)
	if err != nil {
		panic(err)
	}
	return s
}

func TestLinkerValidation(t *testing.T) {
	testCases := []struct {
		input  map[string]string
		errMsg string
	}{
		{
			map[string]string{
				"foo.proto": "import \"foo2.proto\"; message fubar{}",
			},
			`foo.proto:1:8: file not found: foo2.proto`,
		},
		{
			map[string]string{
				"foo.proto":  "import \"foo2.proto\"; message fubar{}",
				"foo2.proto": "import \"foo.proto\"; message baz{}",
			},
			`foo.proto:1:8: cycle found in imports: "foo.proto" -> "foo2.proto" -> "foo.proto"`,
		},
		{
			map[string]string{
				"foo.proto": "message foo {} enum foo { V = 0; }",
			},
			"foo.proto:1:16: duplicate symbol foo: already defined as message",
		},
		{
			map[string]string{
				"foo.proto": "message foo { optional string a = 1; optional string a = 2; }",
			},
			"foo.proto:1:38: duplicate symbol foo.a: already defined as field",
		},
		{
			map[string]string{
				"foo.proto":  "message foo {}",
				"foo2.proto": "enum foo { V = 0; }",
			},
			"foo2.proto:1:1: duplicate symbol foo: already defined as message in \"foo.proto\"",
		},
		{
			map[string]string{
				"foo.proto": "message foo { optional blah a = 1; }",
			},
			"foo.proto:1:24: field foo.a: unknown type blah",
		},
		{
			map[string]string{
				"foo.proto": "message foo { optional bar.baz a = 1; } service bar { rpc baz (foo) returns (foo); }",
			},
			"foo.proto:1:24: field foo.a: invalid type: bar.baz is a method, not a message or enum",
		},
		{
			map[string]string{
				"foo.proto": "message foo { extensions 1 to 2; } extend foo { optional string a = 1; } extend foo { optional int32 b = 1; }",
			},
			"foo.proto:1:106: field b: duplicate extension: a and b are both using tag 1",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; extend foobar { optional string a = 1; }",
			},
			"foo.proto:1:24: unknown extendee type foobar",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; service foobar{} extend foobar { optional string a = 1; }",
			},
			"foo.proto:1:41: extendee is invalid: fu.baz.foobar is a service, not a message",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ extensions 1; } extend foobar { optional string a = 2; }",
			},
			"foo.proto:1:85: field fu.baz.a: tag 2 is not in valid range for extended type fu.baz.foobar",
		},
		{
			map[string]string{
				"foo.proto":  "package fu.baz; import public \"foo2.proto\"; message foobar{ optional baz a = 1; }",
				"foo2.proto": "package fu.baz; import \"foo3.proto\"; message fizzle{ }",
				"foo3.proto": "package fu.baz; message baz{ }",
			},
			"foo.proto:1:70: field fu.baz.foobar.a: unknown type baz",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ repeated string a = 1 [default = \"abc\"]; }",
			},
			"foo.proto:1:56: field fu.baz.foobar.a: default value cannot be set because field is repeated",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ optional foobar a = 1 [default = { a: {} }]; }",
			},
			"foo.proto:1:56: field fu.baz.foobar.a: default value cannot be set because field is a message",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ optional string a = 1 [default = { a: \"abc\" }]; }",
			},
			"foo.proto:1:66: field fu.baz.foobar.a: default value cannot be an aggregate",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ optional string a = 1 [default = 1.234]; }",
			},
			"foo.proto:1:66: field fu.baz.foobar.a: option default: expecting string, got double",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; enum abc { OK=0; NOK=1; } message foobar{ optional abc a = 1 [default = NACK]; }",
			},
			"foo.proto:1:89: field fu.baz.foobar.a: option default: enum fu.baz.abc has no value named NACK",
		},
		{
			map[string]string{
				"foo.proto": "option b = 123;",
			},
			"foo.proto:1:8: option b: field b of google.protobuf.FileOptions does not exist",
		},
		{
			map[string]string{
				"foo.proto": "option (foo.bar) = 123;",
			},
			"foo.proto:1:8: unknown extension foo.bar",
		},
		{
			map[string]string{
				"foo.proto": "option uninterpreted_option = { };",
			},
			"foo.proto:1:8: invalid option 'uninterpreted_option'",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f).b = 123;",
			},
			"foo.proto:5:12: option (f).b: field b of foo does not exist",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f).a = 123;",
			},
			"foo.proto:5:16: option (f).a: expecting string, got integer",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (b) = 123;",
			},
			"foo.proto:5:8: option (b): extension b should extend google.protobuf.FileOptions but instead extends foo",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (foo) = 123;",
			},
			"foo.proto:5:8: invalid extension: foo is a message, not an extension",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (foo.a) = 123;",
			},
			"foo.proto:5:8: invalid extension: foo.a is a field but not an extension",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f) = { a: [ 123 ] };",
			},
			"foo.proto:5:19: option (f): value is an array but field is not repeated",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { repeated string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f) = { a: [ \"a\", \"b\", 123 ] };",
			},
			"foo.proto:5:31: option (f): expecting string, got integer",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f) = { a: \"a\" };\n" +
					"option (f) = { a: \"b\" };",
			},
			"foo.proto:6:8: option (f): non-repeated option field f already set",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f) = { a: \"a\" };\n" +
					"option (f).a = \"b\";",
			},
			"foo.proto:6:12: option (f).a: non-repeated option field a already set",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f) = { a: \"a\" };\n" +
					"option (f).(b) = \"b\";",
			},
			"foo.proto:6:18: option (f).(b): expecting int32, got string",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { required string a = 1; required string b = 2; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f) = { a: \"a\" };\n",
			},
			"foo.proto:1:1: error in file options: some required fields missing: (f).b",
		},
	}
	for i, tc := range testCases {
		acc := func(filename string) (io.ReadCloser, error) {
			f, ok := tc.input[filename]
			if !ok {
				return nil, fmt.Errorf("file not found: %s", filename)
			}
			return ioutil.NopCloser(strings.NewReader(f)), nil
		}
		names := make([]string, 0, len(tc.input))
		for k := range tc.input {
			names = append(names, k)
		}
		_, err := Parser{Accessor: acc}.ParseFiles(names...)
		if err == nil {
			t.Errorf("case %d: expecting validation error %q; instead got no error", i, tc.errMsg)
		} else if err.Error() != tc.errMsg {
			t.Errorf("case %d: expecting validation error %q; instead got: %q", i, tc.errMsg, err)
		}
	}
}

func TestProto3Enums(t *testing.T) {
	file1 := `syntax = "<SYNTAX>"; enum bar { A = 0; B = 1; }`
	file2 := `syntax = "<SYNTAX>"; import "f1.proto"; message foo { <LABEL> bar bar = 1; }`
	getFileContents := func(file, syntax string) string {
		contents := strings.Replace(file, "<SYNTAX>", syntax, 1)
		label := ""
		if syntax == "proto2" {
			label = "optional"
		}
		return strings.Replace(contents, "<LABEL>", label, 1)
	}

	syntaxOptions := []string{"proto2", "proto3"}
	for _, o1 := range syntaxOptions {
		fc1 := getFileContents(file1, o1)

		for _, o2 := range syntaxOptions {
			fc2 := getFileContents(file2, o2)

			// now parse the protos
			acc := func(filename string) (io.ReadCloser, error) {
				var data string
				switch filename {
				case "f1.proto":
					data = fc1
				case "f2.proto":
					data = fc2
				default:
					return nil, fmt.Errorf("file not found: %s", filename)
				}
				return ioutil.NopCloser(strings.NewReader(data)), nil
			}
			_, err := Parser{Accessor: acc}.ParseFiles("f1.proto", "f2.proto")

			if o1 != o2 && o2 == "proto3" {
				expected := "f2.proto:1:54: field foo.bar: cannot use proto2 enum bar in a proto3 message"
				if err == nil {
					t.Errorf("expecting validation error; instead got no error")
				} else if err.Error() != expected {
					t.Errorf("expecting validation error %q; instead got: %q", expected, err)
				}
			} else {
				// other cases succeed (okay to for proto2 to use enum from proto3 file and
				// obviously okay for proto2 importing proto2 and proto3 importing proto3)
				testutil.Ok(t, err)
			}
		}
	}
}
