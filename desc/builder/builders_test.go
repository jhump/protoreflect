package builder

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	_ "github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestSimpleDescriptorsFromScratch(t *testing.T) {
	md, err := desc.LoadMessageDescriptorForMessage((*empty.Empty)(nil))
	testutil.Ok(t, err)

	file := NewFile("foo/bar.proto").SetPackageName("foo.bar")
	en := NewEnum("Options").
		AddValue(NewEnumValue("OPTION_1")).
		AddValue(NewEnumValue("OPTION_2")).
		AddValue(NewEnumValue("OPTION_3"))
	file.AddEnum(en)

	msg := NewMessage("FooRequest").
		AddField(NewField("id", FieldTypeInt64())).
		AddField(NewField("name", FieldTypeString())).
		AddField(NewField("options", FieldTypeEnum(en)).
			SetRepeated())
	file.AddMessage(msg)

	sb := NewService("FooService").
		AddMethod(NewMethod("DoSomething", RpcTypeMessage(msg, false), RpcTypeMessage(msg, false))).
		AddMethod(NewMethod("ReturnThings", RpcTypeImportedMessage(md, false), RpcTypeMessage(msg, true)))
	file.AddService(sb)

	fd, err := file.Build()
	testutil.Ok(t, err)

	testutil.Eq(t, []*desc.FileDescriptor{md.GetFile()}, fd.GetDependencies())
	testutil.Require(t, fd.FindEnum("foo.bar.Options") != nil)
	testutil.Eq(t, 3, len(fd.FindEnum("foo.bar.Options").GetValues()))
	testutil.Require(t, fd.FindMessage("foo.bar.FooRequest") != nil)
	testutil.Eq(t, 3, len(fd.FindMessage("foo.bar.FooRequest").GetFields()))
	testutil.Require(t, fd.FindService("foo.bar.FooService") != nil)
	testutil.Eq(t, 2, len(fd.FindService("foo.bar.FooService").GetMethods()))

	// building the others produces same results
	ed, err := en.Build()
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(ed.AsProto(), fd.FindEnum("foo.bar.Options").AsProto()))

	md, err = msg.Build()
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(md.AsProto(), fd.FindMessage("foo.bar.FooRequest").AsProto()))

	sd, err := sb.Build()
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(sd.AsProto(), fd.FindService("foo.bar.FooService").AsProto()))
}

func TestSimpleDescriptorsFromScratch_SyntheticFiles(t *testing.T) {
	md, err := desc.LoadMessageDescriptorForMessage((*empty.Empty)(nil))
	testutil.Ok(t, err)

	en := NewEnum("Options")
	en.AddValue(NewEnumValue("OPTION_1"))
	en.AddValue(NewEnumValue("OPTION_2"))
	en.AddValue(NewEnumValue("OPTION_3"))

	msg := NewMessage("FooRequest")
	msg.AddField(NewField("id", FieldTypeInt64()))
	msg.AddField(NewField("name", FieldTypeString()))
	msg.AddField(NewField("options", FieldTypeEnum(en)).
		SetRepeated())

	sb := NewService("FooService")
	sb.AddMethod(NewMethod("DoSomething", RpcTypeMessage(msg, false), RpcTypeMessage(msg, false)))
	sb.AddMethod(NewMethod("ReturnThings", RpcTypeImportedMessage(md, false), RpcTypeMessage(msg, true)))

	sd, err := sb.Build()
	testutil.Ok(t, err)
	testutil.Eq(t, "FooService", sd.GetFullyQualifiedName())
	testutil.Eq(t, 2, len(sd.GetMethods()))

	// it imports google/protobuf/empty.proto and a synthetic file that has message
	testutil.Eq(t, 2, len(sd.GetFile().GetDependencies()))
	fd := sd.GetFile().GetDependencies()[0]
	testutil.Eq(t, "google/protobuf/empty.proto", fd.GetName())
	testutil.Eq(t, md.GetFile(), fd)
	fd = sd.GetFile().GetDependencies()[1]
	testutil.Require(t, strings.Contains(fd.GetName(), "generated"))
	testutil.Require(t, fd.FindMessage("FooRequest") != nil)
	testutil.Eq(t, 3, len(fd.FindMessage("FooRequest").GetFields()))

	// this one imports only a synthetic file that has enum
	testutil.Eq(t, 1, len(fd.GetDependencies()))
	fd2 := fd.GetDependencies()[0]
	testutil.Require(t, fd2.FindEnum("Options") != nil)
	testutil.Eq(t, 3, len(fd2.FindEnum("Options").GetValues()))

	// building the others produces same results
	ed, err := en.Build()
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(ed.AsProto(), fd2.FindEnum("Options").AsProto()))

	md, err = msg.Build()
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(md.AsProto(), fd.FindMessage("FooRequest").AsProto()))
}

