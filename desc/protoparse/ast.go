package protoparse

import "fmt"

// This file defines all of the nodes in the proto AST.

type ErrorWithSourcePos struct {
	Underlying error
	Pos        *SourcePos
}

func (e ErrorWithSourcePos) Error() string {
	return fmt.Sprintf("%s:%d:%d: %v", e.Pos.Filename, e.Pos.Line, e.Pos.Col, e.Underlying)
}

type SourcePos struct {
	Filename  string
	Line, Col int
	Offset    int
}

func unknownPos(filename string) *SourcePos {
	return &SourcePos{Filename: filename}
}

type node interface {
	start() *SourcePos
	end() *SourcePos
	leadingComments() []*comment
	trailingComment() []*comment
}

type posRange struct {
	start, end *SourcePos
}

type basicNode struct {
	posRange
	leading  []*comment
	trailing []*comment
}

func (n *basicNode) start() *SourcePos {
	return n.posRange.start
}

func (n *basicNode) end() *SourcePos {
	return n.posRange.end
}

func (n *basicNode) leadingComments() []*comment {
	return n.leading
}

func (n *basicNode) trailingComment() []*comment {
	return n.trailing
}

type comment struct {
	posRange
	text string
}

type basicCompositeNode struct {
	first node
	last  node
}

func (n *basicCompositeNode) start() *SourcePos {
	return n.first.start()
}

func (n *basicCompositeNode) end() *SourcePos {
	return n.last.end()
}

func (n *basicCompositeNode) leadingComments() []*comment {
	return n.first.leadingComments()
}

func (n *basicCompositeNode) trailingComment() []*comment {
	return n.last.trailingComment()
}

func (n *basicCompositeNode) setRange(first, last node) {
	n.first = first
	n.last = last
}

type fileNode struct {
	basicCompositeNode
	syntax *syntaxNode
	decls  []*fileElement

	// These fields are populated after parsing, to make it easier to find them
	// without searching decls. The parse result has a map of descriptors to
	// nodes which makes the other declarations easily discoverable. But these
	// elements do not map to descriptors -- they are just stored as strings in
	// the file descriptor.
	imports []*importNode
	pkg     *packageNode
}

type fileElement struct {
	// a discriminated union: only one field will be set
	imp     *importNode
	pkg     *packageNode
	option  *optionNode
	message *messageNode
	enum    *enumNode
	extend  *extendNode
	service *serviceNode
	empty   *basicNode
}

func (n *fileElement) start() *SourcePos {
	return n.get().start()
}

func (n *fileElement) end() *SourcePos {
	return n.get().end()
}

func (n *fileElement) leadingComments() []*comment {
	return n.get().leadingComments()
}

func (n *fileElement) trailingComment() []*comment {
	return n.get().trailingComment()
}

func (n *fileElement) get() node {
	switch {
	case n.imp != nil:
		return n.imp
	case n.pkg != nil:
		return n.pkg
	case n.option != nil:
		return n.option
	case n.message != nil:
		return n.message
	case n.enum != nil:
		return n.enum
	case n.extend != nil:
		return n.extend
	case n.service != nil:
		return n.service
	default:
		return n.empty
	}
}

type syntaxNode struct {
	basicCompositeNode
	syntax *stringLiteralNode
}

type importNode struct {
	basicCompositeNode
	name   *stringLiteralNode
	public bool
	weak   bool
}

type packageNode struct {
	basicCompositeNode
	name *identNode
}

type identifier string

type identKind int

const (
	identSimpleName identKind = iota
	identQualified
	identTypeName
)

type identNode struct {
	basicNode
	val  string
	kind identKind
}

func (n *identNode) value() interface{} {
	return identifier(n.val)
}

type optionNode struct {
	basicCompositeNode
	name *optionNameNode
	val  valueNode
}

type optionNameNode struct {
	basicCompositeNode
	parts []*optionNamePartNode
}

