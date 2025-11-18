package protobuilder

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	maxSupportedEdition = descriptorpb.Edition_EDITION_2024
)

func computeSyntaxOrEdition(b Builder, opts BuilderOptions) (descriptorpb.Edition, error) {
	constraints := editionConstraints{}
	if err := buildEditionConstraints(b, constraints); err != nil {
		return 0, err
	}
	requiredEds := constraints.required()
	if len(requiredEds) > 1 {
		return 0, editionConstraintRequiredConflict(requiredEds, constraints)
	}
	if len(requiredEds) == 1 {
		return requiredEds[0], nil
	}
	// Try configured default.
	defaultVal := max(descriptorpb.Edition_EDITION_PROTO2, min(maxSupportedEdition, opts.DefaultSyntaxOrEdition))
	if constraints.allowed(defaultVal) {
		return defaultVal, nil
	}
	// If that doesn't work, find the first edition that does work.
	for edition := range supportedEditions() {
		if constraints.allowed(edition) {
			return edition, nil
		}
	}
	return 0, editionConstraintNotAllowedConflict(constraints)
}

type editionConstraintSetting int

const (
	editionAllowed = editionConstraintSetting(iota)
	editionNotAllowed
	editionRequired
)

type editionConstraint struct {
	setting editionConstraintSetting
	why     map[string]struct{}
}

type editionConstraints map[descriptorpb.Edition]editionConstraint

func (ec editionConstraints) set(ed descriptorpb.Edition, val editionConstraintSetting, why1 string, whyOthers ...string) error {
	why := make([]string, 1, len(whyOthers)+1)
	why[0] = why1
	why = append(why, whyOthers...)
	existing := ec[ed]
	var whySet map[string]struct{}
	if val == editionNotAllowed && existing.setting == editionRequired {
		return editionConstraintConflict(ed, slices.Collect(maps.Keys(existing.why)), why)
	} else if val == editionRequired && existing.setting == editionNotAllowed {
		return editionConstraintConflict(ed, why, slices.Collect(maps.Keys(existing.why)))
	} else if val == existing.setting {
		whySet = existing.why
	} else {
		whySet = make(map[string]struct{}, len(why))
	}
	sliceToSet(why, whySet)
	ec[ed] = editionConstraint{
		setting: val,
		why:     whySet,
	}
	return nil
}

func (ec editionConstraints) required() []descriptorpb.Edition {
	var required []descriptorpb.Edition
	for ed, val := range ec {
		if val.setting == editionRequired {
			required = append(required, ed)
		}
	}
	return required
}

func (ec editionConstraints) allowed(edition descriptorpb.Edition) bool {
	return ec[edition].setting != editionNotAllowed
}

func buildEditionConstraints(b Builder, constraints editionConstraints) error {
	for child := range b.Children() {
		if err := buildEditionConstraints(child, constraints); err != nil {
			return err
		}
	}
	switch b := b.(type) {
	case *FileBuilder:
		return buildEditionConstraintsForFile(b, constraints)
	case *MessageBuilder:
		return buildEditionConstraintsForMessage(b, constraints)
	case *FieldBuilder:
		return buildEditionConstraintsForField(b, constraints)
	case *OneofBuilder:
		return buildEditionConstraintsForOneof(b, constraints)
	case *EnumBuilder:
		return buildEditionConstraintsForEnum(b, constraints)
	case *EnumValueBuilder:
		return buildEditionConstraintsForEnumValue(b, constraints)
	case *ServiceBuilder:
		return buildEditionConstraintsForService(b, constraints)
	case *MethodBuilder:
		return buildEditionConstraintsForMethod(b, constraints)
	default:
		return fmt.Errorf("unrecognized builder kind: %T", b)
	}
}

func buildEditionConstraintsForFile(b *FileBuilder, constraints editionConstraints) error {
	return buildEditionConstraintsForOptions(b.Options, constraints)
}

func buildEditionConstraintsForMessage(b *MessageBuilder, constraints editionConstraints) error {
	if err := buildEditionConstraintsForOptions(b.Options, constraints); err != nil {
		return err
	}
	if err := buildEditionConstraintsForVisibility("message", b.Visibility, constraints); err != nil {
		return err
	}
	if len(b.ExtensionRanges) > 0 {
		if err := constraints.set(descriptorpb.Edition_EDITION_PROTO3, editionNotAllowed, "extension ranges"); err != nil {
			return err
		}
	}
	return nil
}

