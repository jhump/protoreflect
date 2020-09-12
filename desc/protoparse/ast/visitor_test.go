package ast

import (
	"sort"
	"testing"

	"github.com/jhump/protoreflect/internal/testutil"
)

// The single visitor returned has all functions set to record the method call.
// The slice of visitors has one element per function, each with exactly one
// function set. So the first visitor can be used to determine the preferred
// function (which should match the node's concrete type). The slice of visitors
// can be used to enumerate ALL matching function calls.
//
// This function is generated via commented-out code at the bottom of this file.
func testVisitors(methodCalled *string) (*Visitor, []*Visitor) {
	v := &Visitor{
		VisitEnumNode: func(*EnumNode) (bool, *Visitor) {
			*methodCalled = "*EnumNode"
			return false, nil
		},
		VisitEnumValueNode: func(*EnumValueNode) (bool, *Visitor) {
			*methodCalled = "*EnumValueNode"
			return false, nil
		},
		VisitFieldDeclNode: func(FieldDeclNode) (bool, *Visitor) {
			*methodCalled = "FieldDeclNode"
			return false, nil
		},
		VisitFieldNode: func(*FieldNode) (bool, *Visitor) {
			*methodCalled = "*FieldNode"
			return false, nil
		},
		VisitGroupNode: func(*GroupNode) (bool, *Visitor) {
			*methodCalled = "*GroupNode"
			return false, nil
		},
		VisitOneOfNode: func(*OneOfNode) (bool, *Visitor) {
			*methodCalled = "*OneOfNode"
			return false, nil
		},
		VisitMapTypeNode: func(*MapTypeNode) (bool, *Visitor) {
			*methodCalled = "*MapTypeNode"
			return false, nil
		},
		VisitMapFieldNode: func(*MapFieldNode) (bool, *Visitor) {
			*methodCalled = "*MapFieldNode"
			return false, nil
		},
		VisitFileNode: func(*FileNode) (bool, *Visitor) {
			*methodCalled = "*FileNode"
			return false, nil
		},
		VisitSyntaxNode: func(*SyntaxNode) (bool, *Visitor) {
			*methodCalled = "*SyntaxNode"
			return false, nil
		},
		VisitImportNode: func(*ImportNode) (bool, *Visitor) {
			*methodCalled = "*ImportNode"
			return false, nil
		},
		VisitPackageNode: func(*PackageNode) (bool, *Visitor) {
			*methodCalled = "*PackageNode"
			return false, nil
		},
		VisitIdentValueNode: func(IdentValueNode) (bool, *Visitor) {
			*methodCalled = "IdentValueNode"
			return false, nil
		},
		VisitIdentNode: func(*IdentNode) (bool, *Visitor) {
			*methodCalled = "*IdentNode"
			return false, nil
		},
		VisitCompoundIdentNode: func(*CompoundIdentNode) (bool, *Visitor) {
			*methodCalled = "*CompoundIdentNode"
			return false, nil
		},
		VisitKeywordNode: func(*KeywordNode) (bool, *Visitor) {
			*methodCalled = "*KeywordNode"
			return false, nil
		},
		VisitMessageDeclNode: func(MessageDeclNode) (bool, *Visitor) {
			*methodCalled = "MessageDeclNode"
			return false, nil
		},
		VisitMessageNode: func(*MessageNode) (bool, *Visitor) {
			*methodCalled = "*MessageNode"
			return false, nil
		},
		VisitExtendNode: func(*ExtendNode) (bool, *Visitor) {
			*methodCalled = "*ExtendNode"
			return false, nil
		},
		VisitNode: func(Node) (bool, *Visitor) {
			*methodCalled = "Node"
			return false, nil
		},
		VisitTerminalNode: func(TerminalNode) (bool, *Visitor) {
			*methodCalled = "TerminalNode"
			return false, nil
		},
		VisitCompositeNode: func(CompositeNode) (bool, *Visitor) {
			*methodCalled = "CompositeNode"
			return false, nil
		},
		VisitRuneNode: func(*RuneNode) (bool, *Visitor) {
			*methodCalled = "*RuneNode"
			return false, nil
		},
		VisitEmptyDeclNode: func(*EmptyDeclNode) (bool, *Visitor) {
			*methodCalled = "*EmptyDeclNode"
			return false, nil
		},
		VisitOptionNode: func(*OptionNode) (bool, *Visitor) {
			*methodCalled = "*OptionNode"
			return false, nil
		},
		VisitOptionNameNode: func(*OptionNameNode) (bool, *Visitor) {
			*methodCalled = "*OptionNameNode"
			return false, nil
		},
		VisitFieldReferenceNode: func(*FieldReferenceNode) (bool, *Visitor) {
			*methodCalled = "*FieldReferenceNode"
			return false, nil
		},
		VisitCompactOptionsNode: func(*CompactOptionsNode) (bool, *Visitor) {
			*methodCalled = "*CompactOptionsNode"
			return false, nil
		},
		VisitExtensionRangeNode: func(*ExtensionRangeNode) (bool, *Visitor) {
			*methodCalled = "*ExtensionRangeNode"
			return false, nil
		},
		VisitRangeNode: func(*RangeNode) (bool, *Visitor) {
			*methodCalled = "*RangeNode"
			return false, nil
		},
		VisitReservedNode: func(*ReservedNode) (bool, *Visitor) {
			*methodCalled = "*ReservedNode"
			return false, nil
		},
		VisitServiceNode: func(*ServiceNode) (bool, *Visitor) {
			*methodCalled = "*ServiceNode"
			return false, nil
		},
		VisitRPCNode: func(*RPCNode) (bool, *Visitor) {
			*methodCalled = "*RPCNode"
			return false, nil
		},
		VisitRPCTypeNode: func(*RPCTypeNode) (bool, *Visitor) {
			*methodCalled = "*RPCTypeNode"
			return false, nil
		},
		VisitValueNode: func(ValueNode) (bool, *Visitor) {
			*methodCalled = "ValueNode"
			return false, nil
		},
		VisitStringValueNode: func(StringValueNode) (bool, *Visitor) {
			*methodCalled = "StringValueNode"
			return false, nil
		},
		VisitStringLiteralNode: func(*StringLiteralNode) (bool, *Visitor) {
			*methodCalled = "*StringLiteralNode"
			return false, nil
		},
		VisitCompoundStringLiteralNode: func(*CompoundStringLiteralNode) (bool, *Visitor) {
			*methodCalled = "*CompoundStringLiteralNode"
			return false, nil
		},
		VisitIntValueNode: func(IntValueNode) (bool, *Visitor) {
			*methodCalled = "IntValueNode"
			return false, nil
		},
		VisitUintLiteralNode: func(*UintLiteralNode) (bool, *Visitor) {
			*methodCalled = "*UintLiteralNode"
			return false, nil
		},
		VisitPositiveUintLiteralNode: func(*PositiveUintLiteralNode) (bool, *Visitor) {
			*methodCalled = "*PositiveUintLiteralNode"
			return false, nil
		},
		VisitNegativeIntLiteralNode: func(*NegativeIntLiteralNode) (bool, *Visitor) {
			*methodCalled = "*NegativeIntLiteralNode"
			return false, nil
		},
		VisitFloatValueNode: func(FloatValueNode) (bool, *Visitor) {
			*methodCalled = "FloatValueNode"
			return false, nil
		},
		VisitFloatLiteralNode: func(*FloatLiteralNode) (bool, *Visitor) {
			*methodCalled = "*FloatLiteralNode"
			return false, nil
		},
		VisitSpecialFloatLiteralNode: func(*SpecialFloatLiteralNode) (bool, *Visitor) {
			*methodCalled = "*SpecialFloatLiteralNode"
			return false, nil
		},
		VisitSignedFloatLiteralNode: func(*SignedFloatLiteralNode) (bool, *Visitor) {
			*methodCalled = "*SignedFloatLiteralNode"
			return false, nil
		},
		VisitBoolLiteralNode: func(*BoolLiteralNode) (bool, *Visitor) {
			*methodCalled = "*BoolLiteralNode"
			return false, nil
		},
		VisitArrayLiteralNode: func(*ArrayLiteralNode) (bool, *Visitor) {
			*methodCalled = "*ArrayLiteralNode"
			return false, nil
		},
		VisitMessageLiteralNode: func(*MessageLiteralNode) (bool, *Visitor) {
			*methodCalled = "*MessageLiteralNode"
			return false, nil
		},
		VisitMessageFieldNode: func(*MessageFieldNode) (bool, *Visitor) {
			*methodCalled = "*MessageFieldNode"
			return false, nil
		},
	}
	others := []*Visitor{
		{
			VisitEnumNode: v.VisitEnumNode,
		},
		{
			VisitEnumValueNode: v.VisitEnumValueNode,
		},
		{
			VisitFieldDeclNode: v.VisitFieldDeclNode,
		},
		{
			VisitFieldNode: v.VisitFieldNode,
		},
		{
			VisitGroupNode: v.VisitGroupNode,
		},
		{
			VisitOneOfNode: v.VisitOneOfNode,
		},
		{
			VisitMapTypeNode: v.VisitMapTypeNode,
		},
		{
			VisitMapFieldNode: v.VisitMapFieldNode,
		},
		{
			VisitFileNode: v.VisitFileNode,
		},
		{
			VisitSyntaxNode: v.VisitSyntaxNode,
		},
		{
			VisitImportNode: v.VisitImportNode,
		},
		{
			VisitPackageNode: v.VisitPackageNode,
		},
		{
			VisitIdentValueNode: v.VisitIdentValueNode,
		},
		{
			VisitIdentNode: v.VisitIdentNode,
		},
		{
			VisitCompoundIdentNode: v.VisitCompoundIdentNode,
		},
		{
			VisitKeywordNode: v.VisitKeywordNode,
		},
		{
			VisitMessageDeclNode: v.VisitMessageDeclNode,
		},
		{
			VisitMessageNode: v.VisitMessageNode,
		},
		{
			VisitExtendNode: v.VisitExtendNode,
		},
		{
			VisitNode: v.VisitNode,
		},
		{
			VisitTerminalNode: v.VisitTerminalNode,
		},
		{
			VisitCompositeNode: v.VisitCompositeNode,
		},
		{
			VisitRuneNode: v.VisitRuneNode,
		},
		{
			VisitEmptyDeclNode: v.VisitEmptyDeclNode,
		},
		{
			VisitOptionNode: v.VisitOptionNode,
		},
		{
			VisitOptionNameNode: v.VisitOptionNameNode,
		},
		{
			VisitFieldReferenceNode: v.VisitFieldReferenceNode,
		},
		{
			VisitCompactOptionsNode: v.VisitCompactOptionsNode,
		},
		{
			VisitExtensionRangeNode: v.VisitExtensionRangeNode,
		},
		{
			VisitRangeNode: v.VisitRangeNode,
		},
		{
			VisitReservedNode: v.VisitReservedNode,
		},
		{
			VisitServiceNode: v.VisitServiceNode,
		},
		{
			VisitRPCNode: v.VisitRPCNode,
		},
		{
			VisitRPCTypeNode: v.VisitRPCTypeNode,
		},
		{
			VisitValueNode: v.VisitValueNode,
		},
		{
			VisitStringValueNode: v.VisitStringValueNode,
		},
		{
			VisitStringLiteralNode: v.VisitStringLiteralNode,
		},
		{
			VisitCompoundStringLiteralNode: v.VisitCompoundStringLiteralNode,
		},
		{
			VisitIntValueNode: v.VisitIntValueNode,
		},
		{
			VisitUintLiteralNode: v.VisitUintLiteralNode,
		},
		{
			VisitPositiveUintLiteralNode: v.VisitPositiveUintLiteralNode,
		},
		{
			VisitNegativeIntLiteralNode: v.VisitNegativeIntLiteralNode,
		},
		{
			VisitFloatValueNode: v.VisitFloatValueNode,
		},
		{
			VisitFloatLiteralNode: v.VisitFloatLiteralNode,
		},
		{
			VisitSpecialFloatLiteralNode: v.VisitSpecialFloatLiteralNode,
		},
		{
			VisitSignedFloatLiteralNode: v.VisitSignedFloatLiteralNode,
		},
		{
			VisitBoolLiteralNode: v.VisitBoolLiteralNode,
		},
		{
			VisitArrayLiteralNode: v.VisitArrayLiteralNode,
		},
		{
			VisitMessageLiteralNode: v.VisitMessageLiteralNode,
		},
		{
			VisitMessageFieldNode: v.VisitMessageFieldNode,
		},
	}
	return v, others
}

