package ast

type OptionDeclNode interface {
	Node
	GetName() Node
	GetValue() ValueNode
}

var _ OptionDeclNode = (*OptionNode)(nil)
var _ OptionDeclNode = (*CompactOptionNode)(nil)

type OptionNode struct {
	compositeNode
	Keyword   *KeywordNode
	OptionBody
	Semicolon *RuneNode
}

func (e *OptionNode) fileElement()    {}
func (e *OptionNode) msgElement()     {}
func (e *OptionNode) oneOfElement()   {}
func (e *OptionNode) enumElement()    {}
func (e *OptionNode) serviceElement() {}
func (e *OptionNode) methodElement()  {}

func NewOptionNode(keyword *KeywordNode, name *OptionNameNode, equals *RuneNode, val ValueNode, semicolon *RuneNode) *OptionNode {
	children := []Node{keyword, name, equals, val, semicolon}
	return &OptionNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword: keyword,
		OptionBody: OptionBody{
			Name: name,
			Equals: equals,
			Val: val,
		},
		Semicolon: semicolon,
	}
}

type OptionBody struct {
	Name      *OptionNameNode
	Equals    *RuneNode
	Val       ValueNode
}

func (n *OptionBody) GetName() Node {
	return n.Name
}

func (n *OptionBody) GetValue() ValueNode {
	return n.Val
}

type OptionNameNode struct {
	compositeNode
	Parts []*FieldReferenceNode
	Dots  []*RuneNode
}

func NewOptionNameNode(parts []*FieldReferenceNode, dots []*RuneNode) *OptionNameNode {
	children := make([]Node, len(parts)*2-1)
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
		Dots: dots,
	}
}

type FieldReferenceNode struct {
	compositeNode
	Open  *RuneNode
	Name  IdentValueNode
	Close *RuneNode
}

func NewFieldReferenceNode(open *RuneNode, name IdentValueNode, close *RuneNode) *FieldReferenceNode {
	var children []Node
	if open != nil {
		children = []Node{open, name, close}
	} else {
		children = []Node{name}
	}
	return &FieldReferenceNode{
		compositeNode: compositeNode{
			children: children,
		},
		Open:  open,
		Name:  name,
		Close: close,
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
	OpenBracket  *RuneNode
	Options      []*CompactOptionNode
	Commas       []*RuneNode
	CloseBracket *RuneNode
}

func NewCompactOptionsNode(open *RuneNode, opts []*CompactOptionNode, commas []*RuneNode, close *RuneNode) *CompactOptionsNode {
	children := make([]Node, len(opts)*2+1)
	children = append(children, open)
	for i, opt := range opts {
		if i > 0 {
			children = append(children, commas[i-1])
		}
		children = append(children, opt)
	}
	children = append(children, close)

	return &CompactOptionsNode{
		compositeNode: compositeNode{
			children: children,
		},
		OpenBracket:  open,
		Options:      opts,
		Commas:       commas,
		CloseBracket: close,
	}
}

type CompactOptionNode struct {
	compositeNode
	OptionBody
}

func NewCompactOptionNode(name *OptionNameNode, equals *RuneNode, val ValueNode) *CompactOptionNode {
	children := []Node{name, equals, val}
	return &CompactOptionNode{
		compositeNode: compositeNode{
			children: children,
		},
		OptionBody: OptionBody{
			Name: name,
			Equals: equals,
			Val: val,
		},
	}
}