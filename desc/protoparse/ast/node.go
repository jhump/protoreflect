package ast

// Node is the interface implemented by all nodes in the AST. It
// provides information about the span of this AST node in terms
// of location in the source file. It also provides information
// about all prior comments (attached as leading comments) and
// optional subsequent comments (attached as trailing comments).
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
var _ TerminalNode = (*BoolLiteralNode)(nil)
var _ TerminalNode = (*SpecialFloatLiteralNode)(nil)
var _ TerminalNode = (*KeywordNode)(nil)
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

type terminalNode struct {
	posRange PosRange
	leading  []Comment
	trailing []Comment
}

func (n *terminalNode) Start() *SourcePos {
	return &n.posRange.Start
}

func (n *terminalNode) End() *SourcePos {
	return &n.posRange.End
}

func (n *terminalNode) LeadingComments() []Comment {
	return n.leading
}

func (n *terminalNode) TrailingComments() []Comment {
	return n.trailing
}

func (n *terminalNode) PopLeadingComment() Comment {
	c := n.leading[0]
	n.leading = n.leading[1:]
	return c
}

func (n *terminalNode) PushTrailingComment(c Comment) {
	n.trailing = append(n.trailing, c)
}

type compositeNode struct {
	children []Node
}

func (n *compositeNode) Children() []Node {
	return n.children
}

func (n *compositeNode) Start() *SourcePos {
	return n.children[0].Start()
}

func (n *compositeNode) End() *SourcePos {
	return n.children[len(n.children)-1].End()
}

func (n *compositeNode) LeadingComments() []Comment {
	return n.children[0].LeadingComments()
}

func (n *compositeNode) TrailingComments() []Comment {
	return n.children[len(n.children)-1].TrailingComments()
}

type RuneNode struct {
	terminalNode
	Rune rune
}

func NewRuneNode(r rune, info TokenInfo) *RuneNode {
	return &RuneNode{
		terminalNode: terminalNode{
			posRange: info.PosRange,
			leading:  info.LeadingComments,
			trailing: info.TrailingComments,
		},
		Rune: r,
	}
}

type EmptyDeclNode struct {
	compositeNode
	Semicolon *RuneNode
}

func NewEmptyDeclNode(semicolon *RuneNode) *EmptyDeclNode {
	return &EmptyDeclNode{
		compositeNode: compositeNode{
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