func TestVisitorAll(t *testing.T) {
	testCases := map[Node][]string{
		(*EnumNode)(nil): {
			"*EnumNode", "CompositeNode", "Node",
		},
		(*EnumValueNode)(nil): {
			"*EnumValueNode", "CompositeNode", "Node",
		},
		(*FieldNode)(nil): {
			"*FieldNode", "FieldDeclNode", "CompositeNode", "Node",
		},
		(*GroupNode)(nil): {
			"*GroupNode", "FieldDeclNode", "MessageDeclNode", "CompositeNode", "Node",
		},
		(*OneOfNode)(nil): {
			"*OneOfNode", "CompositeNode", "Node",
		},
		(*MapTypeNode)(nil): {
			"*MapTypeNode", "CompositeNode", "Node",
		},
		(*MapFieldNode)(nil): {
			"*MapFieldNode", "FieldDeclNode", "MessageDeclNode", "CompositeNode", "Node",
		},
		(*FileNode)(nil): {
			"*FileNode", "CompositeNode", "Node",
		},
		(*SyntaxNode)(nil): {
			"*SyntaxNode", "CompositeNode", "Node",
		},
		(*ImportNode)(nil): {
			"*ImportNode", "CompositeNode", "Node",
		},
		(*PackageNode)(nil): {
			"*PackageNode", "CompositeNode", "Node",
		},
		(*IdentNode)(nil): {
			"*IdentNode", "ValueNode", "IdentValueNode", "TerminalNode", "Node",
		},
		(*CompoundIdentNode)(nil): {
			"*CompoundIdentNode", "ValueNode", "IdentValueNode", "CompositeNode", "Node",
		},
		(*KeywordNode)(nil): {
			"*KeywordNode", "TerminalNode", "Node",
		},
		(*MessageNode)(nil): {
			"*MessageNode", "MessageDeclNode", "CompositeNode", "Node",
		},
		(*ExtendNode)(nil): {
			"*ExtendNode", "CompositeNode", "Node",
		},
		(*RuneNode)(nil): {
			"*RuneNode", "TerminalNode", "Node",
		},
		(*EmptyDeclNode)(nil): {
			"*EmptyDeclNode", "CompositeNode", "Node",
		},
		(*OptionNode)(nil): {
			"*OptionNode", "CompositeNode", "Node",
		},
		(*OptionNameNode)(nil): {
			"*OptionNameNode", "CompositeNode", "Node",
		},
		(*FieldReferenceNode)(nil): {
			"*FieldReferenceNode", "CompositeNode", "Node",
		},
		(*CompactOptionsNode)(nil): {
			"*CompactOptionsNode", "CompositeNode", "Node",
		},
		(*ExtensionRangeNode)(nil): {
			"*ExtensionRangeNode", "CompositeNode", "Node",
		},
		(*RangeNode)(nil): {
			"*RangeNode", "CompositeNode", "Node",
		},
		(*ReservedNode)(nil): {
			"*ReservedNode", "CompositeNode", "Node",
		},
		(*ServiceNode)(nil): {
			"*ServiceNode", "CompositeNode", "Node",
		},
		(*RPCNode)(nil): {
			"*RPCNode", "CompositeNode", "Node",
		},
		(*RPCTypeNode)(nil): {
			"*RPCTypeNode", "CompositeNode", "Node",
		},
		(*StringLiteralNode)(nil): {
			"*StringLiteralNode", "ValueNode", "StringValueNode", "TerminalNode", "Node",
		},
		(*CompoundStringLiteralNode)(nil): {
			"*CompoundStringLiteralNode", "ValueNode", "StringValueNode", "CompositeNode", "Node",
		},
		(*UintLiteralNode)(nil): {
			"*UintLiteralNode", "ValueNode", "IntValueNode", "FloatValueNode", "TerminalNode", "Node",
		},
		(*PositiveUintLiteralNode)(nil): {
			"*PositiveUintLiteralNode", "ValueNode", "IntValueNode", "CompositeNode", "Node",
		},
		(*NegativeIntLiteralNode)(nil): {
			"*NegativeIntLiteralNode", "ValueNode", "IntValueNode", "CompositeNode", "Node",
		},
		(*FloatLiteralNode)(nil): {
			"*FloatLiteralNode", "ValueNode", "FloatValueNode", "TerminalNode", "Node",
		},
		(*SpecialFloatLiteralNode)(nil): {
			"*SpecialFloatLiteralNode", "ValueNode", "FloatValueNode", "TerminalNode", "Node",
		},
		(*SignedFloatLiteralNode)(nil): {
			"*SignedFloatLiteralNode", "ValueNode", "FloatValueNode", "CompositeNode", "Node",
		},
		(*BoolLiteralNode)(nil): {
			"*BoolLiteralNode", "ValueNode", "TerminalNode", "Node",
		},
		(*ArrayLiteralNode)(nil): {
			"*ArrayLiteralNode", "ValueNode", "CompositeNode", "Node",
		},
		(*MessageLiteralNode)(nil): {
			"*MessageLiteralNode", "ValueNode", "CompositeNode", "Node",
		},
		(*MessageFieldNode)(nil): {
			"*MessageFieldNode", "CompositeNode", "Node",
		},
	}

	for n, expectedCalls := range testCases {
		var call string
		v, all := testVisitors(&call)
		_, _ = v.Visit(n)
		testutil.Eq(t, expectedCalls[0], call)
		var allCalls []string
		for _, v := range all {
			call = ""
			_, _ = v.Visit(n)
			if call != "" {
				allCalls = append(allCalls, call)
			}
		}
		sort.Strings(allCalls)
		sort.Strings(expectedCalls)
		testutil.Eq(t, expectedCalls, allCalls)
	}
}

