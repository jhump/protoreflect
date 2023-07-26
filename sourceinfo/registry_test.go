package sourceinfo_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	_ "github.com/jhump/protoreflect/v2/internal/testdata"
	"github.com/jhump/protoreflect/v2/sourceinfo"
)

func TestRegistry(t *testing.T) {
	fdWithout, err := protoregistry.GlobalFiles.FindFileByPath("desc_test1.proto")
	require.NoError(t, err)
	fd, err := sourceinfo.Files.FindFileByPath("desc_test1.proto")
	require.NoError(t, err)
	assert.Equal(t, fd, sourceinfo.WrapFile(fdWithout))
	checkFile(t, fdWithout, fd)
}

func checkFile(t *testing.T, fdWithout, fd protoreflect.FileDescriptor) {
	srcLocs := fd.SourceLocations()
	checkTypes(t, srcLocs, fdWithout, fd, fd)
	for i := 0; i < fd.Services().Len(); i++ {
		sd := fd.Services().Get(i)
		sdWithout := fdWithout.Services().Get(i)
		assert.Equal(t, sd, sourceinfo.WrapService(sdWithout))
		sdByName := fd.Services().ByName(sd.Name())
		assert.Equal(t, sd, sdByName)
		sdByNameWithout := fd.Services().ByName(sd.Name())
		assert.Equal(t, sdByName, sourceinfo.WrapService(sdByNameWithout))
		checkDescriptor(t, srcLocs, sd, fd)
		for j := 0; j < sd.Methods().Len(); j++ {
			mtd := sd.Methods().Get(j)
			mtdByName := sd.Methods().ByName(mtd.Name())
			assert.Equal(t, mtd, mtdByName)
			mtdWithout := sdWithout.Methods().Get(j)
			assert.Equal(t, mtd.Input(), sourceinfo.WrapMessage(mtdWithout.Input()))
			assert.Equal(t, mtd.Output(), sourceinfo.WrapMessage(mtdWithout.Output()))
			checkDescriptor(t, srcLocs, mtd, fd, sd)
		}
	}
}

type typeContainer interface {
	Messages() protoreflect.MessageDescriptors
	Enums() protoreflect.EnumDescriptors
	Extensions() protoreflect.ExtensionDescriptors
}

func checkTypes(t *testing.T, srcLocs protoreflect.SourceLocations, descWithout, desc typeContainer, ancestors ...protoreflect.Descriptor) {
	for i := 0; i < desc.Messages().Len(); i++ {
		md := desc.Messages().Get(i)
		mdWithout := descWithout.Messages().Get(i)
		assert.Equal(t, md, sourceinfo.WrapMessage(mdWithout))
		mdByName := desc.Messages().ByName(md.Name())
		assert.Equal(t, md, mdByName)
		mdByNameWithout := descWithout.Messages().ByName(md.Name())
		assert.Equal(t, mdByName, sourceinfo.WrapMessage(mdByNameWithout))
		msgType := sourceinfo.WrapMessageType(dynamicpb.NewMessageType(mdWithout))
		assert.Equal(t, md, msgType.Descriptor())
		if md.IsMapEntry() {
			// map entry messages do not have generated types or comments
			continue
		}
		registryMsgType, err := sourceinfo.Types.FindMessageByName(md.FullName())
		require.NoError(t, err)
		assert.Equal(t, md, registryMsgType.Descriptor())
		checkMessage(t, srcLocs, mdWithout, md, ancestors...)
	}
	for i := 0; i < desc.Enums().Len(); i++ {
		ed := desc.Enums().Get(i)
		edWithout := descWithout.Enums().Get(i)
		assert.Equal(t, ed, sourceinfo.WrapEnum(edWithout))
		edByName := desc.Enums().ByName(ed.Name())
		assert.Equal(t, ed, edByName)
		edByNameWithout := descWithout.Enums().ByName(ed.Name())
		assert.Equal(t, edByName, sourceinfo.WrapEnum(edByNameWithout))
		enumType := sourceinfo.WrapEnumType(dynamicpb.NewEnumType(edWithout))
		assert.Equal(t, ed, enumType.Descriptor())
		registryEnumType, err := sourceinfo.Types.FindEnumByName(ed.FullName())
		require.NoError(t, err)
		assert.Equal(t, ed, registryEnumType.Descriptor())
		checkEnum(t, srcLocs, edWithout, ed, ancestors...)
	}
	for i := 0; i < desc.Extensions().Len(); i++ {
		extd := desc.Extensions().Get(i)
		assert.True(t, extd.IsExtension())
		extdWithout := descWithout.Extensions().Get(i)
		assert.Equal(t, extd, sourceinfo.WrapExtension(extdWithout))
		extdByName := desc.Extensions().ByName(extd.Name())
		assert.Equal(t, extd, extdByName)
		extdByNameWithout := descWithout.Extensions().ByName(extd.Name())
		assert.Equal(t, extdByName, sourceinfo.WrapExtension(extdByNameWithout))
		extType := sourceinfo.WrapExtensionType(dynamicpb.NewExtensionType(extdWithout))
		assert.Equal(t, extd, extType.TypeDescriptor().Descriptor())
		registryExtType, err := sourceinfo.Types.FindExtensionByName(extd.FullName())
		require.NoError(t, err)
		assert.Equal(t, extd, registryExtType.TypeDescriptor().Descriptor())
		registryExtType, err = sourceinfo.Types.FindExtensionByNumber(extd.ContainingMessage().FullName(), extd.Number())
		require.NoError(t, err)
		assert.Equal(t, extd, registryExtType.TypeDescriptor().Descriptor())
		checkField(t, srcLocs, extdWithout, extd, ancestors...)
	}
}

