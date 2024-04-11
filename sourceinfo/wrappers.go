package sourceinfo

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// These are wrappers around the various interfaces in the
// google.golang.org/protobuf/reflect/protoreflect that all
// make sure to return a FileDescriptor that includes source
// code info.

type fileDescriptor struct {
	protoreflect.FileDescriptor
	locs protoreflect.SourceLocations
}

func (f fileDescriptor) ParentFile() protoreflect.FileDescriptor {
	return f
}

func (f fileDescriptor) Parent() protoreflect.Descriptor {
	return nil
}

func (f fileDescriptor) Imports() protoreflect.FileImports {
	return imports{f.FileDescriptor.Imports()}
}

func (f fileDescriptor) Messages() protoreflect.MessageDescriptors {
	return messages{f.FileDescriptor.Messages()}
}

func (f fileDescriptor) Enums() protoreflect.EnumDescriptors {
	return enums{f.FileDescriptor.Enums()}
}

func (f fileDescriptor) Extensions() protoreflect.ExtensionDescriptors {
	return extensions{f.FileDescriptor.Extensions()}
}

func (f fileDescriptor) Services() protoreflect.ServiceDescriptors {
	return services{f.FileDescriptor.Services()}
}

func (f fileDescriptor) SourceLocations() protoreflect.SourceLocations {
	return f.locs
}

func (f fileDescriptor) Unwrap() protoreflect.Descriptor {
	return f.FileDescriptor
}

type imports struct {
	protoreflect.FileImports
}

func (im imports) Get(i int) protoreflect.FileImport {
	fi := im.FileImports.Get(i)
	return protoreflect.FileImport{
		FileDescriptor: getFile(fi.FileDescriptor),
		IsPublic:       fi.IsPublic,
		IsWeak:         fi.IsWeak,
	}
}

type messages struct {
	protoreflect.MessageDescriptors
}

func (m messages) Get(i int) protoreflect.MessageDescriptor {
	return WrapMessage(m.MessageDescriptors.Get(i))
}

func (m messages) ByName(n protoreflect.Name) protoreflect.MessageDescriptor {
	md := m.MessageDescriptors.ByName(n)
	if md == nil {
		return nil
	}
	return WrapMessage(md)
}

type enums struct {
	protoreflect.EnumDescriptors
}

func (e enums) Get(i int) protoreflect.EnumDescriptor {
	return WrapEnum(e.EnumDescriptors.Get(i))
}

func (e enums) ByName(n protoreflect.Name) protoreflect.EnumDescriptor {
	ed := e.EnumDescriptors.ByName(n)
	if ed == nil {
		return nil
	}
	return WrapEnum(ed)
}

type extensions struct {
	protoreflect.ExtensionDescriptors
}

func (e extensions) Get(i int) protoreflect.ExtensionDescriptor {
	return WrapExtension(e.ExtensionDescriptors.Get(i))
}

func (e extensions) ByName(n protoreflect.Name) protoreflect.ExtensionDescriptor {
	extd := e.ExtensionDescriptors.ByName(n)
	if extd == nil {
		return nil
	}
	return WrapExtension(extd)
}

type services struct {
	protoreflect.ServiceDescriptors
}

func (s services) Get(i int) protoreflect.ServiceDescriptor {
	return WrapService(s.ServiceDescriptors.Get(i))
}

func (s services) ByName(n protoreflect.Name) protoreflect.ServiceDescriptor {
	sd := s.ServiceDescriptors.ByName(n)
	if sd == nil {
		return nil
	}
	return WrapService(sd)
}

type messageDescriptor struct {
	protoreflect.MessageDescriptor
}

func (m messageDescriptor) ParentFile() protoreflect.FileDescriptor {
	return getFile(m.MessageDescriptor.ParentFile())
}

func (m messageDescriptor) Parent() protoreflect.Descriptor {
	d := m.MessageDescriptor.Parent()
	switch d := d.(type) {
	case protoreflect.MessageDescriptor:
		return WrapMessage(d)
	case protoreflect.FileDescriptor:
		return getFile(d)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unexpected descriptor type %T", d))
	}
}

func (m messageDescriptor) Fields() protoreflect.FieldDescriptors {
	return fields{m.MessageDescriptor.Fields()}
}

func (m messageDescriptor) Oneofs() protoreflect.OneofDescriptors {
	return oneofs{m.MessageDescriptor.Oneofs()}
}

func (m messageDescriptor) Enums() protoreflect.EnumDescriptors {
	return enums{m.MessageDescriptor.Enums()}
}

