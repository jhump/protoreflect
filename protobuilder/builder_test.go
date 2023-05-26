package protobuilder

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	_ "github.com/jhump/protoreflect/v2/internal/testdata"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"github.com/jhump/protoreflect/v2/protowrap"
)

func TestSimpleDescriptorsFromScratch(t *testing.T) {
	md := (*emptypb.Empty)(nil).ProtoReflect().Descriptor()

	file := NewFile("foo/bar.proto").SetPackageName("foo.bar")
	en := NewEnum("Options").
		AddValue(NewEnumValue("OPTION_1")).
		AddValue(NewEnumValue("OPTION_2")).
		AddValue(NewEnumValue("OPTION_3"))
	file.AddEnum(en)

	msg := NewMessage("FooRequest").
		AddField(NewField("id", FieldTypeInt64())).
		AddField(NewField("path", FieldTypeString())).
		AddField(NewField("options", FieldTypeEnum(en)).
			SetRepeated())
	file.AddMessage(msg)

	sb := NewService("FooService").
		AddMethod(NewMethod("DoSomething", RpcTypeMessage(msg, false), RpcTypeMessage(msg, false))).
		AddMethod(NewMethod("ReturnThings", RpcTypeImportedMessage(md, false), RpcTypeMessage(msg, true)))
	file.AddService(sb)

	fd, err := file.Build()
	require.NoError(t, err)

	require.Equal(t, 1, fd.Imports().Len())
	require.Equal(t, md.ParentFile(), fd.Imports().Get(0).FileDescriptor)
	require.NotNil(t, fd.Enums().ByName("Options"))
	require.Equal(t, 3, fd.Enums().ByName("Options").Values().Len())
	require.NotNil(t, fd.Messages().ByName("FooRequest"))
	require.Equal(t, 3, fd.Messages().ByName("FooRequest").Fields().Len())
	require.NotNil(t, fd.Services().ByName("FooService"))
	require.Equal(t, 2, fd.Services().ByName("FooService").Methods().Len())

	// building the others produces same results
	ed, err := en.Build()
	require.NoError(t, err)
	diff := cmp.Diff(protowrap.ProtoFromDescriptor(ed), protowrap.ProtoFromDescriptor(fd.Enums().ByName("Options")), protocmp.Transform())
	require.Empty(t, diff)

	md, err = msg.Build()
	require.NoError(t, err)
	diff = cmp.Diff(protowrap.ProtoFromDescriptor(md), protowrap.ProtoFromDescriptor(fd.Messages().ByName("FooRequest")), protocmp.Transform())
	require.Empty(t, diff)

	sd, err := sb.Build()
	require.NoError(t, err)
	diff = cmp.Diff(protowrap.ProtoFromDescriptor(sd), protowrap.ProtoFromDescriptor(fd.Services().ByName("FooService")), protocmp.Transform())
	require.Empty(t, diff)
}

func TestSimpleDescriptorsFromScratch_SyntheticFiles(t *testing.T) {
	md := (*emptypb.Empty)(nil).ProtoReflect().Descriptor()

	en := NewEnum("Options")
	en.AddValue(NewEnumValue("OPTION_1"))
	en.AddValue(NewEnumValue("OPTION_2"))
	en.AddValue(NewEnumValue("OPTION_3"))

	msg := NewMessage("FooRequest")
	msg.AddField(NewField("id", FieldTypeInt64()))
	msg.AddField(NewField("path", FieldTypeString()))
	msg.AddField(NewField("options", FieldTypeEnum(en)).
		SetRepeated())

	sb := NewService("FooService")
	sb.AddMethod(NewMethod("DoSomething", RpcTypeMessage(msg, false), RpcTypeMessage(msg, false)))
	sb.AddMethod(NewMethod("ReturnThings", RpcTypeImportedMessage(md, false), RpcTypeMessage(msg, true)))

	sd, err := sb.Build()
	require.NoError(t, err)
	require.Equal(t, protoreflect.FullName("FooService"), sd.FullName())
	require.Equal(t, 2, sd.Methods().Len())

	// it imports google/protobuf/empty.proto and a synthetic file that has message
	require.Equal(t, 2, sd.ParentFile().Imports().Len())
	fd := sd.ParentFile().Imports().Get(0).FileDescriptor
	require.Equal(t, "google/protobuf/empty.proto", fd.Path())
	require.Equal(t, md.ParentFile(), fd)
	fd = sd.ParentFile().Imports().Get(1).FileDescriptor
	require.True(t, strings.Contains(fd.Path(), "generated"))
	require.NotNil(t, fd.Messages().ByName("FooRequest"))
	require.Equal(t, 3, fd.Messages().ByName("FooRequest").Fields().Len())

	// this one imports only a synthetic file that has enum
	require.Equal(t, 1, fd.Imports().Len())
	fd2 := fd.Imports().Get(0).FileDescriptor
	require.NotNil(t, fd2.Enums().ByName("Options"))
	require.Equal(t, 3, fd2.Enums().ByName("Options").Values().Len())

	// building the others produces same results
	ed, err := en.Build()
	require.NoError(t, err)
	diff := cmp.Diff(protowrap.ProtoFromDescriptor(ed), protowrap.ProtoFromDescriptor(fd2.Enums().ByName("Options")), protocmp.Transform())
	require.Empty(t, diff)

	md, err = msg.Build()
	require.NoError(t, err)
	diff = cmp.Diff(protowrap.ProtoFromDescriptor(md), protowrap.ProtoFromDescriptor(fd.Messages().ByName("FooRequest")), protocmp.Transform())
	require.Empty(t, diff)
}

func TestComplexDescriptorsFromScratch(t *testing.T) {
	mdEmpty := (*emptypb.Empty)(nil).ProtoReflect().Descriptor()
	mdAny := (*anypb.Any)(nil).ProtoReflect().Descriptor()
	mdTimestamp := (*timestamppb.Timestamp)(nil).ProtoReflect().Descriptor()

	mbAny, err := FromMessage(mdAny)
	require.NoError(t, err)

	msgA := NewMessage("FooA").
		AddField(NewField("id", FieldTypeUint64())).
		AddField(NewField("when", FieldTypeImportedMessage(mdTimestamp))).
		AddField(NewField("extras", FieldTypeImportedMessage(mdAny)).
			SetRepeated()).
		AddField(NewField("builder", FieldTypeMessage(mbAny))).
		SetExtensionRanges([]ExtensionRange{{FieldRange: FieldRange{100, 201}}})
	msgA2 := NewMessage("Nnn").
		AddField(NewField("uid1", FieldTypeFixed64())).
		AddField(NewField("uid2", FieldTypeFixed64()))
	NewFile("").
		SetPackageName("foo.bar").
		AddMessage(msgA).
		AddMessage(msgA2)

	msgB := NewMessage("FooB").
		AddField(NewField("foo_a", FieldTypeMessage(msgA)).
			SetRepeated()).
		AddField(NewField("path", FieldTypeString()))
	NewFile("").
		SetPackageName("foo.bar").
		AddMessage(msgB)

	enC := NewEnum("Vals").
		AddValue(NewEnumValue("DEFAULT")).
		AddValue(NewEnumValue("VALUE_A")).
		AddValue(NewEnumValue("VALUE_B")).
		AddValue(NewEnumValue("VALUE_C"))
	msgC := NewMessage("BarBaz").
		AddOneOf(NewOneof("bbb").
			AddChoice(NewField("b1", FieldTypeMessage(msgA))).
			AddChoice(NewField("b2", FieldTypeMessage(msgB)))).
		AddField(NewField("v", FieldTypeEnum(enC)))
	NewFile("some/path/file.proto").
		SetPackageName("foo.baz").
		AddEnum(enC).
		AddMessage(msgC)

	enD := NewEnum("Ppp").
		AddValue(NewEnumValue("P0")).
		AddValue(NewEnumValue("P1")).
		AddValue(NewEnumValue("P2")).
		AddValue(NewEnumValue("P3"))
	exD := NewExtension("ppp", 123, FieldTypeEnum(enD), msgA)
	NewFile("some/other/path/file.proto").
		SetPackageName("foo.biz").
		AddEnum(enD).
		AddExtension(exD)

	msgE := NewMessage("Ppp").
		AddField(NewField("p", FieldTypeEnum(enD))).
		AddField(NewField("n", FieldTypeMessage(msgA2)))
	fd, err := NewFile("").
		SetPackageName("foo.bar").
		AddMessage(msgE).
		AddService(NewService("PppSvc").
			AddMethod(NewMethod("Method1", RpcTypeMessage(msgE, false), RpcTypeImportedMessage(mdEmpty, false))).
			AddMethod(NewMethod("Method2", RpcTypeMessage(msgB, false), RpcTypeMessage(msgC, true)))).
		Build()

	require.NoError(t, err)

	require.Equal(t, 5, fd.Imports().Len())
	// dependencies sorted; those with generated names come last
	depEmpty := fd.Imports().Get(0).FileDescriptor
	require.Equal(t, "google/protobuf/empty.proto", depEmpty.Path())
	require.Equal(t, mdEmpty.ParentFile(), depEmpty)
	depD := fd.Imports().Get(1).FileDescriptor
	require.Equal(t, "some/other/path/file.proto", depD.Path())
	depC := fd.Imports().Get(2).FileDescriptor
	require.Equal(t, "some/path/file.proto", depC.Path())
	depA := fd.Imports().Get(3).FileDescriptor
	require.True(t, strings.Contains(depA.Path(), "generated"))
	depB := fd.Imports().Get(4).FileDescriptor
	require.True(t, strings.Contains(depB.Path(), "generated"))

	// check contents of files
	require.NotNil(t, depA.Messages().ByName("FooA"))
	require.Equal(t, 4, depA.Messages().ByName("FooA").Fields().Len())
	require.NotNil(t, depA.Messages().ByName("Nnn"))
	require.Equal(t, 2, depA.Messages().ByName("Nnn").Fields().Len())
	require.Equal(t, 2, depA.Imports().Len())

	require.NotNil(t, depB.Messages().ByName("FooB"))
	require.Equal(t, 2, depB.Messages().ByName("FooB").Fields().Len())
	require.Equal(t, 1, depB.Imports().Len())

	require.NotNil(t, depC.Messages().ByName("BarBaz"))
	require.Equal(t, 3, depC.Messages().ByName("BarBaz").Fields().Len())
	require.NotNil(t, depC.Enums().ByName("Vals"))
	require.Equal(t, 4, depC.Enums().ByName("Vals").Values().Len())
	require.Equal(t, 2, depC.Imports().Len())

	require.NotNil(t, depD.Enums().ByName("Ppp"))
	require.Equal(t, 4, depD.Enums().ByName("Ppp").Values().Len())
	require.NotNil(t, depD.Extensions().ByName("ppp"))
	require.Equal(t, 1, depD.Imports().Len())

	require.NotNil(t, fd.Messages().ByName("Ppp"))
	require.Equal(t, 2, fd.Messages().ByName("Ppp").Fields().Len())
	require.NotNil(t, fd.Services().ByName("PppSvc"))
	require.Equal(t, 2, fd.Services().ByName("PppSvc").Methods().Len())
}

