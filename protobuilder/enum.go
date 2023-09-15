package protobuilder

import (
	"fmt"
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

// EnumRange is a range of enum numbers. The first element is the start
// of the range, inclusive, and the second element is the end of the range,
// inclusive. (Note: this differs from FieldRange, where the end of the
// range is exclusive.)
type EnumRange [2]protoreflect.EnumNumber

// EnumBuilder is a builder used to construct a protoreflect.EnumDescriptor.
//
// To create a new EnumBuilder, use NewEnum.
type EnumBuilder struct {
	baseBuilder

	Options        *descriptorpb.EnumOptions
	ReservedRanges []EnumRange
	ReservedNames  []protoreflect.Name

	values  []*EnumValueBuilder
	symbols map[protoreflect.Name]*EnumValueBuilder
}

var _ Builder = (*EnumBuilder)(nil)

// NewEnum creates a new EnumBuilder for an enum with the given name. Since the
// new message has no parent element, it also has no package name (e.g. it is in
// the unnamed package, until it is assigned to a file builder that defines a
// package name).
func NewEnum(name protoreflect.Name) *EnumBuilder {
	return &EnumBuilder{
		baseBuilder: baseBuilderWithName(name),
		symbols:     map[protoreflect.Name]*EnumValueBuilder{},
	}
}

// FromEnum returns an EnumBuilder that is effectively a copy of the given
// descriptor.
//
// Note that it is not just the given enum that is copied but its entire file.
// So the caller can get the parent element of the returned builder and the
// result would be a builder that is effectively a copy of the enum descriptor's
// parent.
//
// This means that enum builders created from descriptors do not need to be
// explicitly assigned to a file in order to preserve the original enum's
// package name.
func FromEnum(ed protoreflect.EnumDescriptor) (*EnumBuilder, error) {
	if fb, err := FromFile(ed.ParentFile()); err != nil {
		return nil, err
	} else if eb, ok := fb.findFullyQualifiedElement(ed.FullName()).(*EnumBuilder); ok {
		return eb, nil
	} else {
		return nil, fmt.Errorf("could not find enum %s after converting file %q to builder", ed.FullName(), ed.ParentFile().Path())
	}
}

func fromEnum(ed protoreflect.EnumDescriptor, localEnums map[protoreflect.EnumDescriptor]*EnumBuilder) (*EnumBuilder, error) {
	eb := NewEnum(ed.Name())
	var err error
	eb.Options, err = protoresolve.As[*descriptorpb.EnumOptions](ed.Options())
	if err != nil {
		return nil, err
	}
	ranges := ed.ReservedRanges()
	eb.ReservedRanges = make([]EnumRange, ranges.Len())
	for i, length := 0, ranges.Len(); i < length; i++ {
		eb.ReservedRanges[i] = ranges.Get(i)
	}
	names := ed.ReservedNames()
	eb.ReservedNames = make([]protoreflect.Name, names.Len())
	for i, length := 0, names.Len(); i < length; i++ {
		eb.ReservedNames[i] = names.Get(i)
	}
	setComments(&eb.comments, ed.ParentFile().SourceLocations().ByDescriptor(ed))

	localEnums[ed] = eb

	vals := ed.Values()
	for i, length := 0, vals.Len(); i < length; i++ {
		evd := vals.Get(i)
		if evb, err := fromEnumValue(evd); err != nil {
			return nil, err
		} else if err := eb.TryAddValue(evb); err != nil {
			return nil, err
		}
	}

	return eb, nil
}

// SetName changes this enum's name, returning the enum builder for method
// chaining. If the given new name is not valid (e.g. TrySetName would have
// returned an error) then this method will panic.
func (eb *EnumBuilder) SetName(newName protoreflect.Name) *EnumBuilder {
	if err := eb.TrySetName(newName); err != nil {
		panic(err)
	}
	return eb
}

// TrySetName changes this enum's name. It will return an error if the given new
// name is not a valid protobuf identifier or if the parent builder already has
// an element with the given name.
func (eb *EnumBuilder) TrySetName(newName protoreflect.Name) error {
	return eb.baseBuilder.setName(eb, newName)
}

// SetComments sets the comments associated with the enum. This method returns
// the enum builder, for method chaining.
func (eb *EnumBuilder) SetComments(c Comments) *EnumBuilder {
	eb.comments = c
	return eb
}

// Children returns any builders assigned to this enum builder. These will be
// the enum's values.
func (eb *EnumBuilder) Children() []Builder {
	var ch []Builder
	for _, evb := range eb.values {
		ch = append(ch, evb)
	}
	return ch
}

func (eb *EnumBuilder) findChild(name protoreflect.Name) Builder {
	return eb.symbols[name]
}

func (eb *EnumBuilder) removeChild(b Builder) {
	if p, ok := b.Parent().(*EnumBuilder); !ok || p != eb {
		return
	}
	eb.values = deleteBuilder(b.Name(), eb.values).([]*EnumValueBuilder)
	delete(eb.symbols, b.Name())
	b.setParent(nil)
}

func (eb *EnumBuilder) renamedChild(b Builder, oldName protoreflect.Name) error {
	if p, ok := b.Parent().(*EnumBuilder); !ok || p != eb {
		return nil
	}

	if err := eb.addSymbol(b.(*EnumValueBuilder)); err != nil {
		return err
	}
	delete(eb.symbols, oldName)
	return nil
}

func (eb *EnumBuilder) addSymbol(b *EnumValueBuilder) error {
	if _, ok := eb.symbols[b.Name()]; ok {
		return fmt.Errorf("enum %s already contains value named %q", FullName(eb), b.Name())
	}
	eb.symbols[b.Name()] = b
	return nil
}

// SetOptions sets the enum options for this enum and returns the enum, for
// method chaining.
func (eb *EnumBuilder) SetOptions(options *descriptorpb.EnumOptions) *EnumBuilder {
	eb.Options = options
	return eb
}

// GetValue returns the enum value with the given name. If no such value exists
// in the enum, nil is returned.
func (eb *EnumBuilder) GetValue(name protoreflect.Name) *EnumValueBuilder {
	return eb.symbols[name]
}

// RemoveValue removes the enum value with the given name. If no such value
// exists in the enum, this is a no-op. This returns the enum builder, for
// method chaining.
func (eb *EnumBuilder) RemoveValue(name protoreflect.Name) *EnumBuilder {
	eb.TryRemoveValue(name)
	return eb
}

// TryRemoveValue removes the enum value with the given name and returns false
// if the enum has no such value.
func (eb *EnumBuilder) TryRemoveValue(name protoreflect.Name) bool {
	if evb, ok := eb.symbols[name]; ok {
		eb.removeChild(evb)
		return true
	}
	return false
}

// AddValue adds the given enum value to this enum. If an error prevents the
// value from being added, this method panics. This returns the enum builder,
// for method chaining.
func (eb *EnumBuilder) AddValue(evb *EnumValueBuilder) *EnumBuilder {
	if err := eb.TryAddValue(evb); err != nil {
		panic(err)
	}
	return eb
}

// TryAddValue adds the given enum value to this enum, returning any error that
// prevents the value from being added (such as a name collision with another
// value already added to the enum).
func (eb *EnumBuilder) TryAddValue(evb *EnumValueBuilder) error {
	if err := eb.addSymbol(evb); err != nil {
		return err
	}
	Unlink(evb)
	evb.setParent(eb)
	eb.values = append(eb.values, evb)
	return nil
}

// AddReservedRange adds the given reserved range to this message. The range is
// inclusive of both the start and end, just like defining a range in proto IDL
// source. This returns the message, for method chaining.
func (eb *EnumBuilder) AddReservedRange(start, end protoreflect.EnumNumber) *EnumBuilder {
	eb.ReservedRanges = append(eb.ReservedRanges, EnumRange{start, end})
	return eb
}

// SetReservedRanges replaces all of this enum's reserved ranges with the
// given slice of ranges. This returns the enum, for method chaining.
func (eb *EnumBuilder) SetReservedRanges(ranges []EnumRange) *EnumBuilder {
	eb.ReservedRanges = ranges
	return eb
}

// AddReservedName adds the given name to the list of reserved value names for
// this enum. This returns the enum, for method chaining.
func (eb *EnumBuilder) AddReservedName(name protoreflect.Name) *EnumBuilder {
	eb.ReservedNames = append(eb.ReservedNames, name)
	return eb
}

// SetReservedNames replaces all of this enum's reserved value names with the
// given slice of names. This returns the enum, for method chaining.
func (eb *EnumBuilder) SetReservedNames(names []protoreflect.Name) *EnumBuilder {
	eb.ReservedNames = names
	return eb
}

func (eb *EnumBuilder) buildProto(path []int32, sourceInfo *descriptorpb.SourceCodeInfo) (*descriptorpb.EnumDescriptorProto, error) {
	addCommentsTo(sourceInfo, path, &eb.comments)

	var needNumbersAssigned []*descriptorpb.EnumValueDescriptorProto
	values := make([]*descriptorpb.EnumValueDescriptorProto, 0, len(eb.values))
	for _, evb := range eb.values {
		path := append(path, internal.EnumValuesTag, int32(len(values)))
		evp, err := evb.buildProto(path, sourceInfo)
		if err != nil {
			return nil, err
		}
		values = append(values, evp)
		if !evb.numberSet {
			needNumbersAssigned = append(needNumbersAssigned, evp)
		}
	}

	if len(needNumbersAssigned) > 0 {
		tags := make([]int, len(values)-len(needNumbersAssigned))
		tagsIndex := 0
		for _, evb := range eb.values {
			if evb.numberSet {
				tags[tagsIndex] = int(evb.number)
				tagsIndex++
			}
		}
		sort.Ints(tags)
		t := 0
		ti := sort.Search(len(tags), func(i int) bool {
			return tags[i] >= 0
		})
		if ti < len(tags) {
			tags = tags[ti:]
		}
		for len(needNumbersAssigned) > 0 {
			for len(tags) > 0 && t == tags[0] {
				t++
				tags = tags[1:]
			}
			needNumbersAssigned[0].Number = proto.Int32(int32(t))
			needNumbersAssigned = needNumbersAssigned[1:]
			t++
		}
	}

	resRanges := make([]*descriptorpb.EnumDescriptorProto_EnumReservedRange, len(eb.ReservedRanges))
	for i, r := range eb.ReservedRanges {
		resRanges[i] = &descriptorpb.EnumDescriptorProto_EnumReservedRange{
			Start: proto.Int32(int32(r[0])),
			End:   proto.Int32(int32(r[1])),
		}
	}
	resNames := make([]string, len(eb.ReservedNames))
	for i, name := range eb.ReservedNames {
		resNames[i] = string(name)
	}

	return &descriptorpb.EnumDescriptorProto{
		Name:          proto.String(string(eb.name)),
		Options:       eb.Options,
		Value:         values,
		ReservedRange: resRanges,
		ReservedName:  resNames,
	}, nil
}

// Build constructs an enum descriptor based on the contents of this enum
// builder. If there are any problems constructing the descriptor, including
// resolving symbols referenced by the builder or failing to meet certain
// validation rules, an error is returned.
func (eb *EnumBuilder) Build() (protoreflect.EnumDescriptor, error) {
	ed, err := eb.BuildDescriptor()
	if err != nil {
		return nil, err
	}
	return ed.(protoreflect.EnumDescriptor), nil
}

// BuildDescriptor constructs an enum descriptor based on the contents of this
// enum builder. Most usages will prefer Build() instead, whose return type
// is a concrete descriptor type. This method is present to satisfy the Builder
// interface.
func (eb *EnumBuilder) BuildDescriptor() (protoreflect.Descriptor, error) {
	return doBuild(eb, BuilderOptions{})
}

// EnumValueBuilder is a builder used to construct a protoreflect.EnumValueDescriptor.
// A enum value builder *must* be added to an enum before calling its Build()
// method.
//
// To create a new EnumValueBuilder, use NewEnumValue.
type EnumValueBuilder struct {
	baseBuilder

	number    protoreflect.EnumNumber
	numberSet bool
	Options   *descriptorpb.EnumValueOptions
}

var _ Builder = (*EnumValueBuilder)(nil)

// NewEnumValue creates a new EnumValueBuilder for an enum value with the given
// name. The return value's numeric value will not be set, which means it will
// be auto-assigned when the descriptor is built, unless explicitly set with a
// call to SetNumber.
func NewEnumValue(name protoreflect.Name) *EnumValueBuilder {
	return &EnumValueBuilder{baseBuilder: baseBuilderWithName(name)}
}

// FromEnumValue returns an EnumValueBuilder that is effectively a copy of the
// given descriptor.
//
// Note that it is not just the given enum value that is copied but its entire
// file. So the caller can get the parent element of the returned builder and
// the result would be a builder that is effectively a copy of the enum value
// descriptor's parent enum.
//
// This means that enum value builders created from descriptors do not need to
// be explicitly assigned to a file in order to preserve the original enum
// value's package name.
func FromEnumValue(evd protoreflect.EnumValueDescriptor) (*EnumValueBuilder, error) {
	if fb, err := FromFile(evd.ParentFile()); err != nil {
		return nil, err
	} else if evb, ok := fb.findFullyQualifiedElement(evd.FullName()).(*EnumValueBuilder); ok {
		return evb, nil
	} else {
		return nil, fmt.Errorf("could not find enum value %s after converting file %q to builder", evd.FullName(), evd.ParentFile().Path())
	}
}

func fromEnumValue(evd protoreflect.EnumValueDescriptor) (*EnumValueBuilder, error) {
	evb := NewEnumValue(evd.Name())
	var err error
	evb.Options, err = protoresolve.As[*descriptorpb.EnumValueOptions](evd.Options())
	if err != nil {
		return nil, err
	}
	evb.number = evd.Number()
	evb.numberSet = true
	setComments(&evb.comments, evd.ParentFile().SourceLocations().ByDescriptor(evd))

	return evb, nil
}

// SetName changes this enum value's name, returning the enum value builder for
// method chaining. If the given new name is not valid (e.g. TrySetName would
// have returned an error) then this method will panic.
func (evb *EnumValueBuilder) SetName(newName protoreflect.Name) *EnumValueBuilder {
	if err := evb.TrySetName(newName); err != nil {
		panic(err)
	}
	return evb
}

// TrySetName changes this enum value's name. It will return an error if the
// given new name is not a valid protobuf identifier or if the parent enum
// builder already has an enum value with the given name.
func (evb *EnumValueBuilder) TrySetName(newName protoreflect.Name) error {
	return evb.baseBuilder.setName(evb, newName)
}

// SetComments sets the comments associated with the enum value. This method
// returns the enum value builder, for method chaining.
func (evb *EnumValueBuilder) SetComments(c Comments) *EnumValueBuilder {
	evb.comments = c
	return evb
}

// Children returns nil, since enum values cannot have child elements. It is
// present to satisfy the Builder interface.
func (evb *EnumValueBuilder) Children() []Builder {
	// enum values do not have children
	return nil
}

func (evb *EnumValueBuilder) findChild(_ protoreflect.Name) Builder {
	// enum values do not have children
	return nil
}

func (evb *EnumValueBuilder) removeChild(_ Builder) {
	// enum values do not have children
}

func (evb *EnumValueBuilder) renamedChild(_ Builder, _ protoreflect.Name) error {
	// enum values do not have children
	return nil
}

// SetOptions sets the enum value options for this enum value and returns the
// enum value, for method chaining.
func (evb *EnumValueBuilder) SetOptions(options *descriptorpb.EnumValueOptions) *EnumValueBuilder {
	evb.Options = options
	return evb
}

// Number returns the enum value's numeric value. If the number has not been
// set this returns zero.
func (evb *EnumValueBuilder) Number() protoreflect.EnumNumber {
	return evb.number
}

// HasNumber returns whether or not the enum value's numeric value has been set.
// If it has not been set, it is auto-assigned when the descriptor is built.
func (evb *EnumValueBuilder) HasNumber() bool {
	return evb.numberSet
}

// ClearNumber clears this enum value's numeric value and then returns the enum
// value builder, for method chaining. After being cleared, the number will be
// auto-assigned when the descriptor is built, unless explicitly set by a
// subsequent call to SetNumber.
func (evb *EnumValueBuilder) ClearNumber() *EnumValueBuilder {
	evb.number = 0
	evb.numberSet = false
	return evb
}

// SetNumber changes the numeric value for this enum value and then returns the
// enum value, for method chaining.
func (evb *EnumValueBuilder) SetNumber(number protoreflect.EnumNumber) *EnumValueBuilder {
	evb.number = number
	evb.numberSet = true
	return evb
}

func (evb *EnumValueBuilder) buildProto(path []int32, sourceInfo *descriptorpb.SourceCodeInfo) (*descriptorpb.EnumValueDescriptorProto, error) {
	addCommentsTo(sourceInfo, path, &evb.comments)

	return &descriptorpb.EnumValueDescriptorProto{
		Name:    proto.String(string(evb.name)),
		Number:  proto.Int32(int32(evb.number)),
		Options: evb.Options,
	}, nil
}

// Build constructs an enum value descriptor based on the contents of this enum
// value builder. If there are any problems constructing the descriptor,
// including resolving symbols referenced by the builder or failing to meet
// certain validation rules, an error is returned.
func (evb *EnumValueBuilder) Build() (protoreflect.EnumValueDescriptor, error) {
	evd, err := evb.BuildDescriptor()
	if err != nil {
		return nil, err
	}
	return evd.(protoreflect.EnumValueDescriptor), nil
}

// BuildDescriptor constructs an enum value descriptor based on the contents of
// this enum value builder. Most usages will prefer Build() instead, whose
// return type is a concrete descriptor type. This method is present to satisfy
// the Builder interface.
func (evb *EnumValueBuilder) BuildDescriptor() (protoreflect.Descriptor, error) {
	return doBuild(evb, BuilderOptions{})
}