type optionNamePartNode struct {
	basicCompositeNode
	text        *identNode
	offset      int
	length      int
	isExtension bool
	st, en      *SourcePos
}

func (n *optionNamePartNode) start() *SourcePos {
	if n.isExtension {
		return n.basicCompositeNode.start()
	}
	return n.st
}

func (n *optionNamePartNode) end() *SourcePos {
	if n.isExtension {
		return n.basicCompositeNode.end()
	}
	return n.en
}

func (n *optionNamePartNode) setRange(first, last node) {
	n.basicCompositeNode.setRange(first, last)
	if !n.isExtension {
		st := *first.start()
		st.Col += n.offset
		n.st = &st
		en := st
		en.Col += n.length
		n.en = &en
	}
}

type valueNode interface {
	node
	value() interface{}
}

type stringLiteralNode struct {
	basicNode
	val string
}

func (n *stringLiteralNode) value() interface{} {
	return n.val
}

type intLiteralNode struct {
	basicNode
	val uint64
}

func (n *intLiteralNode) value() interface{} {
	return n.val
}

type negativeIntLiteralNode struct {
	basicCompositeNode
	val int64
}

func (n *negativeIntLiteralNode) value() interface{} {
	return n.val
}

type floatLiteralNode struct {
	basicCompositeNode
	val float64
}

func (n *floatLiteralNode) value() interface{} {
	return n.val
}

type boolLiteralNode struct {
	basicNode
	val bool
}

func (n *boolLiteralNode) value() interface{} {
	return n.val
}

type sliceLiteralNode struct {
	basicCompositeNode
	elements []valueNode
}

func (n *sliceLiteralNode) value() interface{} {
	return n.elements
}

type aggregateLiteralNode struct {
	basicCompositeNode
	elements []*aggregateEntryNode
}

func (n *aggregateLiteralNode) value() interface{} {
	return n.elements
}

type aggregateEntryNode struct {
	basicCompositeNode
	name *aggregateNameNode
	val  valueNode
}

type aggregateNameNode struct {
	basicCompositeNode
	name        *identNode
	isExtension bool
}

func (a *aggregateNameNode) value() string {
	if a.isExtension {
		return "[" + a.name.val + "]"
	} else {
		return a.name.val
	}
}

type fieldDecl interface {
	node
	fieldLabel() node
	fieldName() *identNode
	fieldType() *identNode
	fieldTag() *intLiteralNode
}

type fieldNode struct {
	basicCompositeNode
	label   *labelNode
	fldType *identNode
	name    *identNode
	tag     *intLiteralNode
	options []*optionNode
}

func (n *fieldNode) fieldLabel() node {
	return n.label
}

func (n *fieldNode) fieldName() *identNode {
	return n.name
}

func (n *fieldNode) fieldType() *identNode {
	return n.fldType
}

func (n *fieldNode) fieldTag() *intLiteralNode {
	return n.tag
}

type labelNode struct {
	basicNode
	repeated bool
	required bool
}

type groupNode struct {
	basicCompositeNode
	label *labelNode
	name  *identNode
	tag   *intLiteralNode
	decls []*messageElement

	// This field is populated after parsing, to make it easier to find them
	// without searching decls. The parse result has a map of descriptors to
	// nodes which makes the other declarations easily discoverable. But these
	// elements do not map to descriptors -- they are just stored as strings in
	// the message descriptor.
	reserved []*stringLiteralNode
}

func (n *groupNode) fieldLabel() node {
	return n.label
}

func (n *groupNode) fieldName() *identNode {
	return n.name
}

func (n *groupNode) fieldType() *identNode {
	return n.name
}

func (n *groupNode) fieldTag() *intLiteralNode {
	return n.tag
}

func (n *groupNode) messageName() *identNode {
	return n.name
}

func (n *groupNode) reservedNames() []*stringLiteralNode {
	return n.reserved
}

type oneOfNode struct {
	basicCompositeNode
	name  *identNode
	decls []*oneOfElement
}