func TestCreatingGroupField(t *testing.T) {
	grpMb := NewMessage("GroupA").
		AddField(NewField("path", FieldTypeString())).
		AddField(NewField("id", FieldTypeInt64()))
	grpFlb := NewGroupField(grpMb)

	mb := NewMessage("TestMessage").
		AddField(NewField("foo", FieldTypeBool())).
		AddField(grpFlb)
	md, err := mb.Build()
	require.NoError(t, err)

	require.NotNil(t, md.Fields().ByName("groupa"))
	require.Equal(t, protoreflect.GroupKind, md.Fields().ByName("groupa").Kind())
	nmd := md.Messages().Get(0)
	require.Equal(t, protoreflect.Name("GroupA"), nmd.Name())
	require.Equal(t, nmd, md.Fields().ByName("groupa").Message())

	// try a rename that will fail
	err = grpMb.TrySetName("fooBarBaz")
	require.ErrorContains(t, err, "group path fooBarBaz must start with capital letter")
	// failed rename should not have modified any state
	md2, err := mb.Build()
	require.NoError(t, err)
	diff := cmp.Diff(protowrap.ProtoFromDescriptor(md), protowrap.ProtoFromDescriptor(md2), protocmp.Transform())
	require.Empty(t, diff)
	// another attempt that will fail
	err = grpFlb.TrySetName("foobarbaz")
	require.ErrorContains(t, err, "cannot change path of group field TestMessage.groupa; change path of group instead")
	// again, no state should have been modified
	md2, err = mb.Build()
	require.NoError(t, err)
	diff = cmp.Diff(protowrap.ProtoFromDescriptor(md), protowrap.ProtoFromDescriptor(md2), protocmp.Transform())
	require.Empty(t, diff)

	// and a rename that succeeds
	err = grpMb.TrySetName("FooBarBaz")
	require.NoError(t, err)
	md, err = mb.Build()
	require.NoError(t, err)

	// field also renamed
	require.NotNil(t, md.Fields().ByName("foobarbaz"))
	require.Equal(t, protoreflect.GroupKind, md.Fields().ByName("foobarbaz").Kind())
	nmd = md.Messages().Get(0)
	require.Equal(t, protoreflect.Name("FooBarBaz"), nmd.Name())
	require.Equal(t, nmd, md.Fields().ByName("foobarbaz").Message())
}

func TestCreatingMapField(t *testing.T) {
	mapFlb := NewMapField("countsByName", FieldTypeString(), FieldTypeUint64())
	require.True(t, mapFlb.IsMap())

	mb := NewMessage("TestMessage").
		AddField(NewField("foo", FieldTypeBool())).
		AddField(mapFlb)
	md, err := mb.Build()
	require.NoError(t, err)

	require.NotNil(t, md.Fields().ByName("countsByName"))
	require.True(t, md.Fields().ByName("countsByName").IsMap())
	nmd := md.Messages().Get(0)
	require.Equal(t, protoreflect.Name("CountsByNameEntry"), nmd.Name())
	require.Equal(t, nmd, md.Fields().ByName("countsByName").Message())

	// try a rename that will fail
	err = mapFlb.Type().localMsgType.TrySetName("fooBarBaz")
	require.ErrorContains(t, err, "cannot change path of map entry TestMessage.CountsByNameEntry; change path of field instead")
	// failed rename should not have modified any state
	md2, err := mb.Build()
	require.NoError(t, err)
	diff := cmp.Diff(protowrap.ProtoFromDescriptor(md), protowrap.ProtoFromDescriptor(md2), protocmp.Transform())
	require.Empty(t, diff)

	// and a rename that succeeds
	err = mapFlb.TrySetName("fooBarBaz")
	require.NoError(t, err)
	md, err = mb.Build()
	require.NoError(t, err)

	// map entry also renamed
	require.NotNil(t, md.Fields().ByName("fooBarBaz"))
	require.True(t, md.Fields().ByName("fooBarBaz").IsMap())
	nmd = md.Messages().Get(0)
	require.Equal(t, protoreflect.Name("FooBarBazEntry"), nmd.Name())
	require.Equal(t, nmd, md.Fields().ByName("fooBarBaz").Message())
}

func TestProto3Optional(t *testing.T) {
	mb := NewMessage("Foo")
	flb := NewField("bar", FieldTypeBool()).SetProto3Optional(true)
	mb.AddField(flb)

	_, err := flb.Build()
	require.NotNil(t, err) // file does not have proto3 syntax

	fb := NewFile("foo.proto").SetSyntax(protoreflect.Proto3)
	fb.AddMessage(mb)

	fld, err := flb.Build()
	require.NoError(t, err)

	require.True(t, fld.HasPresence())
	require.NotNil(t, fld.ContainingOneof())
	require.True(t, fld.ContainingOneof().IsSynthetic())
	require.Equal(t, protoreflect.Name("_bar"), fld.ContainingOneof().Name())
}

func TestBuildersFromDescriptors(t *testing.T) {
	for _, s := range []string{"desc_test1.proto", "desc_test2.proto", "desc_test_defaults.proto", "desc_test_options.proto", "desc_test_proto3.proto", "desc_test_wellknowntypes.proto", "nopkg/desc_test_nopkg.proto", "nopkg/desc_test_nopkg_new.proto", "pkg/desc_test_pkg.proto"} {
		fd, err := protoregistry.GlobalFiles.FindFileByPath(s)
		require.NoError(t, err)
		roundTripFile(t, fd)
	}
}

