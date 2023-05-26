package protobuilder

import (
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal"
)

// FieldRange is a range of field numbers. The first element is the start
// of the range, inclusive, and the second element is the end of the range,
// exclusive.
type FieldRange [2]protoreflect.FieldNumber

// ExtensionRange represents a range of extension numbers. It is a FieldRange
// that may have options.
type ExtensionRange struct {
	FieldRange
	Options *descriptorpb.ExtensionRangeOptions
}

// MessageBuilder is a builder used to construct a protoreflect.MessageDescriptor. A
// message builder can define nested messages, enums, and extensions in addition
// to defining the message's fields.
//
// Note that when building a descriptor from a MessageBuilder, not all protobuf
// validation rules are enforced. See the package documentation for more info.
//
// To create a new MessageBuilder, use NewMessage.
type MessageBuilder struct {
	baseBuilder

	Options         *descriptorpb.MessageOptions
	ExtensionRanges []ExtensionRange
	ReservedRanges  []FieldRange
	ReservedNames   []protoreflect.Name

	fieldsAndOneOfs  []Builder
	fieldTags        map[protoreflect.FieldNumber]*FieldBuilder
	nestedMessages   []*MessageBuilder
	nestedExtensions []*FieldBuilder
	nestedEnums      []*EnumBuilder
	symbols          map[protoreflect.Name]Builder
}

var _ Builder = (*MessageBuilder)(nil)

// NewMessage creates a new MessageBuilder for a message with the given path.
// Since the new message has no parent element, it also has no package path
// (e.g. it is in the unnamed package, until it is assigned to a file builder
// that defines a package path).
func NewMessage(name protoreflect.Name) *MessageBuilder {
	return &MessageBuilder{
		baseBuilder: baseBuilderWithName(name),
		fieldTags:   map[protoreflect.FieldNumber]*FieldBuilder{},
		symbols:     map[protoreflect.Name]Builder{},
	}
}

// FromMessage returns a MessageBuilder that is effectively a copy of the given
// descriptor.
//
// Note that it is not just the given message that is copied but its entire
// file. So the caller can get the parent element of the returned builder and
// the result would be a builder that is effectively a copy of the message
// descriptor's parent.
//
// This means that message builders created from descriptors do not need to be
// explicitly assigned to a file in order to preserve the original message's
// package path.
func FromMessage(md protoreflect.MessageDescriptor) (*MessageBuilder, error) {
	if fb, err := FromFile(md.ParentFile()); err != nil {
		return nil, err
	} else if mb, ok := fb.findFullyQualifiedElement(md.FullName()).(*MessageBuilder); ok {
		return mb, nil
	} else {
		return nil, fmt.Errorf("could not find message %s after converting file %q to builder", md.FullName(), md.ParentFile().Path())
	}
}