type oneOfElement struct {
	// a discriminated union: only one field will be set
	option *optionNode
	field  *fieldNode
	empty  *basicNode
}

func (n *oneOfElement) start() *SourcePos {
	return n.get().start()
}

func (n *oneOfElement) end() *SourcePos {
	return n.get().end()
}

func (n *oneOfElement) leadingComments() []*comment {
	return n.get().leadingComments()
}

func (n *oneOfElement) trailingComment() []*comment {
	return n.get().trailingComment()
}

func (n *oneOfElement) get() node {
	switch {
	case n.option != nil:
		return n.option
	case n.field != nil:
		return n.field
	default:
		return n.empty
	}
}

type mapFieldNode struct {
	basicCompositeNode
	mapKeyword *identNode
	keyType    *identNode
	valueType  *identNode
	name       *identNode
	tag        *intLiteralNode
	options    []*optionNode
}

func (n *mapFieldNode) fieldLabel() node {
	return n.mapKeyword
}

func (n *mapFieldNode) fieldName() *identNode {
	return n.name
}

func (n *mapFieldNode) fieldType() *identNode {
	return n.mapKeyword
}

func (n *mapFieldNode) fieldTag() *intLiteralNode {
	return n.tag
}

func (n *mapFieldNode) messageName() *identNode {
	return n.name
}

func (n *mapFieldNode) reservedNames() []*stringLiteralNode {
	return nil
}

func (n *mapFieldNode) keyField() *syntheticMapField {
	tag := &intLiteralNode{
		basicNode: basicNode{
			posRange: posRange{start: n.keyType.start(), end: n.keyType.end()},
		},
		val: 1,
	}
	return &syntheticMapField{ident: n.keyType, tag: tag}
}

func (n *mapFieldNode) valueField() *syntheticMapField {
	tag := &intLiteralNode{
		basicNode: basicNode{
			posRange: posRange{start: n.valueType.start(), end: n.valueType.end()},
		},
		val: 2,
	}
	return &syntheticMapField{ident: n.valueType, tag: tag}
}

type syntheticMapField struct {
	ident *identNode
	tag   *intLiteralNode
}

func (n *syntheticMapField) start() *SourcePos {
	return n.ident.start()
}

func (n *syntheticMapField) end() *SourcePos {
	return n.ident.end()
}

func (n *syntheticMapField) leadingComments() []*comment {
	return nil
}

func (n *syntheticMapField) trailingComment() []*comment {
	return nil
}

func (n *syntheticMapField) fieldLabel() node {
	return n.ident
}

func (n *syntheticMapField) fieldName() *identNode {
	return n.ident
}

func (n *syntheticMapField) fieldType() *identNode {
	return n.ident
}

func (n *syntheticMapField) fieldTag() *intLiteralNode {
	return n.tag
}

type extensionRangeNode struct {
	basicCompositeNode
	ranges  []*rangeNode
	options []*optionNode
}

type rangeNode struct {
	basicCompositeNode
	st, en *intLiteralNode
}

type reservedNode struct {
	basicCompositeNode
	ranges []*rangeNode
	names  []*stringLiteralNode
}

type enumNode struct {
	basicCompositeNode
	name  *identNode
	decls []*enumElement
}

type enumElement struct {
	// a discriminated union: only one field will be set
	option *optionNode
	value  *enumValueNode
	empty  *basicNode
}

func (n *enumElement) start() *SourcePos {
	return n.get().start()
}

func (n *enumElement) end() *SourcePos {
	return n.get().end()
}

func (n *enumElement) leadingComments() []*comment {
	return n.get().leadingComments()
}

func (n *enumElement) trailingComment() []*comment {
	return n.get().trailingComment()
}

func (n *enumElement) get() node {
	switch {
	case n.option != nil:
		return n.option
	case n.value != nil:
		return n.value
	default:
		return n.empty
	}
}