func TestBuildersFromDescriptors_PreserveComments(t *testing.T) {
	files, err := loadProtoset("../internal/testdata/desc_test1.protoset")
	require.NoError(t, err)
	fd, err := files.FindFileByPath("desc_test1.proto")
	require.NoError(t, err)

	fb, err := FromFile(fd)
	require.NoError(t, err)

	count := 0
	var checkBuilderComments func(b Builder)
	checkBuilderComments = func(b Builder) {
		hasComment := true
		switch b := b.(type) {
		case *FileBuilder:
			hasComment = false
		case *FieldBuilder:
			// comments for groups are on the message, not the field
			hasComment = b.Type().Kind() != protoreflect.GroupKind
		case *MessageBuilder:
			// comments for maps are on the field, not the entry message
			if b.Options.GetMapEntry() {
				// we just return to also skip checking child elements
				// (map entry child elements are synthetic and have no comments)
				return
			}
		}

		if hasComment {
			count++
			require.Equal(t, fmt.Sprintf(" Comment for %s\n", b.Name()), b.Comments().LeadingComment,
				"wrong comment for builder %s", FullName(b))
		}
		for _, ch := range b.Children() {
			checkBuilderComments(ch)
		}
	}

	checkBuilderComments(fb)
	// sanity check that we didn't accidentally short-circuit above and fail to check comments
	require.True(t, count > 30, "too few elements checked")

	// now check that they also come out in the resulting descriptor
	fd, err = fb.Build()
	require.NoError(t, err)

	descCount := 0
	var checkDescriptorComments func(d protoreflect.Descriptor)
	checkDescriptorComments = func(d protoreflect.Descriptor) {
		switch d := d.(type) {
		case protoreflect.FileDescriptor:
			msgs := d.Messages()
			for i, length := 0, msgs.Len(); i < length; i++ {
				checkDescriptorComments(msgs.Get(i))
			}
			enums := d.Enums()
			for i, length := 0, enums.Len(); i < length; i++ {
				checkDescriptorComments(enums.Get(i))
			}
			exts := d.Extensions()
			for i, length := 0, exts.Len(); i < length; i++ {
				checkDescriptorComments(exts.Get(i))
			}
			svcs := d.Services()
			for i, length := 0, svcs.Len(); i < length; i++ {
				checkDescriptorComments(svcs.Get(i))
			}
			// files don't have comments, so bail out before check below
			return
		case protoreflect.MessageDescriptor:
			if d.IsMapEntry() {
				// map entry messages have no comments (and neither do their child fields)
				return
			}
			fields := d.Fields()
			for i, length := 0, fields.Len(); i < length; i++ {
				checkDescriptorComments(fields.Get(i))
			}
			msgs := d.Messages()
			for i, length := 0, msgs.Len(); i < length; i++ {
				checkDescriptorComments(msgs.Get(i))
			}
			enums := d.Enums()
			for i, length := 0, enums.Len(); i < length; i++ {
				checkDescriptorComments(enums.Get(i))
			}
			exts := d.Extensions()
			for i, length := 0, exts.Len(); i < length; i++ {
				checkDescriptorComments(exts.Get(i))
			}
			oneofs := d.Oneofs()
			for i, length := 0, oneofs.Len(); i < length; i++ {
				checkDescriptorComments(oneofs.Get(i))
			}
		case protoreflect.FieldDescriptor:
			if d.Kind() == protoreflect.GroupKind {
				// groups comments are on the message, not the field; so bail out before check below
				return
			}
		case protoreflect.EnumDescriptor:
			vals := d.Values()
			for i, length := 0, vals.Len(); i < length; i++ {
				checkDescriptorComments(vals.Get(i))
			}
		case protoreflect.ServiceDescriptor:
			methods := d.Methods()
			for i, length := 0, methods.Len(); i < length; i++ {
				checkDescriptorComments(methods.Get(i))
			}
		}

		descCount++
		require.Equal(t,
			fmt.Sprintf(" Comment for %s\n", d.Name()),
			d.ParentFile().SourceLocations().ByDescriptor(d).LeadingComments,
			"wrong comment for descriptor %s", d.FullName())
	}

	checkDescriptorComments(fd)
	require.Equal(t, count, descCount)
}

func TestBuilder_PreserveAllCommentsAfterBuild(t *testing.T) {
	files := map[string]string{"test.proto": `
syntax = "proto3";

// Leading detached comment for SimpleEnum

// Leading comment for SimpleEnum
enum SimpleEnum {
// Trailing comment for SimpleEnum

  // Leading detached comment for VALUE0

  // Leading comment for VALUE0
  VALUE0 = 0; // Trailing comment for VALUE0
}

// Leading detached comment for SimpleMessage

// Leading comment for SimpleMessage
message SimpleMessage {
// Trailing comment for SimpleMessage

  // Leading detached comment for field1

  // Leading comment for field1
  optional SimpleEnum field1 = 1; // Trailing comment for field1
}
`}

	pa := &protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(files),
		},
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	fds, err := pa.Compile(context.Background(), "test.proto")
	require.NoError(t, err)

	fb, err := FromFile(fds[0])
	require.NoError(t, err)

	fd, err := fb.Build()
	require.NoError(t, err)

	var checkDescriptorComments func(d protoreflect.Descriptor)
	checkDescriptorComments = func(d protoreflect.Descriptor) {
		// fmt.Println(d.FullName(), d.GetSourceInfo().GetLeadingDetachedComments(), d.GetSourceInfo().GetLeadingComments(), d.GetSourceInfo().GetTrailingComments())
		switch d := d.(type) {
		case protoreflect.FileDescriptor:
			msgs := d.Messages()
			for i, length := 0, msgs.Len(); i < length; i++ {
				checkDescriptorComments(msgs.Get(i))
			}
			enums := d.Enums()
			for i, length := 0, enums.Len(); i < length; i++ {
				checkDescriptorComments(enums.Get(i))
			}
			// files don't have comments, so bail out before check below
			return
		case protoreflect.MessageDescriptor:
			if d.IsMapEntry() {
				// map entry messages have no comments (and neither do their child fields)
				return
			}
			fields := d.Fields()
			for i, length := 0, fields.Len(); i < length; i++ {
				checkDescriptorComments(fields.Get(i))
			}
		case protoreflect.FieldDescriptor:
			if d.Kind() == protoreflect.GroupKind {
				// groups comments are on the message, not the field; so bail out before check below
				return
			}
		case protoreflect.EnumDescriptor:
			vals := d.Values()
			for i, length := 0, vals.Len(); i < length; i++ {
				checkDescriptorComments(vals.Get(i))
			}
		}
		require.Equal(t,
			1,
			len(d.ParentFile().SourceLocations().ByDescriptor(d).LeadingDetachedComments),
			"wrong number of leading detached comments for %s", d.FullName())
		require.Equal(t,
			fmt.Sprintf(" Leading detached comment for %s\n", d.Name()),
			d.ParentFile().SourceLocations().ByDescriptor(d).LeadingDetachedComments[0],
			"wrong leading detached comment for descriptor %s", d.FullName())
		require.Equal(t,
			fmt.Sprintf(" Leading comment for %s\n", d.Name()),
			d.ParentFile().SourceLocations().ByDescriptor(d).LeadingComments,
			"wrong leading comment for descriptor %s", d.FullName())
		require.Equal(t,
			fmt.Sprintf(" Trailing comment for %s\n", d.Name()),
			d.ParentFile().SourceLocations().ByDescriptor(d).TrailingComments,
			"wrong trailing comment for descriptor %s", d.FullName())
	}

	checkDescriptorComments(fd)
}

func loadProtoset(path string) (protoresolve.Resolver, error) {
	var fds descriptorpb.FileDescriptorSet
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	bb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(bb, &fds); err != nil {
		return nil, err
	}
	return protowrap.FromFileDescriptorSet(&fds)
}

