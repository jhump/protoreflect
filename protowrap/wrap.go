package protowrap

import (
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// FromFileDescriptorProto is identical to [protodesc.NewFile] except that it
// returns a FileWrapper, not just a [protoreflect.FileDescriptor].
func FromFileDescriptorProto(fd *descriptorpb.FileDescriptorProto, deps protoresolve.DependencyResolver) (FileWrapper, error) {
	file, err := protodesc.NewFile(fd, deps)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{FileDescriptor: file, proto: fd, srcLocs: srcLocsWrapper{SourceLocations: file.SourceLocations()}}, nil
}

// AddToRegistry converts the given proto to a FileWrapper, using reg to resolve
// any imports, and also registers the wrapper with reg.
func AddToRegistry(fd *descriptorpb.FileDescriptorProto, reg protoresolve.DescriptorRegistry) (FileWrapper, error) {
	file, err := FromFileDescriptorProto(fd, reg)
	if err != nil {
		return nil, err
	}
	if err := reg.RegisterFile(file); err != nil {
		return nil, err
	}
	return file, nil
}

// FromFileDescriptorSet is identical to [protodesc.NewFiles] except that all
// descriptors registered with the returned resolver will be FileWrapper instances.
func FromFileDescriptorSet(files *descriptorpb.FileDescriptorSet) (protoresolve.Resolver, error) {
	protosByPath := map[string]*descriptorpb.FileDescriptorProto{}
	for _, fd := range files.File {
		if _, ok := protosByPath[fd.GetName()]; ok {
			return nil, fmt.Errorf("file %q appears in set more than once", fd.GetName())
		}
		protosByPath[fd.GetName()] = fd
	}
	reg := &protoresolve.Registry{}
	for _, fd := range files.File {
		if err := resolveFile(fd, protosByPath, reg); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

func resolveFile(fd *descriptorpb.FileDescriptorProto, protosByPath map[string]*descriptorpb.FileDescriptorProto, reg *protoresolve.Registry) error {
	if _, err := reg.FindFileByPath(fd.GetName()); err == nil {
		// already resolved
		return nil
	}
	// resolve all dependencies
	for _, dep := range fd.GetDependency() {
		depFile := protosByPath[dep]
		if depFile == nil {
			return fmt.Errorf("set is missing file %q (imported by %q)", dep, fd.GetName())
		}
		if err := resolveFile(depFile, protosByPath, reg); err != nil {
			return err
		}
	}
	_, err := AddToRegistry(fd, reg)
	return err
}

type fileWrapper struct {
	protoreflect.FileDescriptor
	proto   *descriptorpb.FileDescriptorProto
	srcLocs srcLocsWrapper
	msgs    msgsWrapper
	enums   enumsWrapper
	exts    extsWrapper
	svcs    svcsWrapper
}

var _ ProtoWrapper = &fileWrapper{}
var _ FileWrapper = &fileWrapper{}
var _ WrappedDescriptor = &fileWrapper{}

func (w *fileWrapper) Unwrap() protoreflect.Descriptor {
	return w.FileDescriptor
}

func (w *fileWrapper) ParentFile() protoreflect.FileDescriptor {
	return w
}

func (w *fileWrapper) Parent() protoreflect.Descriptor {
	return nil
}

func (w *fileWrapper) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *fileWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *fileWrapper) FileDescriptorProto() *descriptorpb.FileDescriptorProto {
	return w.proto
}

func (w *fileWrapper) Messages() protoreflect.MessageDescriptors {
	w.msgs.initFromFile(w)
	return &w.msgs
}

func (w *fileWrapper) Enums() protoreflect.EnumDescriptors {
	w.enums.initFromFile(w)
	return &w.enums
}

func (w *fileWrapper) Extensions() protoreflect.ExtensionDescriptors {
	w.exts.initFromFile(w)
	return &w.exts
}

func (w *fileWrapper) Services() protoreflect.ServiceDescriptors {
	w.svcs.initFromFile(w)
	return &w.svcs
}

func (w *fileWrapper) SourceLocations() protoreflect.SourceLocations {
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
	msgs []*msgWrapper
}

func (w *msgsWrapper) initFromFile(parent *fileWrapper) {
	w.init.Do(func() {
		w.doInit(parent, parent.FileDescriptor.Messages(), parent.proto.MessageType)
	})
}

func (w *msgsWrapper) initFromMessage(parent *msgWrapper) {
	w.init.Do(func() {
		w.doInit(parent, parent.MessageDescriptor.Messages(), parent.proto.NestedType)
	})
}

func (w *msgsWrapper) doInit(parent ProtoWrapper, msgs protoreflect.MessageDescriptors, protos []*descriptorpb.DescriptorProto) {
	length := msgs.Len()
	w.MessageDescriptors = msgs
	w.msgs = make([]*msgWrapper, length)
	for i := 0; i < length; i++ {
		msg := msgs.Get(i)
		w.msgs[i] = &msgWrapper{MessageDescriptor: msg, parent: parent, proto: protos[i]}
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
	enums []*enumWrapper
}

func (w *enumsWrapper) initFromFile(parent *fileWrapper) {
	w.init.Do(func() {
		w.doInit(parent, parent.FileDescriptor.Enums(), parent.proto.EnumType)
	})
}

func (w *enumsWrapper) initFromMessage(parent *msgWrapper) {
	w.init.Do(func() {
		w.doInit(parent, parent.MessageDescriptor.Enums(), parent.proto.EnumType)
	})
}

func (w *enumsWrapper) doInit(parent ProtoWrapper, enums protoreflect.EnumDescriptors, protos []*descriptorpb.EnumDescriptorProto) {
	length := enums.Len()
	w.EnumDescriptors = enums
	w.enums = make([]*enumWrapper, length)
	for i := 0; i < length; i++ {
		en := enums.Get(i)
		w.enums[i] = &enumWrapper{EnumDescriptor: en, parent: parent, proto: protos[i]}
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
	exts []FieldWrapper
}

func (w *extsWrapper) initFromFile(parent *fileWrapper) {
	w.init.Do(func() {
		w.doInit(parent, parent.FileDescriptor.Extensions(), parent.proto.Extension)
	})
}

func (w *extsWrapper) initFromMessage(parent *msgWrapper) {
	w.init.Do(func() {
		w.doInit(parent, parent.MessageDescriptor.Extensions(), parent.proto.Extension)
	})
}

func (w *extsWrapper) doInit(parent ProtoWrapper, exts protoreflect.ExtensionDescriptors, protos []*descriptorpb.FieldDescriptorProto) {
	length := exts.Len()
	w.ExtensionDescriptors = exts
	w.exts = make([]FieldWrapper, length)
	for i := 0; i < length; i++ {
		ext := exts.Get(i)
		fld := &fieldWrapper{FieldDescriptor: ext, parent: parent, proto: protos[i]}
		w.exts[i] = &extWrapper{fieldWrapper: fld, extType: protoresolve.ExtensionType(ext)}
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
	svcs []*svcWrapper
}

func (w *svcsWrapper) initFromFile(parent *fileWrapper) {
	w.init.Do(func() {
		svcs := parent.FileDescriptor.Services()
		length := svcs.Len()
		w.ServiceDescriptors = svcs
		w.svcs = make([]*svcWrapper, length)
		for i := 0; i < length; i++ {
			svc := svcs.Get(i)
			w.svcs[i] = &svcWrapper{ServiceDescriptor: svc, parent: parent, proto: parent.proto.Service[i]}
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
	fields []*fieldWrapper
}

func (w *fieldsWrapper) initFromMessage(parent *msgWrapper) {
	w.init.Do(func() {
		fields := parent.MessageDescriptor.Fields()
		length := fields.Len()
		w.FieldDescriptors = fields
		w.fields = make([]*fieldWrapper, length)
		for i := 0; i < length; i++ {
			field := fields.Get(i)
			w.fields[i] = &fieldWrapper{FieldDescriptor: field, parent: parent, proto: parent.proto.Field[i]}
		}
	})
}

func (w *fieldsWrapper) initFromOneof(parent *oneofWrapper) {
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
	oos []*oneofWrapper
}

func (w *oneofsWrapper) initFromMessage(parent *msgWrapper) {
	w.init.Do(func() {
		oos := parent.MessageDescriptor.Oneofs()
		length := oos.Len()
		w.OneofDescriptors = oos
		w.oos = make([]*oneofWrapper, length)
		for i := 0; i < length; i++ {
			oo := oos.Get(i)
			w.oos[i] = &oneofWrapper{OneofDescriptor: oo, parent: parent, proto: parent.proto.OneofDecl[i]}
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
	vals []*enumValueWrapper
}

func (w *enumValuesWrapper) initFromEnum(parent *enumWrapper) {
	w.init.Do(func() {
		vals := parent.EnumDescriptor.Values()
		length := vals.Len()
		w.EnumValueDescriptors = vals
		w.vals = make([]*enumValueWrapper, length)
		for i := 0; i < length; i++ {
			val := vals.Get(i)
			w.vals[i] = &enumValueWrapper{EnumValueDescriptor: val, parent: parent, proto: parent.proto.Value[i]}
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
	mtds []*mtdWrapper
}

func (w *mtdsWrapper) initFromSvc(parent *svcWrapper) {
	w.init.Do(func() {
		mtds := parent.ServiceDescriptor.Methods()
		length := mtds.Len()
		w.MethodDescriptors = mtds
		w.mtds = make([]*mtdWrapper, length)
		for i := 0; i < length; i++ {
			mtd := mtds.Get(i)
			w.mtds[i] = &mtdWrapper{MethodDescriptor: mtd, parent: parent, proto: parent.proto.Method[i]}
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

type msgWrapper struct {
	protoreflect.MessageDescriptor
	parent ProtoWrapper // either *fileWrapper or *msgWrapper
	proto  *descriptorpb.DescriptorProto
	fields fieldsWrapper
	oneofs oneofsWrapper
	msgs   msgsWrapper
	enums  enumsWrapper
	exts   extsWrapper
}

var _ ProtoWrapper = &msgWrapper{}
var _ MessageWrapper = &msgWrapper{}
var _ WrappedDescriptor = &msgWrapper{}

func (w *msgWrapper) Unwrap() protoreflect.Descriptor {
	return w.MessageDescriptor
}

func (w *msgWrapper) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *msgWrapper) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *msgWrapper) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *msgWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *msgWrapper) MessageDescriptorProto() *descriptorpb.DescriptorProto {
	return w.proto
}

func (w *msgWrapper) Fields() protoreflect.FieldDescriptors {
	w.fields.initFromMessage(w)
	return &w.fields
}

func (w *msgWrapper) Oneofs() protoreflect.OneofDescriptors {
	w.oneofs.initFromMessage(w)
	return &w.oneofs
}

func (w *msgWrapper) Messages() protoreflect.MessageDescriptors {
	w.msgs.initFromMessage(w)
	return &w.msgs
}

func (w *msgWrapper) Enums() protoreflect.EnumDescriptors {
	w.enums.initFromMessage(w)
	return &w.enums
}

func (w *msgWrapper) Extensions() protoreflect.ExtensionDescriptors {
	w.exts.initFromMessage(w)
	return &w.exts
}

type fieldWrapper struct {
	protoreflect.FieldDescriptor
	parent ProtoWrapper // could be *fileWrapper or *msgWrapper
	proto  *descriptorpb.FieldDescriptorProto

	init             sync.Once
	mapKey, mapValue protoreflect.FieldDescriptor
	containingOneof  protoreflect.OneofDescriptor
	defaultEnumValue protoreflect.EnumValueDescriptor
	containingMsg    protoreflect.MessageDescriptor
	enumType         protoreflect.EnumDescriptor
	msgType          protoreflect.MessageDescriptor
}

var _ ProtoWrapper = &fieldWrapper{}
var _ FieldWrapper = &fieldWrapper{}
var _ WrappedDescriptor = &fieldWrapper{}

func (w *fieldWrapper) Unwrap() protoreflect.Descriptor {
	return w.FieldDescriptor
}

func (w *fieldWrapper) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *fieldWrapper) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *fieldWrapper) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *fieldWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *fieldWrapper) FieldDescriptorProto() *descriptorpb.FieldDescriptorProto {
	return w.proto
}

func (w *fieldWrapper) MapKey() protoreflect.FieldDescriptor {
	w.doInit()
	return w.mapKey
}

func (w *fieldWrapper) MapValue() protoreflect.FieldDescriptor {
	w.doInit()
	return w.mapValue
}

func (w *fieldWrapper) DefaultEnumValue() protoreflect.EnumValueDescriptor {
	w.doInit()
	return w.defaultEnumValue
}

func (w *fieldWrapper) ContainingOneof() protoreflect.OneofDescriptor {
	w.doInit()
	return w.containingOneof
}

func (w *fieldWrapper) ContainingMessage() protoreflect.MessageDescriptor {
	w.doInit()
	return w.containingMsg
}

func (w *fieldWrapper) Message() protoreflect.MessageDescriptor {
	w.doInit()
	return w.msgType
}

func (w *fieldWrapper) Enum() protoreflect.EnumDescriptor {
	w.doInit()
	return w.enumType
}

func (w *fieldWrapper) doInit() {
	w.init.Do(func() {
		if mapKey := w.FieldDescriptor.MapKey(); mapKey != nil {
			w.mapKey = findField(mapKey, w.ParentFile())
		}
		if mapVal := w.FieldDescriptor.MapValue(); mapVal != nil {
			w.mapValue = findField(mapVal, w.ParentFile())
		}
		if oo := w.FieldDescriptor.ContainingOneof(); oo != nil {
			parent := w.parent.(MessageWrapper)
			w.containingOneof = parent.Oneofs().Get(oo.Index())
		}
		if !w.IsExtension() {
			w.containingMsg = w.parent.(MessageWrapper)
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

type extWrapper struct {
	*fieldWrapper
	extType protoreflect.ExtensionType
}

var _ ProtoWrapper = &extWrapper{}
var _ FieldWrapper = &extWrapper{}
var _ WrappedDescriptor = &extWrapper{}
var _ protoreflect.ExtensionTypeDescriptor = &extWrapper{}
var _ protoreflect.ExtensionType = &extWrapper{}

func (w *extWrapper) New() protoreflect.Value {
	return w.extType.New()
}

func (w *extWrapper) Zero() protoreflect.Value {
	return w.extType.Zero()
}

func (w *extWrapper) TypeDescriptor() protoreflect.ExtensionTypeDescriptor {
	return w
}

func (w *extWrapper) ValueOf(i interface{}) protoreflect.Value {
	return w.extType.ValueOf(i)
}

func (w *extWrapper) InterfaceOf(value protoreflect.Value) interface{} {
	return w.extType.InterfaceOf(value)
}

func (w *extWrapper) IsValidValue(value protoreflect.Value) bool {
	return w.extType.IsValidValue(value)
}

func (w *extWrapper) IsValidInterface(i interface{}) bool {
	return w.extType.IsValidInterface(i)
}

func (w *extWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *extWrapper) FieldDescriptorProto() *descriptorpb.FieldDescriptorProto {
	return w.proto
}

func (w *extWrapper) Type() protoreflect.ExtensionType {
	return w
}

func (w *extWrapper) Descriptor() protoreflect.ExtensionDescriptor {
	return w
}

type oneofWrapper struct {
	protoreflect.OneofDescriptor
	proto  *descriptorpb.OneofDescriptorProto
	parent *msgWrapper
	fields fieldsWrapper
}

var _ ProtoWrapper = &oneofWrapper{}
var _ OneofWrapper = &oneofWrapper{}
var _ WrappedDescriptor = &oneofWrapper{}

func (w *oneofWrapper) Unwrap() protoreflect.Descriptor {
	return w.OneofDescriptor
}

func (w *oneofWrapper) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *oneofWrapper) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *oneofWrapper) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *oneofWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *oneofWrapper) OneofDescriptorProto() *descriptorpb.OneofDescriptorProto {
	return w.proto
}

func (w *oneofWrapper) Fields() protoreflect.FieldDescriptors {
	w.fields.initFromOneof(w)
	return &w.fields
}

type enumWrapper struct {
	protoreflect.EnumDescriptor
	proto  *descriptorpb.EnumDescriptorProto
	parent ProtoWrapper // either a *fileWrapper or *msgWrapper
	values enumValuesWrapper
}

var _ ProtoWrapper = &enumWrapper{}
var _ EnumWrapper = &enumWrapper{}
var _ WrappedDescriptor = &enumWrapper{}

func (w *enumWrapper) Unwrap() protoreflect.Descriptor {
	return w.EnumDescriptor
}

func (w *enumWrapper) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *enumWrapper) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *enumWrapper) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *enumWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *enumWrapper) EnumDescriptorProto() *descriptorpb.EnumDescriptorProto {
	return w.proto
}

func (w *enumWrapper) Values() protoreflect.EnumValueDescriptors {
	w.values.initFromEnum(w)
	return &w.values
}

type enumValueWrapper struct {
	protoreflect.EnumValueDescriptor
	parent *enumWrapper
	proto  *descriptorpb.EnumValueDescriptorProto
}

var _ ProtoWrapper = &enumValueWrapper{}
var _ EnumValueWrapper = &enumValueWrapper{}
var _ WrappedDescriptor = &enumValueWrapper{}

func (w *enumValueWrapper) Unwrap() protoreflect.Descriptor {
	return w.EnumValueDescriptor
}

func (w *enumValueWrapper) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *enumValueWrapper) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *enumValueWrapper) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *enumValueWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *enumValueWrapper) EnumValueDescriptorProto() *descriptorpb.EnumValueDescriptorProto {
	return w.proto
}

type svcWrapper struct {
	protoreflect.ServiceDescriptor
	parent *fileWrapper
	proto  *descriptorpb.ServiceDescriptorProto
	mtds   mtdsWrapper
}

var _ ProtoWrapper = &svcWrapper{}
var _ ServiceWrapper = &svcWrapper{}
var _ WrappedDescriptor = &svcWrapper{}

func (w *svcWrapper) Unwrap() protoreflect.Descriptor {
	return w.ServiceDescriptor
}

func (w *svcWrapper) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *svcWrapper) ParentFile() protoreflect.FileDescriptor {
	return w.parent
}

func (w *svcWrapper) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *svcWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *svcWrapper) ServiceDescriptorProto() *descriptorpb.ServiceDescriptorProto {
	return w.proto
}

func (w *svcWrapper) Methods() protoreflect.MethodDescriptors {
	w.mtds.initFromSvc(w)
	return &w.mtds
}

type mtdWrapper struct {
	protoreflect.MethodDescriptor
	parent *svcWrapper
	proto  *descriptorpb.MethodDescriptorProto

	init          sync.Once
	input, output protoreflect.MessageDescriptor
}

var _ ProtoWrapper = &mtdWrapper{}
var _ MethodWrapper = &mtdWrapper{}
var _ WrappedDescriptor = &mtdWrapper{}

func (w *mtdWrapper) Unwrap() protoreflect.Descriptor {
	return w.MethodDescriptor
}

func (w *mtdWrapper) Parent() protoreflect.Descriptor {
	return w.parent
}

func (w *mtdWrapper) ParentFile() protoreflect.FileDescriptor {
	return parentFile(w)
}

func (w *mtdWrapper) Options() proto.Message {
	return w.proto.GetOptions()
}

func (w *mtdWrapper) AsProto() proto.Message {
	return w.proto
}

func (w *mtdWrapper) MethodDescriptorProto() *descriptorpb.MethodDescriptorProto {
	return w.proto
}

func (w *mtdWrapper) Input() protoreflect.MessageDescriptor {
	w.doInit()
	return w.input
}

func (w *mtdWrapper) Output() protoreflect.MessageDescriptor {
	w.doInit()
	return w.output
}

func (w *mtdWrapper) doInit() {
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
			panic(fmt.Sprint("unsupported type of parent for field: %T", parent))
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
		panic(fmt.Sprint("unsupported type of parent for message: %T", parent))
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
		panic(fmt.Sprint("unsupported type of parent for enum: %T", parent))
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
