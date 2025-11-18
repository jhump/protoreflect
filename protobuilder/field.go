package protobuilder

import (
	"fmt"
	"iter"
	"strings"
	"unicode"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/internal/fielddefault"
	"github.com/jhump/protoreflect/v2/protomessage"
)

// FieldBuilder is a builder used to construct a protoreflect.FieldDescriptor. A field
// builder is used to create fields and extensions as well as map entry
// messages. It is also used to link groups (defined via a message builder) into
// an enclosing message, associating it with a group field.  A non-extension
// field builder *must* be added to a message before calling its Build() method.
//
// To create a new FieldBuilder, use NewField, NewMapField, NewGroupField,
// NewExtension, or NewExtensionImported (depending on the type of field being
// built).
type FieldBuilder struct {
	baseBuilder
	number protoreflect.FieldNumber

	// msgType is populated for fields that have a "private" message type that
	// isn't expected to be referenced elsewhere. This happens for map fields,
	// where the private message type represents the map entry, and for group
	// fields.
	msgType   *MessageBuilder
	fieldType *FieldType

	Options *descriptorpb.FieldOptions
	// Cardinality indicates if the field is required, optional, or repeated.
	// Required can only be used for fields in files with "proto2" syntax.
	// If the file's syntax is not "proto2", it cannot be set to
	// [protoreflect.Required].
	//
	// To create a required field in files that use "editions", set the
	// FieldPresence field of Options.Features to LEGACY_REQUIRED.
	Cardinality protoreflect.Cardinality

	// Proto3Optional indicates if the field is a proto3 optional field. This
	// only applies to fields in files with "proto3" syntax whose Cardinality
	// is set to [protoreflect.Optional].
	//
	// If the file's syntax is not "proto3", this may not be set to true.
	//
	// This allows setting a field in a proto3 file to have explicit field
	// presence. To manage field presence for fields in files that use
	// "editions", set the FieldPresence field of Options.Features to
	// IMPLICIT.
	Proto3Optional bool

	Default  string
	JsonName string

	foreignExtendee protoreflect.MessageDescriptor
	localExtendee   *MessageBuilder
}

var _ Builder = (*FieldBuilder)(nil)

// NewField creates a new FieldBuilder for a non-extension field with the given
// name and type. To create a map or group field, see NewMapField or
// NewGroupField respectively.
//
// The new field will be optional. See SetCardinality, SetRepeated, and SetRequired
// for changing this aspect of the field. The new field's tag will be zero,
// which means it will be auto-assigned when the descriptor is built. Use
// SetNumber or TrySetNumber to assign an explicit tag number.
func NewField(name protoreflect.Name, typ *FieldType) *FieldBuilder {
	flb := &FieldBuilder{
		baseBuilder: baseBuilderWithName(name),
		fieldType:   typ,
	}
	return flb
}

// NewMapField creates a new FieldBuilder for a non-extension field with the
// given name and whose type is a map of the given key and value types. Map keys
// can be any of the scalar integer types, booleans, or strings. If any other
// type is specified, this function will panic. Map values cannot be groups: if
// a group type is specified, this function will panic.
//
// When this field is added to a message, the associated map entry message type
// will also be added.
//
// The new field's tag will be zero, which means it will be auto-assigned when
// the descriptor is built. Use SetNumber or TrySetNumber to assign an explicit
// tag number.
func NewMapField(name protoreflect.Name, keyTyp, valTyp *FieldType) *FieldBuilder {
	switch keyTyp.fieldType {
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL,
		descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32, descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32, descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		// allowed
	default:
		panic(fmt.Sprintf("Map types cannot have keys of type %v", keyTyp.fieldType))
	}
	if valTyp.fieldType == descriptorpb.FieldDescriptorProto_TYPE_GROUP {
		panic(fmt.Sprintf("Map types cannot have values of type %v", valTyp.fieldType))
	}
	entryMsg := NewMessage(entryTypeName(name))
	keyFlb := NewField("key", keyTyp)
	keyFlb.number = 1
	valFlb := NewField("value", valTyp)
	valFlb.number = 2
	entryMsg.AddField(keyFlb)
	entryMsg.AddField(valFlb)
	entryMsg.Options = &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)}

	flb := NewField(name, FieldTypeMessage(entryMsg)).SetCardinality(protoreflect.Repeated)
	flb.msgType = entryMsg
	entryMsg.setParent(flb)
	return flb
}

