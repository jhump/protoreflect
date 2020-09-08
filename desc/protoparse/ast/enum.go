package ast

import "fmt"

type EnumNode struct {
	compositeNode
	Keyword    *KeywordNode
	Name       *IdentNode
	OpenBrace  *RuneNode
	Options    []*OptionNode
	Values     []*EnumValueNode
	Reserved   []*ReservedNode
	CloseBrace *RuneNode

	AllDecls []EnumElement
}

func (*EnumNode) fileElement() {}
func (*EnumNode) msgElement()  {}

func NewEnumNode(keyword *KeywordNode, name *IdentNode, open *RuneNode, decls []EnumElement, close *RuneNode) *EnumNode {
	children := make([]Node, 0, 4+len(decls))
	children = append(children, keyword, name, open)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, close)

	var opts []*OptionNode
	var vals []*EnumValueNode
	var rsvd []*ReservedNode
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *OptionNode:
			opts = append(opts, decl)
		case *EnumValueNode:
			vals = append(vals, decl)
		case *ReservedNode:
			rsvd = append(rsvd, decl)
		case *EmptyDeclNode:
			// no-op
		default:
			panic(fmt.Sprintf("invalid EnumElement type: %T", decl))
		}
	}

	return &EnumNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:    keyword,
		Name:       name,
		OpenBrace:  open,
		Options:    opts,
		Values:     vals,
		Reserved:   rsvd,
		CloseBrace: close,
		AllDecls:   decls,
	}
}

// EnumElement is an interface implemented by all AST nodes that can
// appear in the body of an enum declaration.
type EnumElement interface {
	Node
	enumElement()
}

var _ EnumElement = (*OptionNode)(nil)
var _ EnumElement = (*EnumValueNode)(nil)
var _ EnumElement = (*ReservedNode)(nil)
var _ EnumElement = (*EmptyDeclNode)(nil)

type EnumValueNode struct {
	compositeNode
	Name      *IdentNode
	Equals    *RuneNode
	Number    IntValueNode
	Options   *CompactOptionsNode
	Semicolon *RuneNode
}

func (e *EnumValueNode) enumElement() {}

func NewEnumValueNode(name *IdentNode, equals *RuneNode, number IntValueNode,opts *CompactOptionsNode, semicolon *RuneNode) *EnumValueNode {
	children := []Node{
		name, equals, number, opts, semicolon,
	}
	return &EnumValueNode{
		compositeNode: compositeNode{
			children: children,
		},
		Name:      name,
		Equals:    equals,
		Number:    number,
		Options:   opts,
		Semicolon: semicolon,
	}
}
