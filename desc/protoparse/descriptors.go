package protoparse

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// This file contains implementations of protoreflect.Descriptor. Note that
// this is a hack since those interfaces have a "doNotImplement" tag
// interface therein. We do just enough to make dynamicpb happy; constructing
// a regular descriptor would fail because we haven't yet interpreted options
// at the point we need these, and some validations will fail if the options
// aren't present.

type fileDescriptor struct {
	protoreflect.FileDescriptor
	proto  *descriptorpb.FileDescriptorProto
	l      *linker
	prefix string
}

func (l *linker) asFileDescriptor(fd *descriptorpb.FileDescriptorProto) *fileDescriptor {
	if ret := l.descriptors[fd]; ret != nil {
		return ret.(*fileDescriptor)
	}
	prefix := fd.GetPackage()
	if prefix != "" {
		prefix += "."
	}
	ret := &fileDescriptor{proto: fd, prefix: prefix, l: l}
	l.descriptors[fd] = ret
	return ret
}

func (f *fileDescriptor) ParentFile() protoreflect.FileDescriptor {
	return f
}

func (f *fileDescriptor) Parent() protoreflect.Descriptor {
	return nil
}

func (f *fileDescriptor) Index() int {
	return 0
}

func (f *fileDescriptor) Syntax() protoreflect.Syntax {
	switch f.proto.GetSyntax() {
	case "proto2", "":
		return protoreflect.Proto2
	case "proto3":
		return protoreflect.Proto3
	default:
		return 0 // ???
	}
}

func (f *fileDescriptor) Name() protoreflect.Name {
	return ""
}

func (f *fileDescriptor) FullName() protoreflect.FullName {
	return f.Package()
}

func (f *fileDescriptor) IsPlaceholder() bool {
	return false
}

func (f *fileDescriptor) Options() protoreflect.ProtoMessage {
	return f.proto.Options
}

func (f *fileDescriptor) Path() string {
	return f.proto.GetName()
}

func (f *fileDescriptor) Package() protoreflect.FullName {
	return protoreflect.FullName(f.proto.GetPackage())
}

func (f *fileDescriptor) Imports() protoreflect.FileImports {
	return &fileImports{parent: f, l: f.l}
}

func (f *fileDescriptor) Enums() protoreflect.EnumDescriptors {
	return &enumDescriptors{file: f, parent: f, enums: f.proto.GetEnumType(), prefix: f.prefix, l: f.l}
}

func (f *fileDescriptor) Messages() protoreflect.MessageDescriptors {
	return &msgDescriptors{file: f, parent: f, msgs: f.proto.GetMessageType(), prefix: f.prefix, l: f.l}
}

func (f *fileDescriptor) Extensions() protoreflect.ExtensionDescriptors {
	return &extDescriptors{file: f, parent: f, exts: f.proto.GetExtension(), prefix: f.prefix, l: f.l}
}

func (f *fileDescriptor) Services() protoreflect.ServiceDescriptors {
	return &svcDescriptors{file: f, svcs: f.proto.GetService(), prefix: f.prefix, l: f.l}
}

func (f *fileDescriptor) SourceLocations() protoreflect.SourceLocations {
	return srcLocs{}
}

type fileImports struct {
	protoreflect.FileImports
	parent *fileDescriptor
	l      *linker
}

func (f *fileImports) Len() int {
	return len(f.parent.proto.Dependency)
}

func (f *fileImports) Get(i int) protoreflect.FileImport {
	dep := f.parent.proto.Dependency[i]
	fd := f.l.files[dep].fd
	desc := f.l.asFileDescriptor(fd)
	isPublic := false
	for _, d := range f.parent.proto.PublicDependency {
		if d == int32(i) {
			isPublic = true
			break
		}
	}
	isWeak := false
	for _, d := range f.parent.proto.WeakDependency {
		if d == int32(i) {
			isWeak = true
			break
		}
	}
	return protoreflect.FileImport{FileDescriptor: desc, IsPublic: isPublic, IsWeak: isWeak}
}

type srcLocs struct {
	protoreflect.SourceLocations
}

func (s srcLocs) Len() int {
	return 0
}

func (s srcLocs) Get(_ int) protoreflect.SourceLocation {
	panic("index out of bounds")
}

func (s srcLocs) ByPath(_ protoreflect.SourcePath) protoreflect.SourceLocation {
	return protoreflect.SourceLocation{}
}

func (s srcLocs) ByDescriptor(_ protoreflect.Descriptor) protoreflect.SourceLocation {
	return protoreflect.SourceLocation{}
}

type msgDescriptors struct {
	protoreflect.MessageDescriptors
	file   *fileDescriptor
	parent protoreflect.Descriptor
	msgs   []*descriptorpb.DescriptorProto
	l      *linker
	prefix string
}

func (m *msgDescriptors) Len() int {
	return len(m.msgs)
}

func (m *msgDescriptors) Get(i int) protoreflect.MessageDescriptor {
	msg := m.msgs[i]
	return m.l.asMessageDescriptor(msg, m.file, m.parent, i, m.prefix+msg.GetName())
}

