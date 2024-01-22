package protoparse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
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

func TestJunkParse(t *testing.T) {
	// inputs that have been found in the past to cause panics by oss-fuzz
	inputs := map[string]string{
		"case-34232": `'';`,
		"case-34238": `.`,
	}
	for name, input := range inputs {
		protoName := fmt.Sprintf("%s.proto", name)
		p := Parser{
			Accessor: FileContentsFromMap(map[string]string{protoName: input}),
		}
		_, err := p.ParseFiles(protoName)
		// we expect this to error... but we don't want it to panic
		testutil.Nok(t, err, "junk input should have returned error")
		t.Logf("error from parse: %v", err)
	}
}

func TestSimpleParse(t *testing.T) {
	protos := map[string]parser.Result{}

	// Just verify that we can successfully parse the same files we use for
	// testing. We do a *very* shallow check of what was parsed because we know
	// it won't be fully correct until after linking. (So that will be tested
	// below, where we parse *and* link.)
	res, err := parseFileForTest("../../internal/testprotos/desc_test1.proto")
	testutil.Ok(t, err)
	fd := res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/desc_test1.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "xtm"))
	testutil.Require(t, hasMessage(fd, "TestMessage"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test2.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/desc_test2.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "groupx"))
	testutil.Require(t, hasMessage(fd, "GroupX"))
	testutil.Require(t, hasMessage(fd, "Frobnitz"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_defaults.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/desc_test_defaults.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "PrimitiveDefaults"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_field_types.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/desc_test_field_types.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "TestEnum"))
	testutil.Require(t, hasMessage(fd, "UnaryFields"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_options.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/desc_test_options.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasExtension(fd, "mfubar"))
	testutil.Require(t, hasEnum(fd, "ReallySimpleEnum"))
	testutil.Require(t, hasMessage(fd, "ReallySimpleMessage"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_proto3.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/desc_test_proto3.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "Proto3Enum"))
	testutil.Require(t, hasService(fd, "TestService"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/desc_test_wellknowntypes.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/desc_test_wellknowntypes.proto", fd.GetName())
	testutil.Eq(t, "testprotos", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "TestWellKnownTypes"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/nopkg/desc_test_nopkg.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/nopkg/desc_test_nopkg.proto", fd.GetName())
	testutil.Eq(t, "", fd.GetPackage())
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/nopkg/desc_test_nopkg_new.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/nopkg/desc_test_nopkg_new.proto", fd.GetName())
	testutil.Eq(t, "", fd.GetPackage())
	testutil.Require(t, hasMessage(fd, "TopLevel"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../../internal/testprotos/pkg/desc_test_pkg.proto")
	testutil.Ok(t, err)
	fd = res.FileDescriptorProto()
	testutil.Eq(t, "../../internal/testprotos/pkg/desc_test_pkg.proto", fd.GetName())
	testutil.Eq(t, "jhump.protoreflect.desc", fd.GetPackage())
	testutil.Require(t, hasEnum(fd, "Foo"))
	testutil.Require(t, hasMessage(fd, "Bar"))
	protos[fd.GetName()] = res

	// We'll also check our fixup logic to make sure it correctly rewrites the
	// names of the files to match corresponding import statementes. This should
	// strip the "../../internal/testprotos/" prefix from each file.
	protos, _ = fixupFilenames(protos)
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

func parseFileForTest(filename string) (parser.Result, error) {
	filenames := []string{filename}
	res, _ := Parser{}.getResolver(filenames)
	rep := reporter.NewHandler(nil)
	results, err := parseToProtos(res, filenames, rep, true)
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func hasExtension(fd *descriptorpb.FileDescriptorProto, name string) bool {
	for _, ext := range fd.Extension {
		if ext.GetName() == name {
			return true
		}
	}
	return false
}

func hasMessage(fd *descriptorpb.FileDescriptorProto, name string) bool {
	for _, md := range fd.MessageType {
		if md.GetName() == name {
			return true
		}
	}
	return false
}

func hasEnum(fd *descriptorpb.FileDescriptorProto, name string) bool {
	for _, ed := range fd.EnumType {
		if ed.GetName() == name {
			return true
		}
	}
	return false
}

func hasService(fd *descriptorpb.FileDescriptorProto, name string) bool {
	for _, sd := range fd.Service {
		if sd.GetName() == name {
			return true
		}
	}
	return false
}

func TestAggregateValueInUninterpretedOptions(t *testing.T) {
	res, err := parseFileForTest("../../internal/testprotos/desc_test_complex.proto")
	testutil.Ok(t, err)
	fd := res.FileDescriptorProto()

	// service TestTestService, method UserAuth; first option
	aggregateValue1 := *fd.Service[0].Method[0].Options.UninterpretedOption[0].AggregateValue
	testutil.Eq(t, `authenticated : true permission : { action : LOGIN entity : "client" }`, aggregateValue1)

	// service TestTestService, method Get; first option
	aggregateValue2 := *fd.Service[0].Method[1].Options.UninterpretedOption[0].AggregateValue
	testutil.Eq(t, `authenticated : true permission : { action : READ entity : "user" }`, aggregateValue2)

	// message Another; first option
	aggregateValue3 := *fd.MessageType[4].Options.UninterpretedOption[0].AggregateValue
	testutil.Eq(t, `foo : "abc" s < name : "foo" , id : 123 > , array : [ 1 , 2 , 3 ] , r : [ < name : "f" > , { name : "s" } , { id : 456 } ] ,`, aggregateValue3)

	// message Test.Nested._NestedNested; second option (rept)
	//  (Test.Nested is at index 1 instead of 0 because of implicit nested message from map field m)
	aggregateValue4 := *fd.MessageType[1].NestedType[1].NestedType[0].Options.UninterpretedOption[1].AggregateValue
	testutil.Eq(t, `foo : "goo" [ foo . bar . Test . Nested . _NestedNested . _garblez ] : "boo"`, aggregateValue4)
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
	t.Run("DependencyIncludedProto", func(t *testing.T) {
		// Create a dependency-aware parser.
		parser := Parser{
			Accessor: FileContentsFromMap(contents),
			LookupImportProto: func(imp string) (*descriptorpb.FileDescriptorProto, error) {
				if imp == "desc_test_wellknowntypes.proto" {
					fileDescriptor, err := desc.LoadFileDescriptor(imp)
					if err != nil {
						return nil, err
					}
					return fileDescriptor.AsFileDescriptorProto(), nil
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
				// It's okay for descriptor.proto to be requested implicitly, but
				// nothing else should make it here since it should instead be
				// retrieved via Accessor.
				if imp != "google/protobuf/descriptor.proto" {
					t.Errorf("LookupImport was called on a filename available to the Accessor: %q", imp)
				}
				return nil, errors.New("unimportant")
			},
		}
		if _, err := parser.ParseFiles("test.proto"); err != nil {
			t.Error(err)
		}
	})
}

func TestParseFilesReturnsKnownExtensions(t *testing.T) {
	accessor := func(filename string) (io.ReadCloser, error) {
		if filename == "desc_test3.proto" {
			return io.NopCloser(strings.NewReader(`
				syntax = "proto3";
				import "desc_test_complex.proto";
				message Foo {
					foo.bar.Simple field = 1;
				}
			`)), nil
		}
		return os.Open(filepath.Join("../../internal/testprotos", filename))
	}
	p := Parser{
		Accessor: accessor,
	}
	fds, err := p.ParseFiles("desc_test3.proto")
	testutil.Ok(t, err)
	fd := fds[0].GetDependencies()[0]
	md := fd.FindMessage("foo.bar.Test.Nested._NestedNested")
	testutil.Require(t, md != nil)
	val, err := proto.GetExtension(md.GetOptions(), testprotos.E_Rept)
	testutil.Ok(t, err)
	_, ok := val.([]*testprotos.Test)
	testutil.Require(t, ok)
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

func TestParseInferImportPaths_SimpleNoOp(t *testing.T) {
	sources := map[string]string{
		"test.proto": `
		syntax = "proto3";
		import "google/protobuf/struct.proto";
		message Foo {
			string name = 1;
			repeated uint64 refs = 2;
			google.protobuf.Struct meta = 3;
		}`,
	}
	p := Parser{
		Accessor:         FileContentsFromMap(sources),
		InferImportPaths: true,
	}
	fds, err := p.ParseFiles("test.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, 1, len(fds))
}

func TestParseInferImportPaths_FixesNestedPaths(t *testing.T) {
	sources := FileContentsFromMap(map[string]string{
		"/foo/bar/a.proto": `
			syntax = "proto3";
			import "baz/b.proto";
			message A {
				B b = 1;
			}`,
		"/foo/bar/baz/b.proto": `
			syntax = "proto3";
			import "baz/c.proto";
			message B {
				C c = 1;
			}`,
		"/foo/bar/baz/c.proto": `
			syntax = "proto3";
			message C {}`,
		"/foo/bar/baz/d.proto": `
			syntax = "proto3";
			import "a.proto";
			message D {
				A a = 1;
			}`,
	})

	testCases := []struct {
		name      string
		cwd       string
		filenames []string
		expect    []string
	}{
		{
			name:      "outside hierarchy",
			cwd:       "/buzz",
			filenames: []string{"../foo/bar/a.proto", "../foo/bar/baz/b.proto", "../foo/bar/baz/c.proto", "../foo/bar/baz/d.proto"},
		},
		{
			name:      "inside hierarchy",
			cwd:       "/foo",
			filenames: []string{"bar/a.proto", "bar/baz/b.proto", "bar/baz/c.proto", "bar/baz/d.proto"},
		},
		{
			name:      "import path root (no-op)",
			cwd:       "/foo/bar",
			filenames: []string{"a.proto", "baz/b.proto", "baz/c.proto", "baz/d.proto"},
		},
		{
			name:      "inside leaf directory",
			cwd:       "/foo/bar/baz",
			filenames: []string{"../a.proto", "b.proto", "c.proto", "d.proto"},
			// NB: Expected names differ from above cases because nothing imports d.proto.
			//     So when inferring the root paths, the fact that d.proto is defined in
			//     the baz sub-directory will not be discovered. That's okay since the parse
			//     operation still succeeds.
			expect: []string{"a.proto", "baz/b.proto", "baz/c.proto", "d.proto"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p := Parser{
				Accessor:         sources,
				ImportPaths:      []string{testCase.cwd, "/foo/bar"},
				InferImportPaths: true,
			}
			fds, err := p.ParseFiles(testCase.filenames...)
			testutil.Ok(t, err)
			testutil.Eq(t, 4, len(fds))
			var expectedNames []string
			if len(testCase.expect) == 0 {
				expectedNames = []string{"a.proto", "baz/b.proto", "baz/c.proto", "baz/d.proto"}
			} else {
				testutil.Eq(t, 4, len(testCase.expect))
				expectedNames = testCase.expect
			}
			// check that they have the expected name
			testutil.Eq(t, expectedNames[0], fds[0].GetName())
			testutil.Eq(t, expectedNames[1], fds[1].GetName())
			testutil.Eq(t, expectedNames[2], fds[2].GetName())
			testutil.Eq(t, expectedNames[3], fds[3].GetName())
		})
	}
}

func TestParseFilesButDoNotLink_DoesNotUseImportPaths(t *testing.T) {
	tempdir, err := os.MkdirTemp("", "protoparse")
	testutil.Ok(t, err)
	defer func() {
		_ = os.RemoveAll(tempdir)
	}()
	err = os.WriteFile(filepath.Join(tempdir, "extra.proto"), []byte("package extra;"), 0644)
	testutil.Ok(t, err)
	mainPath := filepath.Join(tempdir, "main.proto")
	err = os.WriteFile(mainPath, []byte("package main; import \"extra.proto\";"), 0644)
	testutil.Ok(t, err)
	p := Parser{
		ImportPaths: []string{tempdir},
	}
	fds, err := p.ParseFilesButDoNotLink(mainPath)
	testutil.Ok(t, err)
	testutil.Eq(t, 1, len(fds))
}