func (m messageDescriptor) Messages() protoreflect.MessageDescriptors {
	return messages{m.MessageDescriptor.Messages()}
}

func (m messageDescriptor) Extensions() protoreflect.ExtensionDescriptors {
	return extensions{m.MessageDescriptor.Extensions()}
}

func (m messageDescriptor) Unwrap() protoreflect.Descriptor {
	return m.MessageDescriptor
}

type fields struct {
	protoreflect.FieldDescriptors
}

func (f fields) Get(i int) protoreflect.FieldDescriptor {
	return wrapField(f.FieldDescriptors.Get(i))
}

func (f fields) ByName(n protoreflect.Name) protoreflect.FieldDescriptor {
	fld := f.FieldDescriptors.ByName(n)
	if fld == nil {
		return nil
	}
	return wrapField(fld)
}

func (f fields) ByJSONName(n string) protoreflect.FieldDescriptor {
	fld := f.FieldDescriptors.ByJSONName(n)
	if fld == nil {
		return nil
	}
	return wrapField(fld)
}

func (f fields) ByTextName(n string) protoreflect.FieldDescriptor {
	fld := f.FieldDescriptors.ByTextName(n)
	if fld == nil {
		return nil
	}
	return wrapField(fld)
}

func (f fields) ByNumber(n protoreflect.FieldNumber) protoreflect.FieldDescriptor {
	fld := f.FieldDescriptors.ByNumber(n)
	if fld == nil {
		return nil
	}
	return wrapField(fld)
}

type fieldDescriptor struct {
	protoreflect.FieldDescriptor
}

func (f fieldDescriptor) ParentFile() protoreflect.FileDescriptor {
	return getFile(f.FieldDescriptor.ParentFile())
}

func (f fieldDescriptor) Parent() protoreflect.Descriptor {
	d := f.FieldDescriptor.Parent()
	switch d := d.(type) {
	case protoreflect.MessageDescriptor:
		return WrapMessage(d)
	case protoreflect.FileDescriptor:
		return getFile(d)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unexpected descriptor type %T", d))
	}
}

func (f fieldDescriptor) MapKey() protoreflect.FieldDescriptor {
	fd := f.FieldDescriptor.MapKey()
	if fd == nil {
		return nil
	}
	return wrapField(fd)
}

func (f fieldDescriptor) MapValue() protoreflect.FieldDescriptor {
	fd := f.FieldDescriptor.MapValue()
	if fd == nil {
		return nil
	}
	return wrapField(fd)
}

func (f fieldDescriptor) DefaultEnumValue() protoreflect.EnumValueDescriptor {
	evd := f.FieldDescriptor.DefaultEnumValue()
	if evd == nil {
		return nil
	}
	return wrapEnumValue(evd)
}

func (f fieldDescriptor) ContainingOneof() protoreflect.OneofDescriptor {
	ood := f.FieldDescriptor.ContainingOneof()
	if ood == nil {
		return nil
	}
	return wrapOneof(ood)
}

func (f fieldDescriptor) ContainingMessage() protoreflect.MessageDescriptor {
	return WrapMessage(f.FieldDescriptor.ContainingMessage())
}

func (f fieldDescriptor) Enum() protoreflect.EnumDescriptor {
	ed := f.FieldDescriptor.Enum()
	if ed == nil {
		return nil
	}
	return WrapEnum(ed)
}

func (f fieldDescriptor) Message() protoreflect.MessageDescriptor {
	md := f.FieldDescriptor.Message()
	if md == nil {
		return nil
	}
	return WrapMessage(md)
}

func (f fieldDescriptor) Unwrap() protoreflect.Descriptor {
	return f.FieldDescriptor
}

type oneofs struct {
	protoreflect.OneofDescriptors
}

func (o oneofs) Get(i int) protoreflect.OneofDescriptor {
	return wrapOneof(o.OneofDescriptors.Get(i))
}

func (o oneofs) ByName(n protoreflect.Name) protoreflect.OneofDescriptor {
	ood := o.OneofDescriptors.ByName(n)
	if ood == nil {
		return nil
	}
	return wrapOneof(ood)
}

type oneofDescriptor struct {
	protoreflect.OneofDescriptor
}

func (o oneofDescriptor) ParentFile() protoreflect.FileDescriptor {
	return getFile(o.OneofDescriptor.ParentFile())
}

func (o oneofDescriptor) Parent() protoreflect.Descriptor {
	d := o.OneofDescriptor.Parent()
	switch d := d.(type) {
	case protoreflect.MessageDescriptor:
		return WrapMessage(d)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unexpected descriptor type %T", d))
	}
}