// NewGroupField creates a new FieldBuilder for a non-extension field whose type
// is a group with the given definition. The given message's name must start
// with a capital letter, and the resulting field will have the same name but
// converted to all lower-case. If a message is given with a name that starts
// with a lower-case letter, this function will panic.
//
// When this field is added to a message, the associated group message type will
// also be added.
//
// The new field will be optional. See SetCardinality, SetRepeated, and SetRequired
// for changing this aspect of the field. The new field's tag will be zero,
// which means it will be auto-assigned when the descriptor is built. Use
// SetNumber or TrySetNumber to assign an explicit tag number.
func NewGroupField(mb *MessageBuilder) *FieldBuilder {
	if !unicode.IsUpper(rune(mb.name[0])) {
		panic(fmt.Sprintf("group name %s must start with a capital letter", mb.name))
	}
	Unlink(mb)

	ft := &FieldType{
		fieldType:    descriptorpb.FieldDescriptorProto_TYPE_GROUP,
		localMsgType: mb,
	}
	fieldName := protoreflect.Name(strings.ToLower(string(mb.Name())))
	flb := NewField(fieldName, ft)
	flb.msgType = mb
	mb.setParent(flb)
	return flb
}

// NewExtension creates a new FieldBuilder for an extension field with the given
// name, tag, type, and extendee. The extendee given is a message builder.
//
// The new field will be optional. See SetCardinality and SetRepeated for changing
// this aspect of the field.
func NewExtension(name protoreflect.Name, tag protoreflect.FieldNumber, typ *FieldType, extendee *MessageBuilder) *FieldBuilder {
	if extendee == nil {
		panic("extendee cannot be nil")
	}
	flb := NewField(name, typ).SetNumber(tag)
	flb.localExtendee = extendee
	return flb
}

// NewExtensionImported creates a new FieldBuilder for an extension field with
// the given name, tag, type, and extendee. The extendee given is a message
// descriptor.
//
// The new field will be optional. See SetCardinality and SetRepeated for changing
// this aspect of the field.
func NewExtensionImported(name protoreflect.Name, tag protoreflect.FieldNumber, typ *FieldType, extendee protoreflect.MessageDescriptor) *FieldBuilder {
	if extendee == nil {
		panic("extendee cannot be nil")
	}
	flb := NewField(name, typ).SetNumber(tag)
	flb.foreignExtendee = extendee
	return flb
}

// FromField returns a FieldBuilder that is effectively a copy of the given
// descriptor.
//
// Note that it is not just the given field that is copied but its entire file.
// So the caller can get the parent element of the returned builder and the
// result would be a builder that is effectively a copy of the field
// descriptor's parent.
//
// This means that field builders created from descriptors do not need to be
// explicitly assigned to a file in order to preserve the original field's
// package name.
func FromField(fld protoreflect.FieldDescriptor) (*FieldBuilder, error) {
	if fb, err := FromFile(fld.ParentFile()); err != nil {
		return nil, err
	} else if flb, ok := fb.findFullyQualifiedElement(fld.FullName()).(*FieldBuilder); ok {
		return flb, nil
	} else {
		return nil, fmt.Errorf("could not find field %s after converting file %q to builder", fld.FullName(), fld.ParentFile().Path())
	}
}

func fromField(fld protoreflect.FieldDescriptor) (*FieldBuilder, error) {
	ft := fieldTypeFromDescriptor(fld)
	flb := NewField(fld.Name(), ft)
	var err error
	flb.Options, err = protomessage.As[*descriptorpb.FieldOptions](fld.Options())
	if err != nil {
		return nil, err
	}
	flb.Cardinality = fld.Cardinality()
	if flb.Cardinality == protoreflect.Required && fld.ParentFile().Syntax() == protoreflect.Editions {
		// We actually leave "legacy required" fields as optional. The "required" aspect
		// comes from the features set in the field options.
		flb.Cardinality = protoreflect.Optional
	}
	flb.Proto3Optional = fld.ContainingOneof() != nil && fld.ContainingOneof().IsSynthetic()
	flb.Default = fielddefault.DefaultValue(fld)
	if !fld.IsExtension() {
		flb.JsonName = fld.JSONName()
	}
	setComments(&flb.comments, fld.ParentFile().SourceLocations().ByDescriptor(fld))

	if fld.IsExtension() {
		flb.foreignExtendee = fld.ContainingMessage()
	}
	if err := flb.TrySetNumber(fld.Number()); err != nil {
		return nil, err
	}
	return flb, nil
}

