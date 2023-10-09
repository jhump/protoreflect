// Package wrappers contains protoreflect.Descriptor implementations that wrap
// corresponding descriptor protos. This makes it faster and non-lossy to get
// a descriptor proto from the descriptor, unlike use of the protodesc package,
// which is slow (must construct a file descriptor proto hierarchy) and lossy
// (there are some qualities of a descriptor proto that don't impact semantics
// and are thus not represented by a protoreflect.descriptor, so a round-trip
// from proto to descriptor and back will lose some of these qualities).
//
// These are defined here, instead of in the protowrap package, in order to
// prevent an import cycle, since protowrap imports protoresolve, but protoresolve
// has need of these wrappers. So, defined here, they can safely be used from both
// packages without a cycle.
package wrappers

import (
	"fmt"
	"google.golang.org/protobuf/types/dynamicpb"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ProtoWrapper is a protoreflect.Descriptor that wraps an underlying
// descriptor proto. It provides the same interface as Descriptor but
// with one extra operation, to efficiently query for the underlying
// descriptor proto.
//
// See protowrap.ProtoWrapper for more.
type ProtoWrapper interface {
	protoreflect.Descriptor
	AsProto() proto.Message
}

// WrappedDescriptor represents a descriptor that has been wrapped or decorated.
// Its sole method allows recovery of the underlying, original descriptor.
type WrappedDescriptor interface {
	Unwrap() protoreflect.Descriptor
}

// Unwrap unwraps the given descriptor. If it implements WrappedDescriptor,
// the underlying descriptor is returned. Otherwise, d is returned as is.
func Unwrap(d protoreflect.Descriptor) protoreflect.Descriptor {
	w, ok := d.(WrappedDescriptor)
	if !ok {
		// not wrapped
		return d
	}
	unwrapped := w.Unwrap()
	if unwrapped == nil {
		return d
	}
	// Make sure that the unwrapped descriptor matches the incoming type
	switch d.(type) {
	case protoreflect.FileDescriptor:
		return unwrapped.(protoreflect.FileDescriptor)
	case protoreflect.MessageDescriptor:
		return unwrapped.(protoreflect.MessageDescriptor)
	case protoreflect.FieldDescriptor:
		return unwrapped.(protoreflect.FieldDescriptor)
	case protoreflect.OneofDescriptor:
		return unwrapped.(protoreflect.OneofDescriptor)
	case protoreflect.EnumDescriptor:
		return unwrapped.(protoreflect.EnumDescriptor)
	case protoreflect.EnumValueDescriptor:
		return unwrapped.(protoreflect.EnumValueDescriptor)
	case protoreflect.ServiceDescriptor:
		return unwrapped.(protoreflect.ServiceDescriptor)
	case protoreflect.MethodDescriptor:
		return unwrapped.(protoreflect.MethodDescriptor)
	default:
		return unwrapped
	}
}

type File struct {
	protoreflect.FileDescriptor
	proto   *descriptorpb.FileDescriptorProto
	srcLocs srcLocsWrapper
	msgs    msgsWrapper
	enums   enumsWrapper
	exts    extsWrapper
	svcs    svcsWrapper
}

var _ ProtoWrapper = &File{}
var _ WrappedDescriptor = &File{}

func WrapFile(file protoreflect.FileDescriptor, fd *descriptorpb.FileDescriptorProto) *File {
	return &File{FileDescriptor: file, proto: fd, srcLocs: srcLocsWrapper{SourceLocations: file.SourceLocations()}}
}

func (w *File) Unwrap() protoreflect.Descriptor {
	return w.FileDescriptor
}

func (w *File) ParentFile() protoreflect.FileDescriptor {
	return w
}

func (w *File) Parent() protoreflect.Descriptor {
	return nil
}

func (w *File) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *File) AsProto() proto.Message {
	return w.proto
}

func (w *File) FileDescriptorProto() *descriptorpb.FileDescriptorProto {
	return w.proto
}

func (w *File) Messages() protoreflect.MessageDescriptors {
	w.msgs.initFromFile(w)
	return &w.msgs
}

func (w *File) Enums() protoreflect.EnumDescriptors {
	w.enums.initFromFile(w)
	return &w.enums
}