func (o oneofDescriptor) Fields() protoreflect.FieldDescriptors {
	return fields{o.OneofDescriptor.Fields()}
}

func (o oneofDescriptor) Unwrap() protoreflect.Descriptor {
	return o.OneofDescriptor
}

type enumDescriptor struct {
	protoreflect.EnumDescriptor
}

func (e enumDescriptor) ParentFile() protoreflect.FileDescriptor {
	return getFile(e.EnumDescriptor.ParentFile())
}

func (e enumDescriptor) Parent() protoreflect.Descriptor {
	d := e.EnumDescriptor.Parent()
	switch d := d.(type) {
	case protoreflect.MessageDescriptor:
		return WrapMessage(d)
	case protoreflect.FileDescriptor:
		return getFile(d)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unexpected descriptor type %T", d))
	}
}

func (e enumDescriptor) Values() protoreflect.EnumValueDescriptors {
	return enumValues{e.EnumDescriptor.Values()}
}

func (e enumDescriptor) Unwrap() protoreflect.Descriptor {
	return e.EnumDescriptor
}

type enumValues struct {
	protoreflect.EnumValueDescriptors
}

func (e enumValues) Get(i int) protoreflect.EnumValueDescriptor {
	return wrapEnumValue(e.EnumValueDescriptors.Get(i))
}

func (e enumValues) ByName(n protoreflect.Name) protoreflect.EnumValueDescriptor {
	evd := e.EnumValueDescriptors.ByName(n)
	if evd == nil {
		return nil
	}
	return wrapEnumValue(evd)
}

func (e enumValues) ByNumber(n protoreflect.EnumNumber) protoreflect.EnumValueDescriptor {
	evd := e.EnumValueDescriptors.ByNumber(n)
	if evd == nil {
		return nil
	}
	return wrapEnumValue(evd)
}

type enumValueDescriptor struct {
	protoreflect.EnumValueDescriptor
}

func (e enumValueDescriptor) ParentFile() protoreflect.FileDescriptor {
	return getFile(e.EnumValueDescriptor.ParentFile())
}

func (e enumValueDescriptor) Parent() protoreflect.Descriptor {
	d := e.EnumValueDescriptor.Parent()
	switch d := d.(type) {
	case protoreflect.EnumDescriptor:
		return WrapEnum(d)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unexpected descriptor type %T", d))
	}
}

func (e enumValueDescriptor) Unwrap() protoreflect.Descriptor {
	return e.EnumValueDescriptor
}

type extensionTypeDescriptor struct {
	protoreflect.ExtensionTypeDescriptor
}

func (e extensionTypeDescriptor) ParentFile() protoreflect.FileDescriptor {
	return getFile(e.ExtensionTypeDescriptor.ParentFile())
}

func (e extensionTypeDescriptor) Parent() protoreflect.Descriptor {
	d := e.ExtensionTypeDescriptor.Parent()
	switch d := d.(type) {
	case protoreflect.MessageDescriptor:
		return WrapMessage(d)
	case protoreflect.FileDescriptor:
		return getFile(d)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unexpected descriptor type %T", d))
	}
}

func (e extensionTypeDescriptor) MapKey() protoreflect.FieldDescriptor {
	fd := e.ExtensionTypeDescriptor.MapKey()
	if fd == nil {
		return nil
	}
	return wrapField(fd)
}

func (e extensionTypeDescriptor) MapValue() protoreflect.FieldDescriptor {
	fd := e.ExtensionTypeDescriptor.MapValue()
	if fd == nil {
		return nil
	}
	return wrapField(fd)
}

func (e extensionTypeDescriptor) DefaultEnumValue() protoreflect.EnumValueDescriptor {
	evd := e.ExtensionTypeDescriptor.DefaultEnumValue()
	if evd == nil {
		return nil
	}
	return wrapEnumValue(evd)
}

func (e extensionTypeDescriptor) ContainingOneof() protoreflect.OneofDescriptor {
	ood := e.ExtensionTypeDescriptor.ContainingOneof()
	if ood == nil {
		return nil
	}
	return wrapOneof(ood)
}

func (e extensionTypeDescriptor) ContainingMessage() protoreflect.MessageDescriptor {
	return WrapMessage(e.ExtensionTypeDescriptor.ContainingMessage())
}

func (e extensionTypeDescriptor) Enum() protoreflect.EnumDescriptor {
	ed := e.ExtensionTypeDescriptor.Enum()
	if ed == nil {
		return nil
	}
	return WrapEnum(ed)
}

