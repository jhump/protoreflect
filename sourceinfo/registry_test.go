package sourceinfo_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
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
	fdWith, err := sourceinfo.AddSourceInfoToFile(fdWithout)
	require.NoError(t, err)
	assert.Same(t, fd, fdWith)
	checkFile(t, fdWithout, fd)
}

func TestCanUpgrade(t *testing.T) {
	fd, err := protoregistry.GlobalFiles.FindFileByPath("desc_test1.proto")
	require.NoError(t, err)
	require.True(t, sourceinfo.CanUpgrade(fd))

	fd, err = sourceinfo.Files.FindFileByPath("desc_test1.proto")
	require.NoError(t, err)
	require.False(t, sourceinfo.CanUpgrade(fd)) // already has source info

	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				"test.proto": `
				syntax = "proto3";
				package test;
				message Foo {
					string name = 1;
				}
				`,
			}),
		},
	}
	files, err := compiler.Compile(context.Background(), "test.proto")
	require.NoError(t, err)
	require.False(t, sourceinfo.CanUpgrade(files[0])) // already has source info (also not standard impl)
	fdProto := protoutil.ProtoFromFileDescriptor(files[0])
	file, err := protodesc.NewFile(fdProto, nil)
	require.NoError(t, err)
	require.False(t, sourceinfo.CanUpgrade(file)) // already has source info

	fdProto.SourceCodeInfo = nil // strip source info and try again
	file, err = protodesc.NewFile(fdProto, nil)
	require.NoError(t, err)
	require.False(t, sourceinfo.CanUpgrade(file)) // still false; not from gen code
}