// SetName changes this field's name, returning the field builder for method
// chaining. If the given new name is not valid (e.g. TrySetName would have
// returned an error) then this method will panic.
func (flb *FieldBuilder) SetName(newName protoreflect.Name) *FieldBuilder {
	if err := flb.TrySetName(newName); err != nil {
		panic(err)
	}
	return flb
}

// TrySetName changes this field's name. It will return an error if the given
// new name is not a valid protobuf identifier or if the parent builder already
// has an element with the given name.
//
// If the field is a non-extension whose parent is a oneof, the oneof's
// enclosing message is checked for elements with a conflicting name. Despite
// the fact that oneof choices are modeled as children of the oneof builder,
// in the protobuf IDL they are actually all defined in the message's namespace.
func (flb *FieldBuilder) TrySetName(newName protoreflect.Name) error {
	var oldMsgName protoreflect.Name
	if flb.msgType != nil {
		if flb.fieldType.fieldType == descriptorpb.FieldDescriptorProto_TYPE_GROUP {
			return fmt.Errorf("cannot change name of group field %s; change name of group instead", FullName(flb))
		}
		oldMsgName = flb.msgType.name
		msgName := entryTypeName(newName)
		if err := flb.msgType.trySetNameInternal(msgName); err != nil {
			return err
		}
	}
	if err := flb.baseBuilder.setName(flb, newName); err != nil {
		// undo change to map entry name
		if flb.msgType != nil && flb.fieldType.fieldType != descriptorpb.FieldDescriptorProto_TYPE_GROUP {
			flb.msgType.setNameInternal(oldMsgName)
		}
		return err
	}
	return nil
}

func (flb *FieldBuilder) trySetNameInternal(newName protoreflect.Name) error {
	return flb.baseBuilder.setName(flb, newName)
}

func (flb *FieldBuilder) setNameInternal(newName protoreflect.Name) {
	if err := flb.trySetNameInternal(newName); err != nil {
		panic(err)
	}
}

// SetComments sets the comments associated with the field. This method returns
// the field builder, for method chaining.
func (flb *FieldBuilder) SetComments(c Comments) *FieldBuilder {
	flb.comments = c
	return flb
}

func (flb *FieldBuilder) setParent(newParent Builder) {
	flb.baseBuilder.setParent(newParent)
}

// Children returns any builders assigned to this field builder. The only
// kind of children a field can have are message types, that correspond to the
// field's map entry type or group type (for map and group fields respectively).
func (flb *FieldBuilder) Children() iter.Seq[Builder] {
	return func(yield func(Builder) bool) {
		if flb.msgType != nil {
			yield(flb.msgType)
		}
	}
}

func (flb *FieldBuilder) findChild(name protoreflect.Name) Builder {
	if flb.msgType != nil && flb.msgType.name == name {
		return flb.msgType
	}
	return nil
}

func (flb *FieldBuilder) removeChild(b Builder) {
	if mb, ok := b.(*MessageBuilder); ok && mb == flb.msgType {
		flb.msgType = nil
		if p, ok := flb.parent.(*MessageBuilder); ok {
			delete(p.symbols, mb.Name())
		}
	}
}

func (flb *FieldBuilder) renamedChild(b Builder, _ protoreflect.Name) error {
	if flb.msgType != nil {
		var oldFieldName protoreflect.Name
		if flb.fieldType.fieldType == descriptorpb.FieldDescriptorProto_TYPE_GROUP {
			// For groups, we need to rename the field according to the group message's new name
			if !unicode.IsUpper(rune(b.Name()[0])) {
				return fmt.Errorf("group name %s must start with capital letter", b.Name())
			}
			// change field name to be lower-case form of group name
			oldFieldName = flb.name
			fieldName := protoreflect.Name(strings.ToLower(string(b.Name())))
			if err := flb.trySetNameInternal(fieldName); err != nil {
				return err
			}
		}
		if p, ok := flb.parent.(*MessageBuilder); ok {
			if err := p.addSymbol(b); err != nil {
				if flb.fieldType.fieldType == descriptorpb.FieldDescriptorProto_TYPE_GROUP {
					// revert the above field rename
					flb.setNameInternal(oldFieldName)
				}
				return err
			}
		}
	}
	return nil
}

