package ast

// VisitFunc is used to examine a node in the AST when walking the tree.
// It returns a function to use to visit the children of the given node or
// nil to skip the given node's children.
//
// See also the Visitor type.
type VisitFunc func(Node) VisitFunc

// PreOrderWalk conducts a walk of the AST rooted at the given root using
// the given function. It visits a given AST node before it visits that
// node's descandants, hence the name "pre-order".
func PreOrderWalk(root Node, v VisitFunc) bool {
	v = v(root)
	if v == nil {
		return false
	}
	if comp, ok := root.(CompositeNode); ok {
		for _, child := range comp.Children() {
			if !PreOrderWalk(child, v) {
				return false
			}
		}
	}
	return true
}

// PostOrderWalk conducts a walk of the AST rooted at the given root using
// the given function. It visits a given AST node after it visits that
// node's descendants, hence the name "post-order".
func PostOrderWalk(root Node, v VisitFunc) bool {
	if comp, ok := root.(CompositeNode); ok {
		for _, child := range comp.Children() {
			if !PostOrderWalk(child, v) {
				return false
			}
		}
	}
	v = v(root)
	return v != nil
}

type Visitor struct {
	VisitFileNode                func(*FileNode) (bool, *Visitor)
	VisitSyntaxNode              func(*SyntaxNode) (bool, *Visitor)
	VisitPackageNode             func(*PackageNode) (bool, *Visitor)
	VisitImportNode              func(*ImportNode) (bool, *Visitor)
	VisitOptionNode              func(*OptionNode) (bool, *Visitor)
	VisitOptionNameNode          func(*OptionNameNode) (bool, *Visitor)
	VisitFieldReferenceNode      func(*FieldReferenceNode) (bool, *Visitor)
	VisitCompactOptionsNode      func(*CompactOptionsNode) (bool, *Visitor)
	VisitMessageNode             func(*MessageNode) (bool, *Visitor)
	VisitExtendNode              func(*ExtendNode) (bool, *Visitor)
	VisitExtensionRangeNode      func(*ExtensionRangeNode) (bool, *Visitor)
	VisitReservedNode            func(*ReservedNode) (bool, *Visitor)
	VisitRangeNode               func(*RangeNode) (bool, *Visitor)
	VisitFieldNode               func(*FieldNode) (bool, *Visitor)
	VisitGroupNode               func(*GroupNode) (bool, *Visitor)
	VisitMapFieldNode            func(*MapFieldNode) (bool, *Visitor)
	VisitMapTypeNode             func(*MapTypeNode) (bool, *Visitor)
	VisitOneOfNode               func(*OneOfNode) (bool, *Visitor)
	VisitEnumNode                func(*EnumNode) (bool, *Visitor)
	VisitEnumValueNode           func(*EnumValueNode) (bool, *Visitor)
	VisitServiceNode             func(*ServiceNode) (bool, *Visitor)
	VisitRPCNode                 func(*RPCNode) (bool, *Visitor)
	VisitRPCTypeNode             func(*RPCTypeNode) (bool, *Visitor)
	VisitIdentNode               func(*IdentNode) (bool, *Visitor)
	VisitCompoundIdentNode       func(*CompoundIdentNode) (bool, *Visitor)
	VisitStringLiteralNode       func(*StringLiteralNode) (bool, *Visitor)
	VisitCompoundStringNode      func(*CompoundStringNode) (bool, *Visitor)
	VisitUintLiteralNode         func(*UintLiteralNode) (bool, *Visitor)
	VisitCompoundUintNode        func(*CompoundUintNode) (bool, *Visitor)
	VisitNegativeIntNode         func(*NegativeIntNode) (bool, *Visitor)
	VisitFloatLiteralNode        func(*FloatLiteralNode) (bool, *Visitor)
	VisitSpecialFloatLiteralNode func(*SpecialFloatLiteralNode) (bool, *Visitor)
	VisitCompoundFloatNode       func(*CompoundFloatNode) (bool, *Visitor)
	VisitBoolLiteralNode         func(*BoolLiteralNode) (bool, *Visitor)
	VisitSliceLiteralNode        func(*SliceLiteralNode) (bool, *Visitor)
	VisitAggregateLiteralNode    func(*AggregateLiteralNode) (bool, *Visitor)
	VisitRuneNode                func(*RuneNode) (bool, *Visitor)
	VisitEmptyDeclNode           func(*EmptyDeclNode) (bool, *Visitor)

	VisitFieldDeclNode   func(FieldDeclNode) (bool, *Visitor)
	VisitMessageDeclNode func(MessageDeclNode) (bool, *Visitor)
	VisitOptionDeclNode  func(OptionDeclNode) (bool, *Visitor)

	VisitIdentValueNode  func(IdentValueNode) (bool, *Visitor)
	VisitStringValueNode func(StringValueNode) (bool, *Visitor)
	VisitIntValueNode    func(IntValueNode) (bool, *Visitor)
	VisitFloatValueNode  func(FloatValueNode) (bool, *Visitor)
	VisitValueNode       func(ValueNode) (bool, *Visitor)

	VisitTerminalNode  func(TerminalNode) (bool, *Visitor)
	VisitCompositeNode func(CompositeNode) (bool, *Visitor)
	VisitNode          func(Node) (bool, *Visitor)
}

