package ast

type OptionDeclNode interface {
	Node
	GetName() Node
	GetValue() ValueNode
}

var _ OptionDeclNode = (*OptionNode)(nil)
var _ OptionDeclNode = (*CompactOptionNode)(nil)

type OptionNode struct {
	basicCompositeNode
	Keyword   *IdentNode
	OptionBody
	Semicolon *RuneNode
}

func (e *OptionNode) fileElement()    {}
func (e *OptionNode) msgElement()     {}
func (e *OptionNode) oneOfElement()   {}
func (e *OptionNode) enumElement()    {}
func (e *OptionNode) serviceElement() {}
func (e *OptionNode) methodElement()  {}

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
	basicCompositeNode
	Parts []*FieldReferenceNode
	Dots  []*RuneNode
}

type FieldReferenceNode struct {
	basicCompositeNode
	Open  *RuneNode
	Name  *CompoundIdentNode
	Close *RuneNode
}

func (a *FieldReferenceNode) IsExtension() bool {
	return a.Open != nil
}

func (a *FieldReferenceNode) Value() string {
	if a.Open != nil {
		return string(a.Open.Rune) + a.Name.Val + string(a.Close.Rune)
	} else {
		return a.Name.Val
	}
}

type CompactOptionsNode struct {
	basicCompositeNode
	OpenBracket  *RuneNode
	Options      []*CompactOptionNode
	Commas       []*RuneNode
	CloseBracket *RuneNode
}

func (n *CompactOptionsNode) Elements() []*CompactOptionNode {
	if n == nil {
		return nil
	}
	return n.Options
}

type CompactOptionNode struct {
	basicCompositeNode
	OptionBody
}