func roundTripFile(t *testing.T, fd protoreflect.FileDescriptor) {
	// First, recursively verify that every child element can be converted to a
	// Builder and back without loss of fidelity.
	msgs := fd.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		roundTripMessage(t, msgs.Get(0))
	}
	enums := fd.Enums()
	for i, length := 0, enums.Len(); i < length; i++ {
		roundTripEnum(t, enums.Get(0))
	}
	exts := fd.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		roundTripField(t, exts.Get(0))
	}
	svcs := fd.Services()
	for i, length := 0, svcs.Len(); i < length; i++ {
		roundTripService(t, svcs.Get(0))
	}

	// Finally, we check the whole file itself.
	fb, err := FromFile(fd)
	require.NoError(t, err)

	roundTripped, err := fb.Build()
	require.NoError(t, err)

	// Round tripping from a file descriptor to a builder and back will
	// experience some minor changes (that do not impact the semantics of
	// any of the file's contents):
	//  1. The builder sorts dependencies. However the original file
	//     descriptor has dependencies in the order they appear in import
	//     statements in the source file.
	//  2. The builder imports the actual source of all elements and never
	//     uses public imports. The original file, on the other hand, could
	//     use public imports and "indirectly" import other files that way.
	//  3. The builder never emits weak imports.
	//  4. The builder behaves like protoc in that it emits nil as the file
	//     package if none is set. However the new protobuf runtime, when
	//     reconstructing the proto from a protoreflect.FileDescriptor, will
	//     instead emit a pointer to empty string :(
	//  5. The builder tries to preserve SourceCodeInfo, but will not preserve
	//     position information. So that info does not survive round-tripping
	//     (though comments do: there is a separate test for that). Also, the
	//     round-tripped version will have source code info (even though it
	//     may have no comments and zero position info), even if the original
	//     descriptor had none.
	// So we're going to modify the original descriptor in the same ways.
	// That way, a simple proto.Equal() check will suffice to confirm that
	// the file descriptor survived the round trip.

	// The files we are testing have one occurrence of a public import. The
	// file nopkg/desc_test_nopkg.proto declares nothing and public imports
	// nopkg/desc_test_nopkg_new.proto. So any file that depends on the
	// former will be updated to instead depend on the latter (since it is
	// the actual file that declares used elements).
	fdp := protowrap.ProtoFromFileDescriptor(fd)
	needsNopkgNew := false
	hasNoPkgNew := false
	for _, dep := range fdp.Dependency {
		if dep == "nopkg/desc_test_nopkg.proto" {
			needsNopkgNew = true
		}
		if dep == "nopkg/desc_test_nopkg_new.proto" {
			hasNoPkgNew = false
		}
	}
	if needsNopkgNew && !hasNoPkgNew {
		fdp.Dependency = append(fdp.Dependency, "nopkg/desc_test_nopkg_new.proto")
	}

	// Strip any public and weak imports. (The step above should have "fixed"
	// files to handle any actual public import encountered.)
	fdp.PublicDependency = nil
	fdp.WeakDependency = nil

	// Fix the one we loaded so it uses nil as the package instead of an
	// empty string, since that is what builders produce.
	if fdp.GetPackage() == "" {
		fdp.Package = nil
	}
	// Fix the one we loaded so it has syntax set instead of unset, since
	// that is what builders produce.
	if fdp.Syntax == nil {
		fdp.Syntax = proto.String("proto2")
	}

	// Remove source code info: what the builder generates is not expected to
	// match the original source.
	fdp.SourceCodeInfo = nil
	roundTrippedProto := protowrap.ProtoFromFileDescriptor(roundTripped)
	roundTrippedProto.SourceCodeInfo = nil

	// Finally, sort the imports. That way they match the built result (which
	// is always sorted).
	sort.Strings(fdp.Dependency)

	// Now (after tweaking) the original should match the round-tripped descriptor:
	diff := cmp.Diff(fdp, roundTrippedProto, protocmp.Transform())
	require.Empty(t, diff)
}

func roundTripMessage(t *testing.T, md protoreflect.MessageDescriptor) {
	// first recursively validate all nested elements
	fields := md.Fields()
	for i, length := 0, fields.Len(); i < length; i++ {
		roundTripField(t, fields.Get(i))
	}
	oneofs := md.Oneofs()
	for i, length := 0, oneofs.Len(); i < length; i++ {
		ood := oneofs.Get(i)
		oob, err := FromOneof(ood)
		require.NoError(t, err)
		roundTripped, err := oob.Build()
		require.NoError(t, err)
		checkDescriptors(t, ood, roundTripped)
	}
	msgs := md.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		roundTripMessage(t, msgs.Get(i))
	}
	enums := md.Enums()
	for i, length := 0, enums.Len(); i < length; i++ {
		roundTripEnum(t, enums.Get(i))
	}
	exts := md.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		roundTripField(t, exts.Get(i))
	}

	mb, err := FromMessage(md)
	require.NoError(t, err)
	roundTripped, err := mb.Build()
	require.NoError(t, err)
	checkDescriptors(t, md, roundTripped)
}

func roundTripEnum(t *testing.T, ed protoreflect.EnumDescriptor) {
	// first recursively validate all nested elements
	vals := ed.Values()
	for i, length := 0, vals.Len(); i < length; i++ {
		evd := vals.Get(i)
		evb, err := FromEnumValue(evd)
		require.NoError(t, err)
		roundTripped, err := evb.Build()
		require.NoError(t, err)
		checkDescriptors(t, evd, roundTripped)
	}

	eb, err := FromEnum(ed)
	require.NoError(t, err)
	roundTripped, err := eb.Build()
	require.NoError(t, err)
	checkDescriptors(t, ed, roundTripped)
}

func roundTripField(t *testing.T, fld protoreflect.FieldDescriptor) {
	flb, err := FromField(fld)
	require.NoError(t, err)
	roundTripped, err := flb.Build()
	require.NoError(t, err)
	checkDescriptors(t, fld, roundTripped)
}

func roundTripService(t *testing.T, sd protoreflect.ServiceDescriptor) {
	// first recursively validate all nested elements
	methods := sd.Methods()
	for i, length := 0, methods.Len(); i < length; i++ {
		mtd := methods.Get(i)
		mtb, err := FromMethod(mtd)
		require.NoError(t, err)
		roundTripped, err := mtb.Build()
		require.NoError(t, err)
		checkDescriptors(t, mtd, roundTripped)
	}

	sb, err := FromService(sd)
	require.NoError(t, err)
	roundTripped, err := sb.Build()
	require.NoError(t, err)
	checkDescriptors(t, sd, roundTripped)
}

func checkDescriptors(t *testing.T, d1, d2 protoreflect.Descriptor) {
	require.Equal(t, d1.FullName(), d2.FullName())
	diff := cmp.Diff(protowrap.ProtoFromDescriptor(d1), protowrap.ProtoFromDescriptor(d2), protocmp.Transform())
	require.Empty(t, diff)
}

