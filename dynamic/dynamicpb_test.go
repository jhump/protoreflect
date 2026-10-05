package dynamic_test

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	protov2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/jhump/protoreflect/codec"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/internal/testutil"
)

const bridgeProto = `
syntax = "proto2";
package bridge;
import "google/protobuf/any.proto";
message Node {
  optional string name = 1;
  optional Node child = 2;
  repeated Node children = 3;
  map<string, Node> by_name = 4;
  oneof choice {
    int32 num = 5;
    Node other = 6;
  }
  optional group Grp = 7 {
    optional string val = 8;
  }
  optional google.protobuf.Any any = 9;
  extensions 100 to 200;
}
extend Node {
  optional string tag = 100;
  optional Node ext_node = 101;
}
`

func TestToDynamicPB(t *testing.T) {
	fd := parseBridgeProto(t)
	md := fd.FindMessage("bridge.Node")
	msg := newBridgeMessage(t, fd)

	dpb, err := msg.ToDynamicPB()
	testutil.Ok(t, err)
	testutil.Eq(t, md.UnwrapMessage(), dpb.Descriptor())

	js, err := protojson.Marshal(dpb)
	testutil.Ok(t, err)
	// all extensions are present, including in nested messages
	testutil.Eq(t, 2, strings.Count(string(js), `"[bridge.tag]"`), "unexpected JSON: %s", js)
	testutil.Eq(t, 1, strings.Count(string(js), `"[bridge.ext_node]"`), "unexpected JSON: %s", js)
	testutil.Require(t, strings.Contains(string(js), `"val"`), "unexpected JSON: %s", js)
	// unknown fields are preserved too, even though protojson ignores them
	testutil.Require(t, len(dpb.GetUnknown()) > 0)

	// convert back
	er := dynamic.NewExtensionRegistryWithDefaults()
	er.AddExtensionsFromFile(fd)
	roundTripped := dynamic.NewMessageFactoryWithExtensionRegistry(er).NewDynamicMessage(md)
	testutil.Ok(t, roundTripped.ConvertFrom(dpb))
	testutil.Require(t, dynamic.Equal(msg, roundTripped), "%v != %v", msg, roundTripped)
}

func TestConvertToAndFromDynamicPB(t *testing.T) {
	fd := parseBridgeProto(t)
	md := fd.FindMessage("bridge.Node")
	msg := newBridgeMessage(t, fd)
	expected, err := msg.ToDynamicPB()
	testutil.Ok(t, err)

	// These used to fail when extensions were present, or to silently
	// produce empty messages otherwise.
	dpb := dynamicpb.NewMessage(md.UnwrapMessage())
	testutil.Ok(t, msg.ConvertTo(dpb))
	testutil.Require(t, protov2.Equal(expected, dpb))

	dpb = dynamicpb.NewMessage(md.UnwrapMessage())
	testutil.Ok(t, msg.MergeInto(dpb))
	testutil.Require(t, protov2.Equal(expected, dpb))

	other := dynamic.NewMessage(md)
	testutil.Ok(t, other.ConvertFrom(dpb))
	testutil.Eq(t, "root", other.GetFieldByName("name"))
	// This message's factory doesn't know about the extensions, so they are
	// unknown fields, as when unmarshaling. But they can still be accessed.
	testutil.Eq(t, "root-tag", other.GetField(fd.FindExtensionByName("bridge.tag")))

	other = dynamic.NewMessage(md)
	testutil.Ok(t, other.MergeFrom(dpb))
	testutil.Eq(t, "root", other.GetFieldByName("name"))
}

func TestToDynamicPBWithAnyAndProtojson(t *testing.T) {
	fd := parseBridgeProto(t)
	md := fd.FindMessage("bridge.Node")

	inner := dynamic.NewMessage(md)
	inner.SetFieldByName("name", "inside any")
	data, err := inner.Marshal()
	testutil.Ok(t, err)
	msg := dynamic.NewMessage(md)
	msg.SetFieldByName("any", &anypb.Any{TypeUrl: "type.googleapis.com/bridge.Node", Value: data})

	dpb, err := msg.ToDynamicPB()
	testutil.Ok(t, err)
	// as documented on ToDynamicPB, a resolver is needed for protojson to
	// find the type inside the Any
	var files protoregistry.Files
	testutil.Ok(t, files.RegisterFile(fd.UnwrapFile()))
	js, err := protojson.MarshalOptions{Resolver: dynamicpb.NewTypes(&files)}.Marshal(dpb)
	testutil.Ok(t, err)
	testutil.Require(t, strings.Contains(string(js), `"inside any"`), "unexpected JSON: %s", js)
}

func TestToDynamicPBRecursionLimit(t *testing.T) {
	fd := parseBridgeProto(t)
	md := fd.FindMessage("bridge.Node")

	cyclic := dynamic.NewMessage(md)
	cyclic.SetFieldByName("child", cyclic)
	_, err := cyclic.ToDynamicPB()
	testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)
}

func parseBridgeProto(t *testing.T) *desc.FileDescriptor {
	t.Helper()
	parser := protoparse.Parser{
		Accessor: protoparse.FileContentsFromMap(map[string]string{"bridge.proto": bridgeProto}),
	}
	fds, err := parser.ParseFiles("bridge.proto")
	testutil.Ok(t, err)
	_, err = protoregistry.GlobalFiles.FindFileByPath("bridge.proto")
	testutil.Require(t, err != nil, "bridge.proto should not be in the global registry")
	return fds[0]
}

// newBridgeMessage returns a bridge.Node that uses every kind of field,
// including extensions in nested messages and an unknown field.
func newBridgeMessage(t *testing.T, fd *desc.FileDescriptor) *dynamic.Message {
	t.Helper()
	md := fd.FindMessage("bridge.Node")
	tag := fd.FindExtensionByName("bridge.tag")
	extNode := fd.FindExtensionByName("bridge.ext_node")

	child := dynamic.NewMessage(md)
	child.SetFieldByName("name", "child")
	child.SetField(tag, "child-tag")

	mapValue := dynamic.NewMessage(md)
	mapValue.SetFieldByName("num", int32(42))
	extValue := dynamic.NewMessage(md)
	extValue.SetFieldByName("name", "ext")
	mapValue.SetField(extNode, extValue)

	grp := dynamic.NewMessage(md.FindFieldByName("grp").GetMessageType())
	grp.SetFieldByName("val", "group value")

	msg := dynamic.NewMessage(md)
	msg.SetFieldByName("name", "root")
	msg.SetFieldByName("child", child)
	msg.AddRepeatedFieldByName("children", dynamic.NewMessage(md))
	msg.PutMapFieldByName("by_name", "v", mapValue)
	msg.SetFieldByName("grp", grp)
	msg.SetField(tag, "root-tag")
	// field 150 is in the extension range, but there is no such extension
	unknown := protowire.AppendTag(nil, 150, protowire.VarintType)
	unknown = protowire.AppendVarint(unknown, 123)
	testutil.Ok(t, msg.UnmarshalMerge(unknown))
	return msg
}