func (m *msgDescriptors) ByName(s protoreflect.Name) protoreflect.MessageDescriptor {
	for i, msg := range m.msgs {
		if msg.GetName() == string(s) {
			return m.Get(i)
		}
	}
	return nil
}

type msgDescriptor struct {
	protoreflect.MessageDescriptor
	file   *fileDescriptor
	parent protoreflect.Descriptor
	index  int
	proto  *descriptorpb.DescriptorProto
	fqn    string
	l      *linker
}

func (l *linker) asMessageDescriptor(md *descriptorpb.DescriptorProto, file *fileDescriptor, parent protoreflect.Descriptor, index int, fqn string) *msgDescriptor {
	if ret := l.descriptors[md]; ret != nil {
		return ret.(*msgDescriptor)
	}
	ret := &msgDescriptor{file: file, parent: parent, index: index, proto: md, fqn: fqn, l: l}
	l.descriptors[md] = ret
	return ret
}

func (m *msgDescriptor) ParentFile() protoreflect.FileDescriptor {
	return m.file
}

func (m *msgDescriptor) Parent() protoreflect.Descriptor {
	return m.parent
}

func (m *msgDescriptor) Index() int {
	return m.index
}

func (m *msgDescriptor) Syntax() protoreflect.Syntax {
	return m.file.Syntax()
}

func (m *msgDescriptor) Name() protoreflect.Name {
	return protoreflect.Name(m.proto.GetName())
}

func (m *msgDescriptor) FullName() protoreflect.FullName {
	return protoreflect.FullName(m.fqn)
}

func (m *msgDescriptor) IsPlaceholder() bool {
	return false
}

func (m *msgDescriptor) Options() protoreflect.ProtoMessage {
	return m.proto.Options
}

func (m *msgDescriptor) IsMapEntry() bool {
	return m.proto.Options.GetMapEntry()
}

func (m *msgDescriptor) Fields() protoreflect.FieldDescriptors {
	return &fldDescriptors{file: m.file, parent: m, fields: m.proto.GetField(), prefix: m.fqn + ".", l: m.l}
}

func (m *msgDescriptor) Oneofs() protoreflect.OneofDescriptors {
	return &oneofDescriptors{file: m.file, parent: m, oneofs: m.proto.GetOneofDecl(), prefix: m.fqn + ".", l: m.l}
}

func (m *msgDescriptor) ReservedNames() protoreflect.Names {
	return names{s: m.proto.ReservedName}
}

func (m *msgDescriptor) ReservedRanges() protoreflect.FieldRanges {
	return fieldRanges{s: m.proto.ReservedRange}
}

func (m *msgDescriptor) RequiredNumbers() protoreflect.FieldNumbers {
	var indexes fieldNums
	for _, fld := range m.proto.Field {
		if fld.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REQUIRED {
			indexes.s = append(indexes.s, fld.GetNumber())
		}
	}
	return indexes
}

func (m *msgDescriptor) ExtensionRanges() protoreflect.FieldRanges {
	return extRanges{s: m.proto.ExtensionRange}
}

func (m *msgDescriptor) ExtensionRangeOptions(i int) protoreflect.ProtoMessage {
	return m.proto.ExtensionRange[i].Options
}

func (m *msgDescriptor) Enums() protoreflect.EnumDescriptors {
	return &enumDescriptors{file: m.file, parent: m, enums: m.proto.GetEnumType(), prefix: m.fqn + ".", l: m.l}
}

func (m *msgDescriptor) Messages() protoreflect.MessageDescriptors {
	return &msgDescriptors{file: m.file, parent: m, msgs: m.proto.GetNestedType(), prefix: m.fqn + ".", l: m.l}
}

func (m *msgDescriptor) Extensions() protoreflect.ExtensionDescriptors {
	return &extDescriptors{file: m.file, parent: m, exts: m.proto.GetExtension(), prefix: m.fqn + ".", l: m.l}
}

type names struct {
	protoreflect.Names
	s []string
}

func (n names) Len() int {
	return len(n.s)
}

func (n names) Get(i int) protoreflect.Name {
	return protoreflect.Name(n.s[i])
}

func (n names) Has(s protoreflect.Name) bool {
	for _, name := range n.s {
		if name == string(s) {
			return true
		}
	}
	return false
}

type fieldNums struct {
	protoreflect.FieldNumbers
	s []int32
}

func (n fieldNums) Len() int {
	return len(n.s)
}

func (n fieldNums) Get(i int) protoreflect.FieldNumber {
	return protoreflect.FieldNumber(n.s[i])
}

func (n fieldNums) Has(s protoreflect.FieldNumber) bool {
	for _, num := range n.s {
		if num == int32(s) {
			return true
		}
	}
	return false
}

type fieldRanges struct {
	protoreflect.FieldRanges
	s []*descriptorpb.DescriptorProto_ReservedRange
}

func (f fieldRanges) Len() int {
	return len(f.s)
}