func buildEditionConstraintsForField(b *FieldBuilder, constraints editionConstraints) error {
	if err := buildEditionConstraintsForOptions(b.Options, constraints); err != nil {
		return err
	}
	// Only proto2 supports required and group fields.
	if b.Cardinality == protoreflect.Required {
		if err := constraints.set(descriptorpb.Edition_EDITION_PROTO2, editionRequired, "required field"); err != nil {
			return err
		}
	}
	if b.Type().Kind() == protoreflect.GroupKind {
		if err := constraints.set(descriptorpb.Edition_EDITION_PROTO2, editionRequired, "group field"); err != nil {
			return err
		}
	}
	// Only proto3 supports proto3_optional fields.
	if b.Proto3Optional {
		if err := constraints.set(descriptorpb.Edition_EDITION_PROTO3, editionRequired, "proto3 optional field"); err != nil {
			return err
		}
	}
	if b.IsExtension() && !allowedProto3Extendee(b.ExtendeeTypeName()) {
		if err := constraints.set(descriptorpb.Edition_EDITION_PROTO3, editionNotAllowed, "extension field that is not a custom option"); err != nil {
			return err
		}
	}
	return nil
}

func buildEditionConstraintsForOneof(b *OneofBuilder, constraints editionConstraints) error {
	return buildEditionConstraintsForOptions(b.Options, constraints)
}

func buildEditionConstraintsForEnum(b *EnumBuilder, constraints editionConstraints) error {
	if err := buildEditionConstraintsForOptions(b.Options, constraints); err != nil {
		return err
	}
	if err := buildEditionConstraintsForVisibility("enum", b.Visibility, constraints); err != nil {
		return err
	}
	// If there is no zero value, it can't be proto3.
	for _, en := range b.values {
		if en.HasNumber() && en.Number() == 0 {
			return nil
		}
	}
	return constraints.set(descriptorpb.Edition_EDITION_PROTO3, editionNotAllowed, "enum without zero value")
}

func buildEditionConstraintsForEnumValue(b *EnumValueBuilder, constraints editionConstraints) error {
	return buildEditionConstraintsForOptions(b.Options, constraints)
}

func buildEditionConstraintsForService(b *ServiceBuilder, constraints editionConstraints) error {
	return buildEditionConstraintsForOptions(b.Options, constraints)
}

func buildEditionConstraintsForMethod(b *MethodBuilder, constraints editionConstraints) error {
	return buildEditionConstraintsForOptions(b.Options, constraints)
}

func buildEditionConstraintsForOptions(opts proto.Message, constraints editionConstraints) error {
	optsRef := opts.ProtoReflect()
	featuresField := optsRef.Descriptor().Fields().ByName("features")
	if featuresField == nil || !optsRef.Has(featuresField) {
		return nil
	}
	// Non-editions syntaxes cannot refer to the features field.
	if err := constraints.set(descriptorpb.Edition_EDITION_PROTO2, editionNotAllowed, "edition features"); err != nil {
		return err
	}
	return constraints.set(descriptorpb.Edition_EDITION_PROTO3, editionNotAllowed, "edition features")
}

func buildEditionConstraintsForVisibility(typ string, vis descriptorpb.SymbolVisibility, constraints editionConstraints) error {
	if vis == descriptorpb.SymbolVisibility_VISIBILITY_UNSET || vis == descriptorpb.SymbolVisibility_VISIBILITY_EXPORT {
		return nil
	}
	// Only Edition 2024 and newer support limiting symbol visibility.
	if err := constraints.set(descriptorpb.Edition_EDITION_PROTO2, editionNotAllowed, typ+" visibility"); err != nil {
		return err
	}
	if err := constraints.set(descriptorpb.Edition_EDITION_PROTO3, editionNotAllowed, typ+" visibility"); err != nil {
		return err
	}
	return constraints.set(descriptorpb.Edition_EDITION_2023, editionNotAllowed, typ+" visibility")
}

