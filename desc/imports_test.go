package desc_test

import (
	"testing"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	_ "github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestResolveImport(t *testing.T) {
	desc.RegisterImportPath("desc_test1.proto", "foobar/desc_test1.proto")
	testutil.Eq(t, "desc_test1.proto", desc.ResolveImport("foobar/desc_test1.proto"))
	testutil.Eq(t, "foobar/snafu.proto", desc.ResolveImport("foobar/snafu.proto"))

	expectPanic(t, func() {
		desc.RegisterImportPath("", "foobar/desc_test1.proto")
	})
	expectPanic(t, func() {
		desc.RegisterImportPath("desc_test1.proto", "")
	})
	expectPanic(t, func() {
		// not a real registered path
		desc.RegisterImportPath("github.com/jhump/x/y/z/foobar.proto", "x/y/z/foobar.proto")
	})
}

func TestImportResolver(t *testing.T) {
	var r desc.ImportResolver

	expectPanic(t, func() {
		r.RegisterImportPath("", "a/b/c/d.proto")
	})
	expectPanic(t, func() {
		r.RegisterImportPath("d.proto", "")
	})

	// no source constraints
	r.RegisterImportPath("foo/bar.proto", "bar.proto")
	testutil.Eq(t, "foo/bar.proto", r.ResolveImport("test.proto", "bar.proto"))
	testutil.Eq(t, "foo/bar.proto", r.ResolveImport("some/other/source.proto", "bar.proto"))

	// with specific source file
	r.RegisterImportPathFrom("fubar/baz.proto", "baz.proto", "test/test.proto")
	// match
	testutil.Eq(t, "fubar/baz.proto", r.ResolveImport("test/test.proto", "baz.proto"))
	// no match
	testutil.Eq(t, "baz.proto", r.ResolveImport("test.proto", "baz.proto"))
	testutil.Eq(t, "baz.proto", r.ResolveImport("test/test2.proto", "baz.proto"))
	testutil.Eq(t, "baz.proto", r.ResolveImport("some/other/source.proto", "baz.proto"))

	// with specific source file with long path
	r.RegisterImportPathFrom("fubar/frobnitz/baz.proto", "baz.proto", "a/b/c/d/e/f/g/test/test.proto")
	// match
	testutil.Eq(t, "fubar/frobnitz/baz.proto", r.ResolveImport("a/b/c/d/e/f/g/test/test.proto", "baz.proto"))
	// no match
	testutil.Eq(t, "baz.proto", r.ResolveImport("test.proto", "baz.proto"))
	testutil.Eq(t, "baz.proto", r.ResolveImport("test/test2.proto", "baz.proto"))
	testutil.Eq(t, "baz.proto", r.ResolveImport("some/other/source.proto", "baz.proto"))

	// with source path
	r.RegisterImportPathFrom("fubar/frobnitz/snafu.proto", "frobnitz/snafu.proto", "a/b/c/d/e/f/g/h")
	// match
	testutil.Eq(t, "fubar/frobnitz/snafu.proto", r.ResolveImport("a/b/c/d/e/f/g/h/test/test.proto", "frobnitz/snafu.proto"))
	testutil.Eq(t, "fubar/frobnitz/snafu.proto", r.ResolveImport("a/b/c/d/e/f/g/h/abc.proto", "frobnitz/snafu.proto"))
	// no match
	testutil.Eq(t, "frobnitz/snafu.proto", r.ResolveImport("a/b/c/d/e/f/g/test/test.proto", "frobnitz/snafu.proto"))
	testutil.Eq(t, "frobnitz/snafu.proto", r.ResolveImport("test.proto", "frobnitz/snafu.proto"))
	testutil.Eq(t, "frobnitz/snafu.proto", r.ResolveImport("test/test2.proto", "frobnitz/snafu.proto"))
	testutil.Eq(t, "frobnitz/snafu.proto", r.ResolveImport("some/other/source.proto", "frobnitz/snafu.proto"))

	// falls back to global registered paths
	desc.RegisterImportPath("desc_test1.proto", "x/y/z/desc_test1.proto")
	testutil.Eq(t, "desc_test1.proto", r.ResolveImport("a/b/c/d/e/f/g/h/test/test.proto", "x/y/z/desc_test1.proto"))
}

func TestImportResolver_CreateFileDescriptors(t *testing.T) {
	p := protoparse.Parser{
		Accessor: protoparse.FileContentsFromMap(map[string]string{
			"foo/bar.proto": `
				syntax = "proto3";
				package foo;
				message Bar {
					string name = 1;
					uint64 id = 2;
				}
				`,
			// imports above file as just "bar.proto", so we need an
			// import resolver to properly load and link
			"fu/baz.proto": `
				syntax = "proto3";
				package fu;
				import "bar.proto";
				message Baz {
					repeated foo.Bar foobar = 1;
				}
				`,
		}),
		ImportPaths: []string{"foo"},
	}
	fds, err := p.ParseFilesButDoNotLink("foo/bar.proto", "fu/baz.proto")
	testutil.Ok(t, err)

	// Since we didn't link, fu.Baz.foobar field in second file has no type
	// (it can't know whether it's a message or enum until linking is done).
	// So go ahead and fill in the correct type:
	fds[1].MessageType[0].Field[0].Type = descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum()

	// sanity check: make sure linking fails without an import resolver
	_, err = desc.CreateFileDescriptors(fds)
	testutil.Require(t, err != nil)
	testutil.Eq(t, `no such file: "bar.proto"`, err.Error())

	// now try again with resolver
	var r desc.ImportResolver
	r.RegisterImportPath("foo/bar.proto", "bar.proto")
	linkedFiles, err := r.CreateFileDescriptors(fds)
	// success!
	testutil.Ok(t, err)

	// quick check of the resulting files
	fd := linkedFiles["foo/bar.proto"]
	testutil.Require(t, fd != nil)
	md := fd.FindMessage("foo.Bar")
	testutil.Require(t, md != nil)

	fd2 := linkedFiles["fu/baz.proto"]
	testutil.Require(t, fd2 != nil)
	md2 := fd2.FindMessage("fu.Baz")
	testutil.Require(t, md2 != nil)
	fld := md2.FindFieldByNumber(1)
	testutil.Require(t, fld != nil)
	testutil.Eq(t, md, fld.GetMessageType())
	testutil.Eq(t, fd, fd2.GetDependencies()[0])
}

func expectPanic(t *testing.T, fn func()) {
	defer func() {
		p := recover()
		testutil.Require(t, p != nil, "expecting panic")
	}()

	fn()
}