// Number returns this field's tag number, or zero if the tag number will be
// auto-assigned when the field descriptor is built.
func (flb *FieldBuilder) Number() protoreflect.FieldNumber {
	return flb.number
}

// SetNumber changes the numeric tag for this field and then returns the field,
// for method chaining. If the given new tag is not valid (e.g. TrySetNumber
// would have returned an error) then this method will panic.
func (flb *FieldBuilder) SetNumber(tag protoreflect.FieldNumber) *FieldBuilder {
	if err := flb.TrySetNumber(tag); err != nil {
		panic(err)
	}
	return flb
}

// TrySetNumber changes this field's tag number. It will return an error if the
// given new tag is out of valid range or (for non-extension fields) if the
// enclosing message already includes a field with the given tag.
//
// Non-extension fields can be set to zero, which means a proper tag number will
// be auto-assigned when the descriptor is built. Extension field tags, however,
// must be set to a valid non-zero value.
func (flb *FieldBuilder) TrySetNumber(tag protoreflect.FieldNumber) error {
	if tag == flb.number {
		return nil // no change
	}
	if tag < 0 {
		return fmt.Errorf("cannot set tag number for field %s to negative value %d", FullName(flb), tag)
	}
	if tag == 0 && flb.IsExtension() {
		return fmt.Errorf("cannot set tag number for extension %s; only regular fields can be auto-assigned", FullName(flb))
	}
	if tag >= internal.SpecialReservedStart && tag <= internal.SpecialReservedEnd {
		return fmt.Errorf("tag for field %s cannot be in special reserved range %d-%d", FullName(flb), internal.SpecialReservedStart, internal.SpecialReservedEnd)
	}
	if tag > internal.MaxTag {
		return fmt.Errorf("tag for field %s cannot be above max %d", FullName(flb), internal.MaxTag)
	}
	oldTag := flb.number
	flb.number = tag
	if flb.IsExtension() {
		// extension tags are not tracked by builders, so no more to do
		return nil
	}
	switch p := flb.parent.(type) {
	case *OneofBuilder:
		m := p.parent()
		if m != nil {
			if err := m.addTag(flb); err != nil {
				flb.number = oldTag
				return err
			}
			delete(m.fieldTags, oldTag)
		}
	case *MessageBuilder:
		if err := p.addTag(flb); err != nil {
			flb.number = oldTag
			return err
		}
		delete(p.fieldTags, oldTag)
	}
	return nil
}

// SetOptions sets the field options for this field and returns the field, for
// method chaining.
func (flb *FieldBuilder) SetOptions(options *descriptorpb.FieldOptions) *FieldBuilder {
	flb.Options = options
	return flb
}

// SetCardinality sets the label for this field, which can be optional, repeated, or
// required. It returns the field builder, for method chaining.
//
// If the field will be in a file with "proto3" or editions syntax, required is not
// an allowed option. For editions, to set a field to be required, set the field_presence
// feature in the field options to descriptorpb.FeatureSet_LEGACY_REQUIRED.
func (flb *FieldBuilder) SetCardinality(card protoreflect.Cardinality) *FieldBuilder {
	flb.Cardinality = card
	return flb
}

// SetProto3Optional sets whether this is a proto3 optional field. It returns
// the field builder, for method chaining.
//
// This can only be set for fields in files with "proto3" syntax.
func (flb *FieldBuilder) SetProto3Optional(p3o bool) *FieldBuilder {
	flb.Proto3Optional = p3o
	return flb
}

// SetRepeated sets the label for this field to repeated. It returns the field
// builder, for method chaining.
func (flb *FieldBuilder) SetRepeated() *FieldBuilder {
	return flb.SetCardinality(protoreflect.Repeated)
}

// SetRequired sets the label for this field to required. It returns the field
// builder, for method chaining.
//
// This is only allowed for files with "proto2" syntax. For editions, to set a
// field to be required, set the field_presence feature in the field options to
// descriptorpb.FeatureSet_LEGACY_REQUIRED.
func (flb *FieldBuilder) SetRequired() *FieldBuilder {
	return flb.SetCardinality(protoreflect.Required)
}

// SetOptional sets the label for this field to optional. It returns the field
// builder, for method chaining.
func (flb *FieldBuilder) SetOptional() *FieldBuilder {
	return flb.SetCardinality(protoreflect.Optional)
}