func (f fieldRanges) Get(i int) [2]protoreflect.FieldNumber {
	r := f.s[i]
	return [2]protoreflect.FieldNumber{
		protoreflect.FieldNumber(r.GetStart()),
		protoreflect.FieldNumber(r.GetEnd()),
	}
}

func (f fieldRanges) Has(n protoreflect.FieldNumber) bool {
	for _, r := range f.s {
		if r.GetStart() <= int32(n) && r.GetEnd() > int32(n) {
			return true
		}
	}
	return false
}

type extRanges struct {
	protoreflect.FieldRanges
	s []*descriptorpb.DescriptorProto_ExtensionRange
}

func (e extRanges) Len() int {
	return len(e.s)
}

func (e extRanges) Get(i int) [2]protoreflect.FieldNumber {
	r := e.s[i]
	return [2]protoreflect.FieldNumber{
		protoreflect.FieldNumber(r.GetStart()),
		protoreflect.FieldNumber(r.GetEnd()),
	}
}

func (e extRanges) Has(n protoreflect.FieldNumber) bool {
	for _, r := range e.s {
		if r.GetStart() <= int32(n) && r.GetEnd() > int32(n) {
			return true
		}
	}
	return false
}

type enumDescriptors struct {
	protoreflect.EnumDescriptors
	file   *fileDescriptor
	parent protoreflect.Descriptor
	enums  []*descriptorpb.EnumDescriptorProto
	prefix string
	l      *linker
}

func (e *enumDescriptors) Len() int {
	return len(e.enums)
}

func (e *enumDescriptors) Get(i int) protoreflect.EnumDescriptor {
	en := e.enums[i]
	return e.l.asEnumDescriptor(en, e.file, e.parent, i, e.prefix+en.GetName())
}

func (e *enumDescriptors) ByName(s protoreflect.Name) protoreflect.EnumDescriptor {
	for i, en := range e.enums {
		if en.GetName() == string(s) {
			return e.Get(i)
		}
	}
	return nil
}

type enumDescriptor struct {
	protoreflect.EnumDescriptor
	file   *fileDescriptor
	parent protoreflect.Descriptor
	index  int
	proto  *descriptorpb.EnumDescriptorProto
	fqn    string
	l      *linker
}

func (l *linker) asEnumDescriptor(ed *descriptorpb.EnumDescriptorProto, file *fileDescriptor, parent protoreflect.Descriptor, index int, fqn string) *enumDescriptor {
	if ret := l.descriptors[ed]; ret != nil {
		return ret.(*enumDescriptor)
	}
	ret := &enumDescriptor{file: file, parent: parent, index: index, proto: ed, fqn: fqn, l: l}
	l.descriptors[ed] = ret
	return ret
}

func (e *enumDescriptor) ParentFile() protoreflect.FileDescriptor {
	return e.file
}

func (e *enumDescriptor) Parent() protoreflect.Descriptor {
	return e.parent
}

func (e *enumDescriptor) Index() int {
	return e.index
}

func (e *enumDescriptor) Syntax() protoreflect.Syntax {
	return e.file.Syntax()
}

func (e *enumDescriptor) Name() protoreflect.Name {
	return protoreflect.Name(e.proto.GetName())
}

func (e *enumDescriptor) FullName() protoreflect.FullName {
	return protoreflect.FullName(e.fqn)
}

func (e *enumDescriptor) IsPlaceholder() bool {
	return false
}

func (e *enumDescriptor) Options() protoreflect.ProtoMessage {
	return e.proto.Options
}

func (e *enumDescriptor) Values() protoreflect.EnumValueDescriptors {
	// Unlike all other elements, the fully-qualified name of enum values
	// is NOT scoped to their parent element (the enum), but rather to
	// the enum's parent element. This follows C++ scoping rules for
	// enum values.
	prefix := strings.TrimSuffix(e.fqn, e.proto.GetName())
	return &enValDescriptors{file: e.file, parent: e, vals: e.proto.GetValue(), prefix: prefix, l: e.l}
}

func (e *enumDescriptor) ReservedNames() protoreflect.Names {
	return names{s: e.proto.ReservedName}
}

func (e *enumDescriptor) ReservedRanges() protoreflect.EnumRanges {
	return enumRanges{s: e.proto.ReservedRange}
}

type enumRanges struct {
	protoreflect.EnumRanges
	s []*descriptorpb.EnumDescriptorProto_EnumReservedRange
}

func (e enumRanges) Len() int {
	return len(e.s)
}

func (e enumRanges) Get(i int) [2]protoreflect.EnumNumber {
	r := e.s[i]
	return [2]protoreflect.EnumNumber{
		protoreflect.EnumNumber(r.GetStart()),
		protoreflect.EnumNumber(r.GetEnd()),
	}
}

func (e enumRanges) Has(n protoreflect.EnumNumber) bool {
	for _, r := range e.s {
		if r.GetStart() <= int32(n) && r.GetEnd() >= int32(n) {
			return true
		}
	}
	return false
}