func (e extensionTypeDescriptor) Message() protoreflect.MessageDescriptor {
	md := e.ExtensionTypeDescriptor.Message()
	if md == nil {
		return nil
	}
	return WrapMessage(md)
}

func (e extensionTypeDescriptor) Descriptor() protoreflect.ExtensionDescriptor {
	return WrapExtension(e.ExtensionTypeDescriptor.Descriptor())
}

func (e extensionTypeDescriptor) Unwrap() protoreflect.Descriptor {
	return e.ExtensionTypeDescriptor
}

var _ protoreflect.ExtensionTypeDescriptor = extensionTypeDescriptor{}

type serviceDescriptor struct {
	protoreflect.ServiceDescriptor
}

func (s serviceDescriptor) ParentFile() protoreflect.FileDescriptor {
	return getFile(s.ServiceDescriptor.ParentFile())
}

func (s serviceDescriptor) Parent() protoreflect.Descriptor {
	d := s.ServiceDescriptor.Parent()
	switch d := d.(type) {
	case protoreflect.FileDescriptor:
		return getFile(d)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unexpected descriptor type %T", d))
	}
}

func (s serviceDescriptor) Methods() protoreflect.MethodDescriptors {
	return methods{s.ServiceDescriptor.Methods()}
}

func (s serviceDescriptor) Unwrap() protoreflect.Descriptor {
	return s.ServiceDescriptor
}

type methods struct {
	protoreflect.MethodDescriptors
}

func (m methods) Get(i int) protoreflect.MethodDescriptor {
	return wrapMethod(m.MethodDescriptors.Get(i))
}

func (m methods) ByName(n protoreflect.Name) protoreflect.MethodDescriptor {
	mtd := m.MethodDescriptors.ByName(n)
	if mtd == nil {
		return nil
	}
	return wrapMethod(mtd)
}

type methodDescriptor struct {
	protoreflect.MethodDescriptor
}

func (m methodDescriptor) ParentFile() protoreflect.FileDescriptor {
	return getFile(m.MethodDescriptor.ParentFile())
}

func (m methodDescriptor) Parent() protoreflect.Descriptor {
	d := m.MethodDescriptor.Parent()
	switch d := d.(type) {
	case protoreflect.ServiceDescriptor:
		return WrapService(d)
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("unexpected descriptor type %T", d))
	}
}

func (m methodDescriptor) Input() protoreflect.MessageDescriptor {
	return WrapMessage(m.MethodDescriptor.Input())
}

func (m methodDescriptor) Output() protoreflect.MessageDescriptor {
	return WrapMessage(m.MethodDescriptor.Output())
}

func (m methodDescriptor) Unwrap() protoreflect.Descriptor {
	return m.MethodDescriptor
}

type extensionType struct {
	protoreflect.ExtensionType
}

func (e extensionType) TypeDescriptor() protoreflect.ExtensionTypeDescriptor {
	return wrapExtensionTypeDescriptor(e.ExtensionType.TypeDescriptor())
}

type messageType struct {
	protoreflect.MessageType
}

func (m messageType) Descriptor() protoreflect.MessageDescriptor {
	return WrapMessage(m.MessageType.Descriptor())
}

type enumType struct {
	protoreflect.EnumType
}

func (e enumType) Descriptor() protoreflect.EnumDescriptor {
	return WrapEnum(e.EnumType.Descriptor())
}

// WrapFile wraps the given file descriptor so that it will include source
// code info that was registered with this package if the given file was
// processed with protoc-gen-gosrcinfo. Returns fd without wrapping if fd
// already contains source code info.
func WrapFile(fd protoreflect.FileDescriptor) protoreflect.FileDescriptor {
	if wrapper, ok := fd.(fileDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if fd.SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return fd
	}
	return getFile(fd)
}

// WrapMessage wraps the given message descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns md without wrapping if md's
// parent file already contains source code info.
func WrapMessage(md protoreflect.MessageDescriptor) protoreflect.MessageDescriptor {
	if wrapper, ok := md.(messageDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if md.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return md
	}
	if !canWrap(md) {
		return md
	}
	return messageDescriptor{md}
}

func wrapField(fld protoreflect.FieldDescriptor) protoreflect.FieldDescriptor {
	if wrapper, ok := fld.(fieldDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if fld.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return fld
	}
	if !canWrap(fld) {
		return fld
	}
	return fieldDescriptor{fld}
}