func (w *File) Extensions() protoreflect.ExtensionDescriptors {
	w.exts.initFromFile(w)
	return &w.exts
}

func (w *File) Services() protoreflect.ServiceDescriptors {
	w.svcs.initFromFile(w)
	return &w.svcs
}

func (w *File) SourceLocations() protoreflect.SourceLocations {
	return &w.srcLocs
}

type srcLocsWrapper struct {
	protoreflect.SourceLocations
}

func (w *srcLocsWrapper) ByDescriptor(d protoreflect.Descriptor) protoreflect.SourceLocation {
	// The underlying SourceLocations makes a check that the given descriptor belongs to
	// the same file. But if the descriptor is wrapped, its file will be different (its
	// file will be the wrapper, but compared against unwrapped original).
	return w.SourceLocations.ByDescriptor(Unwrap(d))
}

type msgsWrapper struct {
	init sync.Once
	protoreflect.MessageDescriptors
	msgs []*Message
}

func (w *msgsWrapper) initFromFile(parent *File) {
	w.init.Do(func() {
		w.doInit(parent, parent.FileDescriptor.Messages(), parent.proto.MessageType)
	})
}

func (w *msgsWrapper) initFromMessage(parent *Message) {
	w.init.Do(func() {
		w.doInit(parent, parent.MessageDescriptor.Messages(), parent.proto.NestedType)
	})
}

func (w *msgsWrapper) doInit(parent ProtoWrapper, msgs protoreflect.MessageDescriptors, protos []*descriptorpb.DescriptorProto) {
	length := msgs.Len()
	w.MessageDescriptors = msgs
	w.msgs = make([]*Message, length)
	for i := 0; i < length; i++ {
		msg := msgs.Get(i)
		w.msgs[i] = &Message{MessageDescriptor: msg, parent: parent, proto: protos[i]}
	}
}

func (w *msgsWrapper) Get(i int) protoreflect.MessageDescriptor {
	return w.msgs[i]
}

func (w *msgsWrapper) ByName(name protoreflect.Name) protoreflect.MessageDescriptor {
	msg := w.MessageDescriptors.ByName(name)
	if msg == nil {
		return nil
	}
	return w.msgs[msg.Index()]
}

type enumsWrapper struct {
	init sync.Once
	protoreflect.EnumDescriptors
	enums []*Enum
}

func (w *enumsWrapper) initFromFile(parent *File) {
	w.init.Do(func() {
		w.doInit(parent, parent.FileDescriptor.Enums(), parent.proto.EnumType)
	})
}

func (w *enumsWrapper) initFromMessage(parent *Message) {
	w.init.Do(func() {
		w.doInit(parent, parent.MessageDescriptor.Enums(), parent.proto.EnumType)
	})
}

func (w *enumsWrapper) doInit(parent ProtoWrapper, enums protoreflect.EnumDescriptors, protos []*descriptorpb.EnumDescriptorProto) {
	length := enums.Len()
	w.EnumDescriptors = enums
	w.enums = make([]*Enum, length)
	for i := 0; i < length; i++ {
		en := enums.Get(i)
		w.enums[i] = &Enum{EnumDescriptor: en, parent: parent, proto: protos[i]}
	}
}

func (w *enumsWrapper) Get(i int) protoreflect.EnumDescriptor {
	return w.enums[i]
}

func (w *enumsWrapper) ByName(name protoreflect.Name) protoreflect.EnumDescriptor {
	en := w.EnumDescriptors.ByName(name)
	if en == nil {
		return nil
	}
	return w.enums[en.Index()]
}

type extsWrapper struct {
	init sync.Once
	protoreflect.ExtensionDescriptors
	exts []*Extension
}

func (w *extsWrapper) initFromFile(parent *File) {
	w.init.Do(func() {
		w.doInit(parent, parent.FileDescriptor.Extensions(), parent.proto.Extension)
	})
}

func (w *extsWrapper) initFromMessage(parent *Message) {
	w.init.Do(func() {
		w.doInit(parent, parent.MessageDescriptor.Extensions(), parent.proto.Extension)
	})
}