type enValDescriptors struct {
	protoreflect.EnumValueDescriptors
	file   *fileDescriptor
	parent *enumDescriptor
	vals   []*descriptorpb.EnumValueDescriptorProto
	prefix string
	l      *linker
}

func (e *enValDescriptors) Len() int {
	return len(e.vals)
}

func (e *enValDescriptors) Get(i int) protoreflect.EnumValueDescriptor {
	val := e.vals[i]
	return e.l.asEnumValueDescriptor(val, e.file, e.parent, i, e.prefix+val.GetName())
}

func (e *enValDescriptors) ByName(s protoreflect.Name) protoreflect.EnumValueDescriptor {
	for i, en := range e.vals {
		if en.GetName() == string(s) {
			return e.Get(i)
		}
	}
	return nil
}

func (e *enValDescriptors) ByNumber(n protoreflect.EnumNumber) protoreflect.EnumValueDescriptor {
	for i, en := range e.vals {
		if en.GetNumber() == int32(n) {
			return e.Get(i)
		}
	}
	return nil
}

type enValDescriptor struct {
	protoreflect.EnumValueDescriptor
	file   *fileDescriptor
	parent *enumDescriptor
	index  int
	proto  *descriptorpb.EnumValueDescriptorProto
	fqn    string
}

func (l *linker) asEnumValueDescriptor(ed *descriptorpb.EnumValueDescriptorProto, file *fileDescriptor, parent *enumDescriptor, index int, fqn string) *enValDescriptor {
	if ret := l.descriptors[ed]; ret != nil {
		return ret.(*enValDescriptor)
	}
	ret := &enValDescriptor{file: file, parent: parent, index: index, proto: ed, fqn: fqn}
	l.descriptors[ed] = ret
	return ret
}

func (e *enValDescriptor) ParentFile() protoreflect.FileDescriptor {
	return e.file
}

func (e *enValDescriptor) Parent() protoreflect.Descriptor {
	return e.parent
}

func (e *enValDescriptor) Index() int {
	return e.index
}

func (e *enValDescriptor) Syntax() protoreflect.Syntax {
	return e.file.Syntax()
}

func (e *enValDescriptor) Name() protoreflect.Name {
	return protoreflect.Name(e.proto.GetName())
}

func (e *enValDescriptor) FullName() protoreflect.FullName {
	return protoreflect.FullName(e.fqn)
}

func (e *enValDescriptor) IsPlaceholder() bool {
	return false
}

func (e *enValDescriptor) Options() protoreflect.ProtoMessage {
	return e.proto.Options
}

func (e *enValDescriptor) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(e.proto.GetNumber())
}

type extDescriptors struct {
	protoreflect.ExtensionDescriptors
	file   *fileDescriptor
	parent protoreflect.Descriptor
	exts   []*descriptorpb.FieldDescriptorProto
	prefix string
	l      *linker
}

func (e *extDescriptors) Len() int {
	return len(e.exts)
}

func (e *extDescriptors) Get(i int) protoreflect.ExtensionDescriptor {
	fld := e.exts[i]
	return e.l.asFieldDescriptor(fld, e.file, e.parent, i, e.prefix+fld.GetName())
}

func (e *extDescriptors) ByName(s protoreflect.Name) protoreflect.ExtensionDescriptor {
	for i, ext := range e.exts {
		if ext.GetName() == string(s) {
			return e.Get(i)
		}
	}
	return nil
}

type fldDescriptors struct {
	protoreflect.FieldDescriptors
	file   *fileDescriptor
	parent protoreflect.Descriptor
	fields []*descriptorpb.FieldDescriptorProto
	prefix string
	l      *linker
}

func (f *fldDescriptors) Len() int {
	return len(f.fields)
}

func (f *fldDescriptors) Get(i int) protoreflect.FieldDescriptor {
	fld := f.fields[i]
	return f.l.asFieldDescriptor(fld, f.file, f.parent, i, f.prefix+fld.GetName())
}

func (f *fldDescriptors) ByName(s protoreflect.Name) protoreflect.FieldDescriptor {
	for i, fld := range f.fields {
		if fld.GetName() == string(s) {
			return f.Get(i)
		}
	}
	return nil
}

func (f *fldDescriptors) ByJSONName(s string) protoreflect.FieldDescriptor {
	for i, fld := range f.fields {
		if fld.GetJsonName() == s {
			return f.Get(i)
		}
	}
	return nil
}

func (f *fldDescriptors) ByTextName(s string) protoreflect.FieldDescriptor {
	return f.ByName(protoreflect.Name(s))
}

func (f *fldDescriptors) ByNumber(n protoreflect.FieldNumber) protoreflect.FieldDescriptor {
	for i, fld := range f.fields {
		if fld.GetNumber() == int32(n) {
			return f.Get(i)
		}
	}
	return nil
}

type fldDescriptor struct {
	protoreflect.FieldDescriptor
	file   *fileDescriptor
	parent protoreflect.Descriptor
	index  int
	proto  *descriptorpb.FieldDescriptorProto
	fqn    string
	l      *linker
}