func checkFile(t *testing.T, fdWithout, fd protoreflect.FileDescriptor) {
	srcLocs := fd.SourceLocations()
	checkTypes(t, srcLocs, fdWithout, fd, fd)
	for i := 0; i < fd.Services().Len(); i++ {
		sd := fd.Services().Get(i)
		sdWithout := fdWithout.Services().Get(i)
		sdWith, err := sourceinfo.AddSourceInfoToService(sdWithout)
		require.NoError(t, err)
		assert.Same(t, sd, sdWith)
		sdByName := fd.Services().ByName(sd.Name())
		assert.Same(t, sd, sdByName)
		sdByNameWithout := fd.Services().ByName(sd.Name())
		sdByNameWith, err := sourceinfo.AddSourceInfoToService(sdByNameWithout)
		require.NoError(t, err)
		assert.Same(t, sdByName, sdByNameWith)
		checkDescriptor(t, srcLocs, sd, fd)
		for j := 0; j < sd.Methods().Len(); j++ {
			mtd := sd.Methods().Get(j)
			mtdByName := sd.Methods().ByName(mtd.Name())
			assert.Same(t, mtd, mtdByName)
			mtdWithout := sdWithout.Methods().Get(j)
			inputWith, err := sourceinfo.AddSourceInfoToMessage(mtdWithout.Input())
			require.NoError(t, err)
			assert.Same(t, mtd.Input(), inputWith)
			outputWith, err := sourceinfo.AddSourceInfoToMessage(mtdWithout.Output())
			require.NoError(t, err)
			assert.Same(t, mtd.Output(), outputWith)
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
		mdWith, err := sourceinfo.AddSourceInfoToMessage(md)
		require.NoError(t, err)
		assert.Same(t, md, mdWith)
		mdByName := desc.Messages().ByName(md.Name())
		assert.Same(t, md, mdByName)
		mdByNameWithout := descWithout.Messages().ByName(md.Name())
		mdByNameWith, err := sourceinfo.AddSourceInfoToMessage(mdByNameWithout)
		require.NoError(t, err)
		assert.Same(t, mdByName, mdByNameWith)
		msgType, err := sourceinfo.AddSourceInfoToMessageType(dynamicpb.NewMessageType(mdWithout))
		require.NoError(t, err)
		assert.Same(t, md, msgType.Descriptor())
		if md.IsMapEntry() {
			// map entry messages do not have generated types or comments
			continue
		}
		registryMsgType, err := sourceinfo.Types.FindMessageByName(md.FullName())
		require.NoError(t, err)
		assert.Same(t, md, registryMsgType.Descriptor())
		checkMessage(t, srcLocs, mdWithout, md, ancestors...)
	}
	for i := 0; i < desc.Enums().Len(); i++ {
		ed := desc.Enums().Get(i)
		edWithout := descWithout.Enums().Get(i)
		edWith, err := sourceinfo.AddSourceInfoToEnum(ed)
		require.NoError(t, err)
		assert.Same(t, ed, edWith)
		edByName := desc.Enums().ByName(ed.Name())
		assert.Same(t, ed, edByName)
		edByNameWithout := descWithout.Enums().ByName(ed.Name())
		edByNameWith, err := sourceinfo.AddSourceInfoToEnum(edByNameWithout)
		require.NoError(t, err)
		assert.Same(t, edByName, edByNameWith)
		enumType, err := sourceinfo.AddSourceInfoToEnumType(dynamicpb.NewEnumType(edWithout))
		require.NoError(t, err)
		assert.Same(t, ed, enumType.Descriptor())
		registryEnumType, err := sourceinfo.Types.FindEnumByName(ed.FullName())
		require.NoError(t, err)
		assert.Same(t, ed, registryEnumType.Descriptor())
		checkEnum(t, srcLocs, edWithout, ed, ancestors...)
	}
	for i := 0; i < desc.Extensions().Len(); i++ {
		extd := desc.Extensions().Get(i)
		assert.True(t, extd.IsExtension())
		extdWithout := descWithout.Extensions().Get(i)
		extdWith, err := sourceinfo.AddSourceInfoToField(extdWithout)
		require.NoError(t, err)
		assert.Same(t, extd, extdWith)
		extdByName := desc.Extensions().ByName(extd.Name())
		assert.Same(t, extd, extdByName)
		extdByNameWithout := descWithout.Extensions().ByName(extd.Name())
		extdByNameWith, err := sourceinfo.AddSourceInfoToField(extdByNameWithout)
		require.NoError(t, err)
		assert.Same(t, extdByName, extdByNameWith)
		extType, err := sourceinfo.AddSourceInfoToExtensionType(dynamicpb.NewExtensionType(extdWithout))
		require.NoError(t, err)
		assert.Same(t, extd, extType.TypeDescriptor().Descriptor())
		registryExtType, err := sourceinfo.Types.FindExtensionByName(extd.FullName())
		require.NoError(t, err)
		assert.Same(t, extd, registryExtType.TypeDescriptor().Descriptor())
		registryExtType, err = sourceinfo.Types.FindExtensionByNumber(extd.ContainingMessage().FullName(), extd.Number())
		require.NoError(t, err)
		assert.Same(t, extd, registryExtType.TypeDescriptor().Descriptor())
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
		assert.Same(t, ood, oodByName)
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
		assert.Same(t, fld, fldByName)
		fldByJSONName := flds.ByJSONName(fld.JSONName())
		assert.Same(t, fld, fldByJSONName)
		fldByTextName := flds.ByTextName(fld.TextName())
		assert.Same(t, fld, fldByTextName)
		fldByNumber := flds.ByNumber(fld.Number())
		assert.Same(t, fld, fldByNumber)
		fldWithout := fldsWithout.Get(i)
		checkField(t, srcLocs, fldWithout, fld, ancestors...)
	}
}

func checkField(t *testing.T, srcLocs protoreflect.SourceLocations, fldWithout, fld protoreflect.FieldDescriptor, ancestors ...protoreflect.Descriptor) {
	if md := fld.Message(); md != nil {
		mdWithout := fldWithout.Message()
		mdWith, err := sourceinfo.AddSourceInfoToMessage(mdWithout)
		require.NoError(t, err)
		assert.Same(t, md, mdWith)
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
		edWith, err := sourceinfo.AddSourceInfoToEnum(edWithout)
		require.NoError(t, err)
		assert.Same(t, ed, edWith)
	}
	md := fld.ContainingMessage()
	mdWithout := fldWithout.ContainingMessage()
	mdWith, err := sourceinfo.AddSourceInfoToMessage(mdWithout)
	require.NoError(t, err)
	assert.Same(t, md, mdWith)
	if ood := fld.ContainingOneof(); ood != nil {
		assert.Same(t, ood, mdWith.Oneofs().Get(ood.Index()))
	}
	if fld.Kind() == protoreflect.GroupKind && fld.ParentFile().Syntax() == protoreflect.Proto2 {
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
	require.Same(t, d, registryDesc)

	require.Same(t, ancestors[0], d.ParentFile())
	for i := len(ancestors) - 1; i >= 0; i-- {
		d = d.Parent()
		require.Same(t, ancestors[i], d)
	}
}