func checkMessage(t *testing.T, srcLocs protoreflect.SourceLocations, mdWithout, md protoreflect.MessageDescriptor, ancestors ...protoreflect.Descriptor) {
	checkDescriptor(t, srcLocs, md, ancestors...)
	ancestors = append(ancestors, md)
	checkFields(t, srcLocs, mdWithout.Fields(), md.Fields(), ancestors...)
	for i := 0; i < md.Oneofs().Len(); i++ {
		ood := md.Oneofs().Get(i)
		oodByName := md.Oneofs().ByName(ood.Name())
		assert.Equal(t, ood, oodByName)
		oodWithout := mdWithout.Oneofs().Get(i)
		checkFields(t, srcLocs, oodWithout.Fields(), ood.Fields(), ancestors...)
		checkDescriptor(t, srcLocs, ood, ancestors...)
	}
	checkTypes(t, srcLocs, mdWithout, md, ancestors...)
}

func checkFields(t *testing.T, srcLocs protoreflect.SourceLocations, fldsWithout, flds protoreflect.FieldDescriptors, ancestors ...protoreflect.Descriptor) {
	for i := 0; i < flds.Len(); i++ {
		fld := flds.Get(i)
		assert.False(t, fld.IsExtension())
		fldByName := flds.ByName(fld.Name())
		assert.Equal(t, fld, fldByName)
		fldByJSONName := flds.ByJSONName(fld.JSONName())
		assert.Equal(t, fld, fldByJSONName)
		fldByTextName := flds.ByTextName(fld.TextName())
		assert.Equal(t, fld, fldByTextName)
		fldByNumber := flds.ByNumber(fld.Number())
		assert.Equal(t, fld, fldByNumber)
		fldWithout := fldsWithout.Get(i)
		checkField(t, srcLocs, fldWithout, fld, ancestors...)
	}
}

func checkField(t *testing.T, srcLocs protoreflect.SourceLocations, fldWithout, fld protoreflect.FieldDescriptor, ancestors ...protoreflect.Descriptor) {
	if md := fld.Message(); md != nil {
		mdWithout := fldWithout.Message()
		assert.Equal(t, md, sourceinfo.WrapMessage(mdWithout))
	}
	if mapFld := fld.MapKey(); mapFld != nil {
		mapFldWithout := fldWithout.MapKey()
		checkField(t, srcLocs, mapFldWithout, mapFld, append(ancestors, mapFld.ContainingMessage())...)
	}
	if mapFld := fld.MapValue(); mapFld != nil {
		mapFldWithout := fldWithout.MapValue()
		checkField(t, srcLocs, mapFldWithout, mapFld, append(ancestors, mapFld.ContainingMessage())...)
	}
	if ed := fld.Enum(); ed != nil {
		edWithout := fldWithout.Enum()
		assert.Equal(t, ed, sourceinfo.WrapEnum(edWithout))
	}
	if ood := fld.ContainingOneof(); ood != nil {
		assert.Equal(t, ood, sourceinfo.WrapMessage(fldWithout.ContainingMessage()).Oneofs().Get(ood.Index()))
	}
	md := fld.ContainingMessage()
	mdWithout := fldWithout.ContainingMessage()
	assert.Equal(t, md, sourceinfo.WrapMessage(mdWithout))
	if fld.Kind() == protoreflect.GroupKind {
		return // comment is attributed to group message, not field
	}
	if md.IsMapEntry() {
		return // map entry messages have no comments
	}
	checkDescriptor(t, srcLocs, fld, ancestors...)
}

func checkEnum(t *testing.T, srcLocs protoreflect.SourceLocations, edWithout, ed protoreflect.EnumDescriptor, ancestors ...protoreflect.Descriptor) {
	checkDescriptor(t, srcLocs, ed, ancestors...)
	ancestors = append(ancestors, ed)
	for i := 0; i < ed.Values().Len(); i++ {
		evd := ed.Values().Get(i)
		checkDescriptor(t, srcLocs, evd, ancestors...)
	}
}

func checkDescriptor(t *testing.T, srcLocs protoreflect.SourceLocations, d protoreflect.Descriptor, ancestors ...protoreflect.Descriptor) {
	cmt := fmt.Sprintf(" Comment for %s\n", d.Name())
	require.Equal(t, cmt, srcLocs.ByDescriptor(d).LeadingComments)

	registryDesc, err := sourceinfo.Files.FindDescriptorByName(d.FullName())
	require.NoError(t, err)
	require.Equal(t, d, registryDesc)

	require.Equal(t, ancestors[0], d.ParentFile())
	for i := len(ancestors) - 1; i >= 0; i-- {
		d = d.Parent()
		require.Equal(t, ancestors[i], d)
	}
}