func (w *extsWrapper) doInit(parent ProtoWrapper, exts protoreflect.ExtensionDescriptors, protos []*descriptorpb.FieldDescriptorProto) {
	length := exts.Len()
	w.ExtensionDescriptors = exts
	w.exts = make([]*Extension, length)
	for i := 0; i < length; i++ {
		ext := exts.Get(i)
		fld := &Field{FieldDescriptor: ext, parent: parent, proto: protos[i]}

		// Ideally, we'd call protoresolve.ExtensionType, but that would result in an import cycle.
		var extType protoreflect.ExtensionType
		if xtd, ok := ext.(protoreflect.ExtensionTypeDescriptor); ok {
			extType = xtd.Type()
		} else {
			extType = dynamicpb.NewExtensionType(ext)
		}
		w.exts[i] = &Extension{Field: fld, extType: extType}
	}
}

func (w *extsWrapper) Get(i int) protoreflect.ExtensionDescriptor {
	return w.exts[i]
}

func (w *extsWrapper) ByName(name protoreflect.Name) protoreflect.ExtensionDescriptor {
	ext := w.ExtensionDescriptors.ByName(name)
	if ext == nil {
		return nil
	}
	return w.exts[ext.Index()]
}

type svcsWrapper struct {
	init sync.Once
	protoreflect.ServiceDescriptors
	svcs []*Service
}

func (w *svcsWrapper) initFromFile(parent *File) {
	w.init.Do(func() {
		svcs := parent.FileDescriptor.Services()
		length := svcs.Len()
		w.ServiceDescriptors = svcs
		w.svcs = make([]*Service, length)
		for i := 0; i < length; i++ {
			svc := svcs.Get(i)
			w.svcs[i] = &Service{ServiceDescriptor: svc, parent: parent, proto: parent.proto.Service[i]}
		}
	})
}

func (w *svcsWrapper) Get(i int) protoreflect.ServiceDescriptor {
	return w.svcs[i]
}

func (w *svcsWrapper) ByName(name protoreflect.Name) protoreflect.ServiceDescriptor {
	svc := w.ServiceDescriptors.ByName(name)
	if svc == nil {
		return nil
	}
	return w.svcs[svc.Index()]
}

type fieldsWrapper struct {
	init sync.Once
	protoreflect.FieldDescriptors
	fields []*Field
}

func (w *fieldsWrapper) initFromMessage(parent *Message) {
	w.init.Do(func() {
		fields := parent.MessageDescriptor.Fields()
		length := fields.Len()
		w.FieldDescriptors = fields
		w.fields = make([]*Field, length)
		for i := 0; i < length; i++ {
			field := fields.Get(i)
			w.fields[i] = &Field{FieldDescriptor: field, parent: parent, proto: parent.proto.Field[i]}
		}
	})
}

func (w *fieldsWrapper) initFromOneof(parent *Oneof) {
	w.init.Do(func() {
		w.FieldDescriptors = parent.OneofDescriptor.Fields()
		parent.parent.fields.initFromMessage(parent.parent)
		w.fields = parent.parent.fields.fields
	})
}

func (w *fieldsWrapper) Get(i int) protoreflect.FieldDescriptor {
	// We don't do direct access of fields slice here because the indexes
	// for a oneof's fields won't match, since the fields slice is populated
	// from the parent message. So we ask the embedded FieldDescriptors for
	// the value at the index, and then *that* thing's Index() method will
	// return the correct index for our fields slice.
	field := w.FieldDescriptors.Get(i)
	if field == nil {
		return nil
	}
	return w.fields[field.Index()]
}

func (w *fieldsWrapper) ByName(name protoreflect.Name) protoreflect.FieldDescriptor {
	field := w.FieldDescriptors.ByName(name)
	if field == nil {
		return nil
	}
	return w.fields[field.Index()]
}

func (w *fieldsWrapper) ByJSONName(name string) protoreflect.FieldDescriptor {
	field := w.FieldDescriptors.ByJSONName(name)
	if field == nil {
		return nil
	}
	return w.fields[field.Index()]
}

func (w *fieldsWrapper) ByTextName(name string) protoreflect.FieldDescriptor {
	field := w.FieldDescriptors.ByTextName(name)
	if field == nil {
		return nil
	}
	return w.fields[field.Index()]
}