func sliceToSet[E comparable](slice []E, set map[E]struct{}) {
	for _, e := range slice {
		set[e] = struct{}{}
	}
}

func editionConstraintConflict(
	conflict descriptorpb.Edition,
	requiredWhy []string,
	notAllowedWhy []string,
) error {
	return fmt.Errorf("invalid builder: conflicting features in use: %s required due to use of %s, but not allowed due to use of %s",
		editionString(conflict), combine(requiredWhy), combine(notAllowedWhy))
}

func editionConstraintRequiredConflict(
	requiredEditions []descriptorpb.Edition,
	constraints editionConstraints,
) error {
	var requiredWhy, notAllowedWhy []string
	for i, edition := range requiredEditions {
		whySlice := slices.Collect(maps.Keys(constraints[edition].why))
		if i == 0 {
			requiredWhy = whySlice
		} else {
			notAllowedWhy = append(notAllowedWhy, whySlice...)
		}
	}
	return editionConstraintConflict(requiredEditions[0], requiredWhy, notAllowedWhy)
}

func editionConstraintNotAllowedConflict(
	constraints editionConstraints,
) error {
	var reasons strings.Builder
	first := true
	for edition := range supportedEditions() {
		if first {
			first = false
		} else {
			reasons.WriteString("; ")
		}
		reasons.WriteString(editionString(edition))
		reasons.WriteString(" is not allowed due to use of ")
		whySlice := slices.Collect(maps.Keys(constraints[edition].why))
		reasons.WriteString(combine(whySlice))
	}
	return fmt.Errorf("invalid builder: no syntax or edition is valid: %v", reasons)
}

func editionString(ed descriptorpb.Edition) string {
	switch ed {
	case descriptorpb.Edition_EDITION_PROTO2:
		return "syntax proto2"
	case descriptorpb.Edition_EDITION_PROTO3:
		return "syntax proto3"
	default:
		return "edition " + strings.ToLower(strings.TrimPrefix(ed.String(), "EDITION_"))
	}
}

func combine(strs []string) string {
	slices.Sort(strs)
	switch len(strs) {
	case 1:
		return strs[0]
	case 2:
		return strs[0] + " and " + strs[1]
	default:
		var sb strings.Builder
		last := len(strs) - 1
		for i, str := range strs {
			if i == last {
				sb.WriteString(", and ")
			} else if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(str)
		}
		return sb.String()
	}
}

var (
	allowedProto3ExtendeeTypes = map[protoreflect.FullName]struct{}{
		(*descriptorpb.FileOptions)(nil).ProtoReflect().Descriptor().FullName():           {},
		(*descriptorpb.MessageOptions)(nil).ProtoReflect().Descriptor().FullName():        {},
		(*descriptorpb.FieldOptions)(nil).ProtoReflect().Descriptor().FullName():          {},
		(*descriptorpb.OneofOptions)(nil).ProtoReflect().Descriptor().FullName():          {},
		(*descriptorpb.ExtensionRangeOptions)(nil).ProtoReflect().Descriptor().FullName(): {},
		(*descriptorpb.EnumOptions)(nil).ProtoReflect().Descriptor().FullName():           {},
		(*descriptorpb.EnumValueOptions)(nil).ProtoReflect().Descriptor().FullName():      {},
		(*descriptorpb.ServiceOptions)(nil).ProtoReflect().Descriptor().FullName():        {},
		(*descriptorpb.MethodOptions)(nil).ProtoReflect().Descriptor().FullName():         {},
	}
)

func allowedProto3Extendee(extendee protoreflect.FullName) bool {
	_, ok := allowedProto3ExtendeeTypes[extendee]
	return ok
}

func supportedEditions() iter.Seq[descriptorpb.Edition] {
	return func(yield func(descriptorpb.Edition) bool) {
		if !yield(descriptorpb.Edition_EDITION_PROTO2) {
			return
		}
		if !yield(descriptorpb.Edition_EDITION_PROTO3) {
			return
		}
		for edition := descriptorpb.Edition_EDITION_2023; edition <= maxSupportedEdition; edition++ {
			if _, ok := descriptorpb.Edition_name[int32(edition)]; !ok {
				// not a defined value
				continue
			}
			if !yield(edition) {
				return
			}
		}
	}
}