func TestVisitorPriorityOrder(t *testing.T) {
	// This tests a handful of cases, concrete types that implement numerous interfaces,
	// and verifies that the preferred function on the visitor is called when present.
	var call string
	var n Node

	v, _ := testVisitors(&call)
	n = (*StringLiteralNode)(nil)

	v.VisitStringLiteralNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "StringValueNode", call)
	call = ""
	v.VisitStringValueNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "ValueNode", call)
	call = ""
	v.VisitValueNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "TerminalNode", call)
	call = ""
	v.VisitTerminalNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "Node", call)
	call = ""
	v.VisitNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "", call)

	v, _ = testVisitors(&call)
	n = (*CompoundStringLiteralNode)(nil)

	v.VisitCompoundStringLiteralNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "StringValueNode", call)
	call = ""
	v.VisitStringValueNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "ValueNode", call)
	call = ""
	v.VisitValueNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "CompositeNode", call)
	call = ""
	v.VisitCompositeNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "Node", call)
	call = ""
	v.VisitNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "", call)

	v, _ = testVisitors(&call)
	n = (*UintLiteralNode)(nil)

	v.VisitUintLiteralNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "IntValueNode", call)
	call = ""
	v.VisitIntValueNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "FloatValueNode", call)
	call = ""
	v.VisitFloatValueNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "ValueNode", call)
	call = ""
	v.VisitValueNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "TerminalNode", call)
	call = ""
	v.VisitTerminalNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "Node", call)
	call = ""
	v.VisitNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "", call)

	v, _ = testVisitors(&call)
	n = (*GroupNode)(nil)

	v.VisitGroupNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "FieldDeclNode", call)
	call = ""
	v.VisitFieldDeclNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "MessageDeclNode", call)
	call = ""
	v.VisitMessageDeclNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "CompositeNode", call)
	call = ""
	v.VisitCompositeNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "Node", call)
	call = ""
	v.VisitNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "", call)

	v, _ = testVisitors(&call)
	n = (*MapFieldNode)(nil)

	v.VisitMapFieldNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "FieldDeclNode", call)
	call = ""
	v.VisitFieldDeclNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "MessageDeclNode", call)
	call = ""
	v.VisitMessageDeclNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "CompositeNode", call)
	call = ""
	v.VisitCompositeNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "Node", call)
	call = ""
	v.VisitNode = nil
	_, _ = v.Visit(n)
	testutil.Eq(t, "", call)
}