// IsRepeated returns true if this field's label is repeated. Fields created via
// NewMapField will be repeated (since map's are represented "under the hood" as
// a repeated field of map entry messages).
func (flb *FieldBuilder) IsRepeated() bool {
	return flb.Cardinality == protoreflect.Repeated
}

// IsRequired returns true if this field's label is required.
func (flb *FieldBuilder) IsRequired() bool {
	return flb.Cardinality == protoreflect.Required
}

// IsOptional returns true if this field's label is optional.
func (flb *FieldBuilder) IsOptional() bool {
	return flb.Cardinality == protoreflect.Optional
}

// IsMap returns true if this field is a map field.
func (flb *FieldBuilder) IsMap() bool {
	return flb.IsRepeated() &&
		flb.msgType != nil &&
		flb.fieldType.fieldType != descriptorpb.FieldDescriptorProto_TYPE_GROUP &&
		flb.msgType.Options != nil &&
		flb.msgType.Options.GetMapEntry()
}

// Type returns the field's type.
func (flb *FieldBuilder) Type() *FieldType {
	return flb.fieldType
}

// SetType changes the field's type and returns the field builder, for method
// chaining.
func (flb *FieldBuilder) SetType(ft *FieldType) *FieldBuilder {
	flb.fieldType = ft
	if flb.msgType != nil && flb.msgType != ft.localMsgType {
		Unlink(flb.msgType)
	}
	return flb
}

// SetDefaultValue changes the field's type and returns the field builder, for
// method chaining.
func (flb *FieldBuilder) SetDefaultValue(defValue string) *FieldBuilder {
	flb.Default = defValue
	return flb
}

// SetJsonName sets the name used in the field's JSON representation and then
// returns the field builder, for method chaining.
func (flb *FieldBuilder) SetJsonName(jsonName string) *FieldBuilder {
	flb.JsonName = jsonName
	return flb
}

// IsExtension returns true if this is an extension field.
func (flb *FieldBuilder) IsExtension() bool {
	return flb.localExtendee != nil || flb.foreignExtendee != nil
}

// ExtendeeTypeName returns the fully qualified name of the extended message
// or it returns an empty string if this is not an extension field.
func (flb *FieldBuilder) ExtendeeTypeName() protoreflect.FullName {
	if flb.foreignExtendee != nil {
		return flb.foreignExtendee.FullName()
	} else if flb.localExtendee != nil {
		return FullName(flb.localExtendee)
	} else {
		return ""
	}
}

func (flb *FieldBuilder) buildProto(path []int32, sourceInfo *descriptorpb.SourceCodeInfo, isMessageSet bool) (*descriptorpb.FieldDescriptorProto, error) {
	addCommentsTo(sourceInfo, path, &flb.comments)

	if flb.Proto3Optional {
		if flb.ParentFile().Syntax != protoreflect.Proto3 {
			return nil, fmt.Errorf("field %s is not in a proto3 syntax file but is marked as a proto3 optional field", FullName(flb))
		}
		if flb.IsExtension() {
			return nil, fmt.Errorf("field %s: extensions cannot be proto3 optional fields", FullName(flb))
		}
		if _, ok := flb.Parent().(*OneofBuilder); ok {
			return nil, fmt.Errorf("field %s: proto3 optional fields cannot belong to a oneof", FullName(flb))
		}
	}

	var lbl *descriptorpb.FieldDescriptorProto_Label
	if int32(flb.Cardinality) != 0 {
		if flb.ParentFile().Syntax != protoreflect.Proto2 && flb.Cardinality == protoreflect.Required {
			return nil, fmt.Errorf("field %s: only proto2 allows required fields", FullName(flb))
		}
		lbl = (descriptorpb.FieldDescriptorProto_Label)(flb.Cardinality).Enum()
	}
	if flb.ParentFile().Syntax != protoreflect.Proto2 && flb.fieldType.Kind() == protoreflect.GroupKind {
		return nil, fmt.Errorf("field %s: only proto2 allows group fields", FullName(flb))
	}
	var typeName *string
	tn := flb.fieldType.TypeName()
	if tn != "" {
		typeName = proto.String("." + string(tn))
	}
	var extendee *string
	if flb.IsExtension() {
		extendee = proto.String("." + string(flb.ExtendeeTypeName()))
	}
	jsName := flb.JsonName
	if jsName == "" {
		jsName = internal.JsonName(flb.name)
	}
	var def *string
	if flb.Default != "" {
		def = proto.String(flb.Default)
	}
	var proto3Optional *bool
	if flb.Proto3Optional {
		proto3Optional = proto.Bool(true)
	}

	maxTag := internal.GetMaxTag(isMessageSet)
	if flb.number > maxTag {
		return nil, fmt.Errorf("tag for field %s cannot be above max %d", FullName(flb), maxTag)
	}

	fd := &descriptorpb.FieldDescriptorProto{
		Name:           proto.String(string(flb.name)),
		Number:         proto.Int32(int32(flb.number)),
		Options:        flb.Options,
		Label:          lbl,
		Type:           flb.fieldType.fieldType.Enum(),
		TypeName:       typeName,
		JsonName:       proto.String(jsName),
		DefaultValue:   def,
		Extendee:       extendee,
		Proto3Optional: proto3Optional,
	}
	return fd, nil
}

