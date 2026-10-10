package protoprint

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func (p *Printer) qualifyMessageOptionName(pkg, scope, fqn protoreflect.FullName) string {
	// Message options must at least include the message scope, even if the option
	// is inside that message. We do that by requiring we have at least one
	// enclosing skip in the qualified name.
	return p.qualifyElementName(pkg, scope, fqn, 1)
}

func (p *Printer) qualifyExtensionLiteralName(pkg, scope, fqn protoreflect.FullName) string {
	// In message literals, extensions can have package name omitted but may not
	// have any other scopes omitted. We signal that via negative arg.
	return p.qualifyElementName(pkg, scope, fqn, -1)
}

// TODO: Neeeds to be reconciled with https://protobuf.com/docs/language-spec#reference-resolution.
func (p *Printer) qualifyName(pkg, scope, fqn protoreflect.FullName) string {
	return p.qualifyElementName(pkg, scope, fqn, 0)
}

func (p *Printer) qualifyElementName(pkg, scope, fqn protoreflect.FullName, required int) string {
	if p.ForceFullyQualifiedNames {
		// forcing fully-qualified names; make sure to include preceding dot
		if fqn[0] == '.' {
			return string(fqn)
		}
		return fmt.Sprintf(".%s", fqn)
	}

	// compute relative name (so no leading dot)
	if fqn[0] == '.' {
		fqn = fqn[1:]
	}
	if required < 0 {
		scope = pkg + "."
	} else if len(scope) > 0 && scope[len(scope)-1] != '.' {
		scope = scope + "."
	}
	count := 0
	for scope != "" {
		if strings.HasPrefix(string(fqn), string(scope)) && count >= required {
			return string(fqn[len(scope):])
		}
		if scope == pkg+"." {
			break
		}
		pos := strings.LastIndex(string(scope[:len(scope)-1]), ".")
		scope = scope[:pos+1]
		count++
	}
	return string(fqn)
}

// qualifyTypeName is like qualifyName but falls back to a fully-qualified
// leading-dot form if the relative result would begin with a proto keyword
// that takes a different grammar branch at the start of a field
// declaration, extend block, or RPC method type. The keyword set is the
// union of the keywords the spec excludes from field-type identifiers in
// regular, oneof, and extension field declarations, from RPC method type
// identifiers, and from the visibility-modifier position introduced by
// editions.
func (p *Printer) qualifyTypeName(pkg, scope, fqn protoreflect.FullName) string {
	name := p.qualifyName(pkg, scope, fqn)
	if p.ForceFullyQualifiedNames || strings.HasPrefix(name, ".") {
		return name
	}
	first, _, _ := strings.Cut(name, ".")
	switch first {
	case "group",
		"optional", "required", "repeated",
		"map",
		"message", "enum", "oneof",
		"reserved", "extensions", "extend",
		"option",
		"stream",
		// Added in edition 2024.
		"export", "local":
		if fqn[0] != '.' {
			return "." + string(fqn)
		}
		return string(fqn)
	}
	return name
}

func (p *Printer) typeString(fld protoreflect.FieldDescriptor, scope protoreflect.FullName) string {
	if fld.IsMap() {
		return fmt.Sprintf("map<%s, %s>", p.typeString(fld.MapKey(), scope), p.typeString(fld.MapValue(), scope))
	}
	switch fld.Kind() {
	case protoreflect.EnumKind:
		return p.qualifyTypeName(fld.ParentFile().Package(), scope, fld.Enum().FullName())
	case protoreflect.GroupKind:
		if isGroup(fld) {
			return string(fld.Message().Name())
		}
		fallthrough
	case protoreflect.MessageKind:
		return p.qualifyTypeName(fld.ParentFile().Package(), scope, fld.Message().FullName())
	default:
		return fld.Kind().String()
	}
}
