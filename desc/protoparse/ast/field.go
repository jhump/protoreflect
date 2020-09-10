package ast

import "fmt"

// FieldDeclNode is a node in the AST that defines a field. This includes
// normal message fields as well as extensions. There are multiple types
// of AST nodes that declare fields:
//  - *FieldNode
//  - *GroupNode
//  - *MapFieldNode
type FieldDeclNode interface {
	Node
	FieldLabel() Node
	FieldName() Node
	FieldType() Node
	FieldTag() Node
	FieldExtendee() Node
	GetGroupKeyword() Node
	GetOptions() *CompactOptionsNode
}

var _ FieldDeclNode = (*FieldNode)(nil)
var _ FieldDeclNode = (*GroupNode)(nil)
var _ FieldDeclNode = (*MapFieldNode)(nil)
var _ FieldDeclNode = (*SyntheticMapField)(nil)
var _ FieldDeclNode = NoSourceNode{}

type FieldNode struct {
	compositeNode
	Label     FieldLabel
	FldType   IdentValueNode
	Name      *IdentNode
	Equals    *RuneNode
	Tag       *UintLiteralNode
	Options   *CompactOptionsNode
	Semicolon *RuneNode

	Extendee *ExtendNode
}

func (*FieldNode) msgElement()    {}
func (*FieldNode) oneOfElement()  {}
func (*FieldNode) extendElement() {}

func NewFieldNode(label *KeywordNode, fieldType IdentValueNode, name *IdentNode, equals *RuneNode, tag *UintLiteralNode, opts *CompactOptionsNode, semicolon *RuneNode) *FieldNode {
	numChildren := 5
	if label != nil {
		numChildren++
	}
	if opts != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	if label != nil {
		children = append(children, label)
	}
	children = append(children, fieldType, name, equals, tag)
	if opts != nil {
		children = append(children, opts)
	}
	children = append(children, semicolon)

	return &FieldNode{
		compositeNode: compositeNode{
			children: children,
		},
		Label:     newFieldLabel(label),
		FldType:   fieldType,
		Name:      name,
		Equals:    equals,
		Tag:       tag,
		Options:   opts,
		Semicolon: semicolon,
	}
}

func (n *FieldNode) FieldLabel() Node {
	// proto3 fields and fields inside one-ofs will not have a label and we need
	// this check in order to return a nil node -- otherwise we'd return a
	// non-nil node that has a nil pointer value in it :/
	if n.Label.KeywordNode == nil {
		return nil
	}
	return n.Label.KeywordNode
}

func (n *FieldNode) FieldName() Node {
	return n.Name
}

func (n *FieldNode) FieldType() Node {
	return n.FldType
}

func (n *FieldNode) FieldTag() Node {
	return n.Tag
}

func (n *FieldNode) FieldExtendee() Node {
	if n.Extendee != nil {
		return n.Extendee.Extendee
	}
	return nil
}

func (n *FieldNode) GetGroupKeyword() Node {
	return nil
}

func (n *FieldNode) GetOptions() *CompactOptionsNode {
	return n.Options
}

type FieldLabel struct {
	*KeywordNode
	Repeated bool
	Required bool
}

func newFieldLabel(lbl *KeywordNode) FieldLabel {
	repeated, required := false, false
	if lbl != nil {
		repeated = lbl.Val == "repeated"
		required = lbl.Val == "required"
	}
	return FieldLabel{
		KeywordNode: lbl,
		Repeated:    repeated,
		Required:    required,
	}
}

func (f *FieldLabel) IsPresent() bool {
	return f.KeywordNode != nil
}

type GroupNode struct {
	compositeNode
	Label   FieldLabel
	Keyword *KeywordNode
	Name    *IdentNode
	Equals  *RuneNode
	Tag     *UintLiteralNode
	Options *CompactOptionsNode

	MessageBody

	// This field is populated after parsing, to allow lookup of extendee source
	// locations when field extendees cannot be linked. (Otherwise, this is just
	// stored as a string in the field descriptors defined inside the extend
	// block).
	Extendee *ExtendNode
}

