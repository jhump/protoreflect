package ast

type ServiceNode struct {
	basicCompositeNode
	Keyword    *IdentNode
	Name       *IdentNode
	OpenBrace  *RuneNode
	Options    []*OptionNode
	RPCs       []*RPCNode
	CloseBrace *RuneNode

	AllDecls []*ServiceElement
}

func (*ServiceNode) fileElement() {}

type ServiceElement interface {
	Node
	serviceElement()
}

var _ ServiceElement = (*OptionNode)(nil)
var _ ServiceElement = (*RPCNode)(nil)
var _ ServiceElement = (*EmptyDeclNode)(nil)

type RPCNode struct {
	basicCompositeNode
	Keyword    *IdentNode
	Name       *IdentNode
	Input      *RPCTypeNode
	Returns    *IdentNode
	Output     *RPCTypeNode
	Semicolon  *RuneNode
	OpenBrace  *RuneNode
	Options    []*OptionNode
	CloseBrace *RuneNode

	AllDecls []RPCElement
}

func (n *RPCNode) serviceElement() {}

func (n *RPCNode) GetInputType() Node {
	return n.Input.MsgType
}

func (n *RPCNode) GetOutputType() Node {
	return n.Output.MsgType
}

type RPCElement interface {
	Node
	methodElement()
}

var _ RPCElement = (*OptionNode)(nil)
var _ RPCElement = (*EmptyDeclNode)(nil)

type RPCTypeNode struct {
	basicCompositeNode
	OpenParen     *RuneNode
	StreamKeyword *RuneNode
	MsgType       *CompoundIdentNode
	CloseParen    *RuneNode
}