func fromMessage(md protoreflect.MessageDescriptor,
	localMessages map[protoreflect.MessageDescriptor]*MessageBuilder,
	localEnums map[protoreflect.EnumDescriptor]*EnumBuilder) (*MessageBuilder, error) {

	mb := NewMessage(md.Name())
	var err error
	mb.Options, err = as[*descriptorpb.MessageOptions](md.Options())
	if err != nil {
		return nil, err
	}
	ranges := md.ExtensionRanges()
	mb.ExtensionRanges = make([]ExtensionRange, ranges.Len())
	for i, length := 0, ranges.Len(); i < length; i++ {
		opts, err := as[*descriptorpb.ExtensionRangeOptions](md.ExtensionRangeOptions(i))
		if err != nil {
			return nil, err
		}
		mb.ExtensionRanges[i] = ExtensionRange{
			FieldRange: ranges.Get(i),
			Options:    opts,
		}
	}
	ranges = md.ReservedRanges()
	mb.ReservedRanges = make([]FieldRange, ranges.Len())
	for i, length := 0, ranges.Len(); i < length; i++ {
		mb.ReservedRanges[i] = ranges.Get(i)
	}
	names := md.ReservedNames()
	mb.ReservedNames = make([]protoreflect.Name, names.Len())
	for i, length := 0, names.Len(); i < length; i++ {
		mb.ReservedNames[i] = names.Get(i)
	}
	setComments(&mb.comments, md.ParentFile().SourceLocations().ByDescriptor(md))

	localMessages[md] = mb

	srcOneofs := md.Oneofs()
	oneofs := make([]*OneofBuilder, srcOneofs.Len())
	for i, length := 0, srcOneofs.Len(); i < length; i++ {
		ood := srcOneofs.Get(i)
		if ood.IsSynthetic() {
			continue
		}
		if oob, err := fromOneof(ood); err != nil {
			return nil, err
		} else {
			oneofs[i] = oob
		}
	}

	srcFields := md.Fields()
	for i, length := 0, srcFields.Len(); i < length; i++ {
		fld := srcFields.Get(i)
		oo := fld.ContainingOneof()
		if oo != nil && !oo.IsSynthetic() {
			// add one-ofs in the order of their first constituent field
			oob := oneofs[oo.Index()]
			if oob != nil {
				oneofs[oo.Index()] = nil
				if err := mb.TryAddOneOf(oob); err != nil {
					return nil, err
				}
			}
			continue
		}
		if flb, err := fromField(fld); err != nil {
			return nil, err
		} else if err := mb.TryAddField(flb); err != nil {
			return nil, err
		}
	}

	nestedMsgs := md.Messages()
	for i, length := 0, nestedMsgs.Len(); i < length; i++ {
		nmd := nestedMsgs.Get(i)
		if nmb, err := fromMessage(nmd, localMessages, localEnums); err != nil {
			return nil, err
		} else if err := mb.TryAddNestedMessage(nmb); err != nil {
			return nil, err
		}
	}
	nestedEnums := md.Enums()
	for i, length := 0, nestedEnums.Len(); i < length; i++ {
		ed := nestedEnums.Get(i)
		if eb, err := fromEnum(ed, localEnums); err != nil {
			return nil, err
		} else if err := mb.TryAddNestedEnum(eb); err != nil {
			return nil, err
		}
	}
	nestedExts := md.Extensions()
	for i, length := 0, nestedExts.Len(); i < length; i++ {
		exd := nestedExts.Get(i)
		if exb, err := fromField(exd); err != nil {
			return nil, err
		} else if err := mb.TryAddNestedExtension(exb); err != nil {
			return nil, err
		}
	}

	return mb, nil
}

// SetName changes this message's path, returning the message builder for method
// chaining. If the given new path is not valid (e.g. TrySetName would have
// returned an error) then this method will panic.
func (mb *MessageBuilder) SetName(newName protoreflect.Name) *MessageBuilder {
	if err := mb.TrySetName(newName); err != nil {
		panic(err)
	}
	return mb
}

// TrySetName changes this message's path. It will return an error if the given
// new path is not a valid protobuf identifier or if the parent builder already
// has an element with the given path.
//
// If the message is a map or group type whose parent is the corresponding map
// or group field, the parent field's enclosing message is checked for elements
// with a conflicting path. Despite the fact that these message types are
// modeled as children of their associated field builder, in the protobuf IDL
// they are actually all defined in the enclosing message's namespace.
func (mb *MessageBuilder) TrySetName(newName protoreflect.Name) error {
	if p, ok := mb.parent.(*FieldBuilder); ok && p.fieldType.fieldType != descriptorpb.FieldDescriptorProto_TYPE_GROUP {
		return fmt.Errorf("cannot change path of map entry %s; change path of field instead", FullName(mb))
	}
	return mb.trySetNameInternal(newName)
}

func (mb *MessageBuilder) trySetNameInternal(newName protoreflect.Name) error {
	return mb.baseBuilder.setName(mb, newName)
}

func (mb *MessageBuilder) setNameInternal(newName protoreflect.Name) {
	if err := mb.trySetNameInternal(newName); err != nil {
		panic(err)
	}
}

// SetComments sets the comments associated with the message. This method
// returns the message builder, for method chaining.
func (mb *MessageBuilder) SetComments(c Comments) *MessageBuilder {
	mb.comments = c
	return mb
}

// Children returns any builders assigned to this message builder. These will
// include the message's fields and one-ofs as well as any nested messages,
// extensions, and enums.
func (mb *MessageBuilder) Children() []Builder {
	ch := append([]Builder(nil), mb.fieldsAndOneOfs...)
	for _, nmb := range mb.nestedMessages {
		ch = append(ch, nmb)
	}
	for _, exb := range mb.nestedExtensions {
		ch = append(ch, exb)
	}
	for _, eb := range mb.nestedEnums {
		ch = append(ch, eb)
	}
	return ch
}

func (mb *MessageBuilder) findChild(name protoreflect.Name) Builder {
	return mb.symbols[name]
}

