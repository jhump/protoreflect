package ast

import "fmt"

type FileNode struct {
	basicCompositeNode
	Syntax   *SyntaxNode

	Package  []*PackageNode
	Imports  []*ImportNode
	Options  []*OptionNode
	Messages []*MessageNode
	Enums    []*EnumNode
	Extends  []*ExtendNode
	Services []*ServiceNode

	AllDecls    []FileElement
	AllComments []Comment
}

func NewFileElement(syntax *SyntaxNode, decls []FileElement) *FileNode {
	children := make([]Node, 1+len(decls))
	children = append(children, syntax)
	for _, decl := range decls {
		children = append(children, decl)
	}

	var pkgs []*PackageNode
	var imps []*ImportNode
	var opts []*OptionNode
	var msgs []*MessageNode
	var enms []*EnumNode
	var exts []*ExtendNode
	var svcs []*ServiceNode
	for _, decl := range decls {
		switch decl := decl.(type) {
		case *PackageNode:
			pkgs = append(pkgs, decl)
		case *ImportNode:
			imps = append(imps, decl)
		case *OptionNode:
			opts = append(opts, decl)
		case *MessageNode:
			msgs = append(msgs, decl)
		case *EnumNode:
			enms = append(enms, decl)
		case *ExtendNode:
			exts = append(exts, decl)
		case *ServiceNode:
			svcs = append(svcs, decl)
		case *EmptyDeclNode:
			// no-op
		default:
			panic(fmt.Sprintf("invalid FileElement type: %T", decl))
		}
	}

	ret := &FileNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Syntax:   syntax,
		Package:  pkgs,
		Imports:  imps,
		Options:  opts,
		Messages: msgs,
		Enums:    enms,
		Extends:  exts,
		Services: svcs,
		AllDecls: decls,
	}

	v := Visitor{
		VisitTerminalNode: func(n TerminalNode) (bool, *Visitor) {
			ret.AllComments = append(ret.AllComments, n.LeadingComments()...)
			ret.AllComments = append(ret.AllComments, n.TrailingComments()...)
			return false, nil
		},
	}
	PreOrderWalk(ret, v.Visit)

	return ret
}

type FileElement interface {
	Node
	fileElement()
}

var _ FileElement = (*ImportNode)(nil)
var _ FileElement = (*PackageNode)(nil)
var _ FileElement = (*OptionNode)(nil)
var _ FileElement = (*MessageNode)(nil)
var _ FileElement = (*EnumNode)(nil)
var _ FileElement = (*ExtendNode)(nil)
var _ FileElement = (*ServiceNode)(nil)
var _ FileElement = (*EmptyDeclNode)(nil)

type SyntaxNode struct {
	basicCompositeNode
	Keyword   *IdentNode
	Equals    *RuneNode
	Syntax    *CompoundStringNode
	Semicolon *RuneNode
}

func NewSyntaxNode(keyword *IdentNode, equals *RuneNode, syntax *CompoundStringNode, semicolon *RuneNode) *SyntaxNode {
	children := []Node{keyword, equals, syntax, semicolon}
	return &SyntaxNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Keyword:   keyword,
		Equals:    equals,
		Syntax:    syntax,
		Semicolon: semicolon,
	}
}

type ImportNode struct {
	basicCompositeNode
	Keyword   *IdentNode
	Public    *IdentNode
	Weak      *IdentNode
	Name      *CompoundStringNode
	Semicolon *RuneNode
}

func NewImportNode(keyword *IdentNode, public *IdentNode, weak *IdentNode, name *CompoundStringNode, semicolon *RuneNode) *ImportNode {
	numChildren := 3
	if public != nil || weak != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	children = append(children, keyword)
	if public != nil {
		children = append(children, public)
	} else if weak != nil {
		children = append(children, weak)
	}
	children = append(children, name, semicolon)

	return &ImportNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Keyword:   keyword,
		Public:    public,
		Weak:      weak,
		Name:      name,
		Semicolon: semicolon,
	}
}

func (*ImportNode) fileElement() {}

type PackageNode struct {
	basicCompositeNode
	Keyword   *IdentNode
	Name      *CompoundIdentNode
	Semicolon *RuneNode
}

func (*PackageNode) fileElement() {}

func NewPackageNode(keyword *IdentNode, name *CompoundIdentNode, semicolon *RuneNode) *PackageNode {
	children := []Node{keyword, name, semicolon}
	return &PackageNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Keyword:   keyword,
		Name:      name,
		Semicolon: semicolon,
	}
}