func (v *Visitor) Visit(n Node) VisitFunc {
	var ok bool
	var next *Visitor
	switch n := n.(type) {
	case *FileNode:
		if v.VisitFileNode != nil {
			ok, next = v.VisitFileNode(n)
		}
	case *SyntaxNode:
		if v.VisitSyntaxNode != nil {
			ok, next = v.VisitSyntaxNode(n)
		}
	case *PackageNode:
		if v.VisitPackageNode != nil {
			ok, next = v.VisitPackageNode(n)
		}
	case *ImportNode:
		if v.VisitImportNode != nil {
			ok, next = v.VisitImportNode(n)
		}
	case *OptionNode:
		if v.VisitOptionNode != nil {
			ok, next = v.VisitOptionNode(n)
		}
	case *OptionNameNode:
		if v.VisitOptionNameNode != nil {
			ok, next = v.VisitOptionNameNode(n)
		}
	case *FieldReferenceNode:
		if v.VisitFieldReferenceNode != nil {
			ok, next = v.VisitFieldReferenceNode(n)
		}
	case *CompactOptionsNode:
		if v.VisitCompactOptionsNode != nil {
			ok, next = v.VisitCompactOptionsNode(n)
		}
	case *MessageNode:
		if v.VisitMessageNode != nil {
			ok, next = v.VisitMessageNode(n)
		}
	case *ExtendNode:
		if v.VisitExtendNode != nil {
			ok, next = v.VisitExtendNode(n)
		}
	case *ExtensionRangeNode:
		if v.VisitExtensionRangeNode != nil {
			ok, next = v.VisitExtensionRangeNode(n)
		}
	case *ReservedNode:
		if v.VisitReservedNode != nil {
			ok, next = v.VisitReservedNode(n)
		}
	case *RangeNode:
		if v.VisitRangeNode != nil {
			ok, next = v.VisitRangeNode(n)
		}
	case *FieldNode:
		if v.VisitFieldNode != nil {
			ok, next = v.VisitFieldNode(n)
		}
	case *GroupNode:
		if v.VisitGroupNode != nil {
			ok, next = v.VisitGroupNode(n)
		}
	case *MapFieldNode:
		if v.VisitMapFieldNode != nil {
			ok, next = v.VisitMapFieldNode(n)
		}
	case *MapTypeNode:
		if v.VisitMapTypeNode != nil {
			ok, next = v.VisitMapTypeNode(n)
		}
	case *OneOfNode:
		if v.VisitOneOfNode != nil {
			ok, next = v.VisitOneOfNode(n)
		}
	case *EnumNode:
		if v.VisitEnumNode != nil {
			ok, next = v.VisitEnumNode(n)
		}
	case *EnumValueNode:
		if v.VisitEnumValueNode != nil {
			ok, next = v.VisitEnumValueNode(n)
		}
	case *ServiceNode:
		if v.VisitServiceNode != nil {
			ok, next = v.VisitServiceNode(n)
		}
	case *RPCNode:
		if v.VisitRPCNode != nil {
			ok, next = v.VisitRPCNode(n)
		}
	case *RPCTypeNode:
		if v.VisitRPCTypeNode != nil {
			ok, next = v.VisitRPCTypeNode(n)
		}
	case *IdentNode:
		if v.VisitIdentNode != nil {
			ok, next = v.VisitIdentNode(n)
		}
	case *CompoundIdentNode:
		if v.VisitCompoundIdentNode != nil {
			ok, next = v.VisitCompoundIdentNode(n)
		}
	case *StringLiteralNode:
		if v.VisitStringLiteralNode != nil {
			ok, next = v.VisitStringLiteralNode(n)
		}
	case *CompoundStringNode:
		if v.VisitCompoundStringNode != nil {
			ok, next = v.VisitCompoundStringNode(n)
		}
	case *UintLiteralNode:
		if v.VisitUintLiteralNode != nil {
			ok, next = v.VisitUintLiteralNode(n)
		}
	case *CompoundUintNode:
		if v.VisitCompoundUintNode != nil {
			ok, next = v.VisitCompoundUintNode(n)
		}
	case *NegativeIntNode:
		if v.VisitNegativeIntNode != nil {
			ok, next = v.VisitNegativeIntNode(n)
		}
	case *FloatLiteralNode:
		if v.VisitFloatLiteralNode != nil {
			ok, next = v.VisitFloatLiteralNode(n)
		}
	case *SpecialFloatLiteralNode:
		if v.VisitSpecialFloatLiteralNode != nil {
			ok, next = v.VisitSpecialFloatLiteralNode(n)
		}
	case *CompoundFloatNode:
		if v.VisitCompoundFloatNode != nil {
			ok, next = v.VisitCompoundFloatNode(n)
		}
	case *BoolLiteralNode:
		if v.VisitBoolLiteralNode != nil {
			ok, next = v.VisitBoolLiteralNode(n)
		}
	case *SliceLiteralNode:
		if v.VisitSliceLiteralNode != nil {
			ok, next = v.VisitSliceLiteralNode(n)
		}
	case *AggregateLiteralNode:
		if v.VisitAggregateLiteralNode != nil {
			ok, next = v.VisitAggregateLiteralNode(n)
		}
	case *RuneNode:
		if v.VisitRuneNode != nil {
			ok, next = v.VisitRuneNode(n)
		}
	case *EmptyDeclNode:
		if v.VisitEmptyDeclNode != nil {
			ok, next = v.VisitEmptyDeclNode(n)
		}
	case MessageDeclNode:
		if v.VisitMessageDeclNode != nil {
			ok, next = v.VisitMessageDeclNode(n)
		}
	case OptionDeclNode:
		if v.VisitOptionDeclNode != nil {
			ok, next = v.VisitOptionDeclNode(n)
		}
	case FieldDeclNode:
		if v.VisitFieldDeclNode != nil {
			ok, next = v.VisitFieldDeclNode(n)
		}
	case IdentValueNode:
		if v.VisitIdentValueNode != nil {
			ok, next = v.VisitIdentValueNode(n)
		}
	case StringValueNode:
		if v.VisitStringValueNode != nil {
			ok, next = v.VisitStringValueNode(n)
		}
	case IntValueNode:
		if v.VisitIntValueNode != nil {
			ok, next = v.VisitIntValueNode(n)
		}
	case FloatValueNode:
		if v.VisitFloatValueNode != nil {
			ok, next = v.VisitFloatValueNode(n)
		}
	case ValueNode:
		if v.VisitValueNode != nil {
			ok, next = v.VisitValueNode(n)
		}
	case TerminalNode:
		if v.VisitTerminalNode != nil {
			ok, next = v.VisitTerminalNode(n)
		}
	case CompositeNode:
		if v.VisitCompositeNode != nil {
			ok, next = v.VisitCompositeNode(n)
		}
	case Node:
		if v.VisitNode != nil {
			ok, next = v.VisitNode(n)
		}
	default:
		return nil
	}

	if !ok {
		return nil
	}
	if next != nil {
		return next.Visit
	}
	return v.Visit
}