func (w *fieldsWrapper) ByNumber(number protoreflect.FieldNumber) protoreflect.FieldDescriptor {
	field := w.FieldDescriptors.ByNumber(number)
	if field == nil {
		return nil
	}
	return w.fields[field.Index()]
}

type oneofsWrapper struct {
	init sync.Once
	protoreflect.OneofDescriptors
	oos []*Oneof
}

func (w *oneofsWrapper) initFromMessage(parent *Message) {
	w.init.Do(func() {
		oos := parent.MessageDescriptor.Oneofs()
		length := oos.Len()
		w.OneofDescriptors = oos
		w.oos = make([]*Oneof, length)
		for i := 0; i < length; i++ {
			oo := oos.Get(i)
			w.oos[i] = &Oneof{OneofDescriptor: oo, parent: parent, proto: parent.proto.OneofDecl[i]}
		}
	})
}

func (w *oneofsWrapper) Get(i int) protoreflect.OneofDescriptor {
	return w.oos[i]
}

func (w *oneofsWrapper) ByName(name protoreflect.Name) protoreflect.OneofDescriptor {
	oo := w.OneofDescriptors.ByName(name)
	if oo == nil {
		return nil
	}
	return w.oos[oo.Index()]
}

type enumValuesWrapper struct {
	init sync.Once
	protoreflect.EnumValueDescriptors
	vals []*EnumValue
}

func (w *enumValuesWrapper) initFromEnum(parent *Enum) {
	w.init.Do(func() {
		vals := parent.EnumDescriptor.Values()
		length := vals.Len()
		w.EnumValueDescriptors = vals
		w.vals = make([]*EnumValue, length)
		for i := 0; i < length; i++ {
			val := vals.Get(i)
			w.vals[i] = &EnumValue{EnumValueDescriptor: val, parent: parent, proto: parent.proto.Value[i]}
		}
	})
}

func (w *enumValuesWrapper) Get(i int) protoreflect.EnumValueDescriptor {
	return w.vals[i]
}

func (w *enumValuesWrapper) ByName(name protoreflect.Name) protoreflect.EnumValueDescriptor {
	val := w.EnumValueDescriptors.ByName(name)
	if val == nil {
		return nil
	}
	return w.vals[val.Index()]
}

func (w *enumValuesWrapper) ByNumber(number protoreflect.EnumNumber) protoreflect.EnumValueDescriptor {
	val := w.EnumValueDescriptors.ByNumber(number)
	if val == nil {
		return nil
	}
	return w.vals[val.Index()]
}

type mtdsWrapper struct {
	init sync.Once
	protoreflect.MethodDescriptors
	mtds []*Method
}

func (w *mtdsWrapper) initFromSvc(parent *Service) {
	w.init.Do(func() {
		mtds := parent.ServiceDescriptor.Methods()
		length := mtds.Len()
		w.MethodDescriptors = mtds
		w.mtds = make([]*Method, length)
		for i := 0; i < length; i++ {
			mtd := mtds.Get(i)
			w.mtds[i] = &Method{MethodDescriptor: mtd, parent: parent, proto: parent.proto.Method[i]}
		}
	})
}

func (w *mtdsWrapper) Get(i int) protoreflect.MethodDescriptor {
	return w.mtds[i]
}

func (w *mtdsWrapper) ByName(name protoreflect.Name) protoreflect.MethodDescriptor {
	mtd := w.MethodDescriptors.ByName(name)
	if mtd == nil {
		return nil
	}
	return w.mtds[mtd.Index()]
}

type Message struct {
	protoreflect.MessageDescriptor
	parent ProtoWrapper // either *File or *Message
	proto  *descriptorpb.DescriptorProto
	fields fieldsWrapper
	oneofs oneofsWrapper
	msgs   msgsWrapper
	enums  enumsWrapper
	exts   extsWrapper
}

var _ ProtoWrapper = &Message{}
var _ WrappedDescriptor = &Message{}

func (w *Message) Unwrap() protoreflect.Descriptor {
	return w.MessageDescriptor
}

func (w *Message) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *Message) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *Message) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *Message) AsProto() proto.Message {
	return w.proto
}