func TestComplexDescriptorsFromScratch(t *testing.T) {
	mdEmpty, err := desc.LoadMessageDescriptorForMessage((*empty.Empty)(nil))
	testutil.Ok(t, err)
	mdAny, err := desc.LoadMessageDescriptorForMessage((*any.Any)(nil))
	testutil.Ok(t, err)
	mdTimestamp, err := desc.LoadMessageDescriptorForMessage((*timestamp.Timestamp)(nil))
	testutil.Ok(t, err)

	msgA := NewMessage("FooA").
		AddField(NewField("id", FieldTypeUInt64())).
		AddField(NewField("when", FieldTypeImportedMessage(mdTimestamp))).
		AddField(NewField("extras", FieldTypeImportedMessage(mdAny)).
			SetRepeated()).
		SetExtensionRanges([]*dpb.DescriptorProto_ExtensionRange{{Start: proto.Int32(100), End: proto.Int32(201)}})
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
		AddField(NewField("name", FieldTypeString()))
	NewFile("").
		SetPackageName("foo.bar").
		AddMessage(msgB)

	enC := NewEnum("Vals").
		AddValue(NewEnumValue("DEFAULT")).
		AddValue(NewEnumValue("VALUE_A")).
		AddValue(NewEnumValue("VALUE_B")).
		AddValue(NewEnumValue("VALUE_C"))
	msgC := NewMessage("BarBaz").
		AddOneOf(NewOneOf("bbb").
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

	testutil.Ok(t, err)

	testutil.Eq(t, 5, len(fd.GetDependencies()))
	// dependencies sorted; those with generated names come last
	depEmpty := fd.GetDependencies()[0]
	testutil.Eq(t, "google/protobuf/empty.proto", depEmpty.GetName())
	testutil.Eq(t, mdEmpty.GetFile(), depEmpty)
	depD := fd.GetDependencies()[1]
	testutil.Eq(t, "some/other/path/file.proto", depD.GetName())
	depC := fd.GetDependencies()[2]
	testutil.Eq(t, "some/path/file.proto", depC.GetName())
	depA := fd.GetDependencies()[3]
	testutil.Require(t, strings.Contains(depA.GetName(), "generated"))
	depB := fd.GetDependencies()[4]
	testutil.Require(t, strings.Contains(depB.GetName(), "generated"))

	// check contents of files
	testutil.Require(t, depA.FindMessage("foo.bar.FooA") != nil)
	testutil.Eq(t, 3, len(depA.FindMessage("foo.bar.FooA").GetFields()))
	testutil.Require(t, depA.FindMessage("foo.bar.Nnn") != nil)
	testutil.Eq(t, 2, len(depA.FindMessage("foo.bar.Nnn").GetFields()))

	testutil.Require(t, depB.FindMessage("foo.bar.FooB") != nil)
	testutil.Eq(t, 2, len(depB.FindMessage("foo.bar.FooB").GetFields()))

	testutil.Require(t, depC.FindMessage("foo.baz.BarBaz") != nil)
	testutil.Eq(t, 3, len(depC.FindMessage("foo.baz.BarBaz").GetFields()))
	testutil.Require(t, depC.FindEnum("foo.baz.Vals") != nil)
	testutil.Eq(t, 4, len(depC.FindEnum("foo.baz.Vals").GetValues()))

	testutil.Require(t, depD.FindEnum("foo.biz.Ppp") != nil)
	testutil.Eq(t, 4, len(depD.FindEnum("foo.biz.Ppp").GetValues()))
	testutil.Require(t, depD.FindExtensionByName("foo.biz.ppp") != nil)

	testutil.Require(t, fd.FindMessage("foo.bar.Ppp") != nil)
	testutil.Eq(t, 2, len(fd.FindMessage("foo.bar.Ppp").GetFields()))
	testutil.Require(t, fd.FindService("foo.bar.PppSvc") != nil)
	testutil.Eq(t, 2, len(fd.FindService("foo.bar.PppSvc").GetMethods()))
}

func TestCreatingGroupField(t *testing.T) {
	grpMb := NewMessage("GroupA").
		AddField(NewField("name", FieldTypeString())).
		AddField(NewField("id", FieldTypeInt64()))
	grpFlb := NewGroupField(grpMb)

	mb := NewMessage("TestMessage").
		AddField(NewField("foo", FieldTypeBool())).
		AddField(grpFlb)
	md, err := mb.Build()
	testutil.Ok(t, err)

	testutil.Require(t, md.FindFieldByName("groupa") != nil)
	testutil.Eq(t, dpb.FieldDescriptorProto_TYPE_GROUP, md.FindFieldByName("groupa").GetType())
	nmd := md.GetNestedMessageTypes()[0]
	testutil.Eq(t, "GroupA", nmd.GetName())
	testutil.Eq(t, nmd, md.FindFieldByName("groupa").GetMessageType())

	// try a rename that will fail
	err = grpMb.TrySetName("fooBarBaz")
	testutil.Require(t, err != nil)
	testutil.Eq(t, "group name fooBarBaz must start with capital letter", err.Error())
	// failed rename should not have modified any state
	md2, err := mb.Build()
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(md.AsProto(), md2.AsProto()))
	// another attempt that will fail
	err = grpFlb.TrySetName("foobarbaz")
	testutil.Require(t, err != nil)
	testutil.Eq(t, "cannot change name of group field TestMessage.groupa; change name of group instead", err.Error())
	// again, no state should have been modified
	md2, err = mb.Build()
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(md.AsProto(), md2.AsProto()))

	// and a rename that succeeds
	err = grpMb.TrySetName("FooBarBaz")
	testutil.Ok(t, err)
	md, err = mb.Build()
	testutil.Ok(t, err)

	// field also renamed
	testutil.Require(t, md.FindFieldByName("foobarbaz") != nil)
	testutil.Eq(t, dpb.FieldDescriptorProto_TYPE_GROUP, md.FindFieldByName("foobarbaz").GetType())
	nmd = md.GetNestedMessageTypes()[0]
	testutil.Eq(t, "FooBarBaz", nmd.GetName())
	testutil.Eq(t, nmd, md.FindFieldByName("foobarbaz").GetMessageType())
}

