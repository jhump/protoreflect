package ast

import "fmt"

// MessageDeclNode is a node in the AST that defines a message type. This
// includes normal message fields as well as implicit messages:
//  - *MessageNode
//  - *GroupNode (the group is a field and inline message type)
//  - *MapFieldNode (map fields implicitly define a MapEntry message type)
type MessageDeclNode interface {
	Node
	MessageName() Node
}

var _ MessageDeclNode = (*MessageNode)(nil)
var _ MessageDeclNode = (*GroupNode)(nil)
var _ MessageDeclNode = (*MapFieldNode)(nil)

type MessageNode struct {
	compositeNode
	Keyword *KeywordNode
	Name    *IdentNode

	MessageBody
}

func (*MessageNode) fileElement() {}
func (*MessageNode) msgElement()  {}

func NewMessageNode(keyword *KeywordNode, name *IdentNode, open *RuneNode, decls []MessageElement, close *RuneNode) *MessageNode {
	children := make([]Node, 4+len(decls))
	children = append(children, keyword, name, open)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, close)

	ret := &MessageNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword: keyword,
		Name:    name,
	}
	populateMessageBody(&ret.MessageBody, open, decls, close)
	return ret
}

func (n *MessageNode) MessageName() Node {
	return n.Name
}

// MessageBody represents the body of a message. It is used by both
// MessageNodes and GroupNodes.
type MessageBody struct {
	OpenBrace       *RuneNode
	Options         []*OptionNode
	Fields          []*FieldNode
	MapFields       []*MapFieldNode
	Groups          []*GroupNode
	OneOfs          []*OneOfNode
	NestedMessages  []*MessageNode
	Enums           []*EnumNode
	Extends         []*ExtendNode
	ExtensionRanges []*ExtensionRangeNode
	ReservedNode    []*ReservedNode
	CloseBrace      *RuneNode

	AllDecls []MessageElement
}

func populateMessageBody(m *MessageBody, open *RuneNode, decls []MessageElement, close *RuneNode) {
	m.OpenBrace = open
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *OptionNode:
			m.Options = append(m.Options, decl)
		case *FieldNode:
			m.Fields = append(m.Fields, decl)
		case *MapFieldNode:
			m.MapFields = append(m.MapFields, decl)
		case *GroupNode:
			m.Groups = append(m.Groups, decl)
		case *OneOfNode:
			m.OneOfs = append(m.OneOfs, decl)
		case *MessageNode:
			m.NestedMessages = append(m.NestedMessages, decl)
		case *EnumNode:
			m.Enums = append(m.Enums, decl)
		case *ExtendNode:
			m.Extends = append(m.Extends, decl)
		case *ExtensionRangeNode:
			m.ExtensionRanges = append(m.ExtensionRanges, decl)
		case *ReservedNode:
			m.ReservedNode = append(m.ReservedNode, decl)
		case *EmptyDeclNode:
			// no-op
		default:
			panic(fmt.Sprintf("invalid MessageElement type: %T", decl))
		}
	}
	m.CloseBrace = close
}

// MessageElement is an interface implemented by all AST nodes that can
// appear in a message body.
type MessageElement interface {
	Node
	msgElement()
}

var _ MessageElement = (*OptionNode)(nil)
var _ MessageElement = (*FieldNode)(nil)
var _ MessageElement = (*MapFieldNode)(nil)
var _ MessageElement = (*OneOfNode)(nil)
var _ MessageElement = (*GroupNode)(nil)
var _ MessageElement = (*MessageNode)(nil)
var _ MessageElement = (*EnumNode)(nil)
var _ MessageElement = (*ExtendNode)(nil)
var _ MessageElement = (*ExtensionRangeNode)(nil)
var _ MessageElement = (*ReservedNode)(nil)
var _ MessageElement = (*EmptyDeclNode)(nil)

type ExtendNode struct {
	compositeNode
	Keyword    *KeywordNode
	Extendee   IdentValueNode
	OpenBrace  *RuneNode
	Fields     []*FieldNode
	Groups     []*GroupNode
	CloseBrace *RuneNode

	AllDecls []ExtendElement
}

func (*ExtendNode) fileElement() {}
func (*ExtendNode) msgElement()  {}

func NewExtendNode(keyword *KeywordNode, extendee IdentValueNode, open *RuneNode, decls []ExtendElement, close *RuneNode) *ExtendNode {
	children := make([]Node, 4+len(decls))
	children = append(children, keyword, extendee, open)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, close)

	ret := &ExtendNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:    keyword,
		Extendee:   extendee,
		OpenBrace:  open,
		CloseBrace: close,
		AllDecls:   decls,
	}
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *FieldNode:
			ret.Fields = append(ret.Fields, decl)
			decl.Extendee = ret
		case *GroupNode:
			ret.Groups = append(ret.Groups, decl)
			decl.Extendee = ret
		case *EmptyDeclNode:
			// no-op
		default:
			panic(fmt.Sprintf("invalid ExtendElement type: %T", decl))
		}
	}
	return ret
}

// ExtendElement is an interface implemented by all AST nodes that can
// appear in the body of an extends declaration.
type ExtendElement interface {
	Node
	extendElement()
}

var _ ExtendElement = (*FieldNode)(nil)
var _ ExtendElement = (*GroupNode)(nil)
var _ ExtendElement = (*EmptyDeclNode)(nil)