func (w *Message) MessageDescriptorProto() *descriptorpb.DescriptorProto {
	return w.proto
}

func (w *Message) Fields() protoreflect.FieldDescriptors {
	w.fields.initFromMessage(w)
	return &w.fields
}

func (w *Message) Oneofs() protoreflect.OneofDescriptors {
	w.oneofs.initFromMessage(w)
	return &w.oneofs
}

func (w *Message) Messages() protoreflect.MessageDescriptors {
	w.msgs.initFromMessage(w)
	return &w.msgs
}

func (w *Message) Enums() protoreflect.EnumDescriptors {
	w.enums.initFromMessage(w)
	return &w.enums
}

func (w *Message) Extensions() protoreflect.ExtensionDescriptors {
	w.exts.initFromMessage(w)
	return &w.exts
}

type Field struct {
	protoreflect.FieldDescriptor
	parent ProtoWrapper // could be *File or *Message
	proto  *descriptorpb.FieldDescriptorProto

	init             sync.Once
	mapKey, mapValue protoreflect.FieldDescriptor
	containingOneof  protoreflect.OneofDescriptor
	defaultEnumValue protoreflect.EnumValueDescriptor
	containingMsg    protoreflect.MessageDescriptor
	enumType         protoreflect.EnumDescriptor
	msgType          protoreflect.MessageDescriptor
}

var _ ProtoWrapper = &Field{}
var _ WrappedDescriptor = &Field{}

func (w *Field) Unwrap() protoreflect.Descriptor {
	return w.FieldDescriptor
}

func (w *Field) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *Field) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *Field) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *Field) AsProto() proto.Message {
	return w.proto
}

func (w *Field) FieldDescriptorProto() *descriptorpb.FieldDescriptorProto {
	return w.proto
}

func (w *Field) MapKey() protoreflect.FieldDescriptor {
	w.doInit()
	return w.mapKey
}

func (w *Field) MapValue() protoreflect.FieldDescriptor {
	w.doInit()
	return w.mapValue
}

func (w *Field) DefaultEnumValue() protoreflect.EnumValueDescriptor {
	w.doInit()
	return w.defaultEnumValue
}

func (w *Field) ContainingOneof() protoreflect.OneofDescriptor {
	w.doInit()
	return w.containingOneof
}

func (w *Field) ContainingMessage() protoreflect.MessageDescriptor {
	w.doInit()
	return w.containingMsg
}

func (w *Field) Message() protoreflect.MessageDescriptor {
	w.doInit()
	return w.msgType
}

func (w *Field) Enum() protoreflect.EnumDescriptor {
	w.doInit()
	return w.enumType
}

func (w *Field) doInit() {
	w.init.Do(func() {
		if mapKey := w.FieldDescriptor.MapKey(); mapKey != nil {
			w.mapKey = findField(mapKey, w.ParentFile())
		}
		if mapVal := w.FieldDescriptor.MapValue(); mapVal != nil {
			w.mapValue = findField(mapVal, w.ParentFile())
		}
		if oo := w.FieldDescriptor.ContainingOneof(); oo != nil {
			parent := w.parent.(*Message)
			w.containingOneof = parent.Oneofs().Get(oo.Index())
		}
		if !w.IsExtension() {
			w.containingMsg = w.parent.(*Message)
		} else {
			w.containingMsg = maybeFindMessage(w.FieldDescriptor.ContainingMessage(), w.ParentFile())
		}
		if enVal := w.FieldDescriptor.DefaultEnumValue(); enVal != nil {
			w.defaultEnumValue = maybeFindEnumValue(enVal, w.ParentFile())
		}
		if en := w.FieldDescriptor.Enum(); en != nil {
			w.enumType = maybeFindEnum(en, w.ParentFile())
		}
		if msg := w.FieldDescriptor.Message(); msg != nil {
			w.msgType = maybeFindMessage(msg, w.ParentFile())
		}
	})
}

type Extension struct {
	*Field
	extType protoreflect.ExtensionType
}

var _ ProtoWrapper = &Extension{}
var _ WrappedDescriptor = &Extension{}
var _ protoreflect.ExtensionTypeDescriptor = &Extension{}
var _ protoreflect.ExtensionType = &Extension{}