func TestCreatingMapField(t *testing.T) {
	mapFlb := NewMapField("countsByName", FieldTypeString(), FieldTypeUInt64())
	testutil.Require(t, mapFlb.IsMap())

	mb := NewMessage("TestMessage").
		AddField(NewField("foo", FieldTypeBool())).
		AddField(mapFlb)
	md, err := mb.Build()
	testutil.Ok(t, err)

	testutil.Require(t, md.FindFieldByName("countsByName") != nil)
	testutil.Require(t, md.FindFieldByName("countsByName").IsMap())
	nmd := md.GetNestedMessageTypes()[0]
	testutil.Eq(t, "CountsByNameEntry", nmd.GetName())
	testutil.Eq(t, nmd, md.FindFieldByName("countsByName").GetMessageType())

	// try a rename that will fail
	err = mapFlb.GetType().localMsgType.TrySetName("fooBarBaz")
	testutil.Require(t, err != nil)
	testutil.Eq(t, "cannot change name of map entry TestMessage.CountsByNameEntry; change name of field instead", err.Error())
	// failed rename should not have modified any state
	md2, err := mb.Build()
	testutil.Ok(t, err)
	testutil.Require(t, proto.Equal(md.AsProto(), md2.AsProto()))

	// and a rename that succeeds
	err = mapFlb.TrySetName("fooBarBaz")
	testutil.Ok(t, err)
	md, err = mb.Build()
	testutil.Ok(t, err)

	// map entry also renamed
	testutil.Require(t, md.FindFieldByName("fooBarBaz") != nil)
	testutil.Require(t, md.FindFieldByName("fooBarBaz").IsMap())
	nmd = md.GetNestedMessageTypes()[0]
	testutil.Eq(t, "FooBarBazEntry", nmd.GetName())
	testutil.Eq(t, nmd, md.FindFieldByName("fooBarBaz").GetMessageType())
}

func TestBuildersFromDescriptors(t *testing.T) {
	for _, s := range []string{"desc_test1.proto", "desc_test2.proto", "desc_test_defaults.proto", "desc_test_options.proto", "desc_test_proto3.proto", "desc_test_wellknowntypes.proto", "nopkg/desc_test_nopkg.proto", "nopkg/desc_test_nopkg_new.proto", "pkg/desc_test_pkg.proto"} {
		fd, err := desc.LoadFileDescriptor(s)
		testutil.Ok(t, err)
		roundTripFile(t, fd)
	}
}

func TestBuildersFromDescriptors_PreserveComments(t *testing.T) {
	fd, err := loadProtoset("../../internal/testprotos/desc_test1.protoset")
	testutil.Ok(t, err)

	fb, err := FromFile(fd)
	testutil.Ok(t, err)

	count := 0
	var checkBuilderComments func(b Builder)
	checkBuilderComments = func(b Builder) {
		hasComment := true
		switch b := b.(type) {
		case *FileBuilder:
			hasComment = false
		case *FieldBuilder:
			// comments for groups are on the message, not the field
			hasComment = b.GetType().GetType() != dpb.FieldDescriptorProto_TYPE_GROUP
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
			testutil.Eq(t, fmt.Sprintf(" Comment for %s\n", b.GetName()), b.GetComments().LeadingComment,
				"wrong comment for builder %s", GetFullyQualifiedName(b))
		}
		for _, ch := range b.GetChildren() {
			checkBuilderComments(ch)
		}
	}

	checkBuilderComments(fb)
	// sanity check that we didn't accidentally short-circuit above and fail to check comments
	testutil.Require(t, count > 30, "too few elements checked")

	// now check that they also come out in the resulting descriptor
	fd, err = fb.Build()
	testutil.Ok(t, err)

	descCount := 0
	var checkDescriptorComments func(d desc.Descriptor)
	checkDescriptorComments = func(d desc.Descriptor) {
		switch d := d.(type) {
		case *desc.FileDescriptor:
			for _, ch := range d.GetMessageTypes() {
				checkDescriptorComments(ch)
			}
			for _, ch := range d.GetEnumTypes() {
				checkDescriptorComments(ch)
			}
			for _, ch := range d.GetExtensions() {
				checkDescriptorComments(ch)
			}
			for _, ch := range d.GetServices() {
				checkDescriptorComments(ch)
			}
			// files don't have comments, so bail out before check below
			return
		case *desc.MessageDescriptor:
			if d.IsMapEntry() {
				// map entry messages have no comments (and neither do their child fields)
				return
			}
			for _, ch := range d.GetFields() {
				checkDescriptorComments(ch)
			}
			for _, ch := range d.GetNestedMessageTypes() {
				checkDescriptorComments(ch)
			}
			for _, ch := range d.GetNestedEnumTypes() {
				checkDescriptorComments(ch)
			}
			for _, ch := range d.GetNestedExtensions() {
				checkDescriptorComments(ch)
			}
			for _, ch := range d.GetOneOfs() {
				checkDescriptorComments(ch)
			}
		case *desc.FieldDescriptor:
			if d.GetType() == dpb.FieldDescriptorProto_TYPE_GROUP {
				// groups comments are on the message, not hte field; so bail out before check below
				return
			}
		case *desc.EnumDescriptor:
			for _, ch := range d.GetValues() {
				checkDescriptorComments(ch)
			}
		case *desc.ServiceDescriptor:
			for _, ch := range d.GetMethods() {
				checkDescriptorComments(ch)
			}
		}

		descCount++
		testutil.Eq(t, fmt.Sprintf(" Comment for %s\n", d.GetName()), d.GetSourceInfo().GetLeadingComments(),
			"wrong comment for descriptor %s", d.GetFullyQualifiedName())
	}

	checkDescriptorComments(fd)
	testutil.Eq(t, count, descCount)
}

func loadProtoset(path string) (*desc.FileDescriptor, error) {
	var fds dpb.FileDescriptorSet
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bb, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(bb, &fds); err != nil {
		return nil, err
	}
	return desc.CreateFileDescriptorFromSet(&fds)
}

