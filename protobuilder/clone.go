package protobuilder

import (
	"maps"
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Clone returns a deep copy of the given builder. It is the same as
// Builder.Clone, but with a strongly-typed return value.
func Clone[T Builder](b T) T {
	return b.Clone().(T)
}

// cloner makes deep copies of builders. Copying is done in two passes. The
// first copies the builders, recording each copy. The second updates
// references, so that any reference to a builder that was copied refers to
// the copy instead.
type cloner struct {
	// clones maps each builder that was copied to its copy.
	clones map[Builder]Builder
}

func newCloner() *cloner {
	return &cloner{clones: map[Builder]Builder{}}
}

func (c *cloner) file(fb *FileBuilder) *FileBuilder {
	clone := &FileBuilder{
		path:            fb.path,
		Syntax:          fb.Syntax,
		Edition:         fb.Edition,
		Package:         fb.Package,
		Options:         cloneOptions(fb.Options),
		comments:        cloneComments(fb.comments),
		SyntaxComments:  cloneComments(fb.SyntaxComments),
		PackageComments: cloneComments(fb.PackageComments),
		explicitDeps:    maps.Clone(fb.explicitDeps),
		explicitImports: maps.Clone(fb.explicitImports),
	}
	c.clones[fb] = clone
	clone.messages = cloneChildren(fb.messages, clone, c.message)
	clone.extensions = cloneChildren(fb.extensions, clone, c.field)
	clone.enums = cloneChildren(fb.enums, clone, c.enum)
	clone.services = cloneChildren(fb.services, clone, c.service)
	clone.symbols = cloneSymbols(c, fb.symbols)
	fb.origExts.RangeExtensions(func(xt protoreflect.ExtensionType) bool {
		// This can't fail: the extensions came from a valid registry.
		_ = clone.origExts.RegisterExtension(xt)
		return true
	})
	return clone
}

func (c *cloner) message(mb *MessageBuilder) *MessageBuilder {
	clone := &MessageBuilder{
		baseBuilder:     cloneBase(mb.baseBuilder),
		Options:         cloneOptions(mb.Options),
		ExtensionRanges: slices.Clone(mb.ExtensionRanges),
		ReservedRanges:  slices.Clone(mb.ReservedRanges),
		ReservedNames:   slices.Clone(mb.ReservedNames),
	}
	for i := range clone.ExtensionRanges {
		clone.ExtensionRanges[i].Options = cloneOptions(clone.ExtensionRanges[i].Options)
	}
	c.clones[mb] = clone
	clone.fieldsAndOneofs = cloneChildren(mb.fieldsAndOneofs, clone, c.fieldOrOneof)
	clone.nestedMessages = cloneChildren(mb.nestedMessages, clone, c.message)
	clone.nestedExtensions = cloneChildren(mb.nestedExtensions, clone, c.field)
	clone.nestedEnums = cloneChildren(mb.nestedEnums, clone, c.enum)
	clone.fieldTags = cloneSymbols(c, mb.fieldTags)
	clone.symbols = cloneSymbols(c, mb.symbols)
	return clone
}

func (c *cloner) field(flb *FieldBuilder) *FieldBuilder {
	clone := &FieldBuilder{
		baseBuilder:     cloneBase(flb.baseBuilder),
		number:          flb.number,
		Options:         cloneOptions(flb.Options),
		Cardinality:     flb.Cardinality,
		Proto3Optional:  flb.Proto3Optional,
		Default:         flb.Default,
		JSONName:        flb.JSONName,
		foreignExtendee: flb.foreignExtendee,
		localExtendee:   flb.localExtendee,
	}
	if flb.fieldType != nil {
		fieldType := *flb.fieldType
		clone.fieldType = &fieldType
	}
	c.clones[flb] = clone
	if flb.msgType != nil {
		clone.msgType = c.message(flb.msgType)
		clone.msgType.setParent(clone)
	}
	return clone
}

func (c *cloner) oneof(oob *OneofBuilder) *OneofBuilder {
	clone := &OneofBuilder{
		baseBuilder: cloneBase(oob.baseBuilder),
		Options:     cloneOptions(oob.Options),
	}
	c.clones[oob] = clone
	clone.choices = cloneChildren(oob.choices, clone, c.field)
	clone.symbols = cloneSymbols(c, oob.symbols)
	return clone
}

func (c *cloner) enum(eb *EnumBuilder) *EnumBuilder {
	clone := &EnumBuilder{
		baseBuilder:    cloneBase(eb.baseBuilder),
		Options:        cloneOptions(eb.Options),
		ReservedRanges: slices.Clone(eb.ReservedRanges),
		ReservedNames:  slices.Clone(eb.ReservedNames),
	}
	c.clones[eb] = clone
	clone.values = cloneChildren(eb.values, clone, c.enumValue)
	clone.symbols = cloneSymbols(c, eb.symbols)
	return clone
}

func (c *cloner) enumValue(evb *EnumValueBuilder) *EnumValueBuilder {
	clone := &EnumValueBuilder{
		baseBuilder: cloneBase(evb.baseBuilder),
		number:      evb.number,
		numberSet:   evb.numberSet,
		Options:     cloneOptions(evb.Options),
	}
	c.clones[evb] = clone
	return clone
}

func (c *cloner) service(sb *ServiceBuilder) *ServiceBuilder {
	clone := &ServiceBuilder{
		baseBuilder: cloneBase(sb.baseBuilder),
		Options:     cloneOptions(sb.Options),
	}
	c.clones[sb] = clone
	clone.methods = cloneChildren(sb.methods, clone, c.method)
	clone.symbols = cloneSymbols(c, sb.symbols)
	return clone
}

func (c *cloner) method(mtb *MethodBuilder) *MethodBuilder {
	clone := &MethodBuilder{
		baseBuilder: cloneBase(mtb.baseBuilder),
		Options:     cloneOptions(mtb.Options),
		ReqType:     cloneRPCType(mtb.ReqType),
		RespType:    cloneRPCType(mtb.RespType),
	}
	c.clones[mtb] = clone
	return clone
}

// fieldOrOneof copies a field or a oneof, the two kinds of builders in a
// message's fieldsAndOneofs.
func (c *cloner) fieldOrOneof(b Builder) Builder {
	if oob, ok := b.(*OneofBuilder); ok {
		return c.oneof(oob)
	}
	return c.field(b.(*FieldBuilder))
}

// updateReferences changes references to builders that were copied so that
// they refer to the copies. This only needs to examine the copies, since
// those are the only builders whose references should change.
func (c *cloner) updateReferences() {
	for _, b := range c.clones {
		switch b := b.(type) {
		case *FileBuilder:
			if b.explicitDeps != nil {
				deps := make(map[*FileBuilder]struct{}, len(b.explicitDeps))
				for dep := range b.explicitDeps {
					deps[cloneRef(c, dep)] = struct{}{}
				}
				b.explicitDeps = deps
			}
		case *FieldBuilder:
			if b.fieldType != nil {
				b.fieldType.localMsgType = cloneRef(c, b.fieldType.localMsgType)
				b.fieldType.localEnumType = cloneRef(c, b.fieldType.localEnumType)
			}
			b.localExtendee = cloneRef(c, b.localExtendee)
		case *MethodBuilder:
			if b.ReqType != nil {
				b.ReqType.localType = cloneRef(c, b.ReqType.localType)
			}
			if b.RespType != nil {
				b.RespType.localType = cloneRef(c, b.RespType.localType)
			}
		}
	}
}

// cloneRef returns the copy of the given builder if it was copied. Otherwise,
// it returns the given builder.
func cloneRef[T Builder](c *cloner, b T) T {
	if clone, ok := c.clones[b]; ok {
		return clone.(T)
	}
	return b
}

// cloneChildren copies the given children, using the given function, and makes
// the given parent their parent.
func cloneChildren[T Builder](children []T, parent Builder, cloneFn func(T) T) []T {
	clones := make([]T, len(children))
	for i, child := range children {
		clones[i] = cloneFn(child)
		clones[i].setParent(parent)
	}
	return clones
}

// cloneSymbols copies the given map, whose values are builders that have
// already been copied, replacing each value with its copy.
func cloneSymbols[K comparable, V Builder](c *cloner, symbols map[K]V) map[K]V {
	clones := make(map[K]V, len(symbols))
	for k, v := range symbols {
		clones[k] = c.clones[v].(V)
	}
	return clones
}

func cloneBase(b baseBuilder) baseBuilder {
	return baseBuilder{
		name:     b.name,
		comments: cloneComments(b.comments),
	}
}

func cloneComments(comments Comments) Comments {
	comments.LeadingDetachedComments = slices.Clone(comments.LeadingDetachedComments)
	return comments
}

func cloneOptions[T proto.Message](opts T) T {
	return proto.Clone(opts).(T)
}

func cloneRPCType(rpcType *RPCType) *RPCType {
	if rpcType == nil {
		return nil
	}
	clone := *rpcType
	return &clone
}