// Build constructs a field descriptor based on the contents of this field
// builder. If there are any problems constructing the descriptor, including
// resolving symbols referenced by the builder or failing to meet certain
// validation rules, an error is returned.
func (flb *FieldBuilder) Build() (protoreflect.FieldDescriptor, error) {
	d, err := doBuild(flb, BuilderOptions{})
	if err != nil {
		return nil, err
	}
	fld := d.(protoreflect.FieldDescriptor)
	if fld.IsExtension() {
		if xtd, ok := fld.(protoreflect.ExtensionTypeDescriptor); ok {
			return xtd, nil
		}
		return extensionTypeDescriptor{fld, dynamicpb.NewExtensionType(fld)}, nil
	}
	return fld, nil
}

// BuildDescriptor constructs a field descriptor based on the contents of this
// field builder. Most usages will prefer Build() instead, whose return type is
// a concrete descriptor type. This method is present to satisfy the Builder
// interface.
func (flb *FieldBuilder) BuildDescriptor() (protoreflect.Descriptor, error) {
	return flb.Build()
}

type extensionTypeDescriptor struct {
	protoreflect.FieldDescriptor
	xt protoreflect.ExtensionType
}

var _ protoreflect.ExtensionTypeDescriptor = extensionTypeDescriptor{}

func (e extensionTypeDescriptor) Type() protoreflect.ExtensionType {
	return e.xt
}

func (e extensionTypeDescriptor) Descriptor() protoreflect.ExtensionDescriptor {
	return e.FieldDescriptor
}

// OneofBuilder is a builder used to construct a protoreflect.OneOfDescriptor. A oneof
// builder *must* be added to a message before calling its Build() method.
//
// To create a new OneofBuilder, use NewOneof.
type OneofBuilder struct {
	baseBuilder

	Options *descriptorpb.OneofOptions

	choices []*FieldBuilder
	symbols map[protoreflect.Name]*FieldBuilder
}

var _ Builder = (*OneofBuilder)(nil)

// NewOneof creates a new OneofBuilder for a oneof with the given name.
func NewOneof(name protoreflect.Name) *OneofBuilder {
	return &OneofBuilder{
		baseBuilder: baseBuilderWithName(name),
		symbols:     map[protoreflect.Name]*FieldBuilder{},
	}
}

// FromOneof returns a OneofBuilder that is effectively a copy of the given
// descriptor.
//
// Note that it is not just the given oneof that is copied but its entire file.
// So the caller can get the parent element of the returned builder and the
// result would be a builder that is effectively a copy of the oneof
// descriptor's parent message.
//
// This means that oneof builders created from descriptors do not need to be
// explicitly assigned to a file in order to preserve the original oneof's
// package name.
//
// This function returns an error if the given descriptor is synthetic.
func FromOneof(ood protoreflect.OneofDescriptor) (*OneofBuilder, error) {
	if ood.IsSynthetic() {
		return nil, fmt.Errorf("oneof %s is synthetic", ood.FullName())
	}
	if fb, err := FromFile(ood.ParentFile()); err != nil {
		return nil, err
	} else if oob, ok := fb.findFullyQualifiedElement(ood.FullName()).(*OneofBuilder); ok {
		return oob, nil
	} else {
		return nil, fmt.Errorf("could not find oneof %s after converting file %q to builder", ood.FullName(), ood.ParentFile().Path())
	}
}

