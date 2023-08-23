package protoparse

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	protov1 "github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/desc"
	_ "github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestSimpleLink(t *testing.T) {
	fds, err := Parser{ImportPaths: []string{"../../internal/testprotos"}}.ParseFiles("desc_test_complex.proto")
	testutil.Ok(t, err)

	b, err := os.ReadFile("../../internal/testprotos/desc_test_complex.protoset")
	testutil.Ok(t, err)

	var fdSet descriptorpb.FileDescriptorSet
	err = proto.Unmarshal(b, &fdSet)
	testutil.Ok(t, err)

	testutil.Require(t, proto.Equal(fdSet.File[0], protov1.MessageV2(fds[0].AsProto())), "linked descriptor did not match output from protoc:\nwanted: %s\ngot: %s", toString(fdSet.File[0]), toString(protov1.MessageV2(fds[0].AsProto())))
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

func TestProto3Optional(t *testing.T) {
	data, err := os.ReadFile("../../internal/testprotos/proto3_optional/desc_test_proto3_optional.protoset")
	testutil.Ok(t, err)
	var fdset descriptorpb.FileDescriptorSet
	err = proto.Unmarshal(data, &fdset)
	testutil.Ok(t, err)

	var descriptorProto *descriptorpb.FileDescriptorProto
	for _, fd := range fdset.File {
		// not comparing source code info
		fd.SourceCodeInfo = nil

		// we want to use the same descriptor.proto as in protoset, so we don't have to
		// worry about this test breaking when updating to newer versions of the Go
		// descriptor package (which may have a different version of descriptor.proto
		// compiled in).
		if fd.GetName() == "google/protobuf/descriptor.proto" {
			descriptorProto = fd
		}
	}
	testutil.Require(t, descriptorProto != nil, "failed to find google/protobuf/descriptor.proto in protoset")

	exp, err := desc.CreateFileDescriptorFromSet(&fdset)
	testutil.Ok(t, err)

	fds, err := Parser{
		ImportPaths: []string{"../../internal/testprotos"},
		LookupImportProto: func(name string) (*descriptorpb.FileDescriptorProto, error) {
			if name == "google/protobuf/descriptor.proto" {
				return descriptorProto, nil
			}
			return nil, errors.New("not found")
		},
	}.ParseFiles("proto3_optional/desc_test_proto3_optional.proto")
	testutil.Ok(t, err)

	checkFiles(t, fds[0], exp, map[string]struct{}{})
}

func checkFiles(t *testing.T, act, exp *desc.FileDescriptor, checked map[string]struct{}) {
	if _, ok := checked[act.GetName()]; ok {
		// already checked
		return
	}
	checked[act.GetName()] = struct{}{}

	// remove any source code info from expected value, since actual won't have any
	exp.AsFileDescriptorProto().SourceCodeInfo = nil

	testutil.Require(t, proto.Equal(exp.AsFileDescriptorProto(), protov1.MessageV2(act.AsProto())), "linked descriptor did not match output from protoc:\nwanted: %s\ngot: %s", toString(protov1.MessageV2(exp.AsProto())), toString(protov1.MessageV2(act.AsProto())))

	for i, dep := range act.GetDependencies() {
		checkFiles(t, dep, exp.GetDependencies()[i], checked)
	}
}