func wrapOneof(ood protoreflect.OneofDescriptor) protoreflect.OneofDescriptor {
	if wrapper, ok := ood.(oneofDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if ood.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return ood
	}
	if !canWrap(ood) {
		return ood
	}
	return oneofDescriptor{ood}
}

// WrapEnum wraps the given enum descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns ed without wrapping if ed's
// parent file already contains source code info.
func WrapEnum(ed protoreflect.EnumDescriptor) protoreflect.EnumDescriptor {
	if wrapper, ok := ed.(enumDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if ed.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return ed
	}
	if !canWrap(ed) {
		return ed
	}
	return enumDescriptor{ed}
}

func wrapEnumValue(evd protoreflect.EnumValueDescriptor) protoreflect.EnumValueDescriptor {
	if wrapper, ok := evd.(enumValueDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if evd.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return evd
	}
	if !canWrap(evd) {
		return evd
	}
	return enumValueDescriptor{evd}
}

// WrapExtension wraps the given extension descriptor so that it will include
// source code info that was registered with this package if the file it is
// defined in was processed with protoc-gen-gosrcinfo. Returns ed without
// wrapping if extd's parent file already contains source code info.
func WrapExtension(extd protoreflect.ExtensionDescriptor) protoreflect.ExtensionDescriptor {
	if wrapper, ok := extd.(extensionTypeDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if wrapper, ok := extd.(fieldDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if extd.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return extd
	}
	if !canWrap(extd) {
		return extd
	}
	if extType, ok := extd.(protoreflect.ExtensionTypeDescriptor); ok {
		return wrapExtensionTypeDescriptor(extType)
	}
	return fieldDescriptor{extd}
}

func wrapExtensionTypeDescriptor(extd protoreflect.ExtensionTypeDescriptor) protoreflect.ExtensionTypeDescriptor {
	if wrapper, ok := extd.(extensionTypeDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if extd.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return extd
	}
	if !canWrap(extd) {
		return extd
	}
	return extensionTypeDescriptor{extd}
}

// WrapService wraps the given service descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns sd without wrapping if sd's
// parent file already contains source code info.
func WrapService(sd protoreflect.ServiceDescriptor) protoreflect.ServiceDescriptor {
	if wrapper, ok := sd.(serviceDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if sd.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return sd
	}
	if !canWrap(sd) {
		return sd
	}
	return serviceDescriptor{sd}
}

func wrapMethod(mtd protoreflect.MethodDescriptor) protoreflect.MethodDescriptor {
	if wrapper, ok := mtd.(methodDescriptor); ok {
		// already wrapped
		return wrapper
	}
	if mtd.ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return mtd
	}
	if !canWrap(mtd) {
		return mtd
	}
	return methodDescriptor{mtd}
}

// WrapExtensionType wraps the given extension type so that its associated
// descriptor will include source code info that was registered with this package
// if the file it is defined in was processed with protoc-gen-gosrcinfo. Returns
// xt without wrapping if the parent file of xt's descriptor already contains
// source code info.
func WrapExtensionType(xt protoreflect.ExtensionType) protoreflect.ExtensionType {
	if wrapper, ok := xt.(extensionType); ok {
		// already wrapped
		return wrapper
	}
	if xt.TypeDescriptor().ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return xt
	}
	if !canWrap(xt.TypeDescriptor()) {
		return xt
	}
	return extensionType{xt}
}

// WrapMessageType wraps the given message type so that its associated
// descriptor will include source code info that was registered with this package
// if the file it is defined in was processed with protoc-gen-gosrcinfo. Returns
// mt without wrapping if the parent file of mt's descriptor already contains
// source code info.
func WrapMessageType(mt protoreflect.MessageType) protoreflect.MessageType {
	if wrapper, ok := mt.(messageType); ok {
		// already wrapped
		return wrapper
	}
	if mt.Descriptor().ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return mt
	}
	if !canWrap(mt.Descriptor()) {
		return mt
	}
	return messageType{mt}
}

// WrapEnumType wraps the given enum type so that its associated descriptor
// will include source code info that was registered with this package
// if the file it is defined in was processed with protoc-gen-gosrcinfo. Returns
// et without wrapping if the parent file of et's descriptor already contains
// source code info.
func WrapEnumType(et protoreflect.EnumType) protoreflect.EnumType {
	if wrapper, ok := et.(enumType); ok {
		// already wrapped
		return wrapper
	}
	if et.Descriptor().ParentFile().SourceLocations().Len() > 0 {
		// no need to wrap since it includes source info already
		return et
	}
	if !canWrap(et.Descriptor()) {
		return et
	}
	return enumType{et}
}