func (l *linker) asFieldDescriptor(fd *descriptorpb.FieldDescriptorProto, file *fileDescriptor, parent protoreflect.Descriptor, index int, fqn string) *fldDescriptor {
	if ret := l.descriptors[fd]; ret != nil {
		return ret.(*fldDescriptor)
	}
	ret := &fldDescriptor{file: file, parent: parent, index: index, proto: fd, fqn: fqn, l: l}
	l.descriptors[fd] = ret
	return ret
}

func (f *fldDescriptor) ParentFile() protoreflect.FileDescriptor {
	return f.file
}

func (f *fldDescriptor) Parent() protoreflect.Descriptor {
	return f.parent
}

func (f *fldDescriptor) Index() int {
	return f.index
}

func (f *fldDescriptor) Syntax() protoreflect.Syntax {
	return f.file.Syntax()
}

func (f *fldDescriptor) Name() protoreflect.Name {
	return protoreflect.Name(f.proto.GetName())
}

func (f *fldDescriptor) FullName() protoreflect.FullName {
	return protoreflect.FullName(f.fqn)
}

func (f *fldDescriptor) IsPlaceholder() bool {
	return false
}

func (f *fldDescriptor) Options() protoreflect.ProtoMessage {
	return f.proto.Options
}

func (f *fldDescriptor) Number() protoreflect.FieldNumber {
	return protoreflect.FieldNumber(f.proto.GetNumber())
}

func (f *fldDescriptor) Cardinality() protoreflect.Cardinality {
	switch f.proto.GetLabel() {
	case descriptorpb.FieldDescriptorProto_LABEL_REPEATED:
		return protoreflect.Repeated
	case descriptorpb.FieldDescriptorProto_LABEL_REQUIRED:
		return protoreflect.Required
	case descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL:
		return protoreflect.Optional
	default:
		return 0
	}
}

func (f *fldDescriptor) Kind() protoreflect.Kind {
	return protoreflect.Kind(f.proto.GetType())
}

func (f *fldDescriptor) HasJSONName() bool {
	return f.proto.JsonName != nil
}

func (f *fldDescriptor) JSONName() string {
	return f.proto.GetJsonName()
}

func (f *fldDescriptor) TextName() string {
	return string(f.Name())
}

func (f *fldDescriptor) HasPresence() bool {
	if f.Syntax() == protoreflect.Proto2 {
		return true
	}
	if f.Kind() == protoreflect.MessageKind || f.Kind() == protoreflect.GroupKind {
		return true
	}
	if f.proto.OneofIndex != nil {
		return true
	}
	return false
}

func (f *fldDescriptor) IsExtension() bool {
	return f.proto.GetExtendee() != ""
}

func (f *fldDescriptor) HasOptionalKeyword() bool {
	return f.proto.Label != nil && f.proto.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
}

func (f *fldDescriptor) IsWeak() bool {
	return f.proto.Options.GetWeak()
}

func (f *fldDescriptor) IsPacked() bool {
	return f.proto.Options.GetPacked()
}

func (f *fldDescriptor) IsList() bool {
	if f.proto.GetLabel() != descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return false
	}
	return !f.isMapEntry()
}

func (f *fldDescriptor) IsMap() bool {
	if f.proto.GetLabel() != descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return false
	}
	if f.IsExtension() {
		return false
	}
	return f.isMapEntry()
}

func (f *fldDescriptor) isMapEntry() bool {
	if f.proto.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		return false
	}
	return f.Message().IsMapEntry()
}

func (f *fldDescriptor) MapKey() protoreflect.FieldDescriptor {
	if !f.IsMap() {
		return nil
	}
	return f.Message().Fields().ByNumber(1)
}

func (f *fldDescriptor) MapValue() protoreflect.FieldDescriptor {
	if !f.IsMap() {
		return nil
	}
	return f.Message().Fields().ByNumber(2)
}

func (f *fldDescriptor) HasDefault() bool {
	// NB: no need to implement...
	return false
}

func (f *fldDescriptor) Default() protoreflect.Value {
	switch f.Kind() {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind,
		protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(0)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(0)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(0)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(0)
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(0)
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat32(0)
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(false)
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes(nil)
	case protoreflect.StringKind:
		return protoreflect.ValueOfString("")
	case protoreflect.EnumKind:
		return protoreflect.ValueOfEnum(f.Enum().Values().Get(0).Number())
	case protoreflect.GroupKind, protoreflect.MessageKind:
		return protoreflect.ValueOfMessage(dynamicpb.NewMessage(f.Message()))
	default:
		panic(fmt.Sprintf("unknown kind: %v", f.Kind()))
	}
}

func (f *fldDescriptor) DefaultEnumValue() protoreflect.EnumValueDescriptor {
	ed := f.Enum()
	if ed == nil {
		return nil
	}
	return ed.Values().Get(0)
}

