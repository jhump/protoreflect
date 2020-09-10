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

func NewServiceNode(keyword *KeywordNode, name *IdentNode, openBrace *RuneNode, decls []ServiceElement, closeBrace *RuneNode) *ServiceNode {
	children := make([]Node, 0, 4+len(decls))
	children = append(children, keyword, name, openBrace)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, closeBrace)

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
		OpenBrace:  openBrace,
		Options:    opts,
		RPCs:       rpcs,
		CloseBrace: closeBrace,
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

type RPCDeclNode interface {
	Node
	GetInputType() Node
	GetOutputType() Node
}

var _ RPCDeclNode = (*RPCNode)(nil)
var _ RPCDeclNode = NoSourceNode{}

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

func NewRPCNodeWithBody(keyword *KeywordNode, name *IdentNode, input *RPCTypeNode, returns *KeywordNode, output *RPCTypeNode, openBrace *RuneNode, decls []RPCElement, closeBrace *RuneNode) *RPCNode {
	children := make([]Node, 0, 7+len(decls))
	children = append(children, keyword, name, input, returns, output, openBrace)
	children = append(children, openBrace)
	for _, decl := range decls {
		children = append(children, decl)
	}
	children = append(children, closeBrace)

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
		OpenBrace:  openBrace,
		Options:    opts,
		CloseBrace: closeBrace,
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

func NewRPCTypeNode(openParen *RuneNode, stream *KeywordNode, msgType IdentValueNode, closeParen *RuneNode) *RPCTypeNode {
	var children []Node
	if stream != nil {
		children = []Node{openParen, stream, msgType, closeParen}
	} else {
		children = []Node{openParen, msgType, closeParen}
	}

	return &RPCTypeNode{
		compositeNode: compositeNode{
			children: children,
		},
		OpenParen:   openParen,
		Stream:      stream,
		MessageType: msgType,
		CloseParen:  closeParen,
	}
}