//func TestDoGenerate(t *testing.T) {
//	generateVisitors()
//}
//
//func generateVisitors() {
//	// This is manually-curated list of all node types in this package
//	// Not all of them are valid as visitor functions, since we intentionally
//	// omit NoSourceNode, SyntheticMapFieldNode, the various *Element interfaces,
//	// and all of the *DeclNode interfaces that have only one real impl.
//	types := `
//*EnumNode
//EnumElement
//EnumValueDeclNode
//*EnumValueNode
//FieldDeclNode
//*FieldNode
//*FieldLabel
//*GroupNode
//*OneOfNode
//OneOfElement
//*MapTypeNode
//*MapFieldNode
//*SyntheticMapField
//FileDeclNode
//*FileNode
//FileElement
//*SyntaxNode
//*ImportNode
//*PackageNode
//IdentValueNode
//*IdentNode
//*CompoundIdentNode
//*KeywordNode
//MessageDeclNode
//*MessageNode
//MessageElement
//*ExtendNode
//ExtendElement
//Node
//TerminalNode
//CompositeNode
//*RuneNode
//*EmptyDeclNode
//OptionDeclNode
//*OptionNode
//*OptionNameNode
//*FieldReferenceNode
//*CompactOptionsNode
//*ExtensionRangeNode
//RangeDeclNode
//*RangeNode
//*ReservedNode
//*ServiceNode
//ServiceElement
//RPCDeclNode
//*RPCNode
//RPCElement
//*RPCTypeNode
//ValueNode
//StringValueNode
//*StringLiteralNode
//*CompoundStringLiteralNode
//IntValueNode
//*UintLiteralNode
//*PositiveUintLiteralNode
//*NegativeIntLiteralNode
//FloatValueNode
//*FloatLiteralNode
//*SpecialFloatLiteralNode
//*SignedFloatLiteralNode
//*BoolLiteralNode
//*ArrayLiteralNode
//*MessageLiteralNode
//*MessageFieldNode
//`
//	strs := strings.Split(types, "\n")
//	fmt.Println(`func testVisitors(methodCalled *string) (*Visitor, []*Visitor) {`)
//	fmt.Println(`	v := &Visitor{`)
//	for _, str := range strs {
//		if str == "" {
//			continue
//		}
//		name := strings.TrimPrefix(str, "*")
//		fmt.Printf(`		Visit%s: func(%s) (bool, *Visitor) {`, name, str); fmt.Println()
//		fmt.Printf(`			*methodCalled = "%s"`, str); fmt.Println()
//		fmt.Println(`			return false, nil`)
//		fmt.Println(`		},`)
//	}
//	fmt.Println(`	}`)
//	fmt.Println(`	others := []*Visitor{`)
//	for _, str := range strs {
//		if str == "" {
//			continue
//		}
//		name := strings.TrimPrefix(str, "*")
//		fmt.Println(`		{`)
//		fmt.Printf(`			Visit%s: v.Visit%s,`, name, name); fmt.Println()
//		fmt.Println(`		},`)
//	}
//	fmt.Println(`	}`)
//	fmt.Println(`}`)
//}