func (f *fldDescriptor) ContainingOneof() protoreflect.OneofDescriptor {
	if f.IsExtension() {
		return nil
	}
	if f.proto.OneofIndex == nil {
		return nil
	}
	parent := f.parent.(*msgDescriptor)
	index := int(f.proto.GetOneofIndex())
	ood := parent.proto.OneofDecl[index]
	fqn := parent.fqn + "." + ood.GetName()
	return f.l.asOneOfDescriptor(ood, f.file, parent, index, fqn)
}

func (f *fldDescriptor) ContainingMessage() protoreflect.MessageDescriptor {
	if !f.IsExtension() {
		return f.parent.(*msgDescriptor)
	}
	return f.l.findMessageType(f.file.proto, f.proto.GetExtendee())
}

func (f *fldDescriptor) Enum() protoreflect.EnumDescriptor {
	if f.proto.GetType() != descriptorpb.FieldDescriptorProto_TYPE_ENUM {
		return nil
	}
	return f.l.findEnumType(f.file.proto, f.proto.GetTypeName())
}

func (f *fldDescriptor) Message() protoreflect.MessageDescriptor {
	if f.proto.GetType() != descriptorpb.FieldDescriptorProto_TYPE_MESSAGE &&
		f.proto.GetType() != descriptorpb.FieldDescriptorProto_TYPE_GROUP {
		return nil
	}
	return f.l.findMessageType(f.file.proto, f.proto.GetTypeName())
}

func (f *fldDescriptor) Type() protoreflect.ExtensionType {
	return dynamicpb.NewExtensionType(f)
}

func (f *fldDescriptor) Descriptor() protoreflect.ExtensionDescriptor {
	return f
}

type oneofDescriptors struct {
	protoreflect.OneofDescriptors
	file   *fileDescriptor
	parent *msgDescriptor
	oneofs []*descriptorpb.OneofDescriptorProto
	prefix string
	l      *linker
}

func (o *oneofDescriptors) Len() int {
	return len(o.oneofs)
}

func (o *oneofDescriptors) Get(i int) protoreflect.OneofDescriptor {
	oo := o.oneofs[i]
	return o.l.asOneOfDescriptor(oo, o.file, o.parent, i, o.prefix+oo.GetName())
}

func (o *oneofDescriptors) ByName(s protoreflect.Name) protoreflect.OneofDescriptor {
	for i, oo := range o.oneofs {
		if oo.GetName() == string(s) {
			return o.Get(i)
		}
	}
	return nil
}

type oneofDescriptor struct {
	protoreflect.OneofDescriptor
	file   *fileDescriptor
	parent *msgDescriptor
	index  int
	proto  *descriptorpb.OneofDescriptorProto
	fqn    string
	l      *linker
}

func (l *linker) asOneOfDescriptor(ood *descriptorpb.OneofDescriptorProto, file *fileDescriptor, parent *msgDescriptor, index int, fqn string) *oneofDescriptor {
	if ret := l.descriptors[ood]; ret != nil {
		return ret.(*oneofDescriptor)
	}
	ret := &oneofDescriptor{file: file, parent: parent, index: index, proto: ood, fqn: fqn, l: l}
	l.descriptors[ood] = ret
	return ret
}

func (o *oneofDescriptor) ParentFile() protoreflect.FileDescriptor {
	return o.file
}

func (o *oneofDescriptor) Parent() protoreflect.Descriptor {
	return o.parent
}

func (o *oneofDescriptor) Index() int {
	return o.index
}

func (o *oneofDescriptor) Syntax() protoreflect.Syntax {
	return o.file.Syntax()
}

func (o *oneofDescriptor) Name() protoreflect.Name {
	return protoreflect.Name(o.proto.GetName())
}

func (o *oneofDescriptor) FullName() protoreflect.FullName {
	return protoreflect.FullName(o.fqn)
}

func (o *oneofDescriptor) IsPlaceholder() bool {
	return false
}

func (o *oneofDescriptor) Options() protoreflect.ProtoMessage {
	return o.proto.Options
}

func (o *oneofDescriptor) IsSynthetic() bool {
	// NB: no need to implement...
	return false
}

func (o *oneofDescriptor) Fields() protoreflect.FieldDescriptors {
	var fields []*descriptorpb.FieldDescriptorProto
	for _, fld := range o.parent.proto.GetField() {
		if fld.OneofIndex != nil && int(fld.GetOneofIndex()) == o.index {
			fields = append(fields, fld)
		}
	}
	return &fldDescriptors{file: o.file, parent: o.parent, fields: fields, prefix: o.parent.fqn + ".", l: o.l}
}

type svcDescriptors struct {
	protoreflect.ServiceDescriptors
	file   *fileDescriptor
	svcs   []*descriptorpb.ServiceDescriptorProto
	prefix string
	l      *linker
}

func (s *svcDescriptors) Len() int {
	return len(s.svcs)
}

func (s *svcDescriptors) Get(i int) protoreflect.ServiceDescriptor {
	svc := s.svcs[i]
	return &svcDescriptor{file: s.file, index: i, fqn: s.prefix + svc.GetName(), proto: svc, l: s.l}
}