func toString(m proto.Message) string {
	msh := protojson.MarshalOptions{Indent: "  "}
	data, err := msh.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func TestLinkerValidation(t *testing.T) {
	testCases := []struct {
		input  map[string]string
		errMsg string
	}{
		{
			map[string]string{
				"foo.proto":  `syntax = "proto3"; package namespace.a; import "foo2.proto"; import "foo3.proto"; import "foo4.proto"; message Foo{ b.Bar a = 1; b.Baz b = 2; b.Buzz c = 3; }`,
				"foo2.proto": `syntax = "proto3"; package namespace.b; message Bar{}`,
				"foo3.proto": `syntax = "proto3"; package namespace.b; message Baz{}`,
				"foo4.proto": `syntax = "proto3"; package namespace.b; message Buzz{}`,
			},
			"", // should succeed
		},
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
			`foo.proto:1:8: cycle found in imports: "foo.proto" -> "foo2.proto" -> "foo.proto"
					|| foo2.proto:1:8: cycle found in imports: "foo2.proto" -> "foo.proto" -> "foo2.proto"`,
		},
		{
			map[string]string{
				"foo.proto": "enum foo { bar = 1; baz = 2; } enum fu { bar = 1; baz = 2; }",
			},
			`foo.proto:1:42: symbol "bar" already defined at foo.proto:1:12; protobuf uses C++ scoping rules for enum values, so they exist in the scope enclosing the enum`,
		},
		{
			map[string]string{
				"foo.proto": "message foo {} enum foo { V = 0; }",
			},
			`foo.proto:1:21: symbol "foo" already defined at foo.proto:1:9`,
		},
		{
			map[string]string{
				"foo.proto": "message foo { optional string a = 1; optional string a = 2; }",
			},
			`foo.proto:1:54: symbol "foo.a" already defined at foo.proto:1:31`,
		},
		{
			map[string]string{
				"foo.proto":  "message foo {}",
				"foo2.proto": "enum foo { V = 0; }",
			},
			`foo.proto:1:9: symbol "foo" already defined at foo2.proto:1:6
					|| foo2.proto:1:6: symbol "foo" already defined at foo.proto:1:9`,
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
			"foo.proto:1:106: extension with tag 1 for message foo already defined at foo.proto:1:69",
		},
		{
			map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					package google.protobuf;
					message DescriptorProto { }
				`,
			},
			`foo.proto:5:49: symbol "google.protobuf.DescriptorProto" already defined at google/protobuf/descriptor.proto`,
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
				"foo.proto": "message foo{} message bar{} service foobar{ rpc foo(foo) returns (bar); }",
			},
			"foo.proto:1:53: method foobar.foo: invalid request type: foobar.foo is a method, not a message",
		},
		{
			map[string]string{
				"foo.proto": "message foo{} message bar{} service foobar{ rpc foo(bar) returns (foo); }",
			},
			"foo.proto:1:67: method foobar.foo: invalid response type: foobar.foo is a method, not a message",
		},
		{
			map[string]string{
				"foo.proto": "package fu.baz; message foobar{ extensions 1; } extend foobar { optional string a = 2; }",
			},
			"foo.proto:1:85: extension fu.baz.a: tag 2 is not in valid range for extended type fu.baz.foobar",
		},
		{
			map[string]string{
				"foo.proto":  "package fu.baz; import public \"foo2.proto\"; message foobar{ optional baz a = 1; }",
				"foo2.proto": "package fu.baz; import \"foo3.proto\"; message fizzle{ }",
				"foo3.proto": "package fu.baz; message baz{ }",
			},
			"foo.proto:1:70: field fu.baz.foobar.a: unknown type baz; resolved to fu.baz which is not defined; consider using a leading dot",
		},
		{
			map[string]string{
				"foo.proto": `
					syntax = "proto2";
					package foo;
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.FileOptions           { optional string fil_foo = 12000; }
					extend google.protobuf.MessageOptions        { optional string msg_foo = 12000; }
					extend google.protobuf.FieldOptions          { optional string fld_foo = 12000 [(fld_foo) = "extension"]; }
					extend google.protobuf.OneofOptions          { optional string oof_foo = 12000; }
					extend google.protobuf.EnumOptions           { optional string enm_foo = 12000; }
					extend google.protobuf.EnumValueOptions      { optional string env_foo = 12000; }
					extend google.protobuf.ExtensionRangeOptions { optional string ext_foo = 12000; }
					extend google.protobuf.ServiceOptions        { optional string svc_foo = 12000; }
					extend google.protobuf.MethodOptions         { optional string mtd_foo = 12000; }
					option (fil_foo) = "file";
					message Foo {
						option (msg_foo) = "message";
						oneof foo {
							option (oof_foo) = "oneof";
							string bar = 1 [(fld_foo) = "field"];
						}
						extensions 100 to 200 [(ext_foo) = "extensionrange"];
					}
					enum Baz {
						option (enm_foo) = "enum";
						ZERO = 0 [(env_foo) = "enumvalue"];
					}
					service FooService {
						option (svc_foo) = "service";
						rpc Bar(Foo) returns (Foo) {
							option (mtd_foo) = "method";
						}
					}
					`,
			},
			"", // should success
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
			"foo.proto:1:66: field fu.baz.foobar.a: option default: default value cannot be a message",
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
			"foo.proto:6:8: option (f): non-repeated option field (f) already set",
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
		{
			map[string]string{
				"foo.proto": "message Foo { option message_set_wire_format = true; extensions 1 to 100; } extend Foo { optional int32 bar = 1; }",
			},
			"foo.proto:1:99: messages with message-set wire format cannot contain scalar extensions, only messages",
		},
		{
			map[string]string{
				"foo.proto": "message Foo { option message_set_wire_format = true; extensions 1 to 100; } extend Foo { optional Foo bar = 1; }",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "message Foo { option message_set_wire_format = true; extensions 1 to 100; } extend Foo { repeated Foo bar = 1; }",
			},
			"foo.proto:1:90: messages with message-set wire format cannot contain repeated extensions, only optional",
		},
		{
			map[string]string{
				"foo.proto": "message Foo { extensions 1 to max; } extend Foo { optional int32 bar = 536870912; }",
			},
			"foo.proto:1:72: extension bar: tag 536870912 is not in valid range for extended type Foo",
		},
		{
			map[string]string{
				"foo.proto": "message Foo { option message_set_wire_format = true; extensions 1 to max; } extend Foo { optional Foo bar = 536870912; }",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": `syntax = "proto3"; package com.google; import "google/protobuf/wrappers.proto"; message Foo { google.protobuf.StringValue str = 1; }`,
			},
			"foo.proto:1:95: field com.google.Foo.str: unknown type google.protobuf.StringValue; resolved to com.google.protobuf.StringValue which is not defined; consider using a leading dot",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo {\n" +
					"  optional group Bar = 1 { optional string name = 1; }\n" +
					"}\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz { option (foo).bar.name = \"abc\"; }\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo {\n" +
					"  optional group Bar = 1 { optional string name = 1; }\n" +
					"}\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz { option (foo).Bar.name = \"abc\"; }\n",
			},
			"foo.proto:7:28: message Baz: option (foo).Bar.name: field Bar of Foo does not exist",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"extend google.protobuf.MessageOptions {\n" +
					"  optional group Foo = 10001 { optional string name = 1; }\n" +
					"}\n" +
					"message Bar { option (foo).name = \"abc\"; }\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"extend google.protobuf.MessageOptions {\n" +
					"  optional group Foo = 10001 { optional string name = 1; }\n" +
					"}\n" +
					"message Bar { option (Foo).name = \"abc\"; }\n",
			},
			"foo.proto:6:22: message Bar: invalid extension: Foo is a message, not an extension",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo {\n" +
					"  optional group Bar = 1 { optional string name = 1; }\n" +
					"}\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz { option (foo) = { Bar< name: \"abc\" > }; }\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo {\n" +
					"  optional group Bar = 1 { optional string name = 1; }\n" +
					"}\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz { option (foo) = { bar< name: \"abc\" > }; }\n",
			},
			"foo.proto:7:32: message Baz: option (foo): field bar not found (did you mean the group named Bar?)",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { extensions 1 to 10; }\n" +
					"extend Foo { optional group Bar = 10 { optional string name = 1; } }\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz { option (foo) = { [bar]< name: \"abc\" > }; }\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { extensions 1 to 10; }\n" +
					"extend Foo { optional group Bar = 10 { optional string name = 1; } }\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz { option (foo) = { [Bar]< name: \"abc\" > }; }\n",
			},
			"foo.proto:6:33: message Baz: option (foo): invalid extension: Bar is a message, not an extension",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { oneof bar { string baz = 1; string buzz = 2; } }\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz { option (foo) = { baz: \"abc\" buzz: \"xyz\" }; }\n",
			},
			`foo.proto:5:43: message Baz: option (foo): oneof "bar" already has field "baz" set`,
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { oneof bar { string baz = 1; string buzz = 2; } }\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz {\n" +
					"  option (foo).baz = \"abc\";\n" +
					"  option (foo).buzz = \"xyz\";\n" +
					"}",
			},
			`foo.proto:7:16: message Baz: option (foo).buzz: oneof "bar" already has field "baz" set`,
		},
		{
			map[string]string{
				"a.proto": "syntax = \"proto3\";\n" +
					"message m{\n" +
					"  oneof z{\n" +
					"    int64 z=1;\n" +
					"  }\n" +
					"}",
			},
			`a.proto:4:11: symbol "m.z" already defined at a.proto:3:9`,
		},
		{
			map[string]string{
				"a.proto": "syntax=\"proto3\";\n" +
					"message m{\n" +
					"  string z = 1;\n" +
					"  oneof z{int64 b=2;}\n" +
					"}",
			},
			`a.proto:4:9: symbol "m.z" already defined at a.proto:3:10`,
		},
		{
			map[string]string{
				"test.proto": "syntax=\"proto2\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message a { extensions 1 to 100; }\n" +
					"extend google.protobuf.MessageOptions { optional a msga = 10000; }\n" +
					"message b {\n" +
					"  message c {\n" +
					"    extend a { repeated int32 i = 1; repeated float f = 2; }\n" +
					"  }\n" +
					"  option (msga) = {\n" +
					"    [foo.bar.b.c.i]: 123\n" +
					"    [bar.b.c.i]: 234\n" +
					"    [b.c.i]: 345\n" +
					"  };\n" +
					"  option (msga).(foo.bar.b.c.f) = 1.23;\n" +
					"  option (msga).(bar.b.c.f) = 2.34;\n" +
					"  option (msga).(b.c.f) = 3.45;\n" +
					"}",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"test.proto": "syntax=\"proto2\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message a { extensions 1 to 100; }\n" +
					"message b { extensions 1 to 100; }\n" +
					"extend google.protobuf.MessageOptions { optional a msga = 10000; }\n" +
					"message c {\n" +
					"  extend a { optional b b = 1; }\n" +
					"  extend foo.bar.b { repeated int32 i = 1; repeated float f = 2; }\n" +
					"  option (msga) = {\n" +
					"    [foo.bar.c.b] {\n" +
					"      [foo.bar.c.i]: 123\n" +
					"      [bar.c.i]: 234\n" +
					"      [c.i]: 345\n" +
					"    }\n" +
					"  };\n" +
					"  option (msga).(foo.bar.c.b).(foo.bar.c.f) = 1.23;\n" +
					"  option (msga).(foo.bar.c.b).(bar.c.f) = 2.34;\n" +
					"  option (msga).(foo.bar.c.b).(c.f) = 3.45;\n" +
					"}",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"test.proto": "syntax=\"proto2\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message a { extensions 1 to 100; }\n" +
					"extend google.protobuf.MessageOptions { optional a msga = 10000; }\n" +
					"message b {\n" +
					"  message c {\n" +
					"    extend a { repeated int32 i = 1; repeated float f = 2; }\n" +
					"  }\n" +
					"  option (msga) = {\n" +
					"    [c.i]: 456\n" +
					"  };\n" +
					"}",
			},
			"test.proto:11:6: message foo.bar.b: option (foo.bar.msga): unknown extension c.i",
		},
		{
			map[string]string{
				"test.proto": "syntax=\"proto2\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message a { extensions 1 to 100; }\n" +
					"extend google.protobuf.MessageOptions { optional a msga = 10000; }\n" +
					"message b {\n" +
					"  message c {\n" +
					"    extend a { repeated int32 i = 1; repeated float f = 2; }\n" +
					"  }\n" +
					"  option (msga) = {\n" +
					"    [i]: 567\n" +
					"  };\n" +
					"}",
			},
			"test.proto:11:6: message foo.bar.b: option (foo.bar.msga): unknown extension i",
		},
		{
			map[string]string{
				"test.proto": "syntax=\"proto2\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message a { extensions 1 to 100; }\n" +
					"extend google.protobuf.MessageOptions { optional a msga = 10000; }\n" +
					"message b {\n" +
					"  message c {\n" +
					"    extend a { repeated int32 i = 1; repeated float f = 2; }\n" +
					"  }\n" +
					"  option (msga).(c.f) = 4.56;\n" +
					"}",
			},
			"test.proto:10:17: message foo.bar.b: unknown extension c.f",
		},
		{
			map[string]string{
				"test.proto": "syntax=\"proto2\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message a { extensions 1 to 100; }\n" +
					"extend google.protobuf.MessageOptions { optional a msga = 10000; }\n" +
					"message b {\n" +
					"  message c {\n" +
					"    extend a { repeated int32 i = 1; repeated float f = 2; }\n" +
					"  }\n" +
					"  option (msga).(f) = 5.67;\n" +
					"}",
			},
			"test.proto:10:17: message foo.bar.b: unknown extension f",
		},
		{
			map[string]string{
				"a.proto": "syntax=\"proto3\";\nmessage m{\n" +
					"  oneof z{int64 a=1;}\n" +
					"  oneof z{int64 b=2;}\n" +
					"}",
			},
			`a.proto:4:9: symbol "m.z" already defined at a.proto:3:9`,
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { oneof bar { google.protobuf.DescriptorProto baz = 1; google.protobuf.DescriptorProto buzz = 2; } }\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz {\n" +
					"  option (foo).baz.name = \"abc\";\n" +
					"  option (foo).buzz.name = \"xyz\";\n" +
					"}",
			},
			`foo.proto:7:16: message Baz: option (foo).buzz.name: oneof "bar" already has field "baz" set`,
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { oneof bar { google.protobuf.DescriptorProto baz = 1; google.protobuf.DescriptorProto buzz = 2; } }\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz {\n" +
					"  option (foo).baz.options.(foo).baz.name = \"abc\";\n" +
					"  option (foo).baz.options.(foo).buzz.name = \"xyz\";\n" +
					"}",
			},
			`foo.proto:7:34: message Baz: option (foo).baz.options.(foo).buzz.name: oneof "bar" already has field "baz" set`,
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"enum Foo { option allow_alias = true; true = 0; false = 1; True = 0; False = 1; t = 2; f = 3; inf = 4; nan = 5; }\n" +
					"extend google.protobuf.MessageOptions { repeated Foo foo = 10001; }\n" +
					"message Baz {\n" +
					"  option (foo) = true; option (foo) = false;\n" +
					"  option (foo) = t; option (foo) = f;\n" +
					"  option (foo) = True; option (foo) = False;\n" +
					"  option (foo) = inf; option (foo) = nan;\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"extend google.protobuf.MessageOptions { repeated bool foo = 10001; }\n" +
					"message Baz {\n" +
					"  option (foo) = true; option (foo) = false;\n" +
					"  option (foo) = t; option (foo) = f;\n" +
					"  option (foo) = True; option (foo) = False;\n" +
					"}\n",
			},
			"foo.proto:6:18: message Baz: option (foo): expecting bool, got identifier",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { repeated bool b = 1; }\n" +
					"extend google.protobuf.MessageOptions { Foo foo = 10001; }\n" +
					"message Baz {\n" +
					"  option (foo) = {\n" +
					"    b: t     b: f\n" +
					"    b: true  b: false\n" +
					"    b: True  b: False\n" +
					"  };\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { extensions 1 to 10; }\n" +
					"extend Foo { optional bool b = 10; }\n" +
					"extend google.protobuf.MessageOptions { optional Foo foo = 10001; }\n" +
					"message Baz {\n" +
					"  option (foo) = {\n" +
					"    [.b]: true\n" +
					"  };\n" +
					"}\n",
			},
			"foo.proto:8:6: syntax error: unexpected '.'",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/any.proto\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { string a = 1; int32 b = 2; }\n" +
					"extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }\n" +
					"message Baz {\n" +
					"  option (any) = {\n" +
					"    [type.googleapis.com/foo.bar.Foo] <\n" +
					"      a: \"abc\"\n" +
					"      b: 123\n" +
					"    >\n" +
					"  };\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { string a = 1; int32 b = 2; }\n" +
					"extend google.protobuf.MessageOptions { optional Foo f = 10001; }\n" +
					"message Baz {\n" +
					"  option (f) = {\n" +
					"    [type.googleapis.com/foo.bar.Foo] <\n" +
					"      a: \"abc\"\n" +
					"      b: 123\n" +
					"    >\n" +
					"  };\n" +
					"}\n",
			},
			"foo.proto:8:6: message foo.bar.Baz: option (foo.bar.f): type references are only allowed for google.protobuf.Any, but this type is foo.bar.Foo",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/any.proto\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { string a = 1; int32 b = 2; }\n" +
					"extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }\n" +
					"message Baz {\n" +
					"  option (any) = {\n" +
					"    [types.custom.io/foo.bar.Foo] <\n" +
					"      a: \"abc\"\n" +
					"      b: 123\n" +
					"    >\n" +
					"  };\n" +
					"}\n",
			},
			"foo.proto:9:6: message foo.bar.Baz: option (foo.bar.any): could not resolve type reference types.custom.io/foo.bar.Foo",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/any.proto\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { string a = 1; int32 b = 2; }\n" +
					"extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }\n" +
					"message Baz {\n" +
					"  option (any) = {\n" +
					"    [type.googleapis.com/foo.bar.Foo]: 123\n" +
					"  };\n" +
					"}\n",
			},
			"foo.proto:9:40: message foo.bar.Baz: option (foo.bar.any): type references for google.protobuf.Any must have message literal value",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"package foo.bar;\n" +
					"import \"google/protobuf/any.proto\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo { string a = 1; int32 b = 2; }\n" +
					"extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }\n" +
					"message Baz {\n" +
					"  option (any) = {\n" +
					"    [type.googleapis.com/Foo] <\n" +
					"      a: \"abc\"\n" +
					"      b: 123\n" +
					"    >\n" +
					"  };\n" +
					"}\n",
			},
			"foo.proto:9:6: message foo.bar.Baz: option (foo.bar.any): could not resolve type reference type.googleapis.com/Foo",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"extend google.protobuf.MessageOptions {\n" +
					"  string foobar = 10001 [json_name=\"FooBar\"];\n" +
					"}\n",
			},
			"foo.proto:4:26: field foobar: option json_name is not allowed on extensions",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"package foo.foo;\n" +
					"import \"other.proto\";\n" +
					"service Foo { rpc Bar (Baz) returns (Baz); }\n" +
					"message Baz {\n" +
					"  foo.Foo.Bar f = 1;\n" +
					"}\n",
				"other.proto": "syntax = \"proto3\";\n" +
					"package foo;\n" +
					"message Foo {\n" +
					"  enum Bar { ZED = 0; }\n" +
					"}\n",
			},
			"foo.proto:6:3: field foo.foo.Baz.f: invalid type: foo.foo.Foo.Bar is a method, not a message or enum",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"message Foo {\n" +
					"  enum Bar { ZED = 0; }\n" +
					"  message Foo {\n" +
					"    extend google.protobuf.MessageOptions {\n" +
					"      string Bar = 30000;\n" +
					"    }\n" +
					"    Foo.Bar f = 1;\n" +
					"  }\n" +
					"}\n",
			},
			"foo.proto:9:5: field Foo.Foo.f: invalid type: Foo.Foo.Bar is an extension, not a message or enum",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"extend google.protobuf.ServiceOptions {\n" +
					"  string Bar = 30000;\n" +
					"}\n" +
					"message Empty {}\n" +
					"service Foo {\n" +
					"  option (Bar) = \"blah\";\n" +
					"  rpc Bar (Empty) returns (Empty);\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"extend google.protobuf.MethodOptions {\n" +
					"  string Bar = 30000;\n" +
					"}\n" +
					"message Empty {}\n" +
					"service Foo {\n" +
					"  rpc Bar (Empty) returns (Empty) { option (Bar) = \"blah\"; }\n" +
					"}\n",
			},
			"foo.proto:8:44: method Foo.Bar: invalid extension: Bar is a method, not an extension",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/descriptor.proto\";\n" +
					"enum Bar { ZED = 0; }\n" +
					"message Foo {\n" +
					"  extend google.protobuf.MessageOptions {\n" +
					"    string Bar = 30000;\n" +
					"  }\n" +
					"  message Foo {\n" +
					"    Bar f = 1;\n" +
					"  }\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Foo {\n" +
					"  map<string,string> bar = 1;\n" +
					"}\n" +
					"message Baz {\n" +
					"  Foo.BarEntry e = 1;\n" +
					"}\n",
			},
			"foo.proto:6:3: field Baz.e: Foo.BarEntry is a synthetic map entry and may not be referenced explicitly",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"import \"google/protobuf/struct.proto\";\n" +
					"message Foo {\n" +
					"  google.protobuf.Struct.FieldsEntry e = 1;\n" +
					"}\n",
			},
			"foo.proto:4:3: field Foo.e: google.protobuf.Struct.FieldsEntry is a synthetic map entry and may not be referenced explicitly",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Foo {\n" +
					"  string foo = 1;\n" +
					"  string bar = 2 [json_name=\"foo\"];\n" +
					"}\n",
			},
			"foo.proto:4:3: field Foo.bar: custom JSON name \"foo\" conflicts with default JSON name of field foo, defined at foo.proto:3:3",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Blah {\n" +
					"  message Foo {\n" +
					"    string foo = 1;\n" +
					"    string bar = 2 [json_name=\"foo\"];\n" +
					"  }\n" +
					"}\n",
			},
			"foo.proto:5:5: field Foo.bar: custom JSON name \"foo\" conflicts with default JSON name of field foo, defined at foo.proto:4:5",
		}, {
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Foo {\n" +
					"  string foo = 1 [json_name=\"foo_bar\"];\n" +
					"  string bar = 2 [json_name=\"Foo_Bar\"];\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Foo {\n" +
					"  string fooBar = 1;\n" +
					"  string foo_bar = 2;\n" +
					"}\n",
			},
			"foo.proto:4:3: field Foo.foo_bar: default JSON name \"fooBar\" conflicts with default JSON name of field fooBar, defined at foo.proto:3:3",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Foo {\n" +
					"  string fooBar = 1;\n" +
					"  string foo_bar = 2 [json_name=\"fuber\"];\n" +
					"}\n",
			},
			"foo.proto:4:3: field Foo.foo_bar: default JSON name \"fooBar\" conflicts with default JSON name of field fooBar, defined at foo.proto:3:3",
		}, {
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Foo {\n" +
					"  string fooBar = 1;\n" +
					"  string FOO_BAR = 2;\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Foo {\n" +
					"  string fooBar = 1;\n" +
					"  string __foo_bar = 2;\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"message Foo {\n" +
					"  optional string foo = 1 [json_name=\"foo_bar\"];\n" +
					"  optional string bar = 2 [json_name=\"Foo_Bar\"];\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"message Blah {\n" +
					"  message Foo {\n" +
					"    optional string foo = 1 [json_name=\"foo_bar\"];\n" +
					"    optional string bar = 2 [json_name=\"Foo_Bar\"];\n" +
					"  }\n" +
					"}\n",
			},
			"", // should succeed
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"message Foo {\n" +
					"  optional string fooBar = 1;\n" +
					"  optional string foo_bar = 2;\n" +
					"}\n",
			},
			"", // should succeed: only check default JSON names in proto3
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"message Foo {\n" +
					"  optional string fooBar = 1 [json_name=\"fooBar\"];\n" +
					"  optional string foo_bar = 2 [json_name=\"fooBar\"];\n" +
					"}\n",
			},
			"foo.proto:4:3: field Foo.foo_bar: custom JSON name \"fooBar\" conflicts with custom JSON name of field fooBar, defined at foo.proto:3:3",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"message Foo {\n" +
					"  optional string fooBar = 1;\n" +
					"  optional string FOO_BAR = 2;\n" +
					"}\n",
			},
			"", // should succeed: only check default JSON names in proto3
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto2\";\n" +
					"message Foo {\n" +
					"  optional string fooBar = 1;\n" +
					"  optional string __foo_bar = 2;\n" +
					"}\n",
			},
			"", // should succeed: only check default JSON names in proto3
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"enum Foo {\n" +
					"  true = 0;\n" +
					"  TRUE = 1;\n" +
					"}\n",
			},
			"foo.proto:4:3: enum value Foo.TRUE: camel-case name (with optional enum name prefix removed) \"True\" conflicts with camel-case name of enum value true, defined at foo.proto:3:3",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"message Blah {\n" +
					"  enum Foo {\n" +
					"    true = 0;\n" +
					"    TRUE = 1;\n" +
					"  }\n" +
					"}\n",
			},
			"foo.proto:5:5: enum value Foo.TRUE: camel-case name (with optional enum name prefix removed) \"True\" conflicts with camel-case name of enum value true, defined at foo.proto:4:5",
		}, {
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"enum Foo {\n" +
					"  BAR_BAZ = 0;\n" +
					"  Foo_Bar_Baz = 1;\n" +
					"}\n",
			},
			"foo.proto:4:3: enum value Foo.Foo_Bar_Baz: camel-case name (with optional enum name prefix removed) \"BarBaz\" conflicts with camel-case name of enum value BAR_BAZ, defined at foo.proto:3:3",
		},
		{
			map[string]string{
				"foo.proto": "syntax = \"proto3\";\n" +
					"enum Foo {\n" +
					"  option allow_alias = true;\n" +
					"  BAR_BAZ = 0;\n" +
					"  FooBarBaz = 0;\n" +
					"}\n",
			},
			"", // should succeed: not a conflict if both values have same number
		},
	}
	for i, tc := range testCases {
		acc := func(filename string) (io.ReadCloser, error) {
			f, ok := tc.input[filename]
			if !ok {
				return nil, fmt.Errorf("file not found: %s", filename)
			}
			return io.NopCloser(strings.NewReader(f)), nil
		}
		names := make([]string, 0, len(tc.input))
		for k := range tc.input {
			names = append(names, k)
		}
		_, err := Parser{Accessor: acc}.ParseFiles(names...)
		if tc.errMsg == "" {
			if err != nil {
				t.Errorf("case %d: expecting no error; instead got error %q", i, err)
			}
		} else if err == nil {
			t.Errorf("case %d: expecting validation error %q; instead got no error", i, tc.errMsg)
		} else {
			options := strings.Split(tc.errMsg, "||")
			matched := false
			for i := range options {
				options[i] = strings.TrimSpace(options[i])
				if err.Error() == options[i] {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("case %d: expecting validation error %q; instead got: %q", i, strings.Join(options, " || "), err)
			}
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
				return io.NopCloser(strings.NewReader(data)), nil
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

func TestCustomErrorReporterWithLinker(t *testing.T) {
	input := map[string]string{
		"a/b/b.proto": `package a.b;

import "google/protobuf/descriptor.proto";

extend google.protobuf.FieldOptions {
  optional Foo foo = 50001;
}

message Foo {
  optional string bar = 1;
}`,
		"a/c/c.proto": `import "a/b/b.proto";

message ReferencesFooOption {
  optional string baz = 1 [(a.b.foo).bat = "hello"];
}`,
	}
	errMsg := "a/c/c.proto:4:38: field ReferencesFooOption.baz: option (a.b.foo).bat: field bat of a.b.Foo does not exist"

	acc := func(filename string) (io.ReadCloser, error) {
		f, ok := input[filename]
		if !ok {
			return nil, fmt.Errorf("file not found: %s", filename)
		}
		return io.NopCloser(strings.NewReader(f)), nil
	}
	names := make([]string, 0, len(input))
	for k := range input {
		names = append(names, k)
	}
	var errs []error
	_, err := Parser{
		Accessor: acc,
		ErrorReporter: func(errorWithPos ErrorWithPos) error {
			errs = append(errs, errorWithPos)
			// need to return nil to make sure this test case works
			// this will result in us only getting an error from errorHandler.getError()
			// we need to make sure this is called correctly in the linker so that all
			// errors are properly propagated from the return value of linkFiles(), and
			// therefor Parse returns ErrInvalidSource
			return nil
		},
	}.ParseFiles(names...)
	if err != ErrInvalidSource {
		t.Errorf("expecting validation error %v; instead got: %v", ErrInvalidSource, err)
	} else if len(errs) != 1 || errs[0].Error() != errMsg {
		t.Errorf("expecting validation error %q; instead got: %q", errs[0].Error(), errMsg)
	}
}

func TestSyntheticOneOfCollisions(t *testing.T) {
	input := map[string]string{
		"foo1.proto": "syntax = \"proto3\";\n" +
			"message Foo {\n" +
			"  optional string bar = 1;\n" +
			"}\n",
		"foo2.proto": "syntax = \"proto3\";\n" +
			"message Foo {\n" +
			"  optional string bar = 1;\n" +
			"}\n",
	}
	acc := func(filename string) (io.ReadCloser, error) {
		f, ok := input[filename]
		if !ok {
			return nil, fmt.Errorf("file not found: %s", filename)
		}
		return io.NopCloser(strings.NewReader(f)), nil
	}

	var errs []error
	errReporter := func(errorWithPos ErrorWithPos) error {
		errs = append(errs, errorWithPos)
		// need to return nil to accumulate all errors so we can report synthetic
		// oneof collision; otherwise, the link will fail after the first collision
		// and we'll never test the synthetic oneofs
		return nil
	}

	_, err := Parser{
		Accessor:      acc,
		ErrorReporter: errReporter,
	}.ParseFiles("foo1.proto", "foo2.proto")

	testutil.Eq(t, ErrInvalidSource, err)
	expectedOption1 := []string{
		`foo1.proto:2:9: symbol "Foo" already defined at foo2.proto:2:9`,
		`foo1.proto:3:19: symbol "Foo.bar" already defined at foo2.proto:3:19`,
		`foo1.proto:3:19: symbol "Foo._bar" already defined at foo2.proto:3:19`,
	}
	expectedOption2 := []string{
		`foo2.proto:2:9: symbol "Foo" already defined at foo1.proto:2:9`,
		`foo2.proto:3:19: symbol "Foo.bar" already defined at foo1.proto:3:19`,
		`foo2.proto:3:19: symbol "Foo._bar" already defined at foo1.proto:3:19`,
	}

	var actual []string
	for _, err := range errs {
		actual = append(actual, err.Error())
	}
	// Errors expected depend on which file is compiled first. This is mostly deterministic
	// with parallelism of 1, but some things (like enabling -race in tests) can change the
	// expected order.
	if !reflect.DeepEqual(expectedOption1, actual) && !reflect.DeepEqual(expectedOption2, actual) {
		t.Errorf("got errors:\n:%v\nbut wanted EITHER:\n%v\n  OR:\n%v", actual, expectedOption1, expectedOption2)
	}
}

func TestCustomJSONNameWarnings(t *testing.T) {
	testCases := []struct {
		source  string
		warning string
	}{
		{
			source: "syntax = \"proto2\";\n" +
				"message Foo {\n" +
				"  optional string foo_bar = 1;\n" +
				"  optional string fooBar = 2;\n" +
				"}\n",
			warning: "test.proto:4:3: field Foo.fooBar: default JSON name \"fooBar\" conflicts with default JSON name of field foo_bar, defined at test.proto:3:3",
		},
		{
			source: "syntax = \"proto2\";\n" +
				"message Foo {\n" +
				"  optional string foo_bar = 1;\n" +
				"  optional string fooBar = 2;\n" +
				"}\n",
			warning: "test.proto:4:3: field Foo.fooBar: default JSON name \"fooBar\" conflicts with default JSON name of field foo_bar, defined at test.proto:3:3",
		},
		// in nested message
		{
			source: "syntax = \"proto2\";\n" +
				"message Blah { message Foo {\n" +
				"  optional string foo_bar = 1;\n" +
				"  optional string fooBar = 2;\n" +
				"} }\n",
			warning: "test.proto:4:3: field Foo.fooBar: default JSON name \"fooBar\" conflicts with default JSON name of field foo_bar, defined at test.proto:3:3",
		},
		{
			source: "syntax = \"proto2\";\n" +
				"message Blah { message Foo {\n" +
				"  optional string foo_bar = 1;\n" +
				"  optional string fooBar = 2;\n" +
				"} }\n",
			warning: "test.proto:4:3: field Foo.fooBar: default JSON name \"fooBar\" conflicts with default JSON name of field foo_bar, defined at test.proto:3:3",
		},
		// enum values
		{
			source: "syntax = \"proto2\";\n" +
				"enum Foo {\n" +
				"  true = 0;\n" +
				"  TRUE = 1;\n" +
				"}\n",
			warning: "test.proto:4:3: enum value Foo.TRUE: camel-case name (with optional enum name prefix removed) \"True\" conflicts with camel-case name of enum value true, defined at test.proto:3:3",
		},
		{
			source: "syntax = \"proto2\";\n" +
				"enum Foo {\n" +
				"  fooBar_Baz = 0;\n" +
				"  _FOO__BAR_BAZ = 1;\n" +
				"}\n",
			warning: "test.proto:4:3: enum value Foo._FOO__BAR_BAZ: camel-case name (with optional enum name prefix removed) \"BarBaz\" conflicts with camel-case name of enum value fooBar_Baz, defined at test.proto:3:3",
		},
		{
			source: "syntax = \"proto2\";\n" +
				"enum Foo {\n" +
				"  fooBar_Baz = 0;\n" +
				"  FOO__BAR__BAZ__ = 1;\n" +
				"}\n",
			warning: "test.proto:4:3: enum value Foo.FOO__BAR__BAZ__: camel-case name (with optional enum name prefix removed) \"BarBaz\" conflicts with camel-case name of enum value fooBar_Baz, defined at test.proto:3:3",
		},
		{
			source: "syntax = \"proto2\";\n" +
				"enum Foo {\n" +
				"  fooBarBaz = 0;\n" +
				"  _FOO__BAR_BAZ = 1;\n" +
				"}\n",
			warning: "",
		},
		{
			source: "syntax = \"proto2\";\n" +
				"enum Foo {\n" +
				"  option allow_alias = true;\n" +
				"  Bar_Baz = 0;\n" +
				"  _BAR_BAZ_ = 0;\n" +
				"  FOO_BAR_BAZ = 0;\n" +
				"  foobar_baz = 0;\n" +
				"}\n",
			warning: "",
		},
		// in nested message
		{
			source: "syntax = \"proto2\";\n" +
				"message Blah { enum Foo {\n" +
				"  true = 0;\n" +
				"  TRUE = 1;\n" +
				"} }\n",
			warning: "test.proto:4:3: enum value Foo.TRUE: camel-case name (with optional enum name prefix removed) \"True\" conflicts with camel-case name of enum value true, defined at test.proto:3:3",
		},
		{
			source: "syntax = \"proto2\";\n" +
				"message Blah { enum Foo {\n" +
				"  fooBar_Baz = 0;\n" +
				"  _FOO__BAR_BAZ = 1;\n" +
				"} }\n",
			warning: "test.proto:4:3: enum value Foo._FOO__BAR_BAZ: camel-case name (with optional enum name prefix removed) \"BarBaz\" conflicts with camel-case name of enum value fooBar_Baz, defined at test.proto:3:3",
		},
		{
			source: "syntax = \"proto2\";\n" +
				"message Blah { enum Foo {\n" +
				"  option allow_alias = true;\n" +
				"  Bar_Baz = 0;\n" +
				"  _BAR_BAZ_ = 0;\n" +
				"  FOO_BAR_BAZ = 0;\n" +
				"  foobar_baz = 0;\n" +
				"} }\n",
			warning: "",
		},
	}
	for i, tc := range testCases {
		acc := func(filename string) (io.ReadCloser, error) {
			if filename == "test.proto" {
				return io.NopCloser(strings.NewReader(tc.source)), nil
			}
			return nil, fmt.Errorf("file not found: %s", filename)
		}
		var warnings []string
		warnFunc := func(err ErrorWithPos) {
			warnings = append(warnings, err.Error())
		}
		_, err := Parser{Accessor: acc, WarningReporter: warnFunc}.ParseFiles("test.proto")
		if err != nil {
			t.Errorf("case %d: expecting no error; instead got error %q", i, err)
		}
		if tc.warning == "" && len(warnings) > 0 {
			t.Errorf("case %d: expecting no warnings; instead got: %v", i, warnings)
		} else if tc.warning != "" {
			found := false
			for _, w := range warnings {
				if w == tc.warning {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("case %d: expecting warning %q; instead got: %v", i, tc.warning, warnings)
			}
		}
	}
}