func (mb *MessageBuilder) removeChild(b Builder) {
	if p, ok := b.Parent().(*MessageBuilder); !ok || p != mb {
		return
	}

	switch b := b.(type) {
	case *FieldBuilder:
		if b.IsExtension() {
			mb.nestedExtensions = deleteBuilder(b.Name(), mb.nestedExtensions).([]*FieldBuilder)
		} else {
			mb.fieldsAndOneOfs = deleteBuilder(b.Name(), mb.fieldsAndOneOfs).([]Builder)
			delete(mb.fieldTags, b.Number())
			if b.msgType != nil {
				delete(mb.symbols, b.msgType.Name())
			}
		}
	case *OneofBuilder:
		mb.fieldsAndOneOfs = deleteBuilder(b.Name(), mb.fieldsAndOneOfs).([]Builder)
		for _, flb := range b.choices {
			delete(mb.symbols, flb.Name())
			delete(mb.fieldTags, flb.Number())
		}
	case *MessageBuilder:
		mb.nestedMessages = deleteBuilder(b.Name(), mb.nestedMessages).([]*MessageBuilder)
	case *EnumBuilder:
		mb.nestedEnums = deleteBuilder(b.Name(), mb.nestedEnums).([]*EnumBuilder)
	}
	delete(mb.symbols, b.Name())
	b.setParent(nil)
}

func (mb *MessageBuilder) renamedChild(b Builder, oldName protoreflect.Name) error {
	if p, ok := b.Parent().(*MessageBuilder); !ok || p != mb {
		return nil
	}

	if err := mb.addSymbol(b); err != nil {
		return err
	}
	delete(mb.symbols, oldName)
	return nil
}

func (mb *MessageBuilder) addSymbol(b Builder) error {
	if ex, ok := mb.symbols[b.Name()]; ok {
		return fmt.Errorf("message %s already contains element (%T) named %q", FullName(mb), ex, b.Name())
	}
	mb.symbols[b.Name()] = b
	return nil
}

func (mb *MessageBuilder) addTag(flb *FieldBuilder) error {
	if flb.number == 0 {
		return nil
	}
	if ex, ok := mb.fieldTags[flb.Number()]; ok {
		return fmt.Errorf("message %s already contains field with tag %d: %s", FullName(mb), flb.Number(), ex.Name())
	}
	mb.fieldTags[flb.Number()] = flb
	return nil
}

func (mb *MessageBuilder) registerField(flb *FieldBuilder) error {
	if err := mb.addSymbol(flb); err != nil {
		return err
	}
	if err := mb.addTag(flb); err != nil {
		delete(mb.symbols, flb.Name())
		return err
	}
	if flb.msgType != nil {
		if err := mb.addSymbol(flb.msgType); err != nil {
			delete(mb.symbols, flb.Name())
			delete(mb.fieldTags, flb.Number())
			return err
		}
	}
	return nil
}

// GetField returns the field with the given path. If no such field exists in
// the message, nil is returned. The field does not have to be an immediate
// child of this message but could instead be an indirect child via a one-of.
func (mb *MessageBuilder) GetField(name protoreflect.Name) *FieldBuilder {
	b := mb.symbols[name]
	if flb, ok := b.(*FieldBuilder); ok && !flb.IsExtension() {
		return flb
	} else {
		return nil
	}
}

// RemoveField removes the field with the given path. If no such field exists in
// the message, this is a no-op. If the field is part of a one-of, the one-of
// remains assigned to this message and the field is removed from it. This
// returns the message builder, for method chaining.
func (mb *MessageBuilder) RemoveField(name protoreflect.Name) *MessageBuilder {
	mb.TryRemoveField(name)
	return mb
}

// TryRemoveField removes the field with the given path and returns false if the
// message has no such field. If the field is part of a one-of, the one-of
// remains assigned to this message and the field is removed from it.
func (mb *MessageBuilder) TryRemoveField(name protoreflect.Name) bool {
	b := mb.symbols[name]
	if flb, ok := b.(*FieldBuilder); ok && !flb.IsExtension() {
		// parent could be mb, but could also be a one-of
		flb.Parent().removeChild(flb)
		return true
	}
	return false
}

// AddField adds the given field to this message. If an error prevents the field
// from being added, this method panics. If the given field is an extension,
// this method panics. This returns the message builder, for method chaining.
func (mb *MessageBuilder) AddField(flb *FieldBuilder) *MessageBuilder {
	if err := mb.TryAddField(flb); err != nil {
		panic(err)
	}
	return mb
}