func (s *svcDescriptors) ByName(n protoreflect.Name) protoreflect.ServiceDescriptor {
	for i, svc := range s.svcs {
		if svc.GetName() == string(n) {
			return s.Get(i)
		}
	}
	return nil
}

type svcDescriptor struct {
	protoreflect.ServiceDescriptor
	file  *fileDescriptor
	index int
	proto *descriptorpb.ServiceDescriptorProto
	fqn   string
	l     *linker
}

func (s *svcDescriptor) ParentFile() protoreflect.FileDescriptor {
	return s.file
}

func (s *svcDescriptor) Parent() protoreflect.Descriptor {
	return s.file
}

func (s *svcDescriptor) Index() int {
	return s.index
}

func (s *svcDescriptor) Syntax() protoreflect.Syntax {
	return s.file.Syntax()
}

func (s *svcDescriptor) Name() protoreflect.Name {
	return protoreflect.Name(s.proto.GetName())
}

func (s *svcDescriptor) FullName() protoreflect.FullName {
	return protoreflect.FullName(s.fqn)
}

func (s *svcDescriptor) IsPlaceholder() bool {
	return false
}

func (s *svcDescriptor) Options() protoreflect.ProtoMessage {
	return s.proto.Options
}

func (s *svcDescriptor) Methods() protoreflect.MethodDescriptors {
	return &mtdDescriptors{file: s.file, parent: s, mtds: s.proto.GetMethod(), prefix: s.fqn + ".", l: s.l}
}

type mtdDescriptors struct {
	protoreflect.MethodDescriptors
	file   *fileDescriptor
	parent *svcDescriptor
	mtds   []*descriptorpb.MethodDescriptorProto
	prefix string
	l      *linker
}

func (m *mtdDescriptors) Len() int {
	return len(m.mtds)
}

func (m *mtdDescriptors) Get(i int) protoreflect.MethodDescriptor {
	mtd := m.mtds[i]
	return &mtdDescriptor{file: m.file, parent: m.parent, index: i, fqn: m.prefix + mtd.GetName(), proto: mtd, l: m.l}
}

func (m *mtdDescriptors) ByName(n protoreflect.Name) protoreflect.MethodDescriptor {
	for i, svc := range m.mtds {
		if svc.GetName() == string(n) {
			return m.Get(i)
		}
	}
	return nil
}

type mtdDescriptor struct {
	protoreflect.MethodDescriptor
	file   *fileDescriptor
	parent *svcDescriptor
	index  int
	proto  *descriptorpb.MethodDescriptorProto
	fqn    string
	l      *linker
}

func (m *mtdDescriptor) ParentFile() protoreflect.FileDescriptor {
	return m.file
}

func (m *mtdDescriptor) Parent() protoreflect.Descriptor {
	return m.parent
}

func (m *mtdDescriptor) Index() int {
	return m.index
}

func (m *mtdDescriptor) Syntax() protoreflect.Syntax {
	return m.file.Syntax()
}

func (m *mtdDescriptor) Name() protoreflect.Name {
	return protoreflect.Name(m.proto.GetName())
}

func (m *mtdDescriptor) FullName() protoreflect.FullName {
	return protoreflect.FullName(m.fqn)
}

func (m *mtdDescriptor) IsPlaceholder() bool {
	return false
}

func (m *mtdDescriptor) Options() protoreflect.ProtoMessage {
	return m.proto.Options
}

func (m *mtdDescriptor) Input() protoreflect.MessageDescriptor {
	return m.l.findMessageType(m.file.proto, m.proto.GetInputType())
}

func (m *mtdDescriptor) Output() protoreflect.MessageDescriptor {
	return m.l.findMessageType(m.file.proto, m.proto.GetOutputType())
}

func (m *mtdDescriptor) IsStreamingClient() bool {
	return m.proto.GetClientStreaming()
}

func (m *mtdDescriptor) IsStreamingServer() bool {
	return m.proto.GetServerStreaming()
}

func (l *linker) findMessageType(entryPoint *descriptorpb.FileDescriptorProto, fqn string) protoreflect.MessageDescriptor {
	fqn = strings.TrimPrefix(fqn, ".")
	fd, d := l.findElement(entryPoint, fqn)
	msg, ok := d.(*descriptorpb.DescriptorProto)
	if !ok {
		return nil
	}
	if ret := l.descriptors[msg]; ret != nil {
		// don't bother searching for parent if we don't need it...
		return ret.(*msgDescriptor)
	}
	file := l.asFileDescriptor(fd)
	parent, index := findParent(l, file, fqn)
	return l.asMessageDescriptor(msg, file, parent, index, fqn)
}