func (w *Extension) New() protoreflect.Value {
	return w.extType.New()
}

func (w *Extension) Zero() protoreflect.Value {
	return w.extType.Zero()
}

func (w *Extension) TypeDescriptor() protoreflect.ExtensionTypeDescriptor {
	return w
}

func (w *Extension) ValueOf(i interface{}) protoreflect.Value {
	return w.extType.ValueOf(i)
}

func (w *Extension) InterfaceOf(value protoreflect.Value) interface{} {
	return w.extType.InterfaceOf(value)
}

func (w *Extension) IsValidValue(value protoreflect.Value) bool {
	return w.extType.IsValidValue(value)
}

func (w *Extension) IsValidInterface(i interface{}) bool {
	return w.extType.IsValidInterface(i)
}

func (w *Extension) AsProto() proto.Message {
	return w.proto
}

func (w *Extension) FieldDescriptorProto() *descriptorpb.FieldDescriptorProto {
	return w.proto
}

func (w *Extension) Type() protoreflect.ExtensionType {
	return w
}

func (w *Extension) Descriptor() protoreflect.ExtensionDescriptor {
	return w
}

type Oneof struct {
	protoreflect.OneofDescriptor
	proto  *descriptorpb.OneofDescriptorProto
	parent *Message
	fields fieldsWrapper
}

var _ ProtoWrapper = &Oneof{}
var _ WrappedDescriptor = &Oneof{}

func (w *Oneof) Unwrap() protoreflect.Descriptor {
	return w.OneofDescriptor
}

func (w *Oneof) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *Oneof) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *Oneof) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *Oneof) AsProto() proto.Message {
	return w.proto
}

func (w *Oneof) OneofDescriptorProto() *descriptorpb.OneofDescriptorProto {
	return w.proto
}

func (w *Oneof) Fields() protoreflect.FieldDescriptors {
	w.fields.initFromOneof(w)
	return &w.fields
}

type Enum struct {
	protoreflect.EnumDescriptor
	proto  *descriptorpb.EnumDescriptorProto
	parent ProtoWrapper // either a *File or *Message
	values enumValuesWrapper
}

var _ ProtoWrapper = &Enum{}
var _ WrappedDescriptor = &Enum{}

func (w *Enum) Unwrap() protoreflect.Descriptor {
	return w.EnumDescriptor
}

func (w *Enum) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *Enum) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *Enum) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *Enum) AsProto() proto.Message {
	return w.proto
}

func (w *Enum) EnumDescriptorProto() *descriptorpb.EnumDescriptorProto {
	return w.proto
}

func (w *Enum) Values() protoreflect.EnumValueDescriptors {
	w.values.initFromEnum(w)
	return &w.values
}

type EnumValue struct {
	protoreflect.EnumValueDescriptor
	parent *Enum
	proto  *descriptorpb.EnumValueDescriptorProto
}

var _ ProtoWrapper = &EnumValue{}
var _ WrappedDescriptor = &EnumValue{}

func (w *EnumValue) Unwrap() protoreflect.Descriptor {
	return w.EnumValueDescriptor
}

func (w *EnumValue) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *EnumValue) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *EnumValue) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *EnumValue) AsProto() proto.Message {
	return w.proto
}

func (w *EnumValue) EnumValueDescriptorProto() *descriptorpb.EnumValueDescriptorProto {
	return w.proto
}

type Service struct {
	protoreflect.ServiceDescriptor
	parent *File
	proto  *descriptorpb.ServiceDescriptorProto
	mtds   mtdsWrapper
}

var _ ProtoWrapper = &Service{}
var _ WrappedDescriptor = &Service{}

func (w *Service) Unwrap() protoreflect.Descriptor {
	return w.ServiceDescriptor
}

func (w *Service) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *Service) ParentFile() protoreflect.FileDescriptor {
	return w.parent
}

func (w *Service) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *Service) AsProto() proto.Message {
	return w.proto
}

func (w *Service) ServiceDescriptorProto() *descriptorpb.ServiceDescriptorProto {
	return w.proto
}

func (w *Service) Methods() protoreflect.MethodDescriptors {
	w.mtds.initFromSvc(w)
	return &w.mtds
}