// TryAddField adds the given field to this message, returning any error that
// prevents the field from being added (such as a path collision with another
// element already added to the message). An error is returned if the given
// field is an extension field.
func (mb *MessageBuilder) TryAddField(flb *FieldBuilder) error {
	if flb.IsExtension() {
		return fmt.Errorf("field %s is an extension, not a regular field", flb.Name())
	}
	// If we are moving field from a one-of that belongs to this message
	// directly to this message, we have to use different order of operations
	// to prevent failure (otherwise, it looks like it's being added twice).
	// (We do similar if moving the other direction, from message to a one-of
	// that is already assigned to same message.)
	needToUnlinkFirst := mb.isPresentButNotChild(flb)
	if needToUnlinkFirst {
		Unlink(flb)
		if err := mb.registerField(flb); err != nil {
			// Should never happen since, before above Unlink, it was already
			// registered with this message (just indirectly, via a oneof).
			// But if some it DOES happen, the field will now be orphaned :(
			return err
		}
	} else {
		if err := mb.registerField(flb); err != nil {
			return err
		}
		Unlink(flb)
	}
	flb.setParent(mb)
	mb.fieldsAndOneOfs = append(mb.fieldsAndOneOfs, flb)
	return nil
}

// GetOneOf returns the one-of with the given path. If no such one-of exists in
// the message, nil is returned.
func (mb *MessageBuilder) GetOneOf(name protoreflect.Name) *OneofBuilder {
	b := mb.symbols[name]
	if oob, ok := b.(*OneofBuilder); ok {
		return oob
	} else {
		return nil
	}
}

// RemoveOneOf removes the one-of with the given path. If no such one-of exists
// in the message, this is a no-op. This returns the message builder, for method
// chaining.
func (mb *MessageBuilder) RemoveOneOf(name protoreflect.Name) *MessageBuilder {
	mb.TryRemoveOneOf(name)
	return mb
}

// TryRemoveOneOf removes the one-of with the given path and returns false if
// the message has no such one-of.
func (mb *MessageBuilder) TryRemoveOneOf(name protoreflect.Name) bool {
	b := mb.symbols[name]
	if oob, ok := b.(*OneofBuilder); ok {
		mb.removeChild(oob)
		return true
	}
	return false
}

// AddOneOf adds the given one-of to this message. If an error prevents the
// one-of from being added, this method panics. This returns the message
// builder, for method chaining.
func (mb *MessageBuilder) AddOneOf(oob *OneofBuilder) *MessageBuilder {
	if err := mb.TryAddOneOf(oob); err != nil {
		panic(err)
	}
	return mb
}

// TryAddOneOf adds the given one-of to this message, returning any error that
// prevents the one-of from being added (such as a path collision with another
// element already added to the message).
func (mb *MessageBuilder) TryAddOneOf(oob *OneofBuilder) error {
	if err := mb.addSymbol(oob); err != nil {
		return err
	}
	// add nested fields to symbol and tag map
	for i, flb := range oob.choices {
		if err := mb.registerField(flb); err != nil {
			// must undo all additions we've made so far
			delete(mb.symbols, oob.Name())
			for i > 1 {
				i--
				flb := oob.choices[i]
				delete(mb.symbols, flb.Name())
				delete(mb.fieldTags, flb.Number())
			}
			return err
		}
	}
	Unlink(oob)
	oob.setParent(mb)
	mb.fieldsAndOneOfs = append(mb.fieldsAndOneOfs, oob)
	return nil
}

// GetNestedMessage returns the nested message with the given path. If no such
// message exists, nil is returned. The named message must be in this message's
// scope. If the message is nested more deeply, this will return nil. This means
// the message must be a direct child of this message or a child of one of this
// message's fields (e.g. the group type for a group field or a map entry for a
// map field).
func (mb *MessageBuilder) GetNestedMessage(name protoreflect.Name) *MessageBuilder {
	b := mb.symbols[name]
	if nmb, ok := b.(*MessageBuilder); ok {
		return nmb
	} else {
		return nil
	}
}

// RemoveNestedMessage removes the nested message with the given path. If no
// such message exists, this is a no-op. This returns the message builder, for
// method chaining.
func (mb *MessageBuilder) RemoveNestedMessage(name protoreflect.Name) *MessageBuilder {
	mb.TryRemoveNestedMessage(name)
	return mb
}