func (*GroupNode) msgElement()    {}
func (*GroupNode) oneOfElement()  {}
func (*GroupNode) extendElement() {}

func NewGroupNode(label *KeywordNode, keyword *KeywordNode, name *IdentNode, equals *RuneNode, tag *UintLiteralNode, opts *CompactOptionsNode, open *RuneNode, decls []MessageElement, close *RuneNode) *GroupNode {
	numChildren := 6 + len(decls)
	if label != nil {
		numChildren++
	}
	if opts != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	if label != nil {
		children = append(children, label)
	}
	children = append(children, keyword, name, equals, tag)
	if opts != nil {
		children = append(children, opts)
	}
	children = append(children, open)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, close)

	ret := &GroupNode{
		compositeNode: compositeNode{
			children: children,
		},
		Label:   newFieldLabel(label),
		Keyword: keyword,
		Name:    name,
		Equals:  equals,
		Tag:     tag,
		Options: opts,
	}
	populateMessageBody(&ret.MessageBody, open, decls, close)
	return ret
}

func (n *GroupNode) FieldLabel() Node {
	if n.Label.KeywordNode == nil {
		// return nil interface to indicate absence, not a typed nil
		return nil
	}
	return n.Label.KeywordNode
}

func (n *GroupNode) FieldName() Node {
	return n.Name
}

func (n *GroupNode) FieldType() Node {
	return n.Keyword
}

func (n *GroupNode) FieldTag() Node {
	return n.Tag
}

func (n *GroupNode) FieldExtendee() Node {
	if n.Extendee != nil {
		return n.Extendee.Extendee
	}
	return nil
}

func (n *GroupNode) GetGroupKeyword() Node {
	return n.Keyword
}

func (n *GroupNode) GetOptions() *CompactOptionsNode {
	return n.Options
}

func (n *GroupNode) MessageName() Node {
	return n.Name
}

type OneOfNode struct {
	compositeNode
	Keyword    *KeywordNode
	Name       *IdentNode
	OpenBrace  *RuneNode
	Options    []*OptionNode
	Fields     []*FieldNode
	Groups     []*GroupNode
	CloseBrace *RuneNode

	AllDecls []OneOfElement
}

func (*OneOfNode) msgElement() {}

func NewOneOfNode(keyword *KeywordNode, name *IdentNode, open *RuneNode, decls []OneOfElement, close *RuneNode) *OneOfNode {
	children := make([]Node, 0, 4+len(decls))
	children = append(children, keyword, name, open)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, close)

	var opts []*OptionNode
	var flds []*FieldNode
	var grps []*GroupNode
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *OptionNode:
			opts = append(opts, decl)
		case *FieldNode:
			flds = append(flds, decl)
		case *GroupNode:
			grps = append(grps, decl)
		case *EmptyDeclNode:
			// no-op
		default:
			panic(fmt.Sprintf("invalid OneOfElement type: %T", decl))
		}
	}

	return &OneOfNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:    keyword,
		Name:       name,
		OpenBrace:  open,
		Options:    opts,
		Fields:     flds,
		Groups:     grps,
		CloseBrace: close,
		AllDecls:   decls,
	}
}

// OneOfElement is an interface implemented by all AST nodes that can
// appear in the body of a oneof declaration.
type OneOfElement interface {
	Node
	oneOfElement()
}

var _ OneOfElement = (*OptionNode)(nil)
var _ OneOfElement = (*FieldNode)(nil)
var _ OneOfElement = (*GroupNode)(nil)
var _ OneOfElement = (*EmptyDeclNode)(nil)

type MapTypeNode struct {
	compositeNode
	Keyword    *KeywordNode
	OpenAngle  *RuneNode
	KeyType    *IdentNode
	Comma      *RuneNode
	ValueType  IdentValueNode
	CloseAngle *RuneNode
}