func roundTripFile(t *testing.T, fd *desc.FileDescriptor) {
	// First, recursively verify that every child element can be converted to a
	// Builder and back without loss of fidelity.
	for _, md := range fd.GetMessageTypes() {
		roundTripMessage(t, md)
	}
	for _, ed := range fd.GetEnumTypes() {
		roundTripEnum(t, ed)
	}
	for _, exd := range fd.GetExtensions() {
		roundTripField(t, exd)
	}
	for _, sd := range fd.GetServices() {
		roundTripService(t, sd)
	}

	// Finally, we check the whole file itself.
	fb, err := FromFile(fd)
	testutil.Ok(t, err)

	roundTripped, err := fb.Build()
	testutil.Ok(t, err)

	// Round tripping from a file descriptor to a builder and back will
	// experience some minor changes (that do not impact the semantics of
	// any of the file's contents):
	//  1. The builder sorts dependencies. However the original file
	//     descriptor has dependencies in the order they appear in import
	//     statements in the source file.
	//  2. The builder imports the actual source of all elements and never
	//     uses public imports. The original file, on the other hand, could
	//     used public imports and "indirectly" import other files that way.
	//  3. The builder never emits weak imports.
	//  4. The builder tries to preserve SourceCodeInfo, but will not preserve
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
	fdp := fd.AsFileDescriptorProto()
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

	// Remove source code info that the builder generated since the original
	// has none.
	roundTripped.AsFileDescriptorProto().SourceCodeInfo = nil

	// Finally, sort the imports. That way they match the built result (which
	// is always sorted).
	sort.Strings(fdp.Dependency)

	// Now (after tweaking) the original should match the round-tripped descriptor:
	testutil.Require(t, proto.Equal(fdp, roundTripped.AsProto()), "File %q failed round trip.\nExpecting: %s\nGot: %s\n",
		fd.GetName(), proto.MarshalTextString(fdp), proto.MarshalTextString(roundTripped.AsProto()))
}

func roundTripMessage(t *testing.T, md *desc.MessageDescriptor) {
	// first recursively validate all nested elements
	for _, fld := range md.GetFields() {
		roundTripField(t, fld)
	}
	for _, ood := range md.GetOneOfs() {
		oob, err := FromOneOf(ood)
		testutil.Ok(t, err)
		roundTripped, err := oob.Build()
		testutil.Ok(t, err)
		checkDescriptors(t, ood, roundTripped)
	}
	for _, nmd := range md.GetNestedMessageTypes() {
		roundTripMessage(t, nmd)
	}
	for _, ed := range md.GetNestedEnumTypes() {
		roundTripEnum(t, ed)
	}
	for _, exd := range md.GetNestedExtensions() {
		roundTripField(t, exd)
	}

	mb, err := FromMessage(md)
	testutil.Ok(t, err)
	roundTripped, err := mb.Build()
	testutil.Ok(t, err)
	checkDescriptors(t, md, roundTripped)
}

func roundTripEnum(t *testing.T, ed *desc.EnumDescriptor) {
	// first recursively validate all nested elements
	for _, evd := range ed.GetValues() {
		evb, err := FromEnumValue(evd)
		testutil.Ok(t, err)
		roundTripped, err := evb.Build()
		testutil.Ok(t, err)
		checkDescriptors(t, evd, roundTripped)
	}

	eb, err := FromEnum(ed)
	testutil.Ok(t, err)
	roundTripped, err := eb.Build()
	testutil.Ok(t, err)
	checkDescriptors(t, ed, roundTripped)
}

func roundTripField(t *testing.T, fld *desc.FieldDescriptor) {
	flb, err := FromField(fld)
	testutil.Ok(t, err)
	roundTripped, err := flb.Build()
	testutil.Ok(t, err)
	checkDescriptors(t, fld, roundTripped)
}

func roundTripService(t *testing.T, sd *desc.ServiceDescriptor) {
	// first recursively validate all nested elements
	for _, mtd := range sd.GetMethods() {
		mtb, err := FromMethod(mtd)
		testutil.Ok(t, err)
		roundTripped, err := mtb.Build()
		testutil.Ok(t, err)
		checkDescriptors(t, mtd, roundTripped)
	}

	sb, err := FromService(sd)
	testutil.Ok(t, err)
	roundTripped, err := sb.Build()
	testutil.Ok(t, err)
	checkDescriptors(t, sd, roundTripped)
}

func checkDescriptors(t *testing.T, d1, d2 desc.Descriptor) {
	testutil.Eq(t, d1.GetFullyQualifiedName(), d2.GetFullyQualifiedName())
	testutil.Require(t, proto.Equal(d1.AsProto(), d2.AsProto()), "%s failed round trip.\nExpecting: %s\nGot: %s\n",
		d1.GetFullyQualifiedName(), proto.MarshalTextString(d1.AsProto()), proto.MarshalTextString(d2.AsProto()))
}