func fromOneof(ood protoreflect.OneofDescriptor) (*OneofBuilder, error) {
	oob := NewOneof(ood.Name())
	var err error
	oob.Options, err = protomessage.As[*descriptorpb.OneofOptions](ood.Options())
	if err != nil {
		return nil, err
	}
	setComments(&oob.comments, ood.ParentFile().SourceLocations().ByDescriptor(ood))

	fields := ood.Fields()
	for i, length := 0, fields.Len(); i < length; i++ {
		fld := fields.Get(i)
		if flb, err := fromField(fld); err != nil {
			return nil, err
		} else if err := oob.TryAddChoice(flb); err != nil {
			return nil, err
		}
	}

	return oob, nil
}

// SetName changes this oneof's name, returning the oneof builder for method
// chaining. If the given new name is not valid (e.g. TrySetName would have
// returned an error) then this method will panic.
func (oob *OneofBuilder) SetName(newName protoreflect.Name) *OneofBuilder {
	if err := oob.TrySetName(newName); err != nil {
		panic(err)
	}
	return oob
}

// TrySetName changes this oneof's name. It will return an error if the given
// new name is not a valid protobuf identifier or if the parent message builder
// already has an element with the given name.
func (oob *OneofBuilder) TrySetName(newName protoreflect.Name) error {
	return oob.baseBuilder.setName(oob, newName)
}

// SetComments sets the comments associated with the oneof. This method
// returns the oneof builder, for method chaining.
func (oob *OneofBuilder) SetComments(c Comments) *OneofBuilder {
	oob.comments = c
	return oob
}

// Children returns any builders assigned to this oneof builder. These will
// be choices for the oneof, each of which will be a field builder.
func (oob *OneofBuilder) Children() iter.Seq[Builder] {
	return func(yield func(Builder) bool) {
		for _, flb := range oob.choices {
			if !yield(flb) {
				return
			}
		}
	}
}

func (oob *OneofBuilder) parent() *MessageBuilder {
	if oob.baseBuilder.parent == nil {
		return nil
	}
	return oob.baseBuilder.parent.(*MessageBuilder)
}

func (oob *OneofBuilder) findChild(_ protoreflect.Name) Builder {
	// in terms of finding a child by qualified name, fields in the
	// oneof are considered children of the message, not the oneof
	return nil
}

func (oob *OneofBuilder) removeChild(b Builder) {
	if p, ok := b.Parent().(*OneofBuilder); !ok || p != oob {
		return
	}

	if oob.parent() != nil {
		// remove from message's name and tag maps
		flb := b.(*FieldBuilder)
		delete(oob.parent().fieldTags, flb.Number())
		delete(oob.parent().symbols, flb.Name())
		if flb.msgType != nil {
			delete(oob.parent().symbols, flb.msgType.Name())
		}
	}

	oob.choices = deleteBuilder(b.Name(), oob.choices).([]*FieldBuilder)
	delete(oob.symbols, b.Name())
	b.setParent(nil)
}

func (oob *OneofBuilder) renamedChild(b Builder, oldName protoreflect.Name) error {
	if p, ok := b.Parent().(*OneofBuilder); !ok || p != oob {
		return nil
	}

	if err := oob.addSymbol(b.(*FieldBuilder)); err != nil {
		return err
	}

	// update message's name map (to make sure new field name doesn't
	// collide with other kinds of elements in the message)
	if oob.parent() != nil {
		if err := oob.parent().addSymbol(b); err != nil {
			delete(oob.symbols, b.Name())
			return err
		}
		delete(oob.parent().symbols, oldName)
	}

	delete(oob.symbols, oldName)
	return nil
}

func (oob *OneofBuilder) addSymbol(b *FieldBuilder) error {
	if _, ok := oob.symbols[b.Name()]; ok {
		return fmt.Errorf("oneof %s already contains field named %q", FullName(oob), b.Name())
	}
	oob.symbols[b.Name()] = b
	return nil
}

// GetChoice returns the field with the given name. If no such field exists in
// the oneof, nil is returned.
func (oob *OneofBuilder) GetChoice(name protoreflect.Name) *FieldBuilder {
	return oob.symbols[name]
}

// RemoveChoice removes the field with the given name. If no such field exists
// in the oneof, this is a no-op. This returns the oneof builder, for method
// chaining.
func (oob *OneofBuilder) RemoveChoice(name protoreflect.Name) *OneofBuilder {
	oob.TryRemoveChoice(name)
	return oob
}

