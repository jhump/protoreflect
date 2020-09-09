package ast

import "strings"

// Identifier is a possibly-qualified name. This is used to distinguish
// ValueNode values that are references/identifiers vs. those that are
// string literals.
type Identifier string

// IdentValueNode is an AST node that represents an identifier.
type IdentValueNode interface {
	ValueNode
	AsIdentifier() Identifier
}

var _ IdentValueNode = (*IdentNode)(nil)
var _ IdentValueNode = (*CompoundIdentNode)(nil)

type IdentNode struct {
	terminalNode
	Val string
}

func NewIdentNode(val string, info TokenInfo) *IdentNode {
	return &IdentNode{
		terminalNode: info.asTerminalNode(),
		Val:          val,
	}
}

func (n *IdentNode) Value() interface{} {
	return n.AsIdentifier()
}

func (n *IdentNode) AsIdentifier() Identifier {
	return Identifier(n.Val)
}

func (n *IdentNode) AsKeyword() *KeywordNode {
	return (*KeywordNode)(n)
}

type CompoundIdentNode struct {
	compositeNode
	Components []*IdentNode
	Dots       []*RuneNode
	Val        string
}

func NewCompoundIdentNode(components []*IdentNode, dots []*RuneNode) *CompoundIdentNode {
	children := make([]Node, 0, len(components)*2-1)
	var b strings.Builder
	for i, comp := range components {
		if i > 0 {
			dot := dots[i-1]
			children = append(children, dot)
			b.WriteRune(dot.Rune)
		}
		children = append(children, comp)
		b.WriteString(comp.Val)
	}
	return &CompoundIdentNode{
		compositeNode: compositeNode{
			children: children,
		},
		Components: components,
		Dots:       dots,
		Val:        b.String(),
	}
}

func (n *CompoundIdentNode) Value() interface{} {
	return n.AsIdentifier()
}

func (n *CompoundIdentNode) AsIdentifier() Identifier {
	return Identifier(n.Val)
}

type KeywordNode IdentNode

func NewKeywordNode(val string, info TokenInfo) *KeywordNode {
	return &KeywordNode{
		terminalNode: info.asTerminalNode(),
		Val:          val,
	}
}