func TestAddRemoveMoveBuilders(t *testing.T) {
	// add field to one-of
	fld1 := NewField("foo", FieldTypeInt32())
	oo1 := NewOneof("oofoo")
	oo1.AddChoice(fld1)
	checkChildren(t, oo1, fld1)
	require.Equal(t, oo1.GetChoice("foo"), fld1)

	// add one-of w/ field to a message
	msg1 := NewMessage("foo")
	msg1.AddOneOf(oo1)
	checkChildren(t, msg1, oo1)
	require.Equal(t, msg1.GetOneOf("oofoo"), oo1)
	// field remains unchanged
	require.Equal(t, fld1.Parent(), oo1)
	require.Equal(t, oo1.GetChoice("foo"), fld1)
	// field also now registered with msg1
	require.Equal(t, msg1.GetField("foo"), fld1)

	// add empty one-of to message
	oo2 := NewOneof("oobar")
	msg1.AddOneOf(oo2)
	checkChildren(t, msg1, oo1, oo2)
	require.Equal(t, msg1.GetOneOf("oobar"), oo2)
	// now add field to that one-of
	fld2 := NewField("bar", FieldTypeInt32())
	oo2.AddChoice(fld2)
	checkChildren(t, oo2, fld2)
	require.Equal(t, oo2.GetChoice("bar"), fld2)
	// field also now registered with msg1
	require.Equal(t, msg1.GetField("bar"), fld2)

	// add fails due to path collisions
	fld1dup := NewField("foo", FieldTypeInt32())
	err := oo1.TryAddChoice(fld1dup)
	checkFailedAdd(t, err, oo1, fld1dup, "already contains field")
	fld2 = NewField("bar", FieldTypeInt32())
	err = msg1.TryAddField(fld2)
	checkFailedAdd(t, err, msg1, fld2, "already contains element")
	msg2 := NewMessage("oofoo")
	// path collision can be different type
	// (here, nested message conflicts with a one-of)
	err = msg1.TryAddNestedMessage(msg2)
	checkFailedAdd(t, err, msg1, msg2, "already contains element")

	msg2 = NewMessage("baz")
	msg1.AddNestedMessage(msg2)
	checkChildren(t, msg1, oo1, oo2, msg2)
	require.Equal(t, msg1.GetNestedMessage("baz"), msg2)

	// can't add extension or map fields to one-of
	ext1 := NewExtension("abc", 123, FieldTypeInt32(), msg1)
	err = oo1.TryAddChoice(ext1)
	checkFailedAdd(t, err, oo1, ext1, "is an extension, not a regular field")
	err = msg1.TryAddField(ext1)
	checkFailedAdd(t, err, msg1, ext1, "is an extension, not a regular field")
	mapField := NewMapField("abc", FieldTypeInt32(), FieldTypeString())
	err = oo1.TryAddChoice(mapField)
	checkFailedAdd(t, err, oo1, mapField, "cannot add a map field")
	// can add group field though
	groupMsg := NewMessage("Group")
	groupField := NewGroupField(groupMsg)
	oo1.AddChoice(groupField)
	checkChildren(t, oo1, fld1, groupField)
	// adding map and group to msg succeeds
	msg1.AddField(groupField)
	msg1.AddField(mapField)
	checkChildren(t, msg1, oo1, oo2, msg2, groupField, mapField)
	// messages associated with map and group fields are not children of the
	// message, but are in its scope and accessible via GetNestedMessage
	require.Equal(t, msg1.GetNestedMessage("Group"), groupMsg)
	require.Equal(t, msg1.GetNestedMessage("AbcEntry"), mapField.Type().localMsgType)

	// adding extension to message
	ext2 := NewExtension("xyz", 234, FieldTypeInt32(), msg1)
	msg1.AddNestedExtension(ext2)
	checkChildren(t, msg1, oo1, oo2, msg2, groupField, mapField, ext2)
	err = msg1.TryAddNestedExtension(ext1) // path collision
	checkFailedAdd(t, err, msg1, ext1, "already contains element")
	fld3 := NewField("ijk", FieldTypeString())
	err = msg1.TryAddNestedExtension(fld3)
	checkFailedAdd(t, err, msg1, fld3, "is not an extension")

	// add enum values to enum
	enumVal1 := NewEnumValue("A")
	enum1 := NewEnum("bazel")
	enum1.AddValue(enumVal1)
	checkChildren(t, enum1, enumVal1)
	require.Equal(t, enum1.GetValue("A"), enumVal1)
	enumVal2 := NewEnumValue("B")
	enum1.AddValue(enumVal2)
	checkChildren(t, enum1, enumVal1, enumVal2)
	require.Equal(t, enum1.GetValue("B"), enumVal2)
	// fail w/ path collision
	enumVal3 := NewEnumValue("B")
	err = enum1.TryAddValue(enumVal3)
	checkFailedAdd(t, err, enum1, enumVal3, "already contains value")

	msg2.AddNestedEnum(enum1)
	checkChildren(t, msg2, enum1)
	require.Equal(t, msg2.GetNestedEnum("bazel"), enum1)
	ext3 := NewExtension("bazel", 987, FieldTypeString(), msg2)
	err = msg2.TryAddNestedExtension(ext3)
	checkFailedAdd(t, err, msg2, ext3, "already contains element")

	// services and methods
	mtd1 := NewMethod("foo", RpcTypeMessage(msg1, false), RpcTypeMessage(msg1, false))
	svc1 := NewService("FooService")
	svc1.AddMethod(mtd1)
	checkChildren(t, svc1, mtd1)
	require.Equal(t, svc1.GetMethod("foo"), mtd1)
	mtd2 := NewMethod("foo", RpcTypeMessage(msg1, false), RpcTypeMessage(msg1, false))
	err = svc1.TryAddMethod(mtd2)
	checkFailedAdd(t, err, svc1, mtd2, "already contains method")

	// finally, test adding things to  a file
	fb := NewFile("")
	fb.AddMessage(msg1)
	checkChildren(t, fb, msg1)
	require.Equal(t, fb.GetMessage("foo"), msg1)
	fb.AddService(svc1)
	checkChildren(t, fb, msg1, svc1)
	require.Equal(t, fb.GetService("FooService"), svc1)
	enum2 := NewEnum("fizzle")
	fb.AddEnum(enum2)
	checkChildren(t, fb, msg1, svc1, enum2)
	require.Equal(t, fb.GetEnum("fizzle"), enum2)
	ext3 = NewExtension("foosball", 123, FieldTypeInt32(), msg1)
	fb.AddExtension(ext3)
	checkChildren(t, fb, msg1, svc1, enum2, ext3)
	require.Equal(t, fb.GetExtension("foosball"), ext3)

	// errors and path collisions
	err = fb.TryAddExtension(fld3)
	checkFailedAdd(t, err, fb, fld3, "is not an extension")
	msg3 := NewMessage("fizzle")
	err = fb.TryAddMessage(msg3)
	checkFailedAdd(t, err, fb, msg3, "already contains element")
	enum3 := NewEnum("foosball")
	err = fb.TryAddEnum(enum3)
	checkFailedAdd(t, err, fb, enum3, "already contains element")

	// TODO: test moving and removing, too
}

func checkChildren(t *testing.T, parent Builder, children ...Builder) {
	require.Equal(t, len(children), len(parent.Children()), "Wrong number of children for %s (%T)", FullName(parent), parent)
	ch := map[Builder]struct{}{}
	for _, child := range children {
		require.Equal(t, child.Parent(), parent, "Child %s (%T) does not report %s (%T) as its parent", child.Name(), child, FullName(parent), parent)
		ch[child] = struct{}{}
	}
	for _, child := range parent.Children() {
		_, ok := ch[child]
		require.True(t, ok, "Child %s (%T) does appear in list of children for %s (%T)", child.Name(), child, FullName(parent), parent)
	}
}

func checkFailedAdd(t *testing.T, err error, parent Builder, child Builder, errorMsg string) {
	require.ErrorContains(t, err, errorMsg, "Expecting error assigning %s (%T) to %s (%T)", child.Name(), child, FullName(parent), parent)
	require.Equal(t, nil, child.Parent(), "Child %s (%T) should not have a parent after failed add", child.Name(), child)
	for _, ch := range parent.Children() {
		require.True(t, ch != child, "Child %s (%T) should not appear in list of children for %s (%T) but does", child.Name(), child, FullName(parent), parent)
	}
}

func TestRenamingBuilders(t *testing.T) {
	// TODO
}

func TestRenumberingFields(t *testing.T) {
	// TODO
}

var (
	fileOptionsDesc     = (*descriptorpb.FileOptions)(nil).ProtoReflect().Descriptor()
	msgOptionsDesc      = (*descriptorpb.MessageOptions)(nil).ProtoReflect().Descriptor()
	fieldOptionsDesc    = (*descriptorpb.FieldOptions)(nil).ProtoReflect().Descriptor()
	oneofOptionsDesc    = (*descriptorpb.OneofOptions)(nil).ProtoReflect().Descriptor()
	extRangeOptionsDesc = (*descriptorpb.ExtensionRangeOptions)(nil).ProtoReflect().Descriptor()
	enumOptionsDesc     = (*descriptorpb.EnumOptions)(nil).ProtoReflect().Descriptor()
	enumValOptionsDesc  = (*descriptorpb.EnumValueOptions)(nil).ProtoReflect().Descriptor()
	svcOptionsDesc      = (*descriptorpb.ServiceOptions)(nil).ProtoReflect().Descriptor()
	mtdOptionsDesc      = (*descriptorpb.MethodOptions)(nil).ProtoReflect().Descriptor()
)

