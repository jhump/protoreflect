package ast

import "fmt"

// OptionDeclNode is a node in the AST that defines an option. This
// includes both syntaxes for options:
//  - *OptionNode (normal syntax found in files, messages, enums,
//    services, and methods)
//  - *CompactOptionNode (abbreviated syntax found in fields,
//    enum values, extension ranges)
type OptionDeclNode interface {
	Node
	GetName() Node
	GetValue() ValueNode
}

var _ OptionDeclNode = (*OptionNode)(nil)
var _ OptionDeclNode = NoSourceNode{}

type OptionNode struct {
	compositeNode
	Keyword   *KeywordNode
	Name      *OptionNameNode
	Equals    *RuneNode
	Val       ValueNode
	Semicolon *RuneNode
}

func (e *OptionNode) fileElement()    {}
func (e *OptionNode) msgElement()     {}
func (e *OptionNode) oneOfElement()   {}
func (e *OptionNode) enumElement()    {}
func (e *OptionNode) serviceElement() {}
func (e *OptionNode) methodElement()  {}

func NewOptionNode(keyword *KeywordNode, name *OptionNameNode, equals *RuneNode, val ValueNode, semicolon *RuneNode) *OptionNode {
	if keyword == nil {
		panic("keyword is nil")
	}
	if name == nil {
		panic("name is nil")
	}
	if equals == nil {
		panic("equals is nil")
	}
	if val == nil {
		panic("val is nil")
	}
	if semicolon == nil {
		panic("semicolon is nil")
	}
	children := []Node{keyword, name, equals, val, semicolon}
	return &OptionNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:   keyword,
		Name:      name,
		Equals:    equals,
		Val:       val,
		Semicolon: semicolon,
	}
}

func NewCompactOptionNode(name *OptionNameNode, equals *RuneNode, val ValueNode) *OptionNode {
	if name == nil {
		panic("name is nil")
	}
	if equals == nil {
		panic("equals is nil")
	}
	if val == nil {
		panic("val is nil")
	}
	children := []Node{name, equals, val}
	return &OptionNode{
		compositeNode: compositeNode{
			children: children,
		},
		Name:      name,
		Equals:    equals,
		Val:       val,
	}
}

func (n *OptionNode) GetName() Node {
	return n.Name
}

func (n *OptionNode) GetValue() ValueNode {
	return n.Val
}

type OptionNameNode struct {
	compositeNode
	Parts []*FieldReferenceNode
	Dots  []*RuneNode
}

func NewOptionNameNode(parts []*FieldReferenceNode, dots []*RuneNode) *OptionNameNode {
	if len(parts) == 0 {
		panic("must have at least one part")
	}
	if len(dots) != len(parts)-1 {
		panic(fmt.Sprintf("%d parts requires %d dots, not %d", len(parts), len(parts)-1, len(dots)))
	}
	children := make([]Node, 0, len(parts)*2-1)
	for i, part := range parts {
		if part == nil {
			panic(fmt.Sprintf("parts[%d] is nil", i))
		}
		if i > 0 {
			if dots[i-1] == nil {
				panic(fmt.Sprintf("dots[%d] is nil", i-1))
			}
			children = append(children, dots[i-1])
		}
		children = append(children, part)
	}
	return &OptionNameNode{
		compositeNode: compositeNode{
			children: children,
		},
		Parts: parts,
		Dots:  dots,
	}
}

type FieldReferenceNode struct {
	compositeNode
	Open  *RuneNode
	Name  IdentValueNode
	Close *RuneNode
}

func NewFieldReferenceNode(openSym *RuneNode, name IdentValueNode, closeSym *RuneNode) *FieldReferenceNode {
	if name == nil {
		panic("name is nil")
	}
	var children []Node
	if openSym != nil {
		if closeSym == nil {
			panic("closeSym is nil but openSym is not")
		}
		children = []Node{openSym, name, closeSym}
	} else {
		if closeSym != nil {
			panic("openSym is nil but closeSym is not")
		}
		children = []Node{name}
	}
	return &FieldReferenceNode{
		compositeNode: compositeNode{
			children: children,
		},
		Open:  openSym,
		Name:  name,
		Close: closeSym,
	}
}

func (a *FieldReferenceNode) IsExtension() bool {
	return a.Open != nil
}

func (a *FieldReferenceNode) Value() string {
	if a.Open != nil {
		return string(a.Open.Rune) + string(a.Name.AsIdentifier()) + string(a.Close.Rune)
	} else {
		return string(a.Name.AsIdentifier())
	}
}

type CompactOptionsNode struct {
	compositeNode
	OpenBracket *RuneNode
	Options     []*OptionNode
	// Commas represent the separating ',' characters between options. The
	// length of this slice must be exactly len(Options)-1, with each item
	// in Options having a corresponding item in this slice *except the last*
	// (since a trailing comma is not allowed).
	Commas       []*RuneNode
	CloseBracket *RuneNode
}

func NewCompactOptionsNode(openBracket *RuneNode, opts []*OptionNode, commas []*RuneNode, closeBracket *RuneNode) *CompactOptionsNode {
	if openBracket == nil {
		panic("openBracket is nil")
	}
	if closeBracket == nil {
		panic("closeBracket is nil")
	}
	if len(opts) == 0 {
		panic("must have at least one part")
	}
	if len(commas) != len(opts)-1 {
		panic(fmt.Sprintf("%d opts requires %d commas, not %d", len(opts), len(opts)-1, len(commas)))
	}
	children := make([]Node, 0, len(opts)*2+1)
	children = append(children, openBracket)
	for i, opt := range opts {
		if i > 0 {
			if commas[i-1] == nil {
				panic(fmt.Sprintf("commas[%d] is nil", i-1))
			}
			children = append(children, commas[i-1])
		}
		if opt == nil {
			panic(fmt.Sprintf("opts[%d] is nil", i))
		}
		children = append(children, opt)
	}
	children = append(children, closeBracket)

	return &CompactOptionsNode{
		compositeNode: compositeNode{
			children: children,
		},
		OpenBracket:  openBracket,
		Options:      opts,
		Commas:       commas,
		CloseBracket: closeBracket,
	}
}

func (e *CompactOptionsNode) GetElements() []*OptionNode {
	if e == nil {
		return nil
	}
	return e.Options
}
