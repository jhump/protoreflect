package sourceinfo_test

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/desc/sourceinfo"
	"github.com/jhump/protoreflect/internal/testprotos"
	_ "github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestUpdate(t *testing.T) {
	// other descriptor types are indirectly covered by other cases in this file
	t.Run("oneof", func(t *testing.T) {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName("testprotos.AnotherTestMessage.atmoo")
		testutil.Ok(t, err)
		ood, ok := d.(protoreflect.OneofDescriptor)
		testutil.Require(t, ok)

		testutil.Require(t, isZeroSourceLocation(ood.ParentFile().SourceLocations().ByDescriptor(ood))) // no source location
		ood, err = sourceinfo.UpdateDescriptor(ood)
		testutil.Ok(t, err)
		testutil.Require(t, !isZeroSourceLocation(ood.ParentFile().SourceLocations().ByDescriptor(ood))) // now there is a source location
	})
	t.Run("enum value", func(t *testing.T) {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName("testprotos.TestMessage.VALUE1")
		testutil.Ok(t, err)
		evd, ok := d.(protoreflect.EnumValueDescriptor)
		testutil.Require(t, ok)

		testutil.Require(t, isZeroSourceLocation(evd.ParentFile().SourceLocations().ByDescriptor(evd))) // no source location
		evd, err = sourceinfo.UpdateDescriptor(evd)
		testutil.Ok(t, err)
		testutil.Require(t, !isZeroSourceLocation(evd.ParentFile().SourceLocations().ByDescriptor(evd))) // now there is a source location
	})
	t.Run("field", func(t *testing.T) {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName("testprotos.TestMessage.nm")
		testutil.Ok(t, err)
		fld, ok := d.(protoreflect.FieldDescriptor)
		testutil.Require(t, ok)

		testutil.Require(t, isZeroSourceLocation(fld.ParentFile().SourceLocations().ByDescriptor(fld))) // no source location
		fld, err = sourceinfo.UpdateField(fld)
		testutil.Ok(t, err)
		testutil.Require(t, !isZeroSourceLocation(fld.ParentFile().SourceLocations().ByDescriptor(fld))) // now there is a source location
	})
	t.Run("extension", func(t *testing.T) {
		xtd := testprotos.E_Xui.TypeDescriptor()

		testutil.Require(t, isZeroSourceLocation(xtd.ParentFile().SourceLocations().ByDescriptor(xtd))) // no source location
		fld, err := sourceinfo.UpdateField(xtd)
		testutil.Ok(t, err)
		xtd, ok := fld.(protoreflect.ExtensionTypeDescriptor)
		testutil.Require(t, ok)
		testutil.Require(t, !isZeroSourceLocation(xtd.ParentFile().SourceLocations().ByDescriptor(xtd))) // now there is a source location
	})
	t.Run("method", func(t *testing.T) {
		d, err := protoregistry.GlobalFiles.FindDescriptorByName("testprotos.SomeService.SomeRPC")
		testutil.Ok(t, err)
		mtd, ok := d.(protoreflect.MethodDescriptor)
		testutil.Require(t, ok)

		testutil.Require(t, isZeroSourceLocation(mtd.ParentFile().SourceLocations().ByDescriptor(mtd))) // no source location
		mtd, err = sourceinfo.UpdateDescriptor(mtd)
		testutil.Ok(t, err)
		testutil.Require(t, !isZeroSourceLocation(mtd.ParentFile().SourceLocations().ByDescriptor(mtd))) // now there is a source location
	})
}

func TestWrapFile(t *testing.T) {
	file, err := protoregistry.GlobalFiles.FindFileByPath("desc_test1.proto")
	testutil.Ok(t, err)
	testutil.Eq(t, 0, file.SourceLocations().Len())
	file = sourceinfo.WrapFile(file)
	testutil.Neq(t, 0, file.SourceLocations().Len()) // now there are source locations
}

func TestWrapMessage(t *testing.T) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName("testprotos.TestMessage.NestedMessage")
	testutil.Ok(t, err)
	md, ok := d.(protoreflect.MessageDescriptor)
	testutil.Require(t, ok)
	testutil.Require(t, isZeroSourceLocation(md.ParentFile().SourceLocations().ByDescriptor(md))) // no source location
	md = sourceinfo.WrapMessage(md)
	testutil.Require(t, !isZeroSourceLocation(md.ParentFile().SourceLocations().ByDescriptor(md))) // now there is a source location
}

func TestWrapEnum(t *testing.T) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName("testprotos.TestMessage.NestedEnum")
	testutil.Ok(t, err)
	ed, ok := d.(protoreflect.EnumDescriptor)
	testutil.Require(t, ok)
	testutil.Require(t, isZeroSourceLocation(ed.ParentFile().SourceLocations().ByDescriptor(ed))) // no source location
	ed = sourceinfo.WrapEnum(ed)
	testutil.Require(t, !isZeroSourceLocation(ed.ParentFile().SourceLocations().ByDescriptor(ed))) // now there is a source location
}

func TestWrapService(t *testing.T) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName("testprotos.SomeService")
	testutil.Ok(t, err)
	sd, ok := d.(protoreflect.ServiceDescriptor)
	testutil.Require(t, ok)
	testutil.Require(t, isZeroSourceLocation(sd.ParentFile().SourceLocations().ByDescriptor(sd))) // no source location
	sd = sourceinfo.WrapService(sd)
	testutil.Require(t, !isZeroSourceLocation(sd.ParentFile().SourceLocations().ByDescriptor(sd))) // now there is a source location
}

func TestWrapMessageType(t *testing.T) {
	mt, err := protoregistry.GlobalTypes.FindMessageByName("testprotos.TestMessage.NestedMessage")
	testutil.Ok(t, err)
	md := mt.Descriptor()
	testutil.Require(t, isZeroSourceLocation(md.ParentFile().SourceLocations().ByDescriptor(md))) // no source location
	mt = sourceinfo.WrapMessageType(mt)
	md = mt.Descriptor()
	testutil.Require(t, !isZeroSourceLocation(md.ParentFile().SourceLocations().ByDescriptor(md))) // now there is a source location
}

func TestWrapExtensionType(t *testing.T) {
	xt, err := protoregistry.GlobalTypes.FindExtensionByName("testprotos.xtm")
	testutil.Ok(t, err)
	xd := xt.TypeDescriptor()
	testutil.Require(t, isZeroSourceLocation(xd.ParentFile().SourceLocations().ByDescriptor(xd))) // no source location
	xt = sourceinfo.WrapExtensionType(xt)
	xd = xt.TypeDescriptor()
	testutil.Require(t, !isZeroSourceLocation(xd.ParentFile().SourceLocations().ByDescriptor(xd))) // now there is a source location
}

func isZeroSourceLocation(loc protoreflect.SourceLocation) bool {
	return loc.Path == nil &&
		loc.StartLine == 0 && loc.StartColumn == 0 &&
		loc.EndLine == 0 && loc.EndColumn == 0 &&
		loc.LeadingDetachedComments == nil &&
		loc.LeadingComments == "" && loc.TrailingComments == "" &&
		loc.Next == 0
}