// TryRemoveNestedMessage removes the nested message with the given path and
// returns false if this message has no nested message with that path. If the
// named message is a child of a field (e.g. the group type for a group field or
// the map entry for a map field), it is removed from that field and thus
// removed from this message's scope.
func (mb *MessageBuilder) TryRemoveNestedMessage(name protoreflect.Name) bool {
	b := mb.symbols[name]
	if nmb, ok := b.(*MessageBuilder); ok {
		// parent could be mb, but could also be a field (if the message
		// is the field's group or map entry type)
		nmb.Parent().removeChild(nmb)
		return true
	}
	return false
}

// AddNestedMessage adds the given message as a nested child of this message. If
// an error prevents the message from being added, this method panics. This
// returns the message builder, for method chaining.
func (mb *MessageBuilder) AddNestedMessage(nmb *MessageBuilder) *MessageBuilder {
	if err := mb.TryAddNestedMessage(nmb); err != nil {
		panic(err)
	}
	return mb
}

// TryAddNestedMessage adds the given message as a nested child of this message,
// returning any error that prevents the message from being added (such as a
// path collision with another element already added to the message).
func (mb *MessageBuilder) TryAddNestedMessage(nmb *MessageBuilder) error {
	// If we are moving nested message from field (map entry or group type)
	// directly to this message, we have to use different order of operations
	// to prevent failure (otherwise, it looks like it's being added twice).
	// (We don't need to do similar for the other direction, because that isn't
	// possible: you can't add messages to a field, they can only be constructed
	// that way using NewGroupField or NewMapField.)
	needToUnlinkFirst := mb.isPresentButNotChild(nmb)
	if needToUnlinkFirst {
		Unlink(nmb)
		_ = mb.addSymbol(nmb)
	} else {
		if err := mb.addSymbol(nmb); err != nil {
			return err
		}
		Unlink(nmb)
	}
	nmb.setParent(mb)
	mb.nestedMessages = append(mb.nestedMessages, nmb)
	return nil
}

func (mb *MessageBuilder) isPresentButNotChild(b Builder) bool {
	if p, ok := b.Parent().(*MessageBuilder); ok && p == mb {
		// it's a child
		return false
	}
	return mb.symbols[b.Name()] == b
}

// GetNestedExtension returns the nested extension with the given path. If no
// such extension exists, nil is returned. The named extension must be in this
// message's scope. If the extension is nested more deeply, this will return
// nil. This means the extension must be a direct child of this message.
func (mb *MessageBuilder) GetNestedExtension(name protoreflect.Name) *FieldBuilder {
	b := mb.symbols[name]
	if exb, ok := b.(*FieldBuilder); ok && exb.IsExtension() {
		return exb
	} else {
		return nil
	}
}

// RemoveNestedExtension removes the nested extension with the given path. If no
// such extension exists, this is a no-op. This returns the message builder, for
// method chaining.
func (mb *MessageBuilder) RemoveNestedExtension(name protoreflect.Name) *MessageBuilder {
	mb.TryRemoveNestedExtension(name)
	return mb
}

// TryRemoveNestedExtension removes the nested extension with the given path and
// returns false if this message has no nested extension with that path.
func (mb *MessageBuilder) TryRemoveNestedExtension(name protoreflect.Name) bool {
	b := mb.symbols[name]
	if exb, ok := b.(*FieldBuilder); ok && exb.IsExtension() {
		mb.removeChild(exb)
		return true
	}
	return false
}

// AddNestedExtension adds the given extension as a nested child of this
// message. If an error prevents the extension from being added, this method
// panics. This returns the message builder, for method chaining.
func (mb *MessageBuilder) AddNestedExtension(exb *FieldBuilder) *MessageBuilder {
	if err := mb.TryAddNestedExtension(exb); err != nil {
		panic(err)
	}
	return mb
}

// TryAddNestedExtension adds the given extension as a nested child of this
// message, returning any error that prevents the extension from being added
// (such as a path collision with another element already added to the message).
func (mb *MessageBuilder) TryAddNestedExtension(exb *FieldBuilder) error {
	if !exb.IsExtension() {
		return fmt.Errorf("field %s is not an extension", exb.Name())
	}
	if err := mb.addSymbol(exb); err != nil {
		return err
	}
	Unlink(exb)
	exb.setParent(mb)
	mb.nestedExtensions = append(mb.nestedExtensions, exb)
	return nil
}

// GetNestedEnum returns the nested enum with the given path. If no such enum
// exists, nil is returned. The named enum must be in this message's scope. If
// the enum is nested more deeply, this will return nil. This means the enum
// must be a direct child of this message.
func (mb *MessageBuilder) GetNestedEnum(name protoreflect.Name) *EnumBuilder {
	b := mb.symbols[name]
	if eb, ok := b.(*EnumBuilder); ok {
		return eb
	} else {
		return nil
	}
}