type Method struct {
	protoreflect.MethodDescriptor
	parent *Service
	proto  *descriptorpb.MethodDescriptorProto

	init          sync.Once
	input, output protoreflect.MessageDescriptor
}

var _ ProtoWrapper = &Method{}
var _ WrappedDescriptor = &Method{}

func (w *Method) Unwrap() protoreflect.Descriptor {
	return w.MethodDescriptor
}

func (w *Method) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *Method) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *Method) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *Method) AsProto() proto.Message {
	return w.proto
}

func (w *Method) MethodDescriptorProto() *descriptorpb.MethodDescriptorProto {
	return w.proto
}

func (w *Method) Input() protoreflect.MessageDescriptor {
	w.doInit()
	return w.input
}

func (w *Method) Output() protoreflect.MessageDescriptor {
	w.doInit()
	return w.output
}

func (w *Method) doInit() {
	w.init.Do(func() {
		w.input = maybeFindMessage(w.MethodDescriptor.Input(), w.ParentFile())
		w.output = maybeFindMessage(w.MethodDescriptor.Output(), w.ParentFile())
	})
}

func findField(fld protoreflect.FieldDescriptor, root protoreflect.FileDescriptor) protoreflect.FieldDescriptor {
	if fld.IsExtension() {
		switch parent := fld.Parent().(type) {
		case protoreflect.FileDescriptor:
			return root.Extensions().Get(fld.Index())
		case protoreflect.MessageDescriptor:
			msg := findMessage(parent, root)
			return msg.Extensions().Get(fld.Index())
		default:
			panic(fmt.Sprintf("unsupported type of parent for field: %T", parent))
		}
	}
	msg := findMessage(fld.Parent().(protoreflect.MessageDescriptor), root)
	return msg.Fields().Get(fld.Index())
}

func findMessage(msg protoreflect.MessageDescriptor, root protoreflect.FileDescriptor) protoreflect.MessageDescriptor {
	switch parent := msg.Parent().(type) {
	case protoreflect.FileDescriptor:
		return root.Messages().Get(msg.Index())
	case protoreflect.MessageDescriptor:
		p := findMessage(parent, root)
		return p.Messages().Get(msg.Index())
	default:
		panic(fmt.Sprintf("unsupported type of parent for message: %T", parent))
	}
}

func findEnum(en protoreflect.EnumDescriptor, root protoreflect.FileDescriptor) protoreflect.EnumDescriptor {
	switch parent := en.Parent().(type) {
	case protoreflect.FileDescriptor:
		return root.Enums().Get(en.Index())
	case protoreflect.MessageDescriptor:
		p := findMessage(parent, root)
		return p.Enums().Get(en.Index())
	default:
		panic(fmt.Sprintf("unsupported type of parent for enum: %T", parent))
	}
}

func findEnumValue(enVal protoreflect.EnumValueDescriptor, root protoreflect.FileDescriptor) protoreflect.EnumValueDescriptor {
	en := findEnum(enVal.Parent().(protoreflect.EnumDescriptor), root)
	return en.Values().Get(enVal.Index())
}

func maybeFindMessage(msg protoreflect.MessageDescriptor, root protoreflect.FileDescriptor) protoreflect.MessageDescriptor {
	if msg.ParentFile().Path() == root.Path() {
		return findMessage(msg, root)
	}
	return msg
}

func maybeFindEnum(en protoreflect.EnumDescriptor, root protoreflect.FileDescriptor) protoreflect.EnumDescriptor {
	if en.ParentFile().Path() == root.Path() {
		return findEnum(en, root)
	}
	return en
}

func maybeFindEnumValue(enVal protoreflect.EnumValueDescriptor, root protoreflect.FileDescriptor) protoreflect.EnumValueDescriptor {
	if enVal.ParentFile().Path() == root.Path() {
		return findEnumValue(enVal, root)
	}
	return enVal
}

func parentFile(d protoreflect.Descriptor) protoreflect.FileDescriptor {
	for {
		d = d.Parent()
		if d == nil {
			return nil
		}
		if fd, ok := d.(protoreflect.FileDescriptor); ok {
			return fd
		}
	}
}
