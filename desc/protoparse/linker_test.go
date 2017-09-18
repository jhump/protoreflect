package protoparse

import (
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"fmt"
	"github.com/jhump/protoreflect/desc"
	_ "github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
	"io"
	"strings"
)

//go:generate bash -c "protoc test.proto -o ./test.protoset --go_out=../../../../.. && mv test.pb.go test_pb_test.go"

func TestSimpleLink(t *testing.T) {
	fd, err := ParseProtoFileByName("test.proto")
	testutil.Ok(t, err)

	b, err := ioutil.ReadFile("test.protoset")
	testutil.Ok(t, err)
	var files dpb.FileDescriptorSet
	err = proto.Unmarshal(b, &files)
	testutil.Ok(t, err)

	// protoc does not set syntax field if it is proto2, but we do
	// so set it just so we can compare apples to apples
	if files.File[0].Syntax == nil {
		files.File[0].Syntax = proto.String("proto2")
	}
	testutil.Require(t, proto.Equal(files.File[0], fd.AsProto()), "linked descriptor did not match output from protoc:\nwanted: %s\ngot: %s", toString(files.File[0]), toString(fd.AsProto()))
}

func TestMultiFileLink(t *testing.T) {
	for _, name := range []string{"desc_test2.proto", "desc_test_defaults.proto", "desc_test_field_types.proto", "desc_test_options.proto", "desc_test_proto3.proto", "desc_test_wellknowntypes.proto"} {
		fd, err := ParseProtoFileByName(name, "../../internal/testprotos")
		testutil.Ok(t, err)

		exp, err := desc.LoadFileDescriptor(name)
		testutil.Ok(t, err)

		checkFiles(t, fd, exp, map[string]struct{}{})
	}
}