// RemoveNestedEnum removes the nested enum with the given path. If no such enum
// exists, this is a no-op. This returns the message builder, for method
// chaining.
func (mb *MessageBuilder) RemoveNestedEnum(name protoreflect.Name) *MessageBuilder {
	mb.TryRemoveNestedEnum(name)
	return mb
}

// TryRemoveNestedEnum removes the nested enum with the given path and returns
// false if this message has no nested enum with that path.
func (mb *MessageBuilder) TryRemoveNestedEnum(name protoreflect.Name) bool {
	b := mb.symbols[name]
	if eb, ok := b.(*EnumBuilder); ok {
		mb.removeChild(eb)
		return true
	}
	return false
}

// AddNestedEnum adds the given enum as a nested child of this message. If an
// error prevents the enum from being added, this method panics. This returns
// the message builder, for method chaining.
func (mb *MessageBuilder) AddNestedEnum(eb *EnumBuilder) *MessageBuilder {
	if err := mb.TryAddNestedEnum(eb); err != nil {
		panic(err)
	}
	return mb
}

// TryAddNestedEnum adds the given enum as a nested child of this message,
// returning any error that prevents the enum from being added (such as a path
// collision with another element already added to the message).
func (mb *MessageBuilder) TryAddNestedEnum(eb *EnumBuilder) error {
	if err := mb.addSymbol(eb); err != nil {
		return err
	}
	Unlink(eb)
	eb.setParent(mb)
	mb.nestedEnums = append(mb.nestedEnums, eb)
	return nil
}

// SetOptions sets the message options for this message and returns the message,
// for method chaining.
func (mb *MessageBuilder) SetOptions(options *descriptorpb.MessageOptions) *MessageBuilder {
	mb.Options = options
	return mb
}

// AddExtensionRange adds the given extension range to this message. The range
// is inclusive of the start but exclusive of the end. This returns the message,
// for method chaining.
func (mb *MessageBuilder) AddExtensionRange(start, end protoreflect.FieldNumber) *MessageBuilder {
	return mb.AddExtensionRangeWithOptions(start, end, nil)
}

// AddExtensionRangeWithOptions adds the given extension range to this message.
// The range is inclusive of the start but exclusive of the end. This returns the
// message, for method chaining.
func (mb *MessageBuilder) AddExtensionRangeWithOptions(start, end protoreflect.FieldNumber, options *descriptorpb.ExtensionRangeOptions) *MessageBuilder {
	er := ExtensionRange{
		FieldRange: [2]protoreflect.FieldNumber{start, end},
		Options:    options,
	}
	mb.ExtensionRanges = append(mb.ExtensionRanges, er)
	return mb
}

// SetExtensionRanges replaces all of this message's extension ranges with the
// given slice of ranges. This returns the message, for method chaining.
func (mb *MessageBuilder) SetExtensionRanges(ranges []ExtensionRange) *MessageBuilder {
	mb.ExtensionRanges = ranges
	return mb
}

// AddReservedRange adds the given reserved range to this message. The range is
// inclusive of the start but exclusive of the end. This returns the message,
// for method chaining.
func (mb *MessageBuilder) AddReservedRange(start, end protoreflect.FieldNumber) *MessageBuilder {
	rr := FieldRange{start, end}
	mb.ReservedRanges = append(mb.ReservedRanges, rr)
	return mb
}

// SetReservedRanges replaces all of this message's reserved ranges with the
// given slice of ranges. This returns the message, for method chaining.
func (mb *MessageBuilder) SetReservedRanges(ranges []FieldRange) *MessageBuilder {
	mb.ReservedRanges = ranges
	return mb
}

// AddReservedName adds the given path to the list of reserved field names for
// this message. This returns the message, for method chaining.
func (mb *MessageBuilder) AddReservedName(name protoreflect.Name) *MessageBuilder {
	mb.ReservedNames = append(mb.ReservedNames, name)
	return mb
}

// SetReservedNames replaces all of this message's reserved field names with the
// given slice of names. This returns the message, for method chaining.
func (mb *MessageBuilder) SetReservedNames(names []protoreflect.Name) *MessageBuilder {
	mb.ReservedNames = names
	return mb
}