func TestAddRemoveMoveBuilders(t *testing.T) {
	// add field to one-of
	fld1 := NewField("foo", FieldTypeInt32())
	oo1 := NewOneOf("oofoo")
	oo1.AddChoice(fld1)
	checkChildren(t, oo1, fld1)
	testutil.Eq(t, oo1.GetChoice("foo"), fld1)

	// add one-of w/ field to a message
	msg1 := NewMessage("foo")
	msg1.AddOneOf(oo1)
	checkChildren(t, msg1, oo1)
	testutil.Eq(t, msg1.GetOneOf("oofoo"), oo1)
	// field remains unchanged
	testutil.Eq(t, fld1.GetParent(), oo1)
	testutil.Eq(t, oo1.GetChoice("foo"), fld1)
	// field also now registered with msg1
	testutil.Eq(t, msg1.GetField("foo"), fld1)

	// add empty one-of to message
	oo2 := NewOneOf("oobar")
	msg1.AddOneOf(oo2)
	checkChildren(t, msg1, oo1, oo2)
	testutil.Eq(t, msg1.GetOneOf("oobar"), oo2)
	// now add field to that one-of
	fld2 := NewField("bar", FieldTypeInt32())
	oo2.AddChoice(fld2)
	checkChildren(t, oo2, fld2)
	testutil.Eq(t, oo2.GetChoice("bar"), fld2)
	// field also now registered with msg1
	testutil.Eq(t, msg1.GetField("bar"), fld2)

	// add fails due to name collisions
	fld1 = NewField("foo", FieldTypeInt32())
	err := oo1.TryAddChoice(fld1)
	checkFailedAdd(t, err, oo1, fld1, "already contains field")
	fld2 = NewField("bar", FieldTypeInt32())
	err = msg1.TryAddField(fld2)
	checkFailedAdd(t, err, msg1, fld2, "already contains element")
	msg2 := NewMessage("oofoo")
	// name collision can be different type
	// (here, nested message conflicts with a one-of)
	err = msg1.TryAddNestedMessage(msg2)
	checkFailedAdd(t, err, msg1, msg2, "already contains element")

	msg2 = NewMessage("baz")
	msg1.AddNestedMessage(msg2)
	checkChildren(t, msg1, oo1, oo2, msg2)
	testutil.Eq(t, msg1.GetNestedMessage("baz"), msg2)

	// can't add extension, group, or map fields to one-of
	ext1 := NewExtension("abc", 123, FieldTypeInt32(), msg1)
	err = oo1.TryAddChoice(ext1)
	checkFailedAdd(t, err, oo1, ext1, "is an extension, not a regular field")
	err = msg1.TryAddField(ext1)
	checkFailedAdd(t, err, msg1, ext1, "is an extension, not a regular field")
	mapField := NewMapField("abc", FieldTypeInt32(), FieldTypeString())
	err = oo1.TryAddChoice(mapField)
	checkFailedAdd(t, err, oo1, mapField, "cannot add a group or map field")
	groupMsg := NewMessage("Group")
	groupField := NewGroupField(groupMsg)
	err = oo1.TryAddChoice(groupField)
	checkFailedAdd(t, err, oo1, groupField, "cannot add a group or map field")
	// adding map and group to msg succeeds
	msg1.AddField(groupField)
	msg1.AddField(mapField)
	checkChildren(t, msg1, oo1, oo2, msg2, groupField, mapField)
	// messages associated with map and group fields are not children of the
	// message, but are in its scope and accessible via GetNestedMessage
	testutil.Eq(t, msg1.GetNestedMessage("Group"), groupMsg)
	testutil.Eq(t, msg1.GetNestedMessage("AbcEntry"), mapField.GetType().localMsgType)

	// adding extension to message
	ext2 := NewExtension("xyz", 234, FieldTypeInt32(), msg1)
	msg1.AddNestedExtension(ext2)
	checkChildren(t, msg1, oo1, oo2, msg2, groupField, mapField, ext2)
	err = msg1.TryAddNestedExtension(ext1) // name collision
	checkFailedAdd(t, err, msg1, ext1, "already contains element")
	fld3 := NewField("ijk", FieldTypeString())
	err = msg1.TryAddNestedExtension(fld3)
	checkFailedAdd(t, err, msg1, fld3, "is not an extension")

	// add enum values to enum
	enumVal1 := NewEnumValue("A")
	enum1 := NewEnum("bazel")
	enum1.AddValue(enumVal1)
	checkChildren(t, enum1, enumVal1)
	testutil.Eq(t, enum1.GetValue("A"), enumVal1)
	enumVal2 := NewEnumValue("B")
	enum1.AddValue(enumVal2)
	checkChildren(t, enum1, enumVal1, enumVal2)
	testutil.Eq(t, enum1.GetValue("B"), enumVal2)
	// fail w/ name collision
	enumVal3 := NewEnumValue("B")
	err = enum1.TryAddValue(enumVal3)
	checkFailedAdd(t, err, enum1, enumVal3, "already contains value")

	msg2.AddNestedEnum(enum1)
	checkChildren(t, msg2, enum1)
	testutil.Eq(t, msg2.GetNestedEnum("bazel"), enum1)
	ext3 := NewExtension("bazel", 987, FieldTypeString(), msg2)
	err = msg2.TryAddNestedExtension(ext3)
	checkFailedAdd(t, err, msg2, ext3, "already contains element")

	// services and methods
	mtd1 := NewMethod("foo", RpcTypeMessage(msg1, false), RpcTypeMessage(msg1, false))
	svc1 := NewService("FooService")
	svc1.AddMethod(mtd1)
	checkChildren(t, svc1, mtd1)
	testutil.Eq(t, svc1.GetMethod("foo"), mtd1)
	mtd2 := NewMethod("foo", RpcTypeMessage(msg1, false), RpcTypeMessage(msg1, false))
	err = svc1.TryAddMethod(mtd2)
	checkFailedAdd(t, err, svc1, mtd2, "already contains method")

	// finally, test adding things to  a file
	fb := NewFile("")
	fb.AddMessage(msg1)
	checkChildren(t, fb, msg1)
	testutil.Eq(t, fb.GetMessage("foo"), msg1)
	fb.AddService(svc1)
	checkChildren(t, fb, msg1, svc1)
	testutil.Eq(t, fb.GetService("FooService"), svc1)
	enum2 := NewEnum("fizzle")
	fb.AddEnum(enum2)
	checkChildren(t, fb, msg1, svc1, enum2)
	testutil.Eq(t, fb.GetEnum("fizzle"), enum2)
	ext3 = NewExtension("foosball", 123, FieldTypeInt32(), msg1)
	fb.AddExtension(ext3)
	checkChildren(t, fb, msg1, svc1, enum2, ext3)
	testutil.Eq(t, fb.GetExtension("foosball"), ext3)

	// errors and name collisions
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
	testutil.Eq(t, len(children), len(parent.GetChildren()), "Wrong number of children for %s (%T)", GetFullyQualifiedName(parent), parent)
	ch := map[Builder]struct{}{}
	for _, child := range children {
		testutil.Eq(t, child.GetParent(), parent, "Child %s (%T) does not report %s (%T) as its parent", child.GetName(), child, GetFullyQualifiedName(parent), parent)
		ch[child] = struct{}{}
	}
	for _, child := range parent.GetChildren() {
		_, ok := ch[child]
		testutil.Require(t, ok, "Child %s (%T) does appear in list of children for %s (%T)", child.GetName(), child, GetFullyQualifiedName(parent), parent)
	}
}

