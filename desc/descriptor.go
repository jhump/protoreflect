package desc

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

const (
	// NB: It would be nice to use constants from generated code instead of hard-coding these here.
	// But code-gen does not emit these as constants anywhere. The only places they appear in generated
	// code are struct tags on fields of the generated descriptor protos.
	file_messagesTag = 4
	file_enumsTag = 5
	file_servicesTag = 6
	file_extensionsTag = 7
	message_fieldsTag = 2
	message_nestedMessagesTag = 3
	message_enumsTag = 4
	message_extensionsTag = 6
	message_oneOfsTag = 8
	enum_valuesTag = 2
	service_methodsTag = 2
)

// Descriptor is the common interface implemented by all descriptor objects.
type Descriptor interface {
	// GetName returns the name of the object described by the descriptor. This will
	// be a base name that does not include enclosing message names or the package name.
	// For file descriptors, this indicates the path and name to the described file.
	GetName() string
	// GetFullyQualifiedName returns the fully-qualified name of the object described by
	// the descriptor. This will include the package name and any enclosing message names.
	// For file descriptors, this indicates the package that is declared by the file.
	GetFullyQualifiedName() string
	// GetParent returns the enclosing element in a proto source file. If the described
	// object is a top-level object, this returns the file descriptor. Otherwise, it returns
	// the element in which the described object was declared. File descriptors have no
	// parent and return nil.
	GetParent() Descriptor
	// GetFile returns the file descriptor in which this element was declared. File
	// descriptors return themselves.
	GetFile() *FileDescriptor
	// GetOptions returns the options proto containing options for the described element.
	GetOptions() proto.Message
	// GetSourceInfo returns any source code information that was present in the file
	// descriptor. Source code info is optional. If no source code info is available for
	// the element (including if there is none at all in the file descriptor) then this
	// returns nil
	GetSourceInfo() *dpb.SourceCodeInfo_Location
	// AsProto returns the underlying descriptor proto for this descriptor.
	AsProto() proto.Message
}

// FileDescriptor describes a proto source file.
type FileDescriptor struct {
	proto      *dpb.FileDescriptorProto
	symbols    map[string]Descriptor
	deps       []*FileDescriptor
	publicDeps []*FileDescriptor
	weakDeps   []*FileDescriptor
	messages   []*MessageDescriptor
	enums      []*EnumDescriptor
	extensions []*FieldDescriptor
	services   []*ServiceDescriptor
}