func (mb *MessageBuilder) buildProto(path []int32, sourceInfo *descriptorpb.SourceCodeInfo) (*descriptorpb.DescriptorProto, error) {
	addCommentsTo(sourceInfo, path, &mb.comments)

	var needTagsAssigned []*descriptorpb.FieldDescriptorProto
	nestedMessages := make([]*descriptorpb.DescriptorProto, 0, len(mb.nestedMessages))
	oneOfCount := 0
	for _, b := range mb.fieldsAndOneOfs {
		if _, ok := b.(*OneofBuilder); ok {
			oneOfCount++
		}
	}

	fields := make([]*descriptorpb.FieldDescriptorProto, 0, len(mb.fieldsAndOneOfs)-oneOfCount)
	oneOfs := make([]*descriptorpb.OneofDescriptorProto, 0, oneOfCount)

	addField := func(flb *FieldBuilder, fld *descriptorpb.FieldDescriptorProto) error {
		fields = append(fields, fld)
		if flb.number == 0 {
			needTagsAssigned = append(needTagsAssigned, fld)
		}
		if flb.msgType != nil {
			nmpath := append(path, internal.Message_nestedMessagesTag, int32(len(nestedMessages)))
			if entry, err := flb.msgType.buildProto(nmpath, sourceInfo); err != nil {
				return err
			} else {
				nestedMessages = append(nestedMessages, entry)
			}
		}
		return nil
	}

	for _, b := range mb.fieldsAndOneOfs {
		if flb, ok := b.(*FieldBuilder); ok {
			fldpath := append(path, internal.Message_fieldsTag, int32(len(fields)))
			fld, err := flb.buildProto(fldpath, sourceInfo, mb.Options.GetMessageSetWireFormat())
			if err != nil {
				return nil, err
			}
			if err := addField(flb, fld); err != nil {
				return nil, err
			}
		} else {
			oopath := append(path, internal.Message_oneOfsTag, int32(len(oneOfs)))
			oob := b.(*OneofBuilder)
			oobIndex := len(oneOfs)
			ood, err := oob.buildProto(oopath, sourceInfo)
			if err != nil {
				return nil, err
			}
			oneOfs = append(oneOfs, ood)
			for _, flb := range oob.choices {
				path := append(path, internal.Message_fieldsTag, int32(len(fields)))
				fld, err := flb.buildProto(path, sourceInfo, mb.Options.GetMessageSetWireFormat())
				if err != nil {
					return nil, err
				}
				fld.OneofIndex = proto.Int32(int32(oobIndex))
				if err := addField(flb, fld); err != nil {
					return nil, err
				}
			}
		}
	}

	if len(needTagsAssigned) > 0 {
		tags := make([]int, len(fields)-len(needTagsAssigned))
		tagsIndex := 0
		for _, fld := range fields {
			tag := fld.GetNumber()
			if tag != 0 {
				tags[tagsIndex] = int(tag)
				tagsIndex++
			}
		}
		sort.Ints(tags)
		t := 1
		for len(needTagsAssigned) > 0 {
			for len(tags) > 0 && t == tags[0] {
				t++
				tags = tags[1:]
			}
			needTagsAssigned[0].Number = proto.Int32(int32(t))
			needTagsAssigned = needTagsAssigned[1:]
			t++
		}
	}

	for _, nmb := range mb.nestedMessages {
		path := append(path, internal.Message_nestedMessagesTag, int32(len(nestedMessages)))
		if nmd, err := nmb.buildProto(path, sourceInfo); err != nil {
			return nil, err
		} else {
			nestedMessages = append(nestedMessages, nmd)
		}
	}

	nestedExtensions := make([]*descriptorpb.FieldDescriptorProto, 0, len(mb.nestedExtensions))
	for _, exb := range mb.nestedExtensions {
		path := append(path, internal.Message_extensionsTag, int32(len(nestedExtensions)))
		if exd, err := exb.buildProto(path, sourceInfo, isExtendeeMessageSet(exb)); err != nil {
			return nil, err
		} else {
			nestedExtensions = append(nestedExtensions, exd)
		}
	}

	nestedEnums := make([]*descriptorpb.EnumDescriptorProto, 0, len(mb.nestedEnums))
	for _, eb := range mb.nestedEnums {
		path := append(path, internal.Message_enumsTag, int32(len(nestedEnums)))
		if ed, err := eb.buildProto(path, sourceInfo); err != nil {
			return nil, err
		} else {
			nestedEnums = append(nestedEnums, ed)
		}
	}

	extRanges := make([]*descriptorpb.DescriptorProto_ExtensionRange, len(mb.ExtensionRanges))
	for i, r := range mb.ExtensionRanges {
		extRanges[i] = &descriptorpb.DescriptorProto_ExtensionRange{
			Start:   proto.Int32(int32(r.FieldRange[0])),
			End:     proto.Int32(int32(r.FieldRange[1])),
			Options: r.Options,
		}
	}
	resRanges := make([]*descriptorpb.DescriptorProto_ReservedRange, len(mb.ReservedRanges))
	for i, r := range mb.ReservedRanges {
		resRanges[i] = &descriptorpb.DescriptorProto_ReservedRange{
			Start: proto.Int32(int32(r[0])),
			End:   proto.Int32(int32(r[1])),
		}
	}
	resNames := make([]string, len(mb.ReservedNames))
	for i, name := range mb.ReservedNames {
		resNames[i] = string(name)
	}

	md := &descriptorpb.DescriptorProto{
		Name:           proto.String(string(mb.name)),
		Options:        mb.Options,
		Field:          fields,
		OneofDecl:      oneOfs,
		NestedType:     nestedMessages,
		EnumType:       nestedEnums,
		Extension:      nestedExtensions,
		ExtensionRange: extRanges,
		ReservedName:   resNames,
		ReservedRange:  resRanges,
	}

	if mb.ParentFile().Syntax == protoreflect.Proto3 {
		processProto3OptionalFields(md)
	}

	return md, nil
}

