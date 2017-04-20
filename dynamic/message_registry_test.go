package dynamic

import (
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestMessageRegistry_LookupTypes(t *testing.T) {
	mr := &MessageRegistry{}

	// register some types
	md, err := desc.LoadMessageDescriptor("google.protobuf.DescriptorProto")
	testutil.Ok(t, err)
	err = mr.AddMessage("foo.bar/google.protobuf.DescriptorProto", md)
	testutil.Ok(t, err)
	ed := md.GetFile().FindEnum("google.protobuf.FieldDescriptorProto.Type")
	testutil.Require(t, ed != nil)
	err = mr.AddEnum("foo.bar/google.protobuf.FieldDescriptorProto.Type", ed)
	testutil.Ok(t, err)

	// lookups succeed
	msg, err := mr.FindMessageTypeByUrl("foo.bar/google.protobuf.DescriptorProto")
	testutil.Ok(t, err)
	testutil.Eq(t, md, msg)
	en, err := mr.FindEnumTypeByUrl("foo.bar/google.protobuf.FieldDescriptorProto.Type")
	testutil.Ok(t, err)
	testutil.Eq(t, ed, en)

	// right name but wrong domain? not found
	msg, err = mr.FindMessageTypeByUrl("type.googleapis.com/google.protobuf.DescriptorProto")
	testutil.Ok(t, err)
	testutil.Require(t, msg == nil)
	en, err = mr.FindEnumTypeByUrl("type.googleapis.com/google.protobuf.FieldDescriptorProto.Type")
	testutil.Ok(t, err)
	testutil.Require(t, en == nil)

	// wrong type
	_, err = mr.FindMessageTypeByUrl("foo.bar/google.protobuf.FieldDescriptorProto.Type")
	testutil.Require(t, err != nil && strings.Contains(err.Error(), "wanted message, got enum"))
	_, err = mr.FindEnumTypeByUrl("foo.bar/google.protobuf.DescriptorProto")
	testutil.Require(t, err != nil && strings.Contains(err.Error(), "wanted enum, got message"))

	// unmarshal any successfully finds the registered type
	b, err := proto.Marshal(md.AsProto())
	testutil.Ok(t, err)
	a := &any.Any{TypeUrl: "foo.bar/google.protobuf.DescriptorProto", Value: b}
	pm, err := mr.UnmarshalAny(a)
	testutil.Ok(t, err)
	testutil.Ceq(t, md.AsProto(), pm, eqm)
	// we didn't configure the registry with a message factory, so it would have
	// produced a dynamic message instead of a generated message
	testutil.Eq(t, typeOfDynamicMessage, reflect.TypeOf(pm))

	// by default, message registry knows about well-known types
	dur := &duration.Duration{Nanos: 100, Seconds: 1000}
	b, err = proto.Marshal(dur)
	testutil.Ok(t, err)
	a = &any.Any{TypeUrl: "foo.bar/google.protobuf.Duration", Value: b}
	pm, err = mr.UnmarshalAny(a)
	testutil.Ok(t, err)
	testutil.Ceq(t, dur, pm, eqm)
	testutil.Eq(t, reflect.TypeOf((*duration.Duration)(nil)), reflect.TypeOf(pm))
}

func TestMessageRegistry_LookupTypes_WithDefaults(t *testing.T) {
	mr := NewMessageRegistryWithDefaults()

	md, err := desc.LoadMessageDescriptor("google.protobuf.DescriptorProto")
	testutil.Ok(t, err)
	ed := md.GetFile().FindEnum("google.protobuf.FieldDescriptorProto.Type")
	testutil.Require(t, ed != nil)

	// lookups succeed
	msg, err := mr.FindMessageTypeByUrl("type.googleapis.com/google.protobuf.DescriptorProto")
	testutil.Ok(t, err)
	testutil.Eq(t, md, msg)
	// default types don't know their base URL, so will resolve even w/ wrong name
	// (just have to get fully-qualified message name right)
	msg, err = mr.FindMessageTypeByUrl("foo.bar/google.protobuf.DescriptorProto")
	testutil.Ok(t, err)
	testutil.Eq(t, md, msg)

	// sad trombone: no way to lookup "default" enum types, so enums don't resolve
	// without being explicitly registered :(
	en, err := mr.FindEnumTypeByUrl("type.googleapis.com/google.protobuf.FieldDescriptorProto.Type")
	testutil.Ok(t, err)
	testutil.Require(t, en == nil)
	en, err = mr.FindEnumTypeByUrl("foo.bar/google.protobuf.FieldDescriptorProto.Type")
	testutil.Ok(t, err)
	testutil.Require(t, en == nil)

	// unmarshal any successfully finds the registered type
	b, err := proto.Marshal(md.AsProto())
	testutil.Ok(t, err)
	a := &any.Any{TypeUrl: "foo.bar/google.protobuf.DescriptorProto", Value: b}
	pm, err := mr.UnmarshalAny(a)
	testutil.Ok(t, err)
	testutil.Ceq(t, md.AsProto(), pm, eqm)
	// message registry with defaults implies known-type registry with defaults, so
	// it should have marshalled the message into a generated message
	testutil.Eq(t, reflect.TypeOf((*descriptor.DescriptorProto)(nil)), reflect.TypeOf(pm))
}

func TestMessageRegistry_LookupTypes_WithFetcher(t *testing.T) {
	// TODO
}

func TestMessageRegistry_ResolveApiIntoServiceDescriptor(t *testing.T) {
	// TODO
}

func TestMessageRegistry_MarshalAny(t *testing.T) {
	mr := &MessageRegistry{}

	md, err := desc.LoadMessageDescriptor("google.protobuf.DescriptorProto")
	testutil.Ok(t, err)

	// default base URL
	a, err := mr.MarshalAny(md.AsProto())
	testutil.Ok(t, err)
	testutil.Eq(t, "type.googleapis.com/google.protobuf.DescriptorProto", a.TypeUrl)
	var umd descriptor.DescriptorProto
	err = ptypes.UnmarshalAny(a, &umd)
	testutil.Ok(t, err)
	testutil.Ceq(t, md.AsProto(), &umd, eqm)

	// different default
	mr.WithDefaultBaseUrl("foo.com/some/path/")
	a, err = mr.MarshalAny(md.AsProto())
	testutil.Ok(t, err)
	testutil.Eq(t, "foo.com/some/path/google.protobuf.DescriptorProto", a.TypeUrl)

	// custom base URL for package
	mr.AddBaseUrlForElement("bar.com/other/", "google.protobuf")
	a, err = mr.MarshalAny(md.AsProto())
	testutil.Ok(t, err)
	testutil.Eq(t, "bar.com/other/google.protobuf.DescriptorProto", a.TypeUrl)

	// custom base URL for type
	mr.AddBaseUrlForElement("http://baz.com/another/", "google.protobuf.DescriptorProto")
	a, err = mr.MarshalAny(md.AsProto())
	testutil.Ok(t, err)
	testutil.Eq(t, "http://baz.com/another/google.protobuf.DescriptorProto", a.TypeUrl)
}

func TestMessageRegistry_DescriptorsToPTypes(t *testing.T) {
	// TODO
}