func checkFailedAdd(t *testing.T, err error, parent Builder, child Builder, errorMsg string) {
	testutil.Require(t, err != nil, "Expecting error assigning %s (%T) to %s (%T)", child.GetName(), child, GetFullyQualifiedName(parent), parent)
	testutil.Require(t, strings.Contains(err.Error(), errorMsg), "Expecting error assigning %s (%T) to %s (%T) to contain text %q: %q", child.GetName(), child, GetFullyQualifiedName(parent), parent, errorMsg, err.Error())
	testutil.Eq(t, nil, child.GetParent(), "Child %s (%T) should not have a parent after failed add", child.GetName(), child)
	for _, ch := range parent.GetChildren() {
		testutil.Require(t, ch != child, "Child %s (%T) should not appear in list of children for %s (%T) but does", child.GetName(), child, GetFullyQualifiedName(parent), parent)
	}
}

func TestRenamingBuilders(t *testing.T) {
	// TODO
}

func TestRenumberingFields(t *testing.T) {
	// TODO
}

var (
	fileOptionsDesc, msgOptionsDesc, fieldOptionsDesc, oneofOptionsDesc, extRangeOptionsDesc,
	enumOptionsDesc, enumValOptionsDesc, svcOptionsDesc, mtdOptionsDesc *desc.MessageDescriptor
)

