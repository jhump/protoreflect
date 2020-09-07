package ast

// node is the interface implemented by all nodes in the AST
type Node interface {
	Start() *SourcePos
	End() *SourcePos
	LeadingComments() []Comment
	TrailingComments() []Comment
}

type TerminalNode interface {
	Node
	PopLeadingComment() Comment
	PushTrailingComment(Comment)
}

var _ TerminalNode = (*StringLiteralNode)(nil)
var _ TerminalNode = (*UintLiteralNode)(nil)
var _ TerminalNode = (*FloatLiteralNode)(nil)
var _ TerminalNode = (*IdentNode)(nil)
var _ TerminalNode = (*RuneNode)(nil)

type TokenInfo struct {
	PosRange
	LeadingComments  []Comment
	TrailingComments []Comment
}

type CompositeNode interface {
	Node
	Children() []Node
}

type basicNode struct {
	posRange PosRange
	leading  []Comment
	trailing []Comment
}

func (n *basicNode) Start() *SourcePos {
	return &n.posRange.Start
}

func (n *basicNode) End() *SourcePos {
	return &n.posRange.End
}

func (n *basicNode) LeadingComments() []Comment {
	return n.leading
}

func (n *basicNode) TrailingComments() []Comment {
	return n.trailing
}

func (n *basicNode) PopLeadingComment() Comment {
	c := n.leading[0]
	n.leading = n.leading[1:]
	return c
}

func (n *basicNode) PushTrailingComment(c Comment) {
	n.trailing = append(n.trailing, c)
}

type basicCompositeNode struct {
	children []Node
}

func (n *basicCompositeNode) Children() []Node {
	return n.children
}

func (n *basicCompositeNode) Start() *SourcePos {
	return n.children[0].Start()
}

func (n *basicCompositeNode) End() *SourcePos {
	return n.children[len(n.children)-1].End()
}

func (n *basicCompositeNode) LeadingComments() []Comment {
	return n.children[0].LeadingComments()
}

func (n *basicCompositeNode) TrailingComments() []Comment {
	return n.children[len(n.children)-1].TrailingComments()
}

type RuneNode struct {
	basicNode
	Rune rune
}

func NewRuneNode(r rune, info TokenInfo) *RuneNode {
	return &RuneNode{
		basicNode: basicNode{
			posRange: info.PosRange,
			leading:  info.LeadingComments,
			trailing: info.TrailingComments,
		},
		Rune: r,
	}
}

type EmptyDeclNode struct {
	basicCompositeNode
	Semicolon *RuneNode
}

func NewEmptyDeclNode(semicolon *RuneNode) *EmptyDeclNode {
	return &EmptyDeclNode{
		basicCompositeNode: basicCompositeNode{
			children: []Node{semicolon},
		},
		Semicolon: semicolon,
	}
}

func (e *EmptyDeclNode) fileElement()    {}
func (e *EmptyDeclNode) msgElement()     {}
func (e *EmptyDeclNode) extendElement()  {}
func (e *EmptyDeclNode) oneOfElement()   {}
func (e *EmptyDeclNode) enumElement()    {}
func (e *EmptyDeclNode) serviceElement() {}
func (e *EmptyDeclNode) methodElement()  {}