func (l *linker) findEnumType(entryPoint *descriptorpb.FileDescriptorProto, fqn string) protoreflect.EnumDescriptor {
	fqn = strings.TrimPrefix(fqn, ".")
	fd, d := l.findElement(entryPoint, fqn)
	en, ok := d.(*descriptorpb.EnumDescriptorProto)
	if !ok {
		return nil
	}
	if ret := l.descriptors[en]; ret != nil {
		// don't bother searching for parent if we don't need it...
		return ret.(*enumDescriptor)
	}
	file := l.asFileDescriptor(fd)
	parent, index := findParent(l, file, fqn)
	return l.asEnumDescriptor(en, file, parent, index, fqn)
}

func (l *linker) findExtension(entryPoint *descriptorpb.FileDescriptorProto, fqn string) protoreflect.ExtensionDescriptor {
	fqn = strings.TrimPrefix(fqn, ".")
	fd, d := l.findElement(entryPoint, fqn)
	fld, ok := d.(*descriptorpb.FieldDescriptorProto)
	if !ok {
		return nil
	}
	if fld.GetExtendee() == "" {
		// not an extension
		return nil
	}
	if ret := l.descriptors[fld]; ret != nil {
		// don't bother searching for parent if we don't need it...
		return ret.(*fldDescriptor)
	}
	file := l.asFileDescriptor(fd)
	parent, index := findParent(l, file, fqn)
	return l.asFieldDescriptor(fld, file, parent, index, fqn)
}

func (l *linker) findElement(entryPoint *descriptorpb.FileDescriptorProto, fqn string) (*descriptorpb.FileDescriptorProto, proto.Message) {
	importedFd, srcFd, d := l.findElementRecursive(entryPoint, fqn, false, map[*descriptorpb.FileDescriptorProto]struct{}{})
	if importedFd != nil {
		l.markUsed(entryPoint, importedFd)
	}
	return srcFd, d
}

func (l *linker) findElementRecursive(fd *descriptorpb.FileDescriptorProto, fqn string, public bool, checked map[*descriptorpb.FileDescriptorProto]struct{}) (imported *descriptorpb.FileDescriptorProto, final *descriptorpb.FileDescriptorProto, element proto.Message) {
	if _, ok := checked[fd]; ok {
		return nil, nil, nil
	}
	checked[fd] = struct{}{}
	d := l.descriptorPool[fd][fqn]
	if d != nil {
		// not imported, but present in fd
		return nil, fd, d
	}

	// When public = false, we are searching only directly imported symbols. But we
	// also need to search transitive public imports due to semantics of public imports.
	if public {
		for _, dep := range fd.GetPublicDependency() {
			depFile := l.files[fd.GetDependency()[dep]]
			if depFile == nil {
				return nil, nil, nil
			}
			depFd := depFile.fd
			_, srcFd, d := l.findElementRecursive(depFd, fqn, true, checked)
			if d != nil {
				return depFd, srcFd, d
			}
		}
	} else {
		for _, dep := range fd.GetDependency() {
			depFile := l.files[dep]
			if depFile == nil {
				return nil, nil, nil
			}
			depFd := depFile.fd
			_, srcFd, d := l.findElementRecursive(depFd, fqn, true, checked)
			if d != nil {
				return depFd, srcFd, d
			}
		}
	}
	return nil, nil, nil
}

func (l *linker) markUsed(entryPoint, used *descriptorpb.FileDescriptorProto) {
	importsForFile := l.usedImports[entryPoint]
	if importsForFile == nil {
		importsForFile = map[string]struct{}{}
		l.usedImports[entryPoint] = importsForFile
	}
	importsForFile[used.GetName()] = struct{}{}
}

func findParent(l *linker, file *fileDescriptor, fqn string) (protoreflect.Descriptor, int) {
	names := strings.Split(strings.TrimPrefix(fqn, file.prefix), ".")
	if len(names) == 1 {
		for i, en := range file.proto.EnumType {
			if en.GetName() == names[0] {
				return file, i
			}
		}
		for i, ext := range file.proto.Extension {
			if ext.GetName() == names[0] {
				return file, i
			}
		}
	}
	for i, msg := range file.proto.MessageType {
		if msg.GetName() == names[0] {
			if len(names) == 1 {
				return file, i
			}
			md := l.asMessageDescriptor(msg, file, file, i, file.prefix+msg.GetName())
			return findParentMessage(l, md, names[1:])
		}
	}
	return nil, 0
}

func findParentMessage(l *linker, msg *msgDescriptor, names []string) (protoreflect.MessageDescriptor, int) {
	if len(names) == 1 {
		for i, en := range msg.proto.EnumType {
			if en.GetName() == names[0] {
				return msg, i
			}
		}
		for i, ext := range msg.proto.Extension {
			if ext.GetName() == names[0] {
				return msg, i
			}
		}
		for i, fld := range msg.proto.Field {
			if fld.GetName() == names[0] {
				return msg, i
			}
		}
	}
	for i, nested := range msg.proto.NestedType {
		if nested.GetName() == names[0] {
			if len(names) == 1 {
				return msg, i
			}
			md := l.asMessageDescriptor(nested, msg.file, msg, i, msg.fqn+"."+nested.GetName())
			return findParentMessage(l, md, names[1:])
		}
	}
	return nil, 0
}
