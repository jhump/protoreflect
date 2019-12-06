package protoparse

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestEmptyParse(t *testing.T) {
	p := Parser{
		Accessor: func(filename string) (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(nil)), nil
		},
	}
	fd, err := p.ParseFiles("foo.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, 1, len(fd))
	testutil.Eq(t, "foo.proto", fd[0].GetName())
	testutil.Eq(t, 0, len(fd[0].GetDependencies()))
	testutil.Eq(t, 0, len(fd[0].GetMessageTypes()))
	testutil.Eq(t, 0, len(fd[0].GetEnumTypes()))
	testutil.Eq(t, 0, len(fd[0].GetExtensions()))
	testutil.Eq(t, 0, len(fd[0].GetServices()))
}

func TestSimpleParse(t *testing.T) {
	protos := map[string]*parseResult{}

	// Just verify that we can successfully parse the same files we use for
	// testing. We do a *very* shallow check of what was parsed because we know
	// it won't be fully correct until after linking. (So that will be tested
	// below, where we parse *and* link.)
	res, err := parseFileForTest("../../internal/testprotos/desc_test1.proto")
	testutil.Ok(t, err)
	fd := res.fd
	testutil.Eq(t, "../../internal/testprotos/desc_test1.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "xtm"))
	testutil.Require(t, hasMessage(fd, "TestMessage"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test2.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/desc_test2.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "groupx"))
	testutil.Require(t, hasMessage(fd, "GroupX"))
	testutil.Require(t, hasMessage(fd, "Frobnitz"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_defaults.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/desc_test_defaults.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "PrimitiveDefaults"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_field_types.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/desc_test_field_types.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "TestEnum"))
	testutil.Require(t, hasMessage(fd, "UnaryFields"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_options.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/desc_test_options.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "mfubar"))
	testutil.Require(t, hasEnum(fd, "ReallySimpleEnum"))
	testutil.Require(t, hasMessage(fd, "ReallySimpleMessage"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_proto3.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/desc_test_proto3.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "Proto3Enum"))
	testutil.Require(t, hasService(fd, "TestService"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_wellknowntypes.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/desc_test_wellknowntypes.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "TestWellKnownTypes"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/nopkg/desc_test_nopkg.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/nopkg/desc_test_nopkg.proto", fd.GetName())
	testutil.Eq(t, "", fd.GetPackage())
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/nopkg/desc_test_nopkg_new.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/nopkg/desc_test_nopkg_new.proto", fd.GetName())
	testutil.Eq(t, "", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "TopLevel"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/pkg/desc_test_pkg.proto")
	testutil.Ok(t, err)
	fd = res.fd
	testutil.Eq(t, "../../internal/testprotos/pkg/desc_test_pkg.proto", fd.GetName())
	testutil.Eq(t, "jhump.protoreflect.desc", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "Foo"))
	testutil.Require(t, hasMessage(fd, "Bar"))
	protos[fd.GetName()] = res

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

func parseFileForTest(filename string) (*parseResult, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	errs := newErrorHandler(nil)
	res := parseProto(filename, f, errs, true)
	return res, errs.getError()
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
			errMsg:   `test.proto:1:10: syntax value must be 'proto2' or 'proto3'`,
		},
		{
			contents: `message Foo { optional string s = 5000000000; }`,
			errMsg:   `test.proto:1:35: tag number 5000000000 is higher than max allowed tag number (536870911)`,
		},
		{
			contents: `message Foo { optional string s = 19500; }`,
			errMsg:   `test.proto:1:35: tag number 19500 is in disallowed reserved range 19000-19999`,
		},
		{
			contents: `enum Foo { V = 5000000000; }`,
			errMsg:   `test.proto:1:16: constant 5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = -5000000000; }`,
			errMsg:   `test.proto:1:16: constant -5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = 0; reserved 5000000000; }`,
			errMsg:   `test.proto:1:28: constant 5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5000000000; }`,
			errMsg:   `test.proto:1:28: constant -5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = 0; reserved 5000000000 to 5000000001; }`,
			errMsg:   `test.proto:1:28: constant 5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = 0; reserved 5 to 5000000000; }`,
			errMsg:   `test.proto:1:33: constant 5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5000000000 to -5; }`,
			errMsg:   `test.proto:1:28: constant -5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5000000001 to -5000000000; }`,
			errMsg:   `test.proto:1:28: constant -5000000001 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5000000000 to 5; }`,
			errMsg:   `test.proto:1:28: constant -5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5 to 5000000000; }`,
			errMsg:   `test.proto:1:34: constant 5000000000 is out of range for int32 (-2147483648 to 2147483647)`,
		},
		{
			contents: `enum Foo { }`,
			errMsg:   `test.proto:1:1: enum Foo: enums must define at least one value`,
		},
		{
			contents: `message Foo { oneof Bar { } }`,
			errMsg:   `test.proto:1:15: oneof must contain at least one field`,
		},
		{
			contents: `message Foo { extensions 1 to max; } extend Foo { }`,
			errMsg:   `test.proto:1:38: extend sections must define at least one extension`,
		},
		{
			contents: `message Foo { option map_entry = true; }`,
			errMsg:   `test.proto:1:34: message Foo: map_entry option should not be set explicitly; use map type instead`,
		},
		{
			contents: `message Foo { option map_entry = false; }`,
			succeeds: true, // okay if explicit setting is false
		},
		{
			contents: `syntax = "proto2"; message Foo { string s = 1; }`,
			errMsg:   `test.proto:1:41: field Foo.s: field has no label, but proto2 must indicate 'optional' or 'required'`,
		},
		{
			contents: `message Foo { string s = 1; }`, // syntax defaults to proto2
			errMsg:   `test.proto:1:22: field Foo.s: field has no label, but proto2 must indicate 'optional' or 'required'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { optional string s = 1; }`,
			errMsg:   `test.proto:1:34: field Foo.s: field has label LABEL_OPTIONAL, but proto3 must omit labels other than 'repeated'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { required string s = 1; }`,
			errMsg:   `test.proto:1:34: field Foo.s: field has label LABEL_REQUIRED, but proto3 must omit labels other than 'repeated'`,
		},
		{
			contents: `message Foo { extensions 1 to max; } extend Foo { required string sss = 100; }`,
			errMsg:   `test.proto:1:51: field sss: extension fields cannot be 'required'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { optional group Grp = 1 { } }`,
			errMsg:   `test.proto:1:43: field Foo.grp: groups are not allowed in proto3`,
		},
		{
			contents: `syntax = "proto3"; message Foo { extensions 1 to max; }`,
			errMsg:   `test.proto:1:45: message Foo: extension ranges are not allowed in proto3`,
		},
		{
			contents: `syntax = "proto3"; message Foo { string s = 1 [default = "abcdef"]; }`,
			errMsg:   `test.proto:1:48: field Foo.s: default values are not allowed in proto3`,
		},
		{
			contents: `enum Foo { V1 = 1; V2 = 1; }`,
			errMsg:   `test.proto:1:25: enum Foo: values V1 and V2 both have the same numeric value 1; use allow_alias option if intentional`,
		},
		{
			contents: `enum Foo { option allow_alias = true; V1 = 1; V2 = 1; }`,
			succeeds: true,
		},
		{
			contents: `syntax = "proto3"; enum Foo { V1 = 0; reserved 1 to 20; reserved "V2"; }`,
			succeeds: true,
		},
		{
			contents: `enum Foo { V1 = 1; reserved 1 to 20; reserved "V2"; }`,
			errMsg:   `test.proto:1:17: enum Foo: value V1 is using number 1 which is in reserved range 1 to 20`,
		},
		{
			contents: `enum Foo { V1 = 20; reserved 1 to 20; reserved "V2"; }`,
			errMsg:   `test.proto:1:17: enum Foo: value V1 is using number 20 which is in reserved range 1 to 20`,
		},
		{
			contents: `enum Foo { V2 = 0; reserved 1 to 20; reserved "V2"; }`,
			errMsg:   `test.proto:1:12: enum Foo: value V2 is using a reserved name`,
		},
		{
			contents: `enum Foo { V0 = 0; reserved 1 to 20; reserved 21 to 40; reserved "V2"; }`,
			succeeds: true,
		},
		{
			contents: `enum Foo { V0 = 0; reserved 1 to 20; reserved 20 to 40; reserved "V2"; }`,
			errMsg:   `test.proto:1:47: enum Foo: reserved ranges overlap: 1 to 20 and 20 to 40`,
		},
		{
			contents: `syntax = "proto3"; enum Foo { FIRST = 1; }`,
			errMsg:   `test.proto:1:39: enum Foo: proto3 requires that first value in enum have numeric value of 0`,
		},
		{
			contents: `syntax = "proto3"; message Foo { string s = 1; int32 i = 1; }`,
			errMsg:   `test.proto:1:58: message Foo: fields s and i both have the same tag 1`,
		},
		{
			contents: `message Foo { reserved 1 to 10, 10 to 12; }`,
			errMsg:   `test.proto:1:33: message Foo: reserved ranges overlap: 1 to 10 and 10 to 12`,
		},
		{
			contents: `message Foo { extensions 1 to 10, 10 to 12; }`,
			errMsg:   `test.proto:1:35: message Foo: extension ranges overlap: 1 to 10 and 10 to 12`,
		},
		{
			contents: `message Foo { reserved 1 to 10; extensions 10 to 12; }`,
			errMsg:   `test.proto:1:44: message Foo: extension range 10 to 12 overlaps reserved range 1 to 10`,
		},
		{
			contents: `message Foo { reserved 1, 2 to 10, 11 to 20; extensions 21 to 22; }`,
			succeeds: true,
		},
		{
			contents: `message Foo { reserved 10 to 1; }`,
			errMsg:   `test.proto:1:24: range, 10 to 1, is invalid: start must be <= end`,
		},
		{
			contents: `message Foo { extensions 10 to 1; }`,
			errMsg:   `test.proto:1:26: range, 10 to 1, is invalid: start must be <= end`,
		},
		{
			contents: `message Foo { reserved 1 to 5000000000; }`,
			errMsg:   `test.proto:1:29: range end is out-of-range tag: 5000000000 (should be between 0 and 536870911)`,
		},
		{
			contents: `message Foo { extensions 1000000000; }`,
			errMsg:   `test.proto:1:26: range includes out-of-range tag: 1000000000 (should be between 0 and 536870911)`,
		},
		{
			contents: `message Foo { extensions 1000000000 to 1000000001; }`,
			errMsg:   `test.proto:1:26: range start is out-of-range tag: 1000000000 (should be between 0 and 536870911)`,
		},
		{
			contents: `message Foo { extensions 100 to 1000000000; }`,
			errMsg:   `test.proto:1:33: range end is out-of-range tag: 1000000000 (should be between 0 and 536870911)`,
		},
		{
			contents: `message Foo { reserved "foo", "foo"; }`,
			errMsg:   `test.proto:1:31: name "foo" is reserved multiple times`,
		},
		{
			contents: `message Foo { reserved "foo"; optional string foo = 1; }`,
			errMsg:   `test.proto:1:47: message Foo: field foo is using a reserved name`,
		},
		{
			contents: `message Foo { reserved 1 to 10; optional string foo = 1; }`,
			errMsg:   `test.proto:1:55: message Foo: field foo is using tag 1 which is in reserved range 1 to 10`,
		},
		{
			contents: `message Foo { extensions 1 to 10; optional string foo = 1; }`,
			errMsg:   `test.proto:1:57: message Foo: field foo is using tag 1 which is in extension range 1 to 10`,
		},
	}

	for i, tc := range testCases {
		errs := newErrorHandler(nil)
		_ = parseProto("test.proto", strings.NewReader(tc.contents), errs, true)
		err := errs.getError()
		if tc.succeeds {
			testutil.Ok(t, err, "case #%d should succeed", i)
		} else {
			testutil.Nok(t, err, "case #%d should fail", i)
			testutil.Eq(t, tc.errMsg, err.Error(), "case #%d bad error message", i)
		}
	}
}

func TestAggregateValueInUninterpretedOptions(t *testing.T) {
	res, err := parseFileForTest("../../internal/testprotos/desc_test_complex.proto")
	testutil.Ok(t, err)
	fd := res.fd

	aggregateValue1 := *fd.Service[0].Method[0].Options.UninterpretedOption[0].AggregateValue
	testutil.Eq(t, "{ authenticated: true permission{ action: LOGIN entity: \"client\" } }", aggregateValue1)

	aggregateValue2 := *fd.Service[0].Method[1].Options.UninterpretedOption[0].AggregateValue
	testutil.Eq(t, "{ authenticated: true permission{ action: READ entity: \"user\" } }", aggregateValue2)
}

func TestParseFilesMessageComments(t *testing.T) {
	p := Parser{
		IncludeSourceCodeInfo: true,
	}
	protos, err := p.ParseFiles("../../internal/testprotos/desc_test1.proto")
	testutil.Ok(t, err)
	comments := ""
	expected := " Comment for TestMessage\n"
	for _, p := range protos {
		msg := p.FindMessage("testprotos.TestMessage")
		if msg != nil {
			si := msg.GetSourceInfo()
			if si != nil {
				comments = si.GetLeadingComments()
			}
			break
		}
	}
	testutil.Eq(t, expected, comments)
}

func TestParseFilesWithImportsNoImportPath(t *testing.T) {
	relFilePaths := []string{
		"a/b/b1.proto",
		"a/b/b2.proto",
		"c/c.proto",
	}

	pwd, err := os.Getwd()
	testutil.Require(t, err == nil, "%v", err)

	err = os.Chdir("../../internal/testprotos/protoparse")
	testutil.Require(t, err == nil, "%v", err)
	p := Parser{}
	protos, parseErr := p.ParseFiles(relFilePaths...)
	err = os.Chdir(pwd)
	testutil.Require(t, err == nil, "%v", err)
	testutil.Require(t, parseErr == nil, "%v", parseErr)

	testutil.Ok(t, err)
	testutil.Eq(t, len(relFilePaths), len(protos))
}

func TestParseFilesWithDependencies(t *testing.T) {
	// Create some file contents that import a non-well-known proto.
	// (One of the protos in internal/testprotos is fine.)
	contents := map[string]string{
		"test.proto": `
			syntax = "proto3";
			import "desc_test_wellknowntypes.proto";

			message TestImportedType {
				testprotos.TestWellKnownTypes imported_field = 1;
			}
		`,
	}

	// Establish that we *can* parse the source file with a parser that
	// registers the dependency.
	t.Run("DependencyIncluded", func(t *testing.T) {
		// Create a dependency-aware parser.
		parser := Parser{
			Accessor: FileContentsFromMap(contents),
			LookupImport: func(imp string) (*desc.FileDescriptor, error) {
				if imp == "desc_test_wellknowntypes.proto" {
					return desc.LoadFileDescriptor(imp)
				}
				return nil, errors.New("unexpected filename")
			},
		}
		if _, err := parser.ParseFiles("test.proto"); err != nil {
			t.Errorf("Could not parse with a non-well-known import: %v", err)
		}
	})

	// Establish that we *can not* parse the source file with a parser that
	// did not register the dependency.
	t.Run("DependencyExcluded", func(t *testing.T) {
		// Create a dependency-aware parser.
		parser := Parser{
			Accessor: FileContentsFromMap(contents),
		}
		if _, err := parser.ParseFiles("test.proto"); err == nil {
			t.Errorf("Expected parse to fail due to lack of an import.")
		}
	})

	// Establish that the accessor has precedence over LookupImport.
	t.Run("AccessorWins", func(t *testing.T) {
		// Create a dependency-aware parser that should never be called.
		parser := Parser{
			Accessor: FileContentsFromMap(map[string]string{
				"test.proto": `syntax = "proto3";`,
			}),
			LookupImport: func(imp string) (*desc.FileDescriptor, error) {
				t.Errorf("LookupImport was called on a filename available to the Accessor.")
				return nil, errors.New("unimportant")
			},
		}
		if _, err := parser.ParseFiles("test.proto"); err != nil {
			t.Error(err)
		}
	})
}

func TestParseCommentsBeforeDot(t *testing.T) {
	accessor := FileContentsFromMap(map[string]string{
		"test.proto": `
syntax = "proto3";
message Foo {
  // leading comments
  .Foo foo = 1;
}
`,
	})

	p := Parser{
		Accessor:              accessor,
		IncludeSourceCodeInfo: true,
	}
	fds, err := p.ParseFiles("test.proto")
	testutil.Ok(t, err)

	comment := fds[0].GetMessageTypes()[0].GetFields()[0].GetSourceInfo().GetLeadingComments()
	testutil.Eq(t, " leading comments\n", comment)
}