// CreateFileDescriptor instantiates a new file descriptor for the given descriptor proto.
// The file's direct dependencies must be provided. If the given dependencies do not include
// all of the file's dependencies or if the contents of the descriptors are internally
// inconsistent (e.g. contain unresolvable symbols) then an error is returned.
func CreateFileDescriptor(fd *dpb.FileDescriptorProto, deps ...*FileDescriptor) (*FileDescriptor, error) {
	ret := &FileDescriptor{ proto: fd, symbols: map[string]Descriptor{} }
	pkg := fd.GetPackage()

	// populate references to file descriptor dependencies
	files := map[string]*FileDescriptor{}
	for _, f := range deps {
		files[f.proto.GetName()] = f
	}
	ret.deps = make([]*FileDescriptor, len(fd.GetDependency()))
	for i, d := range fd.GetDependency() {
		ret.deps[i] = files[d]
		if ret.deps[i] == nil {
			return nil, fmt.Errorf("Given dependencies did not include %q", d)
		}
	}
	ret.publicDeps = make([]*FileDescriptor, len(fd.GetPublicDependency()))
	for i, pd := range fd.GetPublicDependency() {
		ret.publicDeps[i] = ret.deps[pd]
	}
	ret.weakDeps = make([]*FileDescriptor, len(fd.GetWeakDependency()))
	for i, wd := range fd.GetWeakDependency() {
		ret.weakDeps[i] = ret.deps[wd]
	}

	// populate all tables of child descriptors
	for _, m := range fd.GetMessageType() {
		md, n := createMessageDescriptor(ret, ret, pkg, m, ret.symbols)
		ret.symbols[n] = md
		ret.messages = append(ret.messages, md)
	}
	for _, e := range fd.GetEnumType() {
		ed, n := createEnumDescriptor(ret, ret, pkg, e, ret.symbols)
		ret.symbols[n] = ed
		ret.enums = append(ret.enums, ed)
	}
	for _, ex := range fd.GetExtension() {
		exd, n := createFieldDescriptor(ret, ret, pkg, ex)
		ret.symbols[n] = exd
		ret.extensions = append(ret.extensions, exd)
	}
	for _, s := range fd.GetService() {
		sd, n := createServiceDescriptor(ret, pkg, s, ret.symbols)
		ret.symbols[n] = sd
		ret.services = append(ret.services, sd)
	}
	sourceCodeInfo := map[string]*dpb.SourceCodeInfo_Location{}
	for _, scl := range fd.GetSourceCodeInfo().GetLocation() {
		sourceCodeInfo[pathAsKey(scl.GetPath())] = scl
	}

	// now we can resolve all type references and source code info
	scopes := []scope{fileScope(ret)}
	path := make([]int32, 1, 8)
	path[0] = file_messagesTag
	for i, md := range ret.messages {
		if err := md.resolve(append(path, int32(i)), sourceCodeInfo, scopes); err != nil {
			return nil, err
		}
	}
	path[0] = file_enumsTag
	for i, ed := range ret.enums {
		ed.resolve(append(path, int32(i)), sourceCodeInfo)
	}
	path[0] = file_extensionsTag
	for i, exd := range ret.extensions {
		if err := exd.resolve(append(path, int32(i)), sourceCodeInfo, scopes); err != nil {
			return nil, err
		}
	}
	path[0] = file_servicesTag
	for i, sd := range ret.services {
		if err := sd.resolve(append(path, int32(i)), sourceCodeInfo, scopes); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

// CreateFileDescriptorFromSet creates a descriptor from the given file descriptor set. The
// set's first file will be the returned descriptor. The set's remaining files must comprise
// the full set of transitive dependencies of that first file.
func CreateFileDescriptorFromSet(fds *dpb.FileDescriptorSet) (*FileDescriptor, error) {
	if len(fds.GetFile()) == 0 {
		return nil, errors.New("file descriptor set is empty")
	}
	files := map[string]*dpb.FileDescriptorProto{}
	resolved := map[string]*FileDescriptor{}
	var name string
	for i, fd := range fds.GetFile() {
		if i == 0 {
			name = fd.GetName()
		}
		files[fd.GetName()] = fd
	}
	return createFromSet(name, files, resolved)
}

// createFromSet creates a descriptor for the given filename. It recursively
// creates descriptors for the given file's dependencies.
func createFromSet(filename string, files map[string]*dpb.FileDescriptorProto, resolved map[string]*FileDescriptor) (*FileDescriptor, error) {
	if d, ok := resolved[filename]; ok {
		return d, nil
	}
	fdp := files[filename]
	if fdp == nil {
		return nil, fmt.Errorf("file descriptor set missing a dependency: %s", filename)
	}
	deps := make([]*FileDescriptor, len(fdp.GetDependency()))
	for i, depName := range fdp.GetDependency() {
		if dep, err := createFromSet(depName, files, resolved); err != nil {
			return nil, err
		} else {
			deps[i] = dep
		}
	}
	return CreateFileDescriptor(fdp, deps...)
}

func (fd *FileDescriptor) GetName() string {
	return fd.proto.GetName()
}

func (fd *FileDescriptor) GetFullyQualifiedName() string {
	return fd.proto.GetName()
}

func (fd *FileDescriptor) GetPackage() string {
	return fd.proto.GetPackage()
}

func (fd *FileDescriptor) GetParent() Descriptor {
	return nil
}

func (fd *FileDescriptor) GetFile() *FileDescriptor {
	return fd
}

func (fd *FileDescriptor) GetOptions() proto.Message {
	return fd.proto.GetOptions()
}

func (fd *FileDescriptor) GetFileOptions() *dpb.FileOptions {
	return fd.proto.GetOptions()
}

func (fd *FileDescriptor) GetSourceInfo() *dpb.SourceCodeInfo_Location {
	return nil
}

func (fd *FileDescriptor) AsProto() proto.Message {
	return fd.proto
}

func (fd *FileDescriptor) AsFileDescriptorProto() *dpb.FileDescriptorProto {
	return fd.proto
}

func (fd *FileDescriptor) String() string {
	return fd.proto.String()
}

// GetDependencies returns all of this file's dependencies. These correspond to
// import statements in the file.
func (fd *FileDescriptor) GetDependencies() []*FileDescriptor {
	return fd.deps
}

// GetPublicDependencies returns all of this file's public dependencies. These
// correspond to public import statements in the file.
func (fd *FileDescriptor) GetPublicDependencies() []*FileDescriptor {
	return fd.publicDeps
}

// GetWeakDependencies returns all of this file's public dependencies. These
// correspond to weak import statements in the file.
func (fd *FileDescriptor) GetWeakDependencies() []*FileDescriptor {
	return fd.weakDeps
}

// GetMessageTypes returns all top-level messages declared in this file.
func (fd *FileDescriptor) GetMessageTypes() []*MessageDescriptor {
	return fd.messages
}

// GetEnumTypes returns all top-level enums declared in this file.
func (fd *FileDescriptor) GetEnumTypes() []*EnumDescriptor {
	return fd.enums
}

// GetExtensions returns all top-level extensions declared in this file.
func (fd *FileDescriptor) GetExtensions() []*FieldDescriptor {
	return fd.extensions
}

// GetServices returns all services declared in this file.
func (fd *FileDescriptor) GetServices() []*ServiceDescriptor {
	return fd.services
}

// FindSymbol returns the descriptor contained within this file for the
// element with the given fully-qualified symbol name. If no such element
// exists then this method returns nil.
func (fd *FileDescriptor) FindSymbol(symbol string) Descriptor {
	return fd.symbols[symbol]
}

// MessageDescriptor describes a protocol buffer message.
type MessageDescriptor struct {
	proto      *dpb.DescriptorProto
	parent     Descriptor
	file       *FileDescriptor
	fields     []*FieldDescriptor
	nested     []*MessageDescriptor
	enums      []*EnumDescriptor
	extensions []*FieldDescriptor
	oneOfs     []*OneOfDescriptor
	fqn        string
	sourceInfo *dpb.SourceCodeInfo_Location
}

func createMessageDescriptor(fd *FileDescriptor, parent Descriptor, enclosing string, md *dpb.DescriptorProto, symbols map[string]Descriptor) (*MessageDescriptor, string) {
	msgName := merge(enclosing, md.GetName())
	ret := &MessageDescriptor{ proto: md, parent: parent, file: fd, fqn: msgName }
	for _, f := range md.GetField() {
		fld, n := createFieldDescriptor(fd, ret, msgName, f)
		symbols[n] = fld
		ret.fields = append(ret.fields, fld)
	}
	for _, nm := range md.NestedType {
		nmd, n := createMessageDescriptor(fd, ret, msgName, nm, symbols)
		symbols[n] = nmd
		ret.nested = append(ret.nested, nmd)
	}
	for _, e := range md.EnumType {
		ed, n := createEnumDescriptor(fd, ret, msgName, e, symbols)
		symbols[n] = ed
		ret.enums = append(ret.enums, ed)
	}
	for _, ex := range md.GetExtension() {
		exd, n := createFieldDescriptor(fd, ret, msgName, ex)
		symbols[n] = exd
		ret.extensions = append(ret.extensions, exd)
	}
	for i, o := range md.GetOneofDecl() {
		od, n := createOneOfDescriptor(fd, ret, i, msgName, o)
		symbols[n] = od
		ret.oneOfs = append(ret.oneOfs, od)
	}
	return ret, msgName
}

func (md *MessageDescriptor) resolve(path []int32, sourceCodeInfo map[string]*dpb.SourceCodeInfo_Location, scopes []scope) error {
	md.sourceInfo = sourceCodeInfo[pathAsKey(path)]
	path = append(path, message_nestedMessagesTag)
	scopes = append(scopes, messageScope(md))
	for i, nmd := range md.nested {
		if err := nmd.resolve(append(path, int32(i)), sourceCodeInfo, scopes); err != nil {
			return err
		}
	}
	path[len(path) - 1] = message_enumsTag
	for i, ed := range md.enums {
		ed.resolve(append(path, int32(i)), sourceCodeInfo)
	}
	path[len(path) - 1] = message_fieldsTag
	for i, fld := range md.fields {
		if err := fld.resolve(append(path, int32(i)), sourceCodeInfo, scopes); err != nil {
			return err
		}
	}
	path[len(path) - 1] = message_extensionsTag
	for i, exd := range md.extensions {
		if err := exd.resolve(append(path, int32(i)), sourceCodeInfo, scopes); err != nil {
			return err
		}
	}
	path[len(path) - 1] = message_oneOfsTag
	for i, od := range md.oneOfs {
		od.resolve(append(path, int32(i)), sourceCodeInfo)
	}
	return nil
}

func (md *MessageDescriptor) GetName() string {
	return md.proto.GetName()
}

func (md *MessageDescriptor) GetFullyQualifiedName() string {
	return md.fqn
}

func (md *MessageDescriptor) GetParent() Descriptor {
	return md.parent
}

func (md *MessageDescriptor) GetFile() *FileDescriptor {
	return md.file
}

func (md *MessageDescriptor) GetOptions() proto.Message {
	return md.proto.GetOptions()
}

func (md *MessageDescriptor) GetMessageOptions() *dpb.MessageOptions {
	return md.proto.GetOptions()
}

func (md *MessageDescriptor) GetSourceInfo() *dpb.SourceCodeInfo_Location {
	return md.sourceInfo
}

func (md *MessageDescriptor) AsProto() proto.Message {
	return md.proto
}

func (md *MessageDescriptor) AsDescriptorProto() *dpb.DescriptorProto {
	return md.proto
}

func (md *MessageDescriptor) String() string {
	return md.proto.String()
}

// IsMapEntry returns true if this is a synthetic message type that represents an entry
// in a map field.
func (md *MessageDescriptor) IsMapEntry() bool {
	return md.proto.GetOptions().GetMapEntry()
}

// GetFields returns all of the fields for this message.
func (md *MessageDescriptor) GetFields() []*FieldDescriptor {
	return md.fields
}

// GetNestedMessageTypes returns all of the message types declared inside this message.
func (md *MessageDescriptor) GetNestedMessageTypes() []*MessageDescriptor {
	return md.nested
}

// GetNestedEnumTypes returns all of the enums declared inside this message.
func (md *MessageDescriptor) GetNestedEnumTypes() []*EnumDescriptor {
	return md.enums
}

// GetNestedExtensions returns all of the extensions declared inside this message.
func (md *MessageDescriptor) GetNestedExtensions() []*FieldDescriptor {
	return md.extensions
}

// GetOneOfs returns all of the one-of field sets declared inside this message.
func (md *MessageDescriptor) GetOneOfs() []*OneOfDescriptor {
	return md.oneOfs
}

// FieldDescriptor describes a field of a protocol buffer message.
type FieldDescriptor struct {
	proto      *dpb.FieldDescriptorProto
	parent     Descriptor
	owner      *MessageDescriptor
	file       *FileDescriptor
	oneOf      *OneOfDescriptor
	msgType    *MessageDescriptor
	enumType   *EnumDescriptor
	fqn        string
	sourceInfo *dpb.SourceCodeInfo_Location
}

func createFieldDescriptor(fd *FileDescriptor, parent Descriptor, enclosing string, fld *dpb.FieldDescriptorProto) (*FieldDescriptor, string) {
	fldName := merge(enclosing, fld.GetName())
	ret := &FieldDescriptor{ proto: fld, parent: parent, file: fd, fqn: fldName }
	if fld.GetExtendee() == "" {
		ret.owner = parent.(*MessageDescriptor)
	}
	// owner for extensions, field type (be it message or enum), and one-ofs get resolved later
	return ret, fldName
}

func (fd *FieldDescriptor) resolve(path []int32, sourceCodeInfo map[string]*dpb.SourceCodeInfo_Location, scopes []scope) error {
	fd.sourceInfo = sourceCodeInfo[pathAsKey(path)]
	if fd.proto.GetType() == dpb.FieldDescriptorProto_TYPE_ENUM {
		if desc, err := resolve(fd.file, fd.proto.GetTypeName(), scopes); err != nil {
			return err
		} else {
			fd.enumType = desc.(*EnumDescriptor)
		}
	}
	if fd.proto.GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
		if desc, err := resolve(fd.file, fd.proto.GetTypeName(), scopes); err != nil {
			return err
		} else {
			fd.msgType = desc.(*MessageDescriptor)
		}
	}
	if fd.proto.GetExtendee() != "" {
		if desc, err := resolve(fd.file, fd.proto.GetExtendee(), scopes); err != nil {
			return err
		} else {
			fd.owner = desc.(*MessageDescriptor)
		}
	}
	return nil
}

func (fd *FieldDescriptor) GetName() string {
	return fd.proto.GetName()
}

// GetNumber returns the tag number of this field.
func (fd *FieldDescriptor) GetNumber() int32 {
	return fd.proto.GetNumber()
}

func (fd *FieldDescriptor) GetFullyQualifiedName() string {
	return fd.fqn
}

func (fd *FieldDescriptor) GetParent() Descriptor {
	return fd.parent
}

func (fd *FieldDescriptor) GetFile() *FileDescriptor {
	return fd.file
}

func (fd *FieldDescriptor) GetOptions() proto.Message {
	return fd.proto.GetOptions()
}

func (fd *FieldDescriptor) GetFieldOptions() *dpb.FieldOptions {
	return fd.proto.GetOptions()
}

func (fd *FieldDescriptor) GetSourceInfo() *dpb.SourceCodeInfo_Location {
	return fd.sourceInfo
}

func (fd *FieldDescriptor) AsProto() proto.Message {
	return fd.proto
}

func (fd *FieldDescriptor) AsFieldDescriptorProto() *dpb.FieldDescriptorProto {
	return fd.proto
}

func (fd *FieldDescriptor) String() string {
	return fd.proto.String()
}

// GetOwner returns the message type that this field belongs to. If this is a normal
// field then this is the same as GetParent. But for extensions, this will be the
// extendee message whereas GetParent refers to where the extension was declared.
func (fd *FieldDescriptor) GetOwner() *MessageDescriptor {
	return fd.owner
}

// IsExtension returns true if this is an extension field.
func (fd *FieldDescriptor) IsExtension() bool {
	return fd.proto.GetExtendee() != ""
}

// GetOneOf returns the one-of field set to which this field belongs. If this field
// is not part of a one-of then this method returns nil.
func (fd *FieldDescriptor) GetOneOf() *OneOfDescriptor {
	return fd.oneOf
}

// GetType returns the type of this field. If the type indicates an enum, the
// enum type can be queried via GetEnumType. If the type indicates a message, the
// message type can be queried via GetMessageType.
func (fd *FieldDescriptor) GetType() dpb.FieldDescriptorProto_Type {
	return fd.proto.GetType()
}

// GetLabel returns the label for this field. The label can be required (proto2-only),
// optional (default for proto3), or required.
func (fd *FieldDescriptor) GetLabel() dpb.FieldDescriptorProto_Label {
	return fd.proto.GetLabel()
}

// IsRequired returns true if this field has the "required" label.
func (fd *FieldDescriptor) IsRequired() bool {
	return fd.proto.GetLabel() == dpb.FieldDescriptorProto_LABEL_REQUIRED
}

// IsRepeated returns true if this field has the "repeated" label.
func (fd *FieldDescriptor) IsRepeated() bool {
	return fd.proto.GetLabel() == dpb.FieldDescriptorProto_LABEL_REPEATED
}

// IsMap returns true if this is a map field. If so, it will have the "repeated"
// label its type will be a message that represents a map entry. The map entry
// message will have exactly two fields: tag #1 is  key and tag #2 is the value.
func (fd *FieldDescriptor) IsMap() bool {
	return fd.proto.GetLabel() == dpb.FieldDescriptorProto_LABEL_REPEATED &&
		fd.proto.GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE &&
		fd.GetMessageType().GetMessageOptions().GetMapEntry()
}

// GetMessageType returns the type of this field if it is a message type. If
// this field is not a message type, it returns nil.
func (fd *FieldDescriptor) GetMessageType() *MessageDescriptor {
	return fd.msgType
}

// GetEnumType returns the type of this field if it is an enum type. If this
// field is not an enum type, it returns nil.
func (fd *FieldDescriptor) GetEnumType() *EnumDescriptor {
	return fd.enumType
}

// EnumDescriptor describes an enum declared in a proto file.
type EnumDescriptor struct {
	proto      *dpb.EnumDescriptorProto
	parent     Descriptor
	file       *FileDescriptor
	values     []*EnumValueDescriptor
	fqn        string
	sourceInfo *dpb.SourceCodeInfo_Location
}

func createEnumDescriptor(fd *FileDescriptor, parent Descriptor, enclosing string, ed *dpb.EnumDescriptorProto, symbols map[string]Descriptor) (*EnumDescriptor, string) {
	enumName := merge(enclosing, ed.GetName())
	ret := &EnumDescriptor{ proto: ed, parent: parent, file: fd, fqn: enumName }
	for _, ev := range ed.GetValue() {
		evd, n := createEnumValueDescriptor(fd, ret, enumName, ev)
		symbols[n] = evd
		ret.values = append(ret.values, evd)
	}
	return ret, enumName
}

func (ed *EnumDescriptor) resolve(path []int32, sourceCodeInfo map[string]*dpb.SourceCodeInfo_Location) {
	ed.sourceInfo = sourceCodeInfo[pathAsKey(path)]
	path = append(path, enum_valuesTag)
	for i, evd := range ed.values {
		evd.resolve(append(path, int32(i)), sourceCodeInfo)
	}
}

func (ed *EnumDescriptor) GetName() string {
	return ed.proto.GetName()
}

func (ed *EnumDescriptor) GetFullyQualifiedName() string {
	return ed.fqn
}

func (ed *EnumDescriptor) GetParent() Descriptor {
	return ed.parent
}

func (ed *EnumDescriptor) GetFile() *FileDescriptor {
	return ed.file
}

func (ed *EnumDescriptor) GetOptions() proto.Message {
	return ed.proto.GetOptions()
}

func (ed *EnumDescriptor) GetEnumOptions() *dpb.EnumOptions {
	return ed.proto.GetOptions()
}

func (ed *EnumDescriptor) GetSourceInfo() *dpb.SourceCodeInfo_Location {
	return ed.sourceInfo
}

func (ed *EnumDescriptor) AsProto() proto.Message {
	return ed.proto
}

func (ed *EnumDescriptor) AsEnumDescriptorProto() *dpb.EnumDescriptorProto {
	return ed.proto
}

func (ed *EnumDescriptor) String() string {
	return ed.proto.String()
}

// GetValues returns all of the allowed values defined for this enum.
func (ed *EnumDescriptor) GetValues() []*EnumValueDescriptor {
	return ed.values
}

// EnumValueDescriptor describes an allowed value of an enum declared in a proto file.
type EnumValueDescriptor struct {
	proto      *dpb.EnumValueDescriptorProto
	parent     *EnumDescriptor
	file       *FileDescriptor
	fqn        string
	sourceInfo *dpb.SourceCodeInfo_Location
}

func createEnumValueDescriptor(fd *FileDescriptor, parent *EnumDescriptor, enclosing string, evd *dpb.EnumValueDescriptorProto) (*EnumValueDescriptor, string) {
	valName := merge(enclosing, evd.GetName())
	return &EnumValueDescriptor{ proto: evd, parent: parent, file: fd, fqn: valName }, valName
}

func (vd *EnumValueDescriptor) resolve(path []int32, sourceCodeInfo map[string]*dpb.SourceCodeInfo_Location) {
	vd.sourceInfo = sourceCodeInfo[pathAsKey(path)]
}

func (vd *EnumValueDescriptor) GetName() string {
	return vd.proto.GetName()
}

// GetNumber returns the numeric value associated with this enum value.
func (vd *EnumValueDescriptor) GetNumber() int32 {
	return vd.proto.GetNumber()
}

func (vd *EnumValueDescriptor) GetFullyQualifiedName() string {
	return vd.fqn
}

func (vd *EnumValueDescriptor) GetParent() Descriptor {
	return vd.parent
}

// GetEnum returns the enum in which this enum value is defined.
func (vd *EnumValueDescriptor) GetEnum() *EnumDescriptor {
	return vd.parent
}

func (vd *EnumValueDescriptor) GetFile() *FileDescriptor {
	return vd.file
}

func (vd *EnumValueDescriptor) GetOptions() proto.Message {
	return vd.proto.GetOptions()
}

func (vd *EnumValueDescriptor) GetEnumValueOptions() *dpb.EnumValueOptions {
	return vd.proto.GetOptions()
}

func (vd *EnumValueDescriptor) GetSourceInfo() *dpb.SourceCodeInfo_Location {
	return vd.sourceInfo
}

func (vd *EnumValueDescriptor) AsProto() proto.Message {
	return vd.proto
}

func (vd *EnumValueDescriptor) AsEnumValueDescriptorProto() *dpb.EnumValueDescriptorProto {
	return vd.proto
}

func (vd *EnumValueDescriptor) String() string {
	return vd.proto.String()
}

// ServiceDescriptor describes an RPC service declared in a proto file.
type ServiceDescriptor struct {
	proto      *dpb.ServiceDescriptorProto
	file       *FileDescriptor
	methods    []*MethodDescriptor
	fqn        string
	sourceInfo *dpb.SourceCodeInfo_Location
}

func createServiceDescriptor(fd *FileDescriptor, enclosing string, sd *dpb.ServiceDescriptorProto, symbols map[string]Descriptor) (*ServiceDescriptor, string) {
	serviceName := merge(enclosing, sd.GetName())
	ret := &ServiceDescriptor{ proto: sd, file: fd, fqn: serviceName }
	for _, m := range sd.GetMethod() {
		md, n := createMethodDescriptor(fd, ret, serviceName, m)
		symbols[n] = md
		ret.methods = append(ret.methods, md)
	}
	return ret, serviceName
}

func (sd *ServiceDescriptor) resolve(path []int32, sourceCodeInfo map[string]*dpb.SourceCodeInfo_Location, scopes []scope) error {
	sd.sourceInfo = sourceCodeInfo[pathAsKey(path)]
	path = append(path, service_methodsTag)
	for i, md := range sd.methods {
		if err := md.resolve(append(path, int32(i)), sourceCodeInfo, scopes); err != nil {
			return err
		}
	}
	return nil
}

func (sd *ServiceDescriptor) GetName() string {
	return sd.proto.GetName()
}

func (sd *ServiceDescriptor) GetFullyQualifiedName() string {
	return sd.fqn
}

func (sd *ServiceDescriptor) GetParent() Descriptor {
	return sd.file
}

func (sd *ServiceDescriptor) GetFile() *FileDescriptor {
	return sd.file
}

func (sd *ServiceDescriptor) GetOptions() proto.Message {
	return sd.proto.GetOptions()
}

func (sd *ServiceDescriptor) GetServiceOptions() *dpb.ServiceOptions {
	return sd.proto.GetOptions()
}

func (sd *ServiceDescriptor) GetSourceInfo() *dpb.SourceCodeInfo_Location {
	return sd.sourceInfo
}

func (sd *ServiceDescriptor) AsProto() proto.Message {
	return sd.proto
}

func (sd *ServiceDescriptor) AsServiceDescriptorProto() *dpb.ServiceDescriptorProto {
	return sd.proto
}

func (sd *ServiceDescriptor) String() string {
	return sd.proto.String()
}

// GetMethods returns all of the RPC methods for this service.
func (sd *ServiceDescriptor) GetMethods() []*MethodDescriptor {
	return sd.methods
}

// MethodDescriptor describes an RPC method declared in a proto file.
type MethodDescriptor struct {
	proto      *dpb.MethodDescriptorProto
	parent     *ServiceDescriptor
	file       *FileDescriptor
	inType     *MessageDescriptor
	outType    *MessageDescriptor
	fqn        string
	sourceInfo *dpb.SourceCodeInfo_Location
}

func createMethodDescriptor(fd *FileDescriptor, parent *ServiceDescriptor, enclosing string, md *dpb.MethodDescriptorProto) (*MethodDescriptor, string) {
	// request and response types get resolved later
	methodName := merge(enclosing, md.GetName())
	return &MethodDescriptor{ proto: md, parent: parent, file: fd, fqn: methodName }, methodName
}

func (md *MethodDescriptor) resolve(path []int32, sourceCodeInfo map[string]*dpb.SourceCodeInfo_Location, scopes []scope) error {
	md.sourceInfo = sourceCodeInfo[pathAsKey(path)]
	if desc, err := resolve(md.file, md.proto.GetInputType(), scopes); err != nil {
		return err
	} else {
		md.inType = desc.(*MessageDescriptor)
	}
	if desc, err := resolve(md.file, md.proto.GetOutputType(), scopes); err != nil {
		return err
	} else {
		md.outType = desc.(*MessageDescriptor)
	}
	return nil
}

func (md *MethodDescriptor) GetName() string {
	return md.proto.GetName()
}

func (md *MethodDescriptor) GetFullyQualifiedName() string {
	return md.fqn
}

func (md *MethodDescriptor) GetParent() Descriptor {
	return md.parent
}

// GetService returns the RPC service in which this method is declared.
func (md *MethodDescriptor) GetService() *ServiceDescriptor {
	return md.parent
}

func (md *MethodDescriptor) GetFile() *FileDescriptor {
	return md.file
}

func (md *MethodDescriptor) GetOptions() proto.Message {
	return md.proto.GetOptions()
}

func (md *MethodDescriptor) GetMethodOptions() *dpb.MethodOptions {
	return md.proto.GetOptions()
}

func (md *MethodDescriptor) GetSourceInfo() *dpb.SourceCodeInfo_Location {
	return md.sourceInfo
}

func (md *MethodDescriptor) AsProto() proto.Message {
	return md.proto
}

func (md *MethodDescriptor) AsMethodDescriptorProto() *dpb.MethodDescriptorProto {
	return md.proto
}

func (md *MethodDescriptor) String() string {
	return md.proto.String()
}

// IsServerStreaming returns true if this is a server-streaming method.
func (md *MethodDescriptor) IsServerStreaming() bool {
	return md.proto.GetServerStreaming()
}

// IsClientStreaming returns true if this is a client-streaming method.
func (md *MethodDescriptor) IsClientStreaming() bool {
	return md.proto.GetClientStreaming()
}

// GetInputType returns the input type, or request type, of the RPC method.
func (md *MethodDescriptor) GetInputType() *MessageDescriptor {
	return md.inType
}

// GetOutputType returns the output type, or response type, of the RPC method.
func (md *MethodDescriptor) GetOutputType() *MessageDescriptor {
	return md.outType
}

// OneOfDescriptor describes a one-of field set declared in a protocol buffer message.
type OneOfDescriptor struct {
	proto      *dpb.OneofDescriptorProto
	parent     *MessageDescriptor
	file       *FileDescriptor
	choices    []*FieldDescriptor
	fqn        string
	sourceInfo *dpb.SourceCodeInfo_Location
}

func createOneOfDescriptor(fd *FileDescriptor, parent *MessageDescriptor, index int, enclosing string, od *dpb.OneofDescriptorProto) (*OneOfDescriptor, string) {
	oneOfName := merge(enclosing, od.GetName())
	ret := &OneOfDescriptor{ proto: od, parent: parent, file: fd, fqn: oneOfName }
	for _, f := range parent.fields {
		oi := f.proto.OneofIndex
		if oi != nil && *oi == int32(index) {
			f.oneOf = ret
			ret.choices = append(ret.choices, f)
		}
	}
	return ret, oneOfName
}

func (od *OneOfDescriptor) resolve(path []int32, sourceCodeInfo map[string]*dpb.SourceCodeInfo_Location) {
	od.sourceInfo = sourceCodeInfo[pathAsKey(path)]
}

func (od *OneOfDescriptor) GetName() string {
	return od.proto.GetName()
}

func (od *OneOfDescriptor) GetFullyQualifiedName() string {
	return od.fqn
}

func (od *OneOfDescriptor) GetParent() Descriptor {
	return od.parent
}

// GetOwner returns the message to which this one-of field set belongs.
func (od *OneOfDescriptor) GetOwner() *MessageDescriptor {
	return od.parent
}

func (od *OneOfDescriptor) GetFile() *FileDescriptor {
	return od.file
}

func (od *OneOfDescriptor) GetOptions() proto.Message {
	return od.proto.GetOptions()
}

func (od *OneOfDescriptor) GetOneOfOptions() *dpb.OneofOptions {
	return od.proto.GetOptions()
}

func (od *OneOfDescriptor) GetSourceInfo() *dpb.SourceCodeInfo_Location {
	return od.sourceInfo
}

func (od *OneOfDescriptor) AsProto() proto.Message {
	return od.proto
}

func (od *OneOfDescriptor) AsOneofDescriptorProto() *dpb.OneofDescriptorProto {
	return od.proto
}

func (od *OneOfDescriptor) String() string {
	return od.proto.String()
}

// GetChoices returns the fields that are part of the one-of field set. At most one of
// these fields may be set for a given message.
func (od *OneOfDescriptor) GetChoices() []*FieldDescriptor {
	return od.choices
}

func pathAsKey(path []int32) string {
	var b bytes.Buffer
	first := true
	for _, i := range path {
		if first {
			first = false
		} else {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", i)
	}
	return string(b.Bytes())
}

// scope represents a lexical scope in a proto file in which messages and enums
// can be declared.
type scope func(string) Descriptor

func fileScope(fd *FileDescriptor) scope {
	// we search symbols in this file, but also symbols in other files
	// that have the same package as this file
	pkg := fd.proto.GetPackage()
	fds := collectFilesInPackage(pkg, fd.deps, []*FileDescriptor{ fd })
	return func(name string) Descriptor {
		n := merge(pkg, name)
		for _, fd := range fds {
			if d, ok := fd.symbols[n]; ok {
				return d
			}
		}
		return nil
	}
}

func collectFilesInPackage(pkg string, fds []*FileDescriptor, results []*FileDescriptor) []*FileDescriptor {
	for _, fd := range fds {
		if fd.proto.GetPackage() == pkg {
			results = append(results, fd)
		}
		results = collectFilesInPackage(pkg, fd.publicDeps, results)
	}
	return results
}

func messageScope(md *MessageDescriptor) scope {
	return func(name string) Descriptor {
		n := merge(md.fqn, name)
		if d, ok := md.file.symbols[n]; ok {
			return d
		}
		return nil
	}
}

func resolve(fd *FileDescriptor, name string, scopes []scope) (Descriptor, error) {
	if strings.HasPrefix(name, ".") {
		// already fully-qualified
		d := findSymbol(fd, name[1:], false)
		if d != nil {
			return d, nil
		}
	} else {
		// unqualified, so we look in the enclosing (last) scope first and move
		// towards outermost (first) scope, trying to resolve the symbol
		for i := len(scopes) - 1; i >= 0; i-- {
			d := scopes[i](name)
			if d != nil {
				return d, nil
			}
		}
	}
	return nil, fmt.Errorf("File %q included an unresolvable reference to %q", fd.proto.GetName(), name)
}

func findSymbol(fd *FileDescriptor, name string, public bool) Descriptor {
	d := fd.symbols[name]
	if d != nil {
		return d
	}

	// When public = false, we are searching only directly imported symbols. But we
	// also need to search transitive public imports due to semantics of public imports.
	var deps []*FileDescriptor
	if public {
		deps = fd.publicDeps
	} else {
		deps = fd.deps
	}
	for _, dep := range deps {
		d = findSymbol(dep, name, true)
		if d != nil {
			return d
		}
	}

	return nil
}

func merge(a, b string) string {
	if a == "" {
		return b
	} else {
		return a + "." + b
	}
}