type enumValueNode struct {
	basicCompositeNode
	name    *identNode
	options []*optionNode
	// only one of these two will be set
	number  *intLiteralNode
	numberN *negativeIntLiteralNode
}

type msgDecl interface {
	node
	messageName() *identNode
	reservedNames() []*stringLiteralNode
}

type messageNode struct {
	basicCompositeNode
	name  *identNode
	decls []*messageElement

	// This field is populated after parsing, to make it easier to find them
	// without searching decls. The parse result has a map of descriptors to
	// nodes which makes the other declarations easily discoverable. But these
	// elements do not map to descriptors -- they are just stored as strings in
	// the message descriptor.
	reserved []*stringLiteralNode
}

func (n *messageNode) messageName() *identNode {
	return n.name
}

func (n *messageNode) reservedNames() []*stringLiteralNode {
	return n.reserved
}

type messageElement struct {
	// a discriminated union: only one field will be set
	option         *optionNode
	field          *fieldNode
	mapField       *mapFieldNode
	oneOf          *oneOfNode
	group          *groupNode
	nested         *messageNode
	enum           *enumNode
	extend         *extendNode
	extensionRange *extensionRangeNode
	reserved       *reservedNode
	empty          *basicNode
}

func (n *messageElement) start() *SourcePos {
	return n.get().start()
}

func (n *messageElement) end() *SourcePos {
	return n.get().end()
}

func (n *messageElement) leadingComments() []*comment {
	return n.get().leadingComments()
}

func (n *messageElement) trailingComment() []*comment {
	return n.get().trailingComment()
}

func (n *messageElement) get() node {
	switch {
	case n.option != nil:
		return n.option
	case n.field != nil:
		return n.field
	case n.mapField != nil:
		return n.mapField
	case n.oneOf != nil:
		return n.oneOf
	case n.group != nil:
		return n.group
	case n.nested != nil:
		return n.nested
	case n.enum != nil:
		return n.enum
	case n.extend != nil:
		return n.extend
	case n.extensionRange != nil:
		return n.extensionRange
	case n.reserved != nil:
		return n.reserved
	default:
		return n.empty
	}
}

type extendNode struct {
	basicCompositeNode
	extendee *identNode
	decls    []*extendElement
}

type extendElement struct {
	// a discriminated union: only one field will be set
	field *fieldNode
	group *groupNode
	empty *basicNode
}

func (n *extendElement) start() *SourcePos {
	return n.get().start()
}

func (n *extendElement) end() *SourcePos {
	return n.get().end()
}

func (n *extendElement) leadingComments() []*comment {
	return n.get().leadingComments()
}

func (n *extendElement) trailingComment() []*comment {
	return n.get().trailingComment()
}

func (n *extendElement) get() node {
	switch {
	case n.field != nil:
		return n.field
	case n.group != nil:
		return n.group
	default:
		return n.empty
	}
}

type serviceNode struct {
	basicCompositeNode
	name  *identNode
	decls []*serviceElement
}

type serviceElement struct {
	// a discriminated union: only one field will be set
	option *optionNode
	rpc    *methodNode
	empty  *basicNode
}

func (n *serviceElement) start() *SourcePos {
	return n.get().start()
}

func (n *serviceElement) end() *SourcePos {
	return n.get().end()
}

func (n *serviceElement) leadingComments() []*comment {
	return n.get().leadingComments()
}

func (n *serviceElement) trailingComment() []*comment {
	return n.get().trailingComment()
}

func (n *serviceElement) get() node {
	switch {
	case n.option != nil:
		return n.option
	case n.rpc != nil:
		return n.rpc
	default:
		return n.empty
	}
}

type methodNode struct {
	basicCompositeNode
	name    *identNode
	input   *rpcTypeNode
	output  *rpcTypeNode
	options []*optionNode
}

type rpcTypeNode struct {
	basicCompositeNode
	msgType *identNode
	stream  bool
}