// TryRemoveChoice removes the field with the given name and returns false if
// the oneof has no such field.
func (oob *OneofBuilder) TryRemoveChoice(name protoreflect.Name) bool {
	if flb, ok := oob.symbols[name]; ok {
		oob.removeChild(flb)
		return true
	}
	return false
}

// AddChoice adds the given field to this oneof. If an error prevents the field
// from being added, this method panics. If the given field is an extension,
// this method panics. If the given field is a group or map field or if it is
// not optional (e.g. it is required or repeated), this method panics. This
// returns the oneof builder, for method chaining.
func (oob *OneofBuilder) AddChoice(flb *FieldBuilder) *OneofBuilder {
	if err := oob.TryAddChoice(flb); err != nil {
		panic(err)
	}
	return oob
}

// TryAddChoice adds the given field to this oneof, returning any error that
// prevents the field from being added (such as a name collision with another
// element already added to the enclosing message). An error is returned if the
// given field is an extension field, a map or group field, or repeated or
// required.
func (oob *OneofBuilder) TryAddChoice(flb *FieldBuilder) error {
	if flb.IsExtension() {
		return fmt.Errorf("field %s is an extension, not a regular field", flb.Name())
	}
	if flb.msgType != nil && flb.fieldType.fieldType != descriptorpb.FieldDescriptorProto_TYPE_GROUP {
		return fmt.Errorf("cannot add a map field %q to oneof %s", flb.name, FullName(oob))
	}
	if flb.IsRepeated() || flb.IsRequired() {
		return fmt.Errorf("fields in a oneof must be optional, %s is %v", flb.name, flb.Cardinality)
	}
	if err := oob.addSymbol(flb); err != nil {
		return err
	}
	mb := oob.parent()
	if mb != nil {
		// If we are moving field from a message to a oneof that belongs to the
		// same message, we have to use different order of operations to prevent
		// failure (otherwise, it looks like it's being added twice).
		// (We do similar if moving the other direction, from the oneof into
		// the message to which oneof belongs.)
		needToUnlinkFirst := mb.isPresentButNotChild(flb)
		if needToUnlinkFirst {
			Unlink(flb)
			if err := mb.registerField(flb); err != nil {
				// Should never happen since, before above Unlink, it was already
				// registered with this message.
				// But if somehow it DOES happen, the field will now be orphaned :(
				return err
			}
		} else {
			if err := mb.registerField(flb); err != nil {
				delete(oob.symbols, flb.Name())
				return err
			}
			Unlink(flb)
		}
	}
	flb.setParent(oob)
	oob.choices = append(oob.choices, flb)
	return nil
}

// SetOptions sets the oneof options for this oneof and returns the oneof,
// for method chaining.
func (oob *OneofBuilder) SetOptions(options *descriptorpb.OneofOptions) *OneofBuilder {
	oob.Options = options
	return oob
}

func (oob *OneofBuilder) buildProto(path []int32, sourceInfo *descriptorpb.SourceCodeInfo) (*descriptorpb.OneofDescriptorProto, error) {
	addCommentsTo(sourceInfo, path, &oob.comments)

	for _, flb := range oob.choices {
		if flb.IsRepeated() || flb.IsRequired() {
			return nil, fmt.Errorf("fields in a oneof must be optional, %s is %v", FullName(flb), flb.Cardinality)
		}
	}

	return &descriptorpb.OneofDescriptorProto{
		Name:    proto.String(string(oob.name)),
		Options: oob.Options,
	}, nil
}

// Build constructs a oneof descriptor based on the contents of this oneof
// builder. If there are any problems constructing the descriptor, including
// resolving symbols referenced by the builder or failing to meet certain
// validation rules, an error is returned.
func (oob *OneofBuilder) Build() (protoreflect.OneofDescriptor, error) {
	ood, err := oob.BuildDescriptor()
	if err != nil {
		return nil, err
	}
	return ood.(protoreflect.OneofDescriptor), nil
}

// BuildDescriptor constructs a oneof descriptor based on the contents of this
// oneof builder. Most usages will prefer Build() instead, whose return type is
// a concrete descriptor type. This method is present to satisfy the Builder
// interface.
func (oob *OneofBuilder) BuildDescriptor() (protoreflect.Descriptor, error) {
	return doBuild(oob, BuilderOptions{})
}

func entryTypeName(fieldName protoreflect.Name) protoreflect.Name {
	return protoreflect.Name(internal.InitCap(internal.JsonName(fieldName)) + "Entry")
}
