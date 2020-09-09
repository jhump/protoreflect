package ast

import "fmt"

type ServiceNode struct {
	compositeNode
	Keyword    *KeywordNode
	Name       *IdentNode
	OpenBrace  *RuneNode
	Options    []*OptionNode
	RPCs       []*RPCNode
	CloseBrace *RuneNode

	AllDecls []ServiceElement
}

func (*ServiceNode) fileElement() {}

func NewServiceNode(keyword *KeywordNode, name *IdentNode, open *RuneNode, decls []ServiceElement, close *RuneNode) *ServiceNode {
	children := make([]Node, 4+len(decls))
	children = append(children, keyword, name, open)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, close)

	var opts []*OptionNode
	var rpcs []*RPCNode
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *OptionNode:
			opts = append(opts, decl)
		case *RPCNode:
			rpcs = append(rpcs, decl)
		case *EmptyDeclNode:
			// no-op
		default:
			panic(fmt.Sprintf("invalid ServiceElement type: %T", decl))
		}
	}

	return &ServiceNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:    keyword,
		Name:       name,
		OpenBrace:  open,
		Options:    opts,
		RPCs:       rpcs,
		CloseBrace: close,
		AllDecls:   decls,
	}
}

// ServiceElement is an interface implemented by all AST nodes that can
// appear in the body of a service declaration.
type ServiceElement interface {
	Node
	serviceElement()
}

var _ ServiceElement = (*OptionNode)(nil)
var _ ServiceElement = (*RPCNode)(nil)
var _ ServiceElement = (*EmptyDeclNode)(nil)

type RPCNode struct {
	compositeNode
	Keyword    *KeywordNode
	Name       *IdentNode
	Input      *RPCTypeNode
	Returns    *KeywordNode
	Output     *RPCTypeNode
	Semicolon  *RuneNode
	OpenBrace  *RuneNode
	Options    []*OptionNode
	CloseBrace *RuneNode

	AllDecls []RPCElement
}

func (n *RPCNode) serviceElement() {}

func NewRPCNode(keyword *KeywordNode, name *IdentNode, input *RPCTypeNode, returns *KeywordNode, output *RPCTypeNode, semicolon *RuneNode) *RPCNode {
	children := []Node{keyword, name, input, returns, output, semicolon}
	return &RPCNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:   keyword,
		Name:      name,
		Input:     input,
		Returns:   returns,
		Output:    output,
		Semicolon: semicolon,
	}
}

func NewRPCNodeWithBody(keyword *KeywordNode, name *IdentNode, input *RPCTypeNode, returns *KeywordNode, output *RPCTypeNode, open *RuneNode, decls []RPCElement, close *RuneNode) *RPCNode {
	children := make([]Node, 0, 7+len(decls))
	children = append(children, keyword, name, input, returns, output, open)
	children = append(children, open)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, close)

	var opts []*OptionNode
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *OptionNode:
			opts = append(opts, decl)
		case *EmptyDeclNode:
			// no-op
		default:
			panic(fmt.Sprintf("invalid RPCElement type: %T", decl))
		}
	}

	return &RPCNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:    keyword,
		Name:       name,
		Input:      input,
		Returns:    returns,
		Output:     output,
		OpenBrace:  open,
		Options:    opts,
		CloseBrace: close,
		AllDecls:   decls,
	}
}

func (n *RPCNode) GetInputType() Node {
	return n.Input.MessageType
}

func (n *RPCNode) GetOutputType() Node {
	return n.Output.MessageType
}

// RPCElement is an interface implemented by all AST nodes that can
// appear in the body of an rpc declaration (aka method).
type RPCElement interface {
	Node
	methodElement()
}

var _ RPCElement = (*OptionNode)(nil)
var _ RPCElement = (*EmptyDeclNode)(nil)

type RPCTypeNode struct {
	compositeNode
	OpenParen   *RuneNode
	Stream      *KeywordNode
	MessageType IdentValueNode
	CloseParen  *RuneNode
}

func NewRPCTypeNode(open *RuneNode, stream *KeywordNode, msgType IdentValueNode, close *RuneNode) *RPCTypeNode {
	var children []Node
	if stream != nil {
		children = []Node{open, stream, msgType, close}
	} else {
		children = []Node{open, msgType, close}
	}

	return &RPCTypeNode{
		compositeNode: compositeNode{
			children: children,
		},
		OpenParen:   open,
		Stream:      stream,
		MessageType: msgType,
		CloseParen:  close,
	}
}