func checkFiles(t *testing.T, act, exp *desc.FileDescriptor, checked map[string]struct{}) {
	if _, ok := checked[act.GetName()]; ok {
		// already checked
		return
	}
	checked[act.GetName()] = struct{}{}

	// protoc does not set syntax field if it is proto2, so modify
	// ours to follow suit so we can compare apples to apples
	if act.AsFileDescriptorProto().GetSyntax() == "proto2" {
		act.AsFileDescriptorProto().Syntax = nil
	}

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
			"failed to load imports for \"foo.proto\": file not found: foo2.proto",
		},
		{
			map[string]string{
				"foo.proto":  "import \"foo2.proto\"; message fubar{}",
				"foo2.proto": "import \"foo.proto\"; message baz{}",
			},
			"cycle found in imports: \"foo.proto\" -> \"foo2.proto\" -> \"foo.proto\"",
		},
		{
			map[string]string{
				"foo.proto": "message foo {} enum foo { V = 0; }",
			},
			"file \"foo.proto\": duplicate symbol foo: enum and message",
		},
		{
			map[string]string{
				"foo.proto": "message foo { optional string a = 1; optional string a = 2; }",
			},
			"file \"foo.proto\": duplicate symbol foo.a: field and field",
		},
		{
			map[string]string{
				"foo.proto":  "message foo {}",
				"foo2.proto": "enum foo { V = 0; }",
			},
			"duplicate symbol foo: message in \"foo.proto\" and enum in \"foo2.proto\"",
		},
		{
			map[string]string{
				"foo.proto": "message foo { optional blah a = 1; }",
			},
			"file \"foo.proto\": field foo.a references unknown type: blah",
		},
		{
			map[string]string{
				"foo.proto": "message foo { optional bar.baz a = 1; } service bar { rpc baz (foo) returns (foo); }",
			},
			"file \"foo.proto\": field foo.a has invalid type: bar.baz is a method, not a message or enum",
		},
		{
			map[string]string{
				"foo.proto": "message foo { extensions 1 to 2; } extend foo { optional string a = 1; } extend foo { optional int32 b = 1; }",
			},
			"file \"foo.proto\": duplicate extension for foo: a and b are both using tag 1",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; extend foobar { optional string a = 1; }",
			},
			"file \"foo.proto\": field fu.baz.a extends unknown type: foobar",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; service foobar{} extend foobar { optional string a = 1; }",
			},
			"file \"foo.proto\": field fu.baz.a extends invalid type: fu.baz.foobar is a service, not a message",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ extensions 1; } extend foobar { optional string a = 2; }",
			},
			"file \"foo.proto\": field fu.baz.a tag is not in valid range for extended type fu.baz.foobar: 2",
		},
		{
			map[string]string{
				"foo.proto":  "package fu.baz; import public \"foo2.proto\"; message foobar{ optional baz a = 1; }",
				"foo2.proto": "package fu.baz; import \"foo3.proto\"; message fizzle{ }",
				"foo3.proto": "package fu.baz; message baz{ }",
			},
			"file \"foo.proto\": field fu.baz.foobar.a references unknown type: baz",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ repeated string a = 1 [default = \"abc\"]; }",
			},
			"file \"foo.proto\": default value cannot be set for field fu.baz.foobar.a because it is repeated",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ optional foobar a = 1 [default = { a: {} }]; }",
			},
			"file \"foo.proto\": default value cannot be set for field fu.baz.foobar.a because it is a message",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ optional string a = 1 [default = { a: \"abc\" }]; }",
			},
			"file \"foo.proto\": default value for field fu.baz.foobar.a cannot be an aggregate",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ optional string a = 1 [default = 1.234]; }",
			},
			"file \"foo.proto\": field fu.baz.foobar.a: option default: expecting string, got double",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; enum abc { OK=0; NOK=1; } message foobar{ optional abc a = 1 [default = NACK]; }",
			},
			"file \"foo.proto\": field fu.baz.foobar.a: option default: enum fu.baz.abc has no value named NACK",
		},
		{
			map[string]string{
				"foo.proto": "option b = 123;",
			},
			"file \"foo.proto\": option b: field b of google.protobuf.FileOptions does not exist",
		},
		{
			map[string]string{
				"foo.proto": "option (foo.bar) = 123;",
			},
			"file \"foo.proto\": option (foo.bar): unknown extension: foo.bar",
		},
		{
			map[string]string{
				"foo.proto": "option uninterpreted_option = { };",
			},
			"file \"foo.proto\": invalid option 'uninterpreted_option'",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f).b = 123;",
			},
			"file \"foo.proto\": option (f).b: field b of foo does not exist",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f).a = 123;",
			},
			"file \"foo.proto\": option (f).a: expecting string, got integer",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (b) = 123;",
			},
			"file \"foo.proto\": option (b): extension b should extend google.protobuf.FileOptions but instead extends foo",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (foo) = 123;",
			},
			"file \"foo.proto\": option (foo): invalid extension: foo is a message, not an extension",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (foo.a) = 123;",
			},
			"file \"foo.proto\": option (foo.a): invalid extension: foo.a is a field but not an extension",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { optional string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f) = { a: [ 123 ] };",
			},
			"file \"foo.proto\": option (f) at a: value is an array but field is not repeated",
		},
		{
			map[string]string{
				"foo.proto": "import \"google/protobuf/descriptor.proto\";\n" +
					"message foo { repeated string a = 1; extensions 10 to 20; }\n" +
					"extend foo { optional int32 b = 10; }\n" +
					"extend google.protobuf.FileOptions { optional foo f = 20000; }\n" +
					"option (f) = { a: [ \"a\", \"b\", 123 ] };",
			},
			"file \"foo.proto\": option (f) at a[2]: expecting string, got integer",
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
			"file \"foo.proto\": option (f): non-repeated option field f already set",
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
			"file \"foo.proto\": option (f).a: non-repeated option field a already set",
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
			"file \"foo.proto\": option (f).(b): expecting int32, got string/bytes",
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
		_, err := ParseProtoFiles(acc, names...)
		if err == nil || !strings.Contains(err.Error(), tc.errMsg) {
			t.Errorf("case %d: expecting validation error %q; instead got: %v", i, tc.errMsg, err)
		}
	}
}
