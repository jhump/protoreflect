package protoparse

import (
	"sort"
	"strings"
	"testing"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestSimpleParse(t *testing.T) {
	protos := map[string]*dpb.FileDescriptorProto{}

	// Just verify that we can successfully parse the same files we use for
	// testing. We do a *very* shallow check of what was parsed because we know
	// it won't be fully correct until after linking. (So that will be tested
	// below, where we parse *and* link.)
	fd, err := parseProtoFile("../../internal/testprotos/desc_test1.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/desc_test1.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "xtm"))
	testutil.Require(t, hasMessage(fd, "TestMessage"))
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/desc_test2.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/desc_test2.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "groupx"))
	testutil.Require(t, hasMessage(fd, "GroupX"))
	testutil.Require(t, hasMessage(fd, "Frobnitz"))
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/desc_test_defaults.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/desc_test_defaults.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "PrimitiveDefaults"))
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/desc_test_field_types.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/desc_test_field_types.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "TestEnum"))
	testutil.Require(t, hasMessage(fd, "UnaryFields"))
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/desc_test_options.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/desc_test_options.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "mfubar"))
	testutil.Require(t, hasEnum(fd, "ReallySimpleEnum"))
	testutil.Require(t, hasMessage(fd, "ReallySimpleMessage"))
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/desc_test_proto3.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/desc_test_proto3.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "Proto3Enum"))
	testutil.Require(t, hasService(fd, "TestService"))
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/desc_test_wellknowntypes.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/desc_test_wellknowntypes.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "TestWellKnownTypes"))
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/nopkg/desc_test_nopkg.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/nopkg/desc_test_nopkg.proto", fd.GetName())
	testutil.Eq(t, "", fd.GetPackage())
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/nopkg/desc_test_nopkg_new.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/nopkg/desc_test_nopkg_new.proto", fd.GetName())
	testutil.Eq(t, "", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "TopLevel"))
	protos[fd.GetName()] = fd

	fd, err = parseProtoFile("../../internal/testprotos/pkg/desc_test_pkg.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, "../../internal/testprotos/pkg/desc_test_pkg.proto", fd.GetName())
	testutil.Eq(t, "jhump.protoreflect.desc", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "Foo"))
	testutil.Require(t, hasMessage(fd, "Bar"))
	protos[fd.GetName()] = fd

	// We'll also check our fixup logic to make sure it correctly rewrites the
	// names of the files to match corresponding import statementes. This should
	// strip the "../../internal/testprotos/" prefix from each file.
	protos = fixupFilenames(protos)
	var actual []string
	for n := range protos {
		actual = append(actual, n)
	}
	sort.Strings(actual)
	expected := []string{
		"desc_test1.proto",
		"desc_test2.proto",
		"desc_test_defaults.proto",
		"desc_test_field_types.proto",
		"desc_test_options.proto",
		"desc_test_proto3.proto",
		"desc_test_wellknowntypes.proto",
		"nopkg/desc_test_nopkg.proto",
		"nopkg/desc_test_nopkg_new.proto",
		"pkg/desc_test_pkg.proto",
	}
	testutil.Eq(t, expected, actual)
}

func hasExtension(fd *dpb.FileDescriptorProto, name string) bool {
	for _, ext := range fd.Extension {
		if ext.GetName() == name {
			return true
		}
	}
	return false
}

func hasMessage(fd *dpb.FileDescriptorProto, name string) bool {
	for _, md := range fd.MessageType {
		if md.GetName() == name {
			return true
		}
	}
	return false
}

func hasEnum(fd *dpb.FileDescriptorProto, name string) bool {
	for _, ed := range fd.EnumType {
		if ed.GetName() == name {
			return true
		}
	}
	return false
}

func hasService(fd *dpb.FileDescriptorProto, name string) bool {
	for _, sd := range fd.Service {
		if sd.GetName() == name {
			return true
		}
	}
	return false
}

