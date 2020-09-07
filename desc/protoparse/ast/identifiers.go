package ast

import "strings"

type Identifier string

type IdentValueNode interface {
	ValueNode
	AsIdentifier() Identifier
}

var _ IdentValueNode = (*IdentNode)(nil)
var _ IdentValueNode = (*CompoundIdentNode)(nil)

type IdentNode struct {
	basicNode
	Val string
}

func NewIdentNode(val string, info TokenInfo) *IdentNode {
	return &IdentNode{
		basicNode: basicNode{
			posRange: info.PosRange,
			leading:  info.LeadingComments,
			trailing: info.TrailingComments,
		},
		Val: val,
	}
}

func (n *IdentNode) Value() interface{} {
	return n.AsIdentifier()
}

func (n *IdentNode) AsIdentifier() Identifier {
	return Identifier(n.Val)
}

type CompoundIdentNode struct {
	basicCompositeNode
	Components []*IdentNode
	Dots       []*RuneNode
	Val        string
}

func NewCompoundIdentNode(components []*IdentNode, dots []*RuneNode) *CompoundIdentNode {
	children := make([]Node, 0, len(components)*2 - 1)
	var b strings.Builder
	for i, comp := range components {
		children = append(children, comp)
		b.WriteString(comp.Val)
		if i > 0 {
			dot := dots[i-1]
			children = append(children, dot)
			b.WriteRune(dot.Rune)
		}
	}
	return &CompoundIdentNode{
		basicCompositeNode: basicCompositeNode{
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