func TestCustomOptionsDiscoveredInSameFile(t *testing.T) {
	// Add option for every type to file
	file := NewFile("foo.proto")

	fileOpt := NewExtensionImported("file_foo", 54321, FieldTypeString(), fileOptionsDesc)
	file.AddExtension(fileOpt)

	msgOpt := NewExtensionImported("msg_foo", 54321, FieldTypeString(), msgOptionsDesc)
	file.AddExtension(msgOpt)

	fieldOpt := NewExtensionImported("field_foo", 54321, FieldTypeString(), fieldOptionsDesc)
	file.AddExtension(fieldOpt)

	oneofOpt := NewExtensionImported("oneof_foo", 54321, FieldTypeString(), oneofOptionsDesc)
	file.AddExtension(oneofOpt)

	extRangeOpt := NewExtensionImported("ext_range_foo", 54321, FieldTypeString(), extRangeOptionsDesc)
	file.AddExtension(extRangeOpt)

	enumOpt := NewExtensionImported("enum_foo", 54321, FieldTypeString(), enumOptionsDesc)
	file.AddExtension(enumOpt)

	enumValOpt := NewExtensionImported("enum_val_foo", 54321, FieldTypeString(), enumValOptionsDesc)
	file.AddExtension(enumValOpt)

	svcOpt := NewExtensionImported("svc_foo", 54321, FieldTypeString(), svcOptionsDesc)
	file.AddExtension(svcOpt)

	mtdOpt := NewExtensionImported("mtd_foo", 54321, FieldTypeString(), mtdOptionsDesc)
	file.AddExtension(mtdOpt)

	// Now we can test referring to these and making sure they show up correctly
	// in built descriptors

	t.Run("file options", func(t *testing.T) {
		fb := clone(t, file)
		fb.Options = &descriptorpb.FileOptions{}
		ext, err := fileOpt.Build()
		require.NoError(t, err)
		fb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))
		checkBuildWithLocalExtensions(t, fb)
	})

	t.Run("message options", func(t *testing.T) {
		mb := NewMessage("Foo")
		mb.Options = &descriptorpb.MessageOptions{}
		ext, err := msgOpt.Build()
		require.NoError(t, err)
		mb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

		fb := clone(t, file)
		fb.AddMessage(mb)
		checkBuildWithLocalExtensions(t, mb)
	})

	t.Run("field options", func(t *testing.T) {
		flb := NewField("foo", FieldTypeString())
		flb.Options = &descriptorpb.FieldOptions{}
		// fields must be connected to a message
		mb := NewMessage("Foo").AddField(flb)
		ext, err := fieldOpt.Build()
		require.NoError(t, err)
		flb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

		fb := clone(t, file)
		fb.AddMessage(mb)
		checkBuildWithLocalExtensions(t, flb)
	})

	t.Run("oneof options", func(t *testing.T) {
		oob := NewOneof("oo")
		oob.AddChoice(NewField("foo", FieldTypeString()))
		oob.Options = &descriptorpb.OneofOptions{}
		// oneofs must be connected to a message
		mb := NewMessage("Foo").AddOneOf(oob)
		ext, err := oneofOpt.Build()
		require.NoError(t, err)
		oob.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

		fb := clone(t, file)
		fb.AddMessage(mb)
		checkBuildWithLocalExtensions(t, oob)
	})

	t.Run("extension range options", func(t *testing.T) {
		var erOpts descriptorpb.ExtensionRangeOptions
		ext, err := extRangeOpt.Build()
		require.NoError(t, err)
		erOpts.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))
		mb := NewMessage("foo").AddExtensionRangeWithOptions(100, 200, &erOpts)

		fb := clone(t, file)
		fb.AddMessage(mb)
		checkBuildWithLocalExtensions(t, mb)
	})

	t.Run("enum options", func(t *testing.T) {
		eb := NewEnum("Foo")
		eb.AddValue(NewEnumValue("FOO"))
		eb.Options = &descriptorpb.EnumOptions{}
		ext, err := enumOpt.Build()
		require.NoError(t, err)
		eb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

		fb := clone(t, file)
		fb.AddEnum(eb)
		checkBuildWithLocalExtensions(t, eb)
	})

	t.Run("enum val options", func(t *testing.T) {
		evb := NewEnumValue("FOO")
		// enum values must be connected to an enum
		eb := NewEnum("Foo").AddValue(evb)
		evb.Options = &descriptorpb.EnumValueOptions{}
		ext, err := enumValOpt.Build()
		require.NoError(t, err)
		evb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

		fb := clone(t, file)
		fb.AddEnum(eb)
		checkBuildWithLocalExtensions(t, evb)
	})

	t.Run("service options", func(t *testing.T) {
		sb := NewService("Foo")
		sb.Options = &descriptorpb.ServiceOptions{}
		ext, err := svcOpt.Build()
		require.NoError(t, err)
		sb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

		fb := clone(t, file)
		fb.AddService(sb)
		checkBuildWithLocalExtensions(t, sb)
	})

	t.Run("method options", func(t *testing.T) {
		req := NewMessage("Request")
		resp := NewMessage("Response")
		mtb := NewMethod("Foo",
			RpcTypeMessage(req, false),
			RpcTypeMessage(resp, false))
		// methods must be connected to a service
		sb := NewService("Bar").AddMethod(mtb)
		mtb.Options = &descriptorpb.MethodOptions{}
		ext, err := mtdOpt.Build()
		require.NoError(t, err)
		mtb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

		fb := clone(t, file)
		fb.AddService(sb).AddMessage(req).AddMessage(resp)
		checkBuildWithLocalExtensions(t, mtb)
	})
}

func checkBuildWithLocalExtensions(t *testing.T, builder Builder) {
	// requiring options and succeeding (since they are defined locally)
	var opts BuilderOptions
	opts.RequireInterpretedOptions = true
	d, err := opts.Build(builder)
	require.NoError(t, err)
	// since they are defined locally, no extra imports
	require.Equal(t, 1, d.ParentFile().Imports().Len())
	require.Equal(t, "google/protobuf/descriptor.proto", d.ParentFile().Imports().Get(0).Path())
}

func TestCustomOptionsDiscoveredInDependencies(t *testing.T) {
	// Add option for every type to file
	file := NewFile("options.proto")

	fileOpt := NewExtensionImported("file_foo", 54321, FieldTypeString(), fileOptionsDesc)
	file.AddExtension(fileOpt)

	msgOpt := NewExtensionImported("msg_foo", 54321, FieldTypeString(), msgOptionsDesc)
	file.AddExtension(msgOpt)

	fieldOpt := NewExtensionImported("field_foo", 54321, FieldTypeString(), fieldOptionsDesc)
	file.AddExtension(fieldOpt)

	oneofOpt := NewExtensionImported("oneof_foo", 54321, FieldTypeString(), oneofOptionsDesc)
	file.AddExtension(oneofOpt)

	extRangeOpt := NewExtensionImported("ext_range_foo", 54321, FieldTypeString(), extRangeOptionsDesc)
	file.AddExtension(extRangeOpt)

	enumOpt := NewExtensionImported("enum_foo", 54321, FieldTypeString(), enumOptionsDesc)
	file.AddExtension(enumOpt)

	enumValOpt := NewExtensionImported("enum_val_foo", 54321, FieldTypeString(), enumValOptionsDesc)
	file.AddExtension(enumValOpt)

	svcOpt := NewExtensionImported("svc_foo", 54321, FieldTypeString(), svcOptionsDesc)
	file.AddExtension(svcOpt)

	mtdOpt := NewExtensionImported("mtd_foo", 54321, FieldTypeString(), mtdOptionsDesc)
	file.AddExtension(mtdOpt)

	fileDesc, err := file.Build()
	require.NoError(t, err)

	// Now we can test referring to these and making sure they show up correctly
	// in built descriptors
	for name, useBuilder := range map[string]bool{"descriptor": false, "builder": true} {
		newFile := func() *FileBuilder {
			fb := NewFile("foo.proto")
			if useBuilder {
				fb.AddDependency(file)
			} else {
				fb.AddImportedDependency(fileDesc)
			}
			return fb
		}
		t.Run(name, func(t *testing.T) {
			t.Run("file options", func(t *testing.T) {
				fb := newFile()
				fb.Options = &descriptorpb.FileOptions{}
				ext, err := fileOpt.Build()
				require.NoError(t, err)
				fb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))
				checkBuildWithImportedExtensions(t, fb)
			})

			t.Run("message options", func(t *testing.T) {
				mb := NewMessage("Foo")
				mb.Options = &descriptorpb.MessageOptions{}
				ext, err := msgOpt.Build()
				require.NoError(t, err)
				mb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

				fb := newFile()
				fb.AddMessage(mb)
				checkBuildWithImportedExtensions(t, mb)
			})

			t.Run("field options", func(t *testing.T) {
				flb := NewField("foo", FieldTypeString())
				flb.Options = &descriptorpb.FieldOptions{}
				// fields must be connected to a message
				mb := NewMessage("Foo").AddField(flb)
				ext, err := fieldOpt.Build()
				require.NoError(t, err)
				flb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

				fb := newFile()
				fb.AddMessage(mb)
				checkBuildWithImportedExtensions(t, flb)
			})

			t.Run("oneof options", func(t *testing.T) {
				oob := NewOneof("oo")
				oob.AddChoice(NewField("foo", FieldTypeString()))
				oob.Options = &descriptorpb.OneofOptions{}
				// oneofs must be connected to a message
				mb := NewMessage("Foo").AddOneOf(oob)
				ext, err := oneofOpt.Build()
				require.NoError(t, err)
				oob.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

				fb := newFile()
				fb.AddMessage(mb)
				checkBuildWithImportedExtensions(t, oob)
			})

			t.Run("extension range options", func(t *testing.T) {
				var erOpts descriptorpb.ExtensionRangeOptions
				ext, err := extRangeOpt.Build()
				require.NoError(t, err)
				erOpts.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))
				mb := NewMessage("foo").AddExtensionRangeWithOptions(100, 200, &erOpts)

				fb := newFile()
				fb.AddMessage(mb)
				checkBuildWithImportedExtensions(t, mb)
			})

			t.Run("enum options", func(t *testing.T) {
				eb := NewEnum("Foo")
				eb.AddValue(NewEnumValue("FOO"))
				eb.Options = &descriptorpb.EnumOptions{}
				ext, err := enumOpt.Build()
				require.NoError(t, err)
				eb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

				fb := newFile()
				fb.AddEnum(eb)
				checkBuildWithImportedExtensions(t, eb)
			})

			t.Run("enum val options", func(t *testing.T) {
				evb := NewEnumValue("FOO")
				// enum values must be connected to an enum
				eb := NewEnum("Foo").AddValue(evb)
				evb.Options = &descriptorpb.EnumValueOptions{}
				ext, err := enumValOpt.Build()
				require.NoError(t, err)
				evb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

				fb := newFile()
				fb.AddEnum(eb)
				checkBuildWithImportedExtensions(t, evb)
			})

			t.Run("service options", func(t *testing.T) {
				sb := NewService("Foo")
				sb.Options = &descriptorpb.ServiceOptions{}
				ext, err := svcOpt.Build()
				require.NoError(t, err)
				sb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

				fb := newFile()
				fb.AddService(sb)
				checkBuildWithImportedExtensions(t, sb)
			})

			t.Run("method options", func(t *testing.T) {
				req := NewMessage("Request")
				resp := NewMessage("Response")
				mtb := NewMethod("Foo",
					RpcTypeMessage(req, false),
					RpcTypeMessage(resp, false))
				// methods must be connected to a service
				sb := NewService("Bar").AddMethod(mtb)
				mtb.Options = &descriptorpb.MethodOptions{}
				ext, err := mtdOpt.Build()
				require.NoError(t, err)
				mtb.Options.ProtoReflect().Set(ext, protoreflect.ValueOfString("fubar"))

				fb := newFile()
				fb.AddService(sb).AddMessage(req).AddMessage(resp)
				checkBuildWithImportedExtensions(t, mtb)
			})
		})
	}
}

