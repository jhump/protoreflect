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
var _ MessageDeclNode = NoSourceNode{}

type MessageNode struct {
	compositeNode
	Keyword *KeywordNode
	Name    *IdentNode
	MessageBody
}

func (*MessageNode) fileElement() {}
func (*MessageNode) msgElement()  {}

func NewMessageNode(keyword *KeywordNode, name *IdentNode, openBrace *RuneNode, decls []MessageElement, closeBrace *RuneNode) *MessageNode {
	if keyword == nil {
		panic("keyword is nil")
	}
	if name == nil {
		panic("name is nil")
	}
	if openBrace == nil {
		panic("openBrace is nil")
	}
	if closeBrace == nil {
		panic("closeBrace is nil")
	}
	children := make([]Node, 0, 4+len(decls))
	children = append(children, keyword, name, openBrace)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, closeBrace)

	ret := &MessageNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword: keyword,
		Name:    name,
	}
	populateMessageBody(&ret.MessageBody, openBrace, decls, closeBrace)
	return ret
}

func (n *MessageNode) MessageName() Node {
	return n.Name
}

// MessageBody represents the body of a message. It is used by both
// MessageNodes and GroupNodes.
type MessageBody struct {
	OpenBrace  *RuneNode
	Decls      []MessageElement
	CloseBrace *RuneNode
}

func populateMessageBody(m *MessageBody, openBrace *RuneNode, decls []MessageElement, closeBrace *RuneNode) {
	m.OpenBrace = openBrace
	m.Decls = decls
	for _, decl := range decls {
		switch decl.(type) {
		case *OptionNode, *FieldNode, *MapFieldNode, *GroupNode, *OneOfNode,
			*MessageNode, *EnumNode, *ExtendNode, *ExtensionRangeNode,
			*ReservedNode, *EmptyDeclNode:
		default:
			panic(fmt.Sprintf("invalid MessageElement type: %T", decl))
		}
	}
	m.CloseBrace = closeBrace
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
	Decls      []ExtendElement
	CloseBrace *RuneNode
}

func (*ExtendNode) fileElement() {}
func (*ExtendNode) msgElement()  {}

func NewExtendNode(keyword *KeywordNode, extendee IdentValueNode, openBrace *RuneNode, decls []ExtendElement, closeBrace *RuneNode) *ExtendNode {
	if keyword == nil {
		panic("keyword is nil")
	}
	if extendee == nil {
		panic("extendee is nil")
	}
	if openBrace == nil {
		panic("openBrace is nil")
	}
	if closeBrace == nil {
		panic("closeBrace is nil")
	}
	children := make([]Node, 0, 4+len(decls))
	children = append(children, keyword, extendee, openBrace)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, closeBrace)

	ret := &ExtendNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:    keyword,
		Extendee:   extendee,
		OpenBrace:  openBrace,
		Decls:      decls,
		CloseBrace: closeBrace,
	}
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *FieldNode:
			decl.Extendee = ret
		case *GroupNode:
			decl.Extendee = ret
		case *EmptyDeclNode:
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