// Build constructs a message descriptor based on the contents of this message
// builder. If there are any problems constructing the descriptor, including
// resolving symbols referenced by the builder or failing to meet certain
// validation rules, an error is returned.
func (mb *MessageBuilder) Build() (protoreflect.MessageDescriptor, error) {
	md, err := mb.BuildDescriptor()
	if err != nil {
		return nil, err
	}
	return md.(protoreflect.MessageDescriptor), nil
}

// BuildDescriptor constructs a message descriptor based on the contents of this
// message builder. Most usages will prefer Build() instead, whose return type
// is a concrete descriptor type. This method is present to satisfy the Builder
// interface.
func (mb *MessageBuilder) BuildDescriptor() (protoreflect.Descriptor, error) {
	return doBuild(mb, BuilderOptions{})
}

// processProto3OptionalFields adds synthetic oneofs to the given message descriptor
// for each proto3 optional field. It also updates the fields to have the correct
// oneof index reference.
func processProto3OptionalFields(msgd *descriptorpb.DescriptorProto) {
	var allNames map[string]struct{}
	for _, fd := range msgd.Field {
		if fd.GetProto3Optional() {
			// lazy init the set of all names
			if allNames == nil {
				allNames = map[string]struct{}{}
				for _, fd := range msgd.Field {
					allNames[fd.GetName()] = struct{}{}
				}
				for _, od := range msgd.OneofDecl {
					allNames[od.GetName()] = struct{}{}
				}
				// NB: protoc only considers names of other fields and oneofs
				// when computing the synthetic oneof name. But that feels like
				// a bug, since it means it could generate a name that conflicts
				// with some other symbol defined in the message. If it's decided
				// that's NOT a bug and is desirable, then we should remove the
				// following four loops to mimic protoc's behavior.
				for _, xd := range msgd.Extension {
					allNames[xd.GetName()] = struct{}{}
				}
				for _, ed := range msgd.EnumType {
					allNames[ed.GetName()] = struct{}{}
					for _, evd := range ed.Value {
						allNames[evd.GetName()] = struct{}{}
					}
				}
				for _, fd := range msgd.NestedType {
					allNames[fd.GetName()] = struct{}{}
				}
				for _, n := range msgd.ReservedName {
					allNames[n] = struct{}{}
				}
			}

			// Compute a name for the synthetic oneof. This uses the same
			// algorithm as used in protoc:
			//  https://github.com/protocolbuffers/protobuf/blob/74ad62759e0a9b5a21094f3fb9bb4ebfaa0d1ab8/src/google/protobuf/compiler/parser.cc#L785-L803
			ooName := fd.GetName()
			if !strings.HasPrefix(ooName, "_") {
				ooName = "_" + ooName
			}
			for {
				_, ok := allNames[ooName]
				if !ok {
					// found a unique name
					allNames[ooName] = struct{}{}
					break
				}
				ooName = "X" + ooName
			}

			fd.OneofIndex = proto.Int32(int32(len(msgd.OneofDecl)))
			ood := &descriptorpb.OneofDescriptorProto{Name: proto.String(ooName)}
			msgd.OneofDecl = append(msgd.OneofDecl, ood)
		}
	}
}