func checkBuildWithImportedExtensions(t *testing.T, builder Builder) {
	// requiring options and succeeding (since they are defined in explicit import)
	var opts BuilderOptions
	opts.RequireInterpretedOptions = true
	d, err := opts.Build(builder)
	require.NoError(t, err)
	// the only import is for the custom options
	require.Equal(t, 1, d.ParentFile().Imports().Len())
	require.Equal(t, "options.proto", d.ParentFile().Imports().Get(0).Path())
}

func TestUseOfExtensionRegistry(t *testing.T) {
	// Add option for every type to extension registry
	var exts protoregistry.Types

	fileOpt, err := NewExtensionImported("file_foo", 54321, FieldTypeString(), fileOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(fileOpt))
	require.NoError(t, err)

	msgOpt, err := NewExtensionImported("msg_foo", 54321, FieldTypeString(), msgOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(msgOpt))
	require.NoError(t, err)

	fieldOpt, err := NewExtensionImported("field_foo", 54321, FieldTypeString(), fieldOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(fieldOpt))
	require.NoError(t, err)

	oneofOpt, err := NewExtensionImported("oneof_foo", 54321, FieldTypeString(), oneofOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(oneofOpt))
	require.NoError(t, err)

	extRangeOpt, err := NewExtensionImported("ext_range_foo", 54321, FieldTypeString(), extRangeOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(extRangeOpt))
	require.NoError(t, err)

	enumOpt, err := NewExtensionImported("enum_foo", 54321, FieldTypeString(), enumOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(enumOpt))
	require.NoError(t, err)

	enumValOpt, err := NewExtensionImported("enum_val_foo", 54321, FieldTypeString(), enumValOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(enumValOpt))
	require.NoError(t, err)

	svcOpt, err := NewExtensionImported("svc_foo", 54321, FieldTypeString(), svcOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(svcOpt))
	require.NoError(t, err)

	mtdOpt, err := NewExtensionImported("mtd_foo", 54321, FieldTypeString(), mtdOptionsDesc).Build()
	require.NoError(t, err)
	err = exts.RegisterExtension(protoresolve.ExtensionType(mtdOpt))
	require.NoError(t, err)

	// Now we can test referring to these and making sure they show up correctly
	// in built descriptors

	t.Run("file options", func(t *testing.T) {
		fb := NewFile("foo.proto")
		fb.Options = &descriptorpb.FileOptions{}
		fb.Options.ProtoReflect().SetUnknown(unrecognizedFieldString(fileOpt, "fubar"))
		checkBuildWithExtensions(t, &exts, fileOpt.ParentFile(), fb)
	})

	t.Run("message options", func(t *testing.T) {
		mb := NewMessage("Foo")
		mb.Options = &descriptorpb.MessageOptions{}
		mb.Options.ProtoReflect().SetUnknown(unrecognizedFieldString(msgOpt, "fubar"))
		checkBuildWithExtensions(t, &exts, msgOpt.ParentFile(), mb)
	})

	t.Run("field options", func(t *testing.T) {
		flb := NewField("foo", FieldTypeString())
		flb.Options = &descriptorpb.FieldOptions{}
		// fields must be connected to a message
		NewMessage("Foo").AddField(flb)
		flb.Options.ProtoReflect().SetUnknown(unrecognizedFieldString(fieldOpt, "fubar"))
		checkBuildWithExtensions(t, &exts, fieldOpt.ParentFile(), flb)
	})

	t.Run("oneof options", func(t *testing.T) {
		oob := NewOneof("oo")
		oob.AddChoice(NewField("foo", FieldTypeString()))
		oob.Options = &descriptorpb.OneofOptions{}
		// oneofs must be connected to a message
		NewMessage("Foo").AddOneOf(oob)
		oob.Options.ProtoReflect().SetUnknown(unrecognizedFieldString(oneofOpt, "fubar"))
		checkBuildWithExtensions(t, &exts, oneofOpt.ParentFile(), oob)
	})

	t.Run("extension range options", func(t *testing.T) {
		var erOpts descriptorpb.ExtensionRangeOptions
		erOpts.ProtoReflect().SetUnknown(unrecognizedFieldString(extRangeOpt, "fubar"))
		mb := NewMessage("foo").AddExtensionRangeWithOptions(100, 200, &erOpts)
		checkBuildWithExtensions(t, &exts, extRangeOpt.ParentFile(), mb)
	})

	t.Run("enum options", func(t *testing.T) {
		eb := NewEnum("Foo")
		eb.AddValue(NewEnumValue("FOO"))
		eb.Options = &descriptorpb.EnumOptions{}
		eb.Options.ProtoReflect().SetUnknown(unrecognizedFieldString(enumOpt, "fubar"))
		checkBuildWithExtensions(t, &exts, enumOpt.ParentFile(), eb)
	})

	t.Run("enum val options", func(t *testing.T) {
		evb := NewEnumValue("FOO")
		// enum values must be connected to an enum
		NewEnum("Foo").AddValue(evb)
		evb.Options = &descriptorpb.EnumValueOptions{}
		evb.Options.ProtoReflect().SetUnknown(unrecognizedFieldString(enumValOpt, "fubar"))
		checkBuildWithExtensions(t, &exts, enumValOpt.ParentFile(), evb)
	})

	t.Run("service options", func(t *testing.T) {
		sb := NewService("Foo")
		sb.Options = &descriptorpb.ServiceOptions{}
		sb.Options.ProtoReflect().SetUnknown(unrecognizedFieldString(svcOpt, "fubar"))
		checkBuildWithExtensions(t, &exts, svcOpt.ParentFile(), sb)
	})

	t.Run("method options", func(t *testing.T) {
		mtb := NewMethod("Foo",
			RpcTypeMessage(NewMessage("Request"), false),
			RpcTypeMessage(NewMessage("Response"), false))
		// methods must be connected to a service
		NewService("Bar").AddMethod(mtb)
		mtb.Options = &descriptorpb.MethodOptions{}
		mtb.Options.ProtoReflect().SetUnknown(unrecognizedFieldString(mtdOpt, "fubar"))
		checkBuildWithExtensions(t, &exts, mtdOpt.ParentFile(), mtb)
	})
}

func unrecognizedFieldString(ext protoreflect.FieldDescriptor, str string) protoreflect.RawFields {
	var f protoreflect.RawFields
	f = protowire.AppendTag(f, ext.Number(), protowire.BytesType)
	return protowire.AppendString(f, str)
}