func TestBasicValidation(t *testing.T) {
	testCases := []struct {
		contents string
		succeeds bool
		errMsg   string
	}{
		{
			contents: `syntax = "proto1";`,
			errMsg:   `syntax value must be 'proto2' or 'proto3'`,
		},
		{
			contents: `message Foo { optional string s = 5000000000; }`,
			errMsg:   `higher than max allowed tag number`,
		},
		{
			contents: `message Foo { optional string s = 19500; }`,
			errMsg:   `in disallowed reserved range`,
		},
		{
			contents: `enum Foo { V = 5000000000; }`,
			errMsg:   `is out of range for int32`,
		},
		{
			contents: `enum Foo { V = -5000000000; }`,
			errMsg:   `is out of range for int32`,
		},
		{
			contents: `enum Foo { }`,
			errMsg:   `enums must define at least one value`,
		},
		{
			contents: `message Foo { oneof Bar { } }`,
			errMsg:   `oneof must contain at least one field`,
		},
		{
			contents: `message Foo { extensions 1 to max; } extend Foo { }`,
			errMsg:   `extend sections must define at least one extension`,
		},
		{
			contents: `message Foo { option map_entry = true; }`,
			errMsg:   `map_entry option should not be set explicitly; use map type instead`,
		},
		{
			contents: `message Foo { option map_entry = false; }`,
			succeeds: true, // okay if explicit setting is false
		},
		{
			contents: `syntax = "proto2"; message Foo { string s = 1; }`,
			errMsg:   `field has no label, but proto2 must indicate 'optional' or 'required'`,
		},
		{
			contents: `message Foo { string s = 1; }`, // syntax defaults to proto2
			errMsg:   `field has no label, but proto2 must indicate 'optional' or 'required'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { optional string s = 1; }`,
			errMsg:   `field has label LABEL_OPTIONAL, but proto3 should omit labels other than 'repeated'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { required string s = 1; }`,
			errMsg:   `field has label LABEL_REQUIRED, but proto3 should omit labels other than 'repeated'`,
		},
		{
			contents: `message Foo { extensions 1 to max; } extend Foo { required string sss = 100; }`,
			errMsg:   `extension fields cannot be 'required'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { optional group Grp = 1 { } }`,
			errMsg:   `groups are not allowed in proto3`,
		},
		{
			contents: `syntax = "proto3"; message Foo { extensions 1 to max; }`,
			errMsg:   `extension ranges are not allowed in proto3`,
		},
		{
			contents: `syntax = "proto3"; message Foo { string s = 1 [default = "abcdef"]; }`,
			errMsg:   `default values are not allowed in proto3`,
		},
		{
			contents: `enum Foo { V1 = 1; V2 = 1; }`,
			errMsg:   `values V1 and V2 both have the same numeric value 1; use allow_alias option if intentional`,
		},
		{
			contents: `enum Foo { option allow_alias = true; V1 = 1; V2 = 1; }`,
			succeeds: true,
		},
		{
			contents: `syntax = "proto3"; enum Foo { FIRST = 1; }`,
			errMsg:   `proto3 requires that first value in enum have numeric value of 0`,
		},
		{
			contents: `syntax = "proto3"; message Foo { string s = 1; int32 i = 1; }`,
			errMsg:   `fields s and i both have the same tag 1`,
		},
		{
			contents: `message Foo { reserved 1 to 10, 10 to 12; }`,
			errMsg:   `reserved ranges overlap: 1 to 10 and 10 to 12`,
		},
		{
			contents: `message Foo { extensions 1 to 10, 10 to 12; }`,
			errMsg:   `extension ranges overlap: 1 to 10 and 10 to 12`,
		},
		{
			contents: `message Foo { reserved 1 to 10; extensions 10 to 12; }`,
			errMsg:   `extension range 10 to 12 overlaps reserved range 1 to 10`,
		},
		{
			contents: `message Foo { reserved 1, 2 to 10, 11 to 20; extensions 21 to 22; }`,
			succeeds: true,
		},
		{
			contents: `message Foo { reserved 10 to 1; }`,
			errMsg:   `range, 10 to 1, is invalid: start must be <= end`,
		},
		{
			contents: `message Foo { extensions 10 to 1; }`,
			errMsg:   `range, 10 to 1, is invalid: start must be <= end`,
		},
		{
			contents: `message Foo { reserved 1 to 5000000000; }`,
			errMsg:   `range end is out-of-range tag`,
		},
		{
			contents: `message Foo { extensions 1000000000; }`,
			errMsg:   `range includes out-of-range tag`,
		},
		{
			contents: `message Foo { extensions 1000000000 to 1000000001; }`,
			errMsg:   `range start is out-of-range tag`,
		},
		{
			contents: `message Foo { extensions 1000000000 to 1000000001; }`,
			errMsg:   `range start is out-of-range tag`,
		},
		{
			contents: `message Foo { reserved "foo", "foo"; }`,
			errMsg:   `field "foo" is reserved multiple times`,
		},
		{
			contents: `message Foo { reserved "foo"; optional string foo = 1; }`,
			errMsg:   `field foo is using a reserved name`,
		},
		{
			contents: `message Foo { reserved 1 to 10; optional string foo = 1; }`,
			errMsg:   `field foo is using tag 1 which is in reserved range 1 to 10`,
		},
		{
			contents: `message Foo { extensions 1 to 10; optional string foo = 1; }`,
			errMsg:   `field foo is using tag 1 which is in extension range 1 to 10`,
		},
	}

	for i, tc := range testCases {
		_, err := parseProto("test.proto", strings.NewReader(tc.contents), map[string][]*aggregate{})
		if tc.succeeds {
			testutil.Ok(t, err, "case #%d should succeed", i)
		} else {
			testutil.Require(t, err != nil, "case #%d should fail", i)
			testutil.Require(t, strings.Contains(err.Error(), tc.errMsg), "case #%d should contain %q but does not: %q", i, tc.errMsg, err.Error())
		}
	}
}
