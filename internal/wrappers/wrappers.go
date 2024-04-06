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
	"sync"

	"google.golang.org/protobuf/reflect/protodesc"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
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

// FileWrapper is a protoreflect.FileDescriptor that wraps an underlying
// FileDescriptorProto. It is like ProtoWrapper, but specific to files
// and has a strongly-typed accessor method.
//
// See protowrap.FileWrapper for more.
type FileWrapper interface {
	protoreflect.FileDescriptor
	FileDescriptorProto() *descriptorpb.FileDescriptorProto
}

// ProtoFromFileDescriptor extracts a descriptor proto from the given "rich"
// descriptor.
//
// See protowrap.ProtoFromFileDescriptor for more.
func ProtoFromFileDescriptor(d protoreflect.FileDescriptor) *descriptorpb.FileDescriptorProto {
	if imp, ok := d.(protoreflect.FileImport); ok {
		d = imp.FileDescriptor
	}
	if res, ok := d.(FileWrapper); ok {
		return res.FileDescriptorProto()
	}
	if res, ok := d.(ProtoWrapper); ok {
		if fd, ok := res.AsProto().(*descriptorpb.FileDescriptorProto); ok {
			return fd
		}
	}
	return protodesc.ToFileDescriptorProto(d)
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

// File is a wrapper around a FileDescriptor that provides convenient
// access to the underlying FileDescriptorProto.
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

// WrapFile wraps the given FileDescriptor in a *File. The given
// *FileDescriptorProto, fd,  is assumed to be the underlying
// descriptor proto from which file was produced.
func WrapFile(file protoreflect.FileDescriptor, fd *descriptorpb.FileDescriptorProto) *File {
	return &File{FileDescriptor: file, proto: fd, srcLocs: srcLocsWrapper{SourceLocations: file.SourceLocations()}}
}

// Unwrap implements the WrappedDescriptor interface.
func (w *File) Unwrap() protoreflect.Descriptor {
	return w.FileDescriptor
}

// ParentFile implements the FileDescriptor interface.
func (w *File) ParentFile() protoreflect.FileDescriptor {
	return w
}

// Parent implements the FileDescriptor interface.
func (w *File) Parent() protoreflect.Descriptor {
	return nil
}

// Options implements the FileDescriptor interface.
func (w *File) Options() proto.Message {
	return w.proto.GetOptions()
}

// AsProto implements the ProtoWrapper interface. Also
// see FileDescriptorProto.
func (w *File) AsProto() proto.Message {
	return w.proto
}

// FileDescriptorProto provides access to the underlying
// descriptor proto.
func (w *File) FileDescriptorProto() *descriptorpb.FileDescriptorProto {
	return w.proto
}

// Messages implements the FileDescriptor interface.
func (w *File) Messages() protoreflect.MessageDescriptors {
	w.msgs.initFromFile(w)
	return &w.msgs
}

// Enums implements the FileDescriptor interface.
func (w *File) Enums() protoreflect.EnumDescriptors {
	w.enums.initFromFile(w)
	return &w.enums
}

// Extensions implements the FileDescriptor interface.
func (w *File) Extensions() protoreflect.ExtensionDescriptors {
	w.exts.initFromFile(w)
	return &w.exts
}

// Services implements the FileDescriptor interface.
func (w *File) Services() protoreflect.ServiceDescriptors {
	w.svcs.initFromFile(w)
	return &w.svcs
}

// SourceLocations implements the FileDescriptor interface.
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

// Message is a wrapper around a MessageDescriptor that provides convenient
// access to the underlying DescriptorProto.
//
// This is the concrete type of message descriptors returned from instances
// of *File. All messages in the hierarchy of a *File will have this type.
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

// Unwrap implements the WrappedDescriptor interface.
func (w *Message) Unwrap() protoreflect.Descriptor {
	return w.MessageDescriptor
}

// Parent implements the MessageDescriptor interface.
func (w *Message) Parent() protoreflect.Descriptor {
	return w.parent
}

// ParentFile implements the MessageDescriptor interface.
func (w *Message) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

// Options implements the MessageDescriptor interface.
func (w *Message) Options() proto.Message {
	return w.proto.GetOptions()
}

// AsProto implements the ProtoWrapper interface. Also
// see MessageDescriptorProto.
func (w *Message) AsProto() proto.Message {
	return w.proto
}

// MessageDescriptorProto provides access to the underlying
// descriptor proto.
func (w *Message) MessageDescriptorProto() *descriptorpb.DescriptorProto {
	return w.proto
}

// Fields implements the MessageDescriptor interface.
func (w *Message) Fields() protoreflect.FieldDescriptors {
	w.fields.initFromMessage(w)
	return &w.fields
}

// Oneofs implements the MessageDescriptor interface.
func (w *Message) Oneofs() protoreflect.OneofDescriptors {
	w.oneofs.initFromMessage(w)
	return &w.oneofs
}

// Messages implements the MessageDescriptor interface.
func (w *Message) Messages() protoreflect.MessageDescriptors {
	w.msgs.initFromMessage(w)
	return &w.msgs
}

// Enums implements the MessageDescriptor interface.
func (w *Message) Enums() protoreflect.EnumDescriptors {
	w.enums.initFromMessage(w)
	return &w.enums
}

// Extensions implements the MessageDescriptor interface.
func (w *Message) Extensions() protoreflect.ExtensionDescriptors {
	w.exts.initFromMessage(w)
	return &w.exts
}

// Field is a wrapper around a FieldDescriptor that provides convenient
// access to the underlying FieldDescriptorProto.
//
// This is the concrete type of field descriptors returned from instances
// of *Message. All (non-extension) fields in the hierarchy of a *File will
// have this type.
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

// Unwrap implements the WrappedDescriptor interface.
func (w *Field) Unwrap() protoreflect.Descriptor {
	return w.FieldDescriptor
}

// Parent implements the FieldDescriptor interface.
func (w *Field) Parent() protoreflect.Descriptor {
	return w.parent
}

// ParentFile implements the FieldDescriptor interface.
func (w *Field) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

// Options implements the FieldDescriptor interface.
func (w *Field) Options() proto.Message {
	return w.proto.GetOptions()
}

// AsProto implements the ProtoWrapper interface. Also
// see FieldDescriptorProto.
func (w *Field) AsProto() proto.Message {
	return w.proto
}

// FieldDescriptorProto provides access to the underlying
// descriptor proto.
func (w *Field) FieldDescriptorProto() *descriptorpb.FieldDescriptorProto {
	return w.proto
}

// MapKey implements the FieldDescriptor interface.
func (w *Field) MapKey() protoreflect.FieldDescriptor {
	w.doInit()
	return w.mapKey
}

// MapValue implements the FieldDescriptor interface.
func (w *Field) MapValue() protoreflect.FieldDescriptor {
	w.doInit()
	return w.mapValue
}

// DefaultEnumValue implements the FieldDescriptor interface.
func (w *Field) DefaultEnumValue() protoreflect.EnumValueDescriptor {
	w.doInit()
	return w.defaultEnumValue
}

// ContainingOneof implements the FieldDescriptor interface.
func (w *Field) ContainingOneof() protoreflect.OneofDescriptor {
	w.doInit()
	return w.containingOneof
}

// ContainingMessage implements the FieldDescriptor interface.
func (w *Field) ContainingMessage() protoreflect.MessageDescriptor {
	w.doInit()
	return w.containingMsg
}

// Message implements the FieldDescriptor interface.
func (w *Field) Message() protoreflect.MessageDescriptor {
	w.doInit()
	return w.msgType
}

// Enum implements the FieldDescriptor interface.
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

// Extension is a wrapper around a FieldDescriptor that provides convenient
// access to the underlying FieldDescriptorProto. This type is used to
// represent extension fields; *Field is used to represent normal
// (non-extension) fields.
//
// In addition to protoreflect.FieldDescriptor, this type also implements
// protoreflect.ExtensionTypeDescriptor. If the FieldDescriptor this wraps
// did not implemented ExtensionTypeDescriptor, these methods are
// implemented in terms of a dynamic extension type (e.g. dynamicpb.NewExtensionType).
//
// This is the concrete type of field descriptors returned from instances
// of *File. All extension fields in the hierarchy of a *File will have this
// type.
type Extension struct {
	*Field
	extType protoreflect.ExtensionType
}

var _ ProtoWrapper = &Extension{}
var _ WrappedDescriptor = &Extension{}
var _ protoreflect.ExtensionTypeDescriptor = &Extension{}
var _ protoreflect.ExtensionType = &Extension{}

// New implements the ExtensionType interface.
func (w *Extension) New() protoreflect.Value {
	return w.extType.New()
}

// Zero implements the ExtensionType interface.
func (w *Extension) Zero() protoreflect.Value {
	return w.extType.Zero()
}

// TypeDescriptor implements the ExtensionType interface.
func (w *Extension) TypeDescriptor() protoreflect.ExtensionTypeDescriptor {
	return w
}

// ValueOf implements the ExtensionType interface.
func (w *Extension) ValueOf(i interface{}) protoreflect.Value {
	return w.extType.ValueOf(i)
}

// InterfaceOf implements the ExtensionType interface.
func (w *Extension) InterfaceOf(value protoreflect.Value) interface{} {
	return w.extType.InterfaceOf(value)
}

// IsValidValue implements the ExtensionType interface.
func (w *Extension) IsValidValue(value protoreflect.Value) bool {
	return w.extType.IsValidValue(value)
}

// IsValidInterface implements the ExtensionType interface.
func (w *Extension) IsValidInterface(i interface{}) bool {
	return w.extType.IsValidInterface(i)
}

// AsProto implements the ProtoWrapper interface. Also
// see FieldDescriptorProto.
func (w *Extension) AsProto() proto.Message {
	return w.proto
}

// FieldDescriptorProto provides access to the underlying
// descriptor proto.
func (w *Extension) FieldDescriptorProto() *descriptorpb.FieldDescriptorProto {
	return w.proto
}

// Type implements the ExtensionTypeDescriptor interface.
func (w *Extension) Type() protoreflect.ExtensionType {
	return w
}

// Descriptor implements the ExtensionTypeDescriptor interface.
func (w *Extension) Descriptor() protoreflect.ExtensionDescriptor {
	return w
}

// Oneof is a wrapper around a OneofDescriptor that provides convenient
// access to the underlying OneofDescriptorProto.
//
// This is the concrete type of oneof descriptors returned from instances
// of *Message. All oneofs in the hierarchy of a *File will have this type.
type Oneof struct {
	protoreflect.OneofDescriptor
	proto  *descriptorpb.OneofDescriptorProto
	parent *Message
	fields fieldsWrapper
}

var _ ProtoWrapper = &Oneof{}
var _ WrappedDescriptor = &Oneof{}

// Unwrap implements the WrappedDescriptor interface.
func (w *Oneof) Unwrap() protoreflect.Descriptor {
	return w.OneofDescriptor
}

// Parent implements the OneofDescriptor interface.
func (w *Oneof) Parent() protoreflect.Descriptor {
	return w.parent
}

// ParentFile implements the OneofDescriptor interface.
func (w *Oneof) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

// Options implements the OneofDescriptor interface.
func (w *Oneof) Options() proto.Message {
	return w.proto.GetOptions()
}

// AsProto implements the ProtoWrapper interface. Also
// see OneofDescriptorProto.
func (w *Oneof) AsProto() proto.Message {
	return w.proto
}

// OneofDescriptorProto provides access to the underlying
// descriptor proto.
func (w *Oneof) OneofDescriptorProto() *descriptorpb.OneofDescriptorProto {
	return w.proto
}

// Fields implements the OneofDescriptor interface.
func (w *Oneof) Fields() protoreflect.FieldDescriptors {
	w.fields.initFromOneof(w)
	return &w.fields
}

// Enum is a wrapper around an EnumDescriptor that provides convenient
// access to the underlying EnumDescriptorProto.
//
// This is the concrete type of enum descriptors returned from instances
// of *File. All enums in the hierarchy of a *File will have this type.
type Enum struct {
	protoreflect.EnumDescriptor
	proto  *descriptorpb.EnumDescriptorProto
	parent ProtoWrapper // either a *File or *Message
	values enumValuesWrapper
}

var _ ProtoWrapper = &Enum{}
var _ WrappedDescriptor = &Enum{}

// Unwrap implements the WrappedDescriptor interface.
func (w *Enum) Unwrap() protoreflect.Descriptor {
	return w.EnumDescriptor
}

// Parent implements the EnumDescriptor interface.
func (w *Enum) Parent() protoreflect.Descriptor {
	return w.parent
}

// ParentFile implements the EnumDescriptor interface.
func (w *Enum) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

// Options implements the EnumDescriptor interface.
func (w *Enum) Options() proto.Message {
	return w.proto.GetOptions()
}

// AsProto implements the ProtoWrapper interface. Also
// see EnumDescriptorProto.
func (w *Enum) AsProto() proto.Message {
	return w.proto
}

// EnumDescriptorProto provides access to the underlying
// descriptor proto.
func (w *Enum) EnumDescriptorProto() *descriptorpb.EnumDescriptorProto {
	return w.proto
}

// Values implements the EnumDescriptor interface.
func (w *Enum) Values() protoreflect.EnumValueDescriptors {
	w.values.initFromEnum(w)
	return &w.values
}

// EnumValue is a wrapper around an EnumValueDescriptor that provides
// convenient access to the underlying EnumValueDescriptorProto.
//
// This is the concrete type of enum value descriptors returned from
// instances of *Enum. All enum values in the hierarchy of a *File will
// have this type.
type EnumValue struct {
	protoreflect.EnumValueDescriptor
	parent *Enum
	proto  *descriptorpb.EnumValueDescriptorProto
}

var _ ProtoWrapper = &EnumValue{}
var _ WrappedDescriptor = &EnumValue{}

// Unwrap implements the WrappedDescriptor interface.
func (w *EnumValue) Unwrap() protoreflect.Descriptor {
	return w.EnumValueDescriptor
}

// Parent implements the EnumValueDescriptor interface.
func (w *EnumValue) Parent() protoreflect.Descriptor {
	return w.parent
}

// ParentFile implements the EnumValueDescriptor interface.
func (w *EnumValue) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

// Options implements the EnumValueDescriptor interface.
func (w *EnumValue) Options() proto.Message {
	return w.proto.GetOptions()
}

// AsProto implements the ProtoWrapper interface. Also
// see EnumValueDescriptorProto.
func (w *EnumValue) AsProto() proto.Message {
	return w.proto
}

// EnumValueDescriptorProto provides access to the underlying
// descriptor proto.
func (w *EnumValue) EnumValueDescriptorProto() *descriptorpb.EnumValueDescriptorProto {
	return w.proto
}

// Service is a wrapper around a ServiceDescriptor that provides convenient
// access to the underlying ServiceDescriptorProto.
//
// This is the concrete type of service descriptors returned from instances
// of *File.
type Service struct {
	protoreflect.ServiceDescriptor
	parent *File
	proto  *descriptorpb.ServiceDescriptorProto
	mtds   mtdsWrapper
}

var _ ProtoWrapper = &Service{}
var _ WrappedDescriptor = &Service{}

// Unwrap implements the WrappedDescriptor interface.
func (w *Service) Unwrap() protoreflect.Descriptor {
	return w.ServiceDescriptor
}

// Parent implements the ServiceDescriptor interface.
func (w *Service) Parent() protoreflect.Descriptor {
	return w.parent
}

// ParentFile implements the ServiceDescriptor interface.
func (w *Service) ParentFile() protoreflect.FileDescriptor {
	return w.parent
}

// Options implements the ServiceDescriptor interface.
func (w *Service) Options() proto.Message {
	return w.proto.GetOptions()
}

// AsProto implements the ProtoWrapper interface. Also
// see ServiceDescriptorProto.
func (w *Service) AsProto() proto.Message {
	return w.proto
}

// ServiceDescriptorProto provides access to the underlying
// descriptor proto.
func (w *Service) ServiceDescriptorProto() *descriptorpb.ServiceDescriptorProto {
	return w.proto
}

// Methods implements the ServiceDescriptor interface.
func (w *Service) Methods() protoreflect.MethodDescriptors {
	w.mtds.initFromSvc(w)
	return &w.mtds
}

// Method is a wrapper around a MethodDescriptor that provides convenient
// access to the underlying MethodDescriptorProto.
//
// This is the concrete type of method descriptors returned from instances
// of *Service.
type Method struct {
	protoreflect.MethodDescriptor
	parent *Service
	proto  *descriptorpb.MethodDescriptorProto

	init          sync.Once
	input, output protoreflect.MessageDescriptor
}

var _ ProtoWrapper = &Method{}
var _ WrappedDescriptor = &Method{}

// Unwrap implements the WrappedDescriptor interface.
func (w *Method) Unwrap() protoreflect.Descriptor {
	return w.MethodDescriptor
}

// Parent implements the MethodDescriptor interface.
func (w *Method) Parent() protoreflect.Descriptor {
	return w.parent
}

// ParentFile implements the MethodDescriptor interface.
func (w *Method) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

// Options implements the MethodDescriptor interface.
func (w *Method) Options() proto.Message {
	return w.proto.GetOptions()
}

// AsProto implements the ProtoWrapper interface. Also
// see MethodDescriptorProto.
func (w *Method) AsProto() proto.Message {
	return w.proto
}

// MethodDescriptorProto provides access to the underlying
// descriptor proto.
func (w *Method) MethodDescriptorProto() *descriptorpb.MethodDescriptorProto {
	return w.proto
}

// Input implements the MethodDescriptor interface.
func (w *Method) Input() protoreflect.MessageDescriptor {
	w.doInit()
	return w.input
}

// Output implements the MethodDescriptor interface.
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