func init() {
	var err error
	fileOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.FileOptions)(nil))
	if err != nil {
		panic(err)
	}
	msgOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.MessageOptions)(nil))
	if err != nil {
		panic(err)
	}
	fieldOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.FieldOptions)(nil))
	if err != nil {
		panic(err)
	}
	oneofOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.OneofOptions)(nil))
	if err != nil {
		panic(err)
	}
	extRangeOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.ExtensionRangeOptions)(nil))
	if err != nil {
		panic(err)
	}
	enumOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.EnumOptions)(nil))
	if err != nil {
		panic(err)
	}
	enumValOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.EnumValueOptions)(nil))
	if err != nil {
		panic(err)
	}
	svcOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.ServiceOptions)(nil))
	if err != nil {
		panic(err)
	}
	mtdOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*dpb.MethodOptions)(nil))
	if err != nil {
		panic(err)
	}
}

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
		fb.Options = &dpb.FileOptions{}
		ext, err := fileOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(fb.Options, ext, "fubar")
		testutil.Ok(t, err)
		checkBuildWithLocalExtensions(t, fb)
	})

	t.Run("message options", func(t *testing.T) {
		mb := NewMessage("Foo")
		mb.Options = &dpb.MessageOptions{}
		ext, err := msgOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(mb.Options, ext, "fubar")
		testutil.Ok(t, err)

		fb := clone(t, file)
		fb.AddMessage(mb)
		checkBuildWithLocalExtensions(t, mb)
	})

	t.Run("field options", func(t *testing.T) {
		flb := NewField("foo", FieldTypeString())
		flb.Options = &dpb.FieldOptions{}
		// fields must be connected to a message
		mb := NewMessage("Foo").AddField(flb)
		ext, err := fieldOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(flb.Options, ext, "fubar")
		testutil.Ok(t, err)

		fb := clone(t, file)
		fb.AddMessage(mb)
		checkBuildWithLocalExtensions(t, flb)
	})

	t.Run("oneof options", func(t *testing.T) {
		oob := NewOneOf("oo")
		oob.Options = &dpb.OneofOptions{}
		// oneofs must be connected to a message
		mb := NewMessage("Foo").AddOneOf(oob)
		ext, err := oneofOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(oob.Options, ext, "fubar")
		testutil.Ok(t, err)

		fb := clone(t, file)
		fb.AddMessage(mb)
		checkBuildWithLocalExtensions(t, oob)
	})

	t.Run("extension range options", func(t *testing.T) {
		var erOpts dpb.ExtensionRangeOptions
		ext, err := extRangeOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(&erOpts, ext, "fubar")
		testutil.Ok(t, err)
		mb := NewMessage("foo").AddExtensionRangeWithOptions(100, 200, &erOpts)

		fb := clone(t, file)
		fb.AddMessage(mb)
		checkBuildWithLocalExtensions(t, mb)
	})

	t.Run("enum options", func(t *testing.T) {
		eb := NewEnum("Foo")
		eb.Options = &dpb.EnumOptions{}
		ext, err := enumOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(eb.Options, ext, "fubar")
		testutil.Ok(t, err)

		fb := clone(t, file)
		fb.AddEnum(eb)
		checkBuildWithLocalExtensions(t, eb)
	})

	t.Run("enum val options", func(t *testing.T) {
		evb := NewEnumValue("FOO")
		// enum values must be connected to an enum
		eb := NewEnum("Foo").AddValue(evb)
		evb.Options = &dpb.EnumValueOptions{}
		ext, err := enumValOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(evb.Options, ext, "fubar")
		testutil.Ok(t, err)

		fb := clone(t, file)
		fb.AddEnum(eb)
		checkBuildWithLocalExtensions(t, evb)
	})

	t.Run("service options", func(t *testing.T) {
		sb := NewService("Foo")
		sb.Options = &dpb.ServiceOptions{}
		ext, err := svcOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(sb.Options, ext, "fubar")
		testutil.Ok(t, err)

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
		mtb.Options = &dpb.MethodOptions{}
		ext, err := mtdOpt.Build()
		testutil.Ok(t, err)
		err = dynamic.SetExtension(mtb.Options, ext, "fubar")
		testutil.Ok(t, err)

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
	testutil.Ok(t, err)
	// since they are defined locally, no extra imports
	testutil.Eq(t, []string{"google/protobuf/descriptor.proto"}, d.GetFile().AsFileDescriptorProto().GetDependency())
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
	testutil.Ok(t, err)

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
				fb.Options = &dpb.FileOptions{}
				ext, err := fileOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(fb.Options, ext, "fubar")
				testutil.Ok(t, err)
				checkBuildWithImportedExtensions(t, fb)
			})

			t.Run("message options", func(t *testing.T) {
				mb := NewMessage("Foo")
				mb.Options = &dpb.MessageOptions{}
				ext, err := msgOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(mb.Options, ext, "fubar")
				testutil.Ok(t, err)

				fb := newFile()
				fb.AddMessage(mb)
				checkBuildWithImportedExtensions(t, mb)
			})

			t.Run("field options", func(t *testing.T) {
				flb := NewField("foo", FieldTypeString())
				flb.Options = &dpb.FieldOptions{}
				// fields must be connected to a message
				mb := NewMessage("Foo").AddField(flb)
				ext, err := fieldOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(flb.Options, ext, "fubar")
				testutil.Ok(t, err)

				fb := newFile()
				fb.AddMessage(mb)
				checkBuildWithImportedExtensions(t, flb)
			})

			t.Run("oneof options", func(t *testing.T) {
				oob := NewOneOf("oo")
				oob.Options = &dpb.OneofOptions{}
				// oneofs must be connected to a message
				mb := NewMessage("Foo").AddOneOf(oob)
				ext, err := oneofOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(oob.Options, ext, "fubar")
				testutil.Ok(t, err)

				fb := newFile()
				fb.AddMessage(mb)
				checkBuildWithImportedExtensions(t, oob)
			})

			t.Run("extension range options", func(t *testing.T) {
				var erOpts dpb.ExtensionRangeOptions
				ext, err := extRangeOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(&erOpts, ext, "fubar")
				testutil.Ok(t, err)
				mb := NewMessage("foo").AddExtensionRangeWithOptions(100, 200, &erOpts)

				fb := newFile()
				fb.AddMessage(mb)
				checkBuildWithImportedExtensions(t, mb)
			})

			t.Run("enum options", func(t *testing.T) {
				eb := NewEnum("Foo")
				eb.Options = &dpb.EnumOptions{}
				ext, err := enumOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(eb.Options, ext, "fubar")
				testutil.Ok(t, err)

				fb := newFile()
				fb.AddEnum(eb)
				checkBuildWithImportedExtensions(t, eb)
			})

			t.Run("enum val options", func(t *testing.T) {
				evb := NewEnumValue("FOO")
				// enum values must be connected to an enum
				eb := NewEnum("Foo").AddValue(evb)
				evb.Options = &dpb.EnumValueOptions{}
				ext, err := enumValOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(evb.Options, ext, "fubar")
				testutil.Ok(t, err)

				fb := newFile()
				fb.AddEnum(eb)
				checkBuildWithImportedExtensions(t, evb)
			})

			t.Run("service options", func(t *testing.T) {
				sb := NewService("Foo")
				sb.Options = &dpb.ServiceOptions{}
				ext, err := svcOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(sb.Options, ext, "fubar")
				testutil.Ok(t, err)

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
				mtb.Options = &dpb.MethodOptions{}
				ext, err := mtdOpt.Build()
				testutil.Ok(t, err)
				err = dynamic.SetExtension(mtb.Options, ext, "fubar")
				testutil.Ok(t, err)

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
	testutil.Ok(t, err)
	// the only import is for the custom options
	testutil.Eq(t, []string{"options.proto"}, d.GetFile().AsFileDescriptorProto().GetDependency())
}

func TestUseOfExtensionRegistry(t *testing.T) {
	// Add option for every type to extension registry
	var exts dynamic.ExtensionRegistry

	fileOpt, err := NewExtensionImported("file_foo", 54321, FieldTypeString(), fileOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(fileOpt)
	testutil.Ok(t, err)

	msgOpt, err := NewExtensionImported("msg_foo", 54321, FieldTypeString(), msgOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(msgOpt)
	testutil.Ok(t, err)

	fieldOpt, err := NewExtensionImported("field_foo", 54321, FieldTypeString(), fieldOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(fieldOpt)
	testutil.Ok(t, err)

	oneofOpt, err := NewExtensionImported("oneof_foo", 54321, FieldTypeString(), oneofOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(oneofOpt)
	testutil.Ok(t, err)

	extRangeOpt, err := NewExtensionImported("ext_range_foo", 54321, FieldTypeString(), extRangeOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(extRangeOpt)
	testutil.Ok(t, err)

	enumOpt, err := NewExtensionImported("enum_foo", 54321, FieldTypeString(), enumOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(enumOpt)
	testutil.Ok(t, err)

	enumValOpt, err := NewExtensionImported("enum_val_foo", 54321, FieldTypeString(), enumValOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(enumValOpt)
	testutil.Ok(t, err)

	svcOpt, err := NewExtensionImported("svc_foo", 54321, FieldTypeString(), svcOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(svcOpt)
	testutil.Ok(t, err)

	mtdOpt, err := NewExtensionImported("mtd_foo", 54321, FieldTypeString(), mtdOptionsDesc).Build()
	testutil.Ok(t, err)
	err = exts.AddExtension(mtdOpt)
	testutil.Ok(t, err)

	// Now we can test referring to these and making sure they show up correctly
	// in built descriptors

	t.Run("file options", func(t *testing.T) {
		fb := NewFile("foo.proto")
		fb.Options = &dpb.FileOptions{}
		err = dynamic.SetExtension(fb.Options, fileOpt, "fubar")
		testutil.Ok(t, err)
		checkBuildWithExtensions(t, &exts, fileOpt.GetFile(), fb)
	})

	t.Run("message options", func(t *testing.T) {
		mb := NewMessage("Foo")
		mb.Options = &dpb.MessageOptions{}
		err = dynamic.SetExtension(mb.Options, msgOpt, "fubar")
		testutil.Ok(t, err)
		checkBuildWithExtensions(t, &exts, msgOpt.GetFile(), mb)
	})

	t.Run("field options", func(t *testing.T) {
		flb := NewField("foo", FieldTypeString())
		flb.Options = &dpb.FieldOptions{}
		// fields must be connected to a message
		NewMessage("Foo").AddField(flb)
		err = dynamic.SetExtension(flb.Options, fieldOpt, "fubar")
		testutil.Ok(t, err)
		checkBuildWithExtensions(t, &exts, fieldOpt.GetFile(), flb)
	})

	t.Run("oneof options", func(t *testing.T) {
		oob := NewOneOf("oo")
		oob.Options = &dpb.OneofOptions{}
		// oneofs must be connected to a message
		NewMessage("Foo").AddOneOf(oob)
		err = dynamic.SetExtension(oob.Options, oneofOpt, "fubar")
		testutil.Ok(t, err)
		checkBuildWithExtensions(t, &exts, oneofOpt.GetFile(), oob)
	})

	t.Run("extension range options", func(t *testing.T) {
		var erOpts dpb.ExtensionRangeOptions
		err = dynamic.SetExtension(&erOpts, extRangeOpt, "fubar")
		testutil.Ok(t, err)
		mb := NewMessage("foo").AddExtensionRangeWithOptions(100, 200, &erOpts)
		checkBuildWithExtensions(t, &exts, extRangeOpt.GetFile(), mb)
	})

	t.Run("enum options", func(t *testing.T) {
		eb := NewEnum("Foo")
		eb.Options = &dpb.EnumOptions{}
		err = dynamic.SetExtension(eb.Options, enumOpt, "fubar")
		testutil.Ok(t, err)
		checkBuildWithExtensions(t, &exts, enumOpt.GetFile(), eb)
	})

	t.Run("enum val options", func(t *testing.T) {
		evb := NewEnumValue("FOO")
		// enum values must be connected to an enum
		NewEnum("Foo").AddValue(evb)
		evb.Options = &dpb.EnumValueOptions{}
		err = dynamic.SetExtension(evb.Options, enumValOpt, "fubar")
		testutil.Ok(t, err)
		checkBuildWithExtensions(t, &exts, enumValOpt.GetFile(), evb)
	})

	t.Run("service options", func(t *testing.T) {
		sb := NewService("Foo")
		sb.Options = &dpb.ServiceOptions{}
		err = dynamic.SetExtension(sb.Options, svcOpt, "fubar")
		testutil.Ok(t, err)
		checkBuildWithExtensions(t, &exts, svcOpt.GetFile(), sb)
	})

	t.Run("method options", func(t *testing.T) {
		mtb := NewMethod("Foo",
			RpcTypeMessage(NewMessage("Request"), false),
			RpcTypeMessage(NewMessage("Response"), false))
		// methods must be connected to a service
		NewService("Bar").AddMethod(mtb)
		mtb.Options = &dpb.MethodOptions{}
		err = dynamic.SetExtension(mtb.Options, mtdOpt, "fubar")
		testutil.Ok(t, err)
		checkBuildWithExtensions(t, &exts, mtdOpt.GetFile(), mtb)
	})
}

func checkBuildWithExtensions(t *testing.T, exts *dynamic.ExtensionRegistry, expected *desc.FileDescriptor, builder Builder) {
	// without interpreting custom option
	d, err := builder.BuildDescriptor()
	testutil.Ok(t, err)
	for _, dep := range d.GetFile().GetDependencies() {
		testutil.Neq(t, expected, dep)
	}
	numDeps := len(d.GetFile().GetDependencies())

	// requiring options (and failing)
	var opts BuilderOptions
	opts.RequireInterpretedOptions = true
	_, err = opts.Build(builder)
	testutil.Require(t, err != nil)

	// able to interpret options via extension registry
	opts.Extensions = exts
	d, err = opts.Build(builder)
	testutil.Ok(t, err)
	testutil.Eq(t, numDeps+1, len(d.GetFile().GetDependencies()))
	found := false
	for _, dep := range d.GetFile().GetDependencies() {
		if expected == dep {
			found = true
			break
		}
	}
	testutil.Require(t, found)
}

func TestRemoveField(t *testing.T) {
	msg := NewMessage("FancyMessage").
		AddField(NewField("one", FieldTypeInt64())).
		AddField(NewField("two", FieldTypeString())).
		AddField(NewField("three", FieldTypeString()))

	ok := msg.TryRemoveField("two")
	children := msg.GetChildren()

	testutil.Require(t, ok)
	testutil.Eq(t, 2, len(children))
	testutil.Eq(t, "one", children[0].GetName())
	testutil.Eq(t, "three", children[1].GetName())
}

func clone(t *testing.T, fb *FileBuilder) *FileBuilder {
	fd, err := fb.Build()
	testutil.Ok(t, err)
	fb, err = FromFile(fd)
	testutil.Ok(t, err)
	return fb
}
