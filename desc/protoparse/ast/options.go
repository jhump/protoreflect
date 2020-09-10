package ast

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
	numChildren := 3
	if keyword != nil {
		numChildren++
	}
	if semicolon != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	if keyword != nil {
		children = append(children, keyword)
	}
	children = append(children, name, equals, val)
	if semicolon != nil {
		children = append(children, semicolon)
	}
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
	children := make([]Node, 0, len(parts)*2-1)
	for i, part := range parts {
		if i > 0 {
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
	var children []Node
	if openSym != nil {
		children = []Node{openSym, name, closeSym}
	} else {
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
	children := make([]Node, 0, len(opts)*2+1)
	children = append(children, openBracket)
	for i, opt := range opts {
		if i > 0 {
			children = append(children, commas[i-1])
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