func checkBuildWithExtensions(t *testing.T, exts protoresolve.ExtensionTypeResolver, expected protoreflect.FileDescriptor, builder Builder) {
	// without interpreting custom option
	d, err := builder.BuildDescriptor()
	require.NoError(t, err)
	deps := d.ParentFile().Imports()
	for i, length := 0, deps.Len(); i < length; i++ {
		dep := deps.Get(i)
		require.NotEqual(t, expected, dep)
	}
	numDeps := d.ParentFile().Imports().Len()

	// requiring options (and failing)
	var opts BuilderOptions
	opts.RequireInterpretedOptions = true
	_, err = opts.Build(builder)
	require.NotNil(t, err)

	// able to interpret options via extension registry
	opts.Resolver = exts
	d, err = opts.Build(builder)
	require.NoError(t, err)
	require.Equal(t, numDeps+1, d.ParentFile().Imports().Len())
	found := false
	deps = d.ParentFile().Imports()
	for i, length := 0, deps.Len(); i < length; i++ {
		dep := deps.Get(i).FileDescriptor
		if expected == dep {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestRemoveField(t *testing.T) {
	msg := NewMessage("FancyMessage").
		AddField(NewField("one", FieldTypeInt64())).
		AddField(NewField("two", FieldTypeString())).
		AddField(NewField("three", FieldTypeString()))

	ok := msg.TryRemoveField("two")
	children := msg.Children()

	require.True(t, ok)
	require.Equal(t, 2, len(children))
	require.Equal(t, protoreflect.Name("one"), children[0].Name())
	require.Equal(t, protoreflect.Name("three"), children[1].Name())
}

func TestInterleavedFieldNumbers(t *testing.T) {
	msg := NewMessage("MessageWithInterleavedFieldNumbers").
		AddField(NewField("one", FieldTypeInt64()).SetNumber(1)).
		AddField(NewField("two", FieldTypeInt64())).
		AddField(NewField("three", FieldTypeString()).SetNumber(3)).
		AddField(NewField("four", FieldTypeInt64())).
		AddField(NewField("five", FieldTypeString()).SetNumber(5))

	md, err := msg.Build()
	require.NoError(t, err)

	require.NotNil(t, md.Fields().ByName("one"))
	require.Equal(t, protoreflect.FieldNumber(1), md.Fields().ByName("one").Number())

	require.NotNil(t, md.Fields().ByName("two"))
	require.Equal(t, protoreflect.FieldNumber(2), md.Fields().ByName("two").Number())

	require.NotNil(t, md.Fields().ByName("three"))
	require.Equal(t, protoreflect.FieldNumber(3), md.Fields().ByName("three").Number())

	require.NotNil(t, md.Fields().ByName("four"))
	require.Equal(t, protoreflect.FieldNumber(4), md.Fields().ByName("four").Number())

	require.NotNil(t, md.Fields().ByName("five"))
	require.Equal(t, protoreflect.FieldNumber(5), md.Fields().ByName("five").Number())
}

func clone(t *testing.T, fb *FileBuilder) *FileBuilder {
	fd, err := fb.Build()
	require.NoError(t, err)
	fb, err = FromFile(fd)
	require.NoError(t, err)
	return fb
}

func TestPruneDependencies(t *testing.T) {
	extDesc, err := NewExtensionImported("foo", 20001, FieldTypeString(), msgOptionsDesc).Build()
	require.NoError(t, err)

	msgOpts := &descriptorpb.MessageOptions{}
	msgOpts.ProtoReflect().Set(extDesc, protoreflect.ValueOfString("bar"))

	emptyDesc := (*emptypb.Empty)(nil).ProtoReflect().Descriptor()

	// we have to explicitly import the file for the custom option
	fileB := NewFile("").AddImportedDependency(extDesc.ParentFile())
	msgB := NewMessage("Foo").
		AddField(NewField("a", FieldTypeImportedMessage(emptyDesc))).
		SetOptions(msgOpts)
	fileDesc, err := fileB.AddMessage(msgB).Build()
	require.NoError(t, err)

	// The file for msgDesc should have two imports: one for the custom option and
	//   one for empty.proto.
	require.Equal(t, 2, fileDesc.Imports().Len())
	require.Equal(t, "google/protobuf/empty.proto", fileDesc.Imports().Get(0).Path())
	require.Equal(t, extDesc.ParentFile().Path(), fileDesc.Imports().Get(1).Path())

	// If we now remove the message's field, both imports are still there even
	// though the import for empty.proto is now unused.
	fileB, err = FromFile(fileDesc)
	require.NoError(t, err)
	fileB.GetMessage("Foo").RemoveField("a")
	newFileDesc, err := fileB.Build()
	require.NoError(t, err)
	require.Equal(t, 2, newFileDesc.Imports().Len())
	require.Equal(t, "google/protobuf/empty.proto", newFileDesc.Imports().Get(0).Path())
	require.Equal(t, extDesc.ParentFile().Path(), newFileDesc.Imports().Get(1).Path())

	// But if we prune unused dependencies, we'll see the import for empty.proto
	// gone. The other import for the custom option should be preserved.
	fileB, err = FromFile(fileDesc)
	require.NoError(t, err)
	fileB.GetMessage("Foo").RemoveField("a")
	newFileDesc, err = fileB.PruneUnusedDependencies().Build()
	require.NoError(t, err)
	require.Equal(t, 1, newFileDesc.Imports().Len())
	require.Equal(t, extDesc.ParentFile().Path(), newFileDesc.Imports().Get(0).Path())
}

func TestInvalid(t *testing.T) {
	testCases := []struct {
		name          string
		builder       func() Builder
		expectedError string
	}{
		{
			name: "required in proto3",
			builder: func() Builder {
				return NewFile("foo.proto").
					SetSyntax(protoreflect.Proto3).
					AddMessage(
						NewMessage("Foo").AddField(NewField("foo", FieldTypeBool()).SetRequired()),
					)
			},
			expectedError: "proto3 does not allow required fields",
		},
		{
			name: "extension range in proto3",
			builder: func() Builder {
				return NewFile("foo.proto").
					SetSyntax(protoreflect.Proto3).
					AddMessage(
						NewMessage("Foo").AddExtensionRange(100, 1000),
					)
			},
			expectedError: "proto3 semantics cannot have extension ranges",
		},
		{
			name: "group in proto3",
			builder: func() Builder {
				return NewFile("foo.proto").
					SetSyntax(protoreflect.Proto3).
					AddMessage(
						NewMessage("Foo").AddField(NewGroupField(NewMessage("Bar"))),
					)
			},
			// NB: This is the actual error message returned by the protobuf runtime. It is
			//     misleading since it says proto2 instead of proto3.
			expectedError: "invalid group: invalid under proto2 semantics",
		},
		{
			name: "default value in proto3",
			builder: func() Builder {
				return NewFile("foo.proto").
					SetSyntax(protoreflect.Proto3).
					AddMessage(
						NewMessage("Foo").AddField(NewField("foo", FieldTypeString()).SetDefaultValue("abc")),
					)
			},
			expectedError: "invalid default: cannot be specified under proto3 semantics",
		},
		{
			name: "extension tag outside range",
			builder: func() Builder {
				msg := NewMessage("Foo").AddExtensionRange(100, 1000)
				return NewFile("foo.proto").
					AddMessage(msg).
					AddExtension(NewExtension("foo", 1, FieldTypeString(), msg))
			},
			expectedError: "non-extension field number: 1",
		},
		{
			name: "non-extension tag in extension range",
			builder: func() Builder {
				return NewFile("foo.proto").
					AddMessage(NewMessage("Foo").
						AddField(NewField("foo", FieldTypeBool()).SetNumber(100)).
						AddExtensionRange(100, 1000))
			},
			expectedError: "number 100 in extension range",
		},
		{
			name: "tag in reserved range",
			builder: func() Builder {
				return NewFile("foo.proto").
					AddMessage(NewMessage("Foo").
						AddField(NewField("foo", FieldTypeBool()).SetNumber(100)).
						AddReservedRange(100, 1000))
			},
			expectedError: "must not use reserved number 100",
		},
		{
			name: "field has reserved path",
			builder: func() Builder {
				return NewFile("foo.proto").
					AddMessage(NewMessage("Foo").
						AddField(NewField("foo", FieldTypeBool())).
						AddReservedName("foo"))
			},
			expectedError: "must not use reserved name",
		},
		{
			name: "ranges overlap",
			builder: func() Builder {
				return NewFile("foo.proto").
					AddMessage(NewMessage("Foo").
						AddReservedRange(100, 1000).
						AddExtensionRange(200, 300))
			},
			expectedError: "reserved and extension ranges has overlapping ranges",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.builder().BuildDescriptor()
			require.ErrorContains(t, err, testCase.expectedError, "unexpected error: want %q, got %q", testCase.expectedError, err.Error())
		})
	}
}