func NewMapTypeNode(keyword *KeywordNode, open *RuneNode, keyType *IdentNode, comma *RuneNode, valType IdentValueNode, close *RuneNode) *MapTypeNode {
	children := []Node{keyword, open, keyType, comma, valType, close}
	return &MapTypeNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:    keyword,
		OpenAngle:  open,
		KeyType:    keyType,
		Comma:      comma,
		ValueType:  valType,
		CloseAngle: close,
	}
}

type MapFieldNode struct {
	compositeNode
	MapType   *MapTypeNode
	Name      *IdentNode
	Equals    *RuneNode
	Tag       *UintLiteralNode
	Options   *CompactOptionsNode
	Semicolon *RuneNode
}

func (*MapFieldNode) msgElement() {}

func NewMapFieldNode(mapType *MapTypeNode, name *IdentNode, equals *RuneNode, tag *UintLiteralNode, opts *CompactOptionsNode, semicolon *RuneNode) *MapFieldNode {
	numChildren := 5
	if opts != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	children = append(children, mapType, name, equals, tag)
	if opts != nil {
		children = append(children, opts)
	}
	children = append(children, semicolon)

	return &MapFieldNode{
		compositeNode: compositeNode{
			children: children,
		},
		MapType:   mapType,
		Name:      name,
		Equals:    equals,
		Tag:       tag,
		Options:   opts,
		Semicolon: semicolon,
	}
}

func (n *MapFieldNode) FieldLabel() Node {
	return nil
}

func (n *MapFieldNode) FieldName() Node {
	return n.Name
}

func (n *MapFieldNode) FieldType() Node {
	return n.MapType
}

func (n *MapFieldNode) FieldTag() Node {
	return n.Tag
}

func (n *MapFieldNode) FieldExtendee() Node {
	return nil
}

func (n *MapFieldNode) GetGroupKeyword() Node {
	return nil
}

func (n *MapFieldNode) GetOptions() *CompactOptionsNode {
	return n.Options
}

func (n *MapFieldNode) MessageName() Node {
	return n.Name
}

func (n *MapFieldNode) KeyField() *SyntheticMapField {
	return NewSyntheticMapField(n.MapType.KeyType, 1)
}

func (n *MapFieldNode) ValueField() *SyntheticMapField {
	return NewSyntheticMapField(n.MapType.ValueType, 2)
}

// SyntheticMapField is not an actual node in the AST but a synthetic node
// that implements FieldDeclNode. These are used to represent the implicit
// field declarations of the "key" and "value" fields in a map entry.
type SyntheticMapField struct {
	Ident IdentValueNode
	Tag   *UintLiteralNode
}

// NewSyntheticMapField creates a new *SyntheticMapField for the given
// identifier (either a key or value type in a map declaration) and tag
// number (1 for key, 2 for value).
func NewSyntheticMapField(ident IdentValueNode, tagNum uint64) *SyntheticMapField {
	tag := &UintLiteralNode{
		terminalNode: terminalNode{
			posRange: PosRange{Start: *ident.Start(), End: *ident.End()},
		},
		Val: tagNum,
	}
	return &SyntheticMapField{Ident: ident, Tag: tag}
}

func (n *SyntheticMapField) Start() *SourcePos {
	return n.Ident.Start()
}

func (n *SyntheticMapField) End() *SourcePos {
	return n.Ident.End()
}

func (n *SyntheticMapField) LeadingComments() []Comment {
	return nil
}

func (n *SyntheticMapField) TrailingComments() []Comment {
	return nil
}

func (n *SyntheticMapField) FieldLabel() Node {
	return n.Ident
}

func (n *SyntheticMapField) FieldName() Node {
	return n.Ident
}

func (n *SyntheticMapField) FieldType() Node {
	return n.Ident
}

func (n *SyntheticMapField) FieldTag() Node {
	return n.Tag
}

func (n *SyntheticMapField) FieldExtendee() Node {
	return nil
}

func (n *SyntheticMapField) GetGroupKeyword() Node {
	return nil
}

func (n *SyntheticMapField) GetOptions() *CompactOptionsNode {
	return nil
}
