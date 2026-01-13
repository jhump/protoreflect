// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ast

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The single visitor returned has all functions set to record the method call.
// The slice of visitors has one element per function, each with exactly one
// function set. So the first visitor can be used to determine the preferred
// function (which should match the node's concrete type). The slice of visitors
// can be used to enumerate ALL matching function calls.
//
// This function is generated via commented-out code at the bottom of this file.
func testVisitors(methodCalled *string) (*SimpleVisitor, []*SimpleVisitor) {
	v := &SimpleVisitor{
		DoVisitEnumNode: func(*EnumNode) error {
			*methodCalled = "*EnumNode"
			return nil
		},
		DoVisitEnumValueNode: func(*EnumValueNode) error {
			*methodCalled = "*EnumValueNode"
			return nil
		},
		DoVisitFieldDeclNode: func(FieldDeclNode) error {
			*methodCalled = "FieldDeclNode"
			return nil
		},
		DoVisitFieldNode: func(*FieldNode) error {
			*methodCalled = "*FieldNode"
			return nil
		},
		DoVisitGroupNode: func(*GroupNode) error {
			*methodCalled = "*GroupNode"
			return nil
		},
		DoVisitOneofNode: func(*OneofNode) error {
			*methodCalled = "*OneofNode"
			return nil
		},
		DoVisitMapTypeNode: func(*MapTypeNode) error {
			*methodCalled = "*MapTypeNode"
			return nil
		},
		DoVisitMapFieldNode: func(*MapFieldNode) error {
			*methodCalled = "*MapFieldNode"
			return nil
		},
		DoVisitFileNode: func(*FileNode) error {
			*methodCalled = "*FileNode"
			return nil
		},
		DoVisitSyntaxNode: func(*SyntaxNode) error {
			*methodCalled = "*SyntaxNode"
			return nil
		},
		DoVisitImportNode: func(*ImportNode) error {
			*methodCalled = "*ImportNode"
			return nil
		},
		DoVisitPackageNode: func(*PackageNode) error {
			*methodCalled = "*PackageNode"
			return nil
		},
		DoVisitIdentValueNode: func(IdentValueNode) error {
			*methodCalled = "IdentValueNode"
			return nil
		},
		DoVisitIdentNode: func(*IdentNode) error {
			*methodCalled = "*IdentNode"
			return nil
		},
		DoVisitCompoundIdentNode: func(*CompoundIdentNode) error {
			*methodCalled = "*CompoundIdentNode"
			return nil
		},
		DoVisitKeywordNode: func(*KeywordNode) error {
			*methodCalled = "*KeywordNode"
			return nil
		},
		DoVisitMessageDeclNode: func(MessageDeclNode) error {
			*methodCalled = "MessageDeclNode"
			return nil
		},
		DoVisitMessageNode: func(*MessageNode) error {
			*methodCalled = "*MessageNode"
			return nil
		},
		DoVisitExtendNode: func(*ExtendNode) error {
			*methodCalled = "*ExtendNode"
			return nil
		},
		DoVisitNode: func(Node) error {
			*methodCalled = "Node"
			return nil
		},
		DoVisitTerminalNode: func(TerminalNode) error {
			*methodCalled = "TerminalNode"
			return nil
		},
		DoVisitCompositeNode: func(CompositeNode) error {
			*methodCalled = "CompositeNode"
			return nil
		},
		DoVisitRuneNode: func(*RuneNode) error {
			*methodCalled = "*RuneNode"
			return nil
		},
		DoVisitEmptyDeclNode: func(*EmptyDeclNode) error {
			*methodCalled = "*EmptyDeclNode"
			return nil
		},
		DoVisitOptionNode: func(*OptionNode) error {
			*methodCalled = "*OptionNode"
			return nil
		},
		DoVisitOptionNameNode: func(*OptionNameNode) error {
			*methodCalled = "*OptionNameNode"
			return nil
		},
		DoVisitFieldReferenceNode: func(*FieldReferenceNode) error {
			*methodCalled = "*FieldReferenceNode"
			return nil
		},
		DoVisitCompactOptionsNode: func(*CompactOptionsNode) error {
			*methodCalled = "*CompactOptionsNode"
			return nil
		},
		DoVisitExtensionRangeNode: func(*ExtensionRangeNode) error {
			*methodCalled = "*ExtensionRangeNode"
			return nil
		},
		DoVisitRangeNode: func(*RangeNode) error {
			*methodCalled = "*RangeNode"
			return nil
		},
		DoVisitReservedNode: func(*ReservedNode) error {
			*methodCalled = "*ReservedNode"
			return nil
		},
		DoVisitServiceNode: func(*ServiceNode) error {
			*methodCalled = "*ServiceNode"
			return nil
		},
		DoVisitRPCNode: func(*RPCNode) error {
			*methodCalled = "*RPCNode"
			return nil
		},
		DoVisitRPCTypeNode: func(*RPCTypeNode) error {
			*methodCalled = "*RPCTypeNode"
			return nil
		},
		DoVisitValueNode: func(ValueNode) error {
			*methodCalled = "ValueNode"
			return nil
		},
		DoVisitStringValueNode: func(StringValueNode) error {
			*methodCalled = "StringValueNode"
			return nil
		},
		DoVisitStringLiteralNode: func(*StringLiteralNode) error {
			*methodCalled = "*StringLiteralNode"
			return nil
		},
		DoVisitCompoundStringLiteralNode: func(*CompoundStringLiteralNode) error {
			*methodCalled = "*CompoundStringLiteralNode"
			return nil
		},
		DoVisitIntValueNode: func(IntValueNode) error {
			*methodCalled = "IntValueNode"
			return nil
		},
		DoVisitUintLiteralNode: func(*UintLiteralNode) error {
			*methodCalled = "*UintLiteralNode"
			return nil
		},
		DoVisitNegativeIntLiteralNode: func(*NegativeIntLiteralNode) error {
			*methodCalled = "*NegativeIntLiteralNode"
			return nil
		},
		DoVisitFloatValueNode: func(FloatValueNode) error {
			*methodCalled = "FloatValueNode"
			return nil
		},
		DoVisitFloatLiteralNode: func(*FloatLiteralNode) error {
			*methodCalled = "*FloatLiteralNode"
			return nil
		},
		DoVisitSpecialFloatLiteralNode: func(*SpecialFloatLiteralNode) error {
			*methodCalled = "*SpecialFloatLiteralNode"
			return nil
		},
		DoVisitSignedFloatLiteralNode: func(*SignedFloatLiteralNode) error {
			*methodCalled = "*SignedFloatLiteralNode"
			return nil
		},
		DoVisitArrayLiteralNode: func(*ArrayLiteralNode) error {
			*methodCalled = "*ArrayLiteralNode"
			return nil
		},
		DoVisitMessageLiteralNode: func(*MessageLiteralNode) error {
			*methodCalled = "*MessageLiteralNode"
			return nil
		},
		DoVisitMessageFieldNode: func(*MessageFieldNode) error {
			*methodCalled = "*MessageFieldNode"
			return nil
		},
	}
	others := []*SimpleVisitor{
		{
			DoVisitEnumNode: v.DoVisitEnumNode,
		},
		{
			DoVisitEnumValueNode: v.DoVisitEnumValueNode,
		},
		{
			DoVisitFieldDeclNode: v.DoVisitFieldDeclNode,
		},
		{
			DoVisitFieldNode: v.DoVisitFieldNode,
		},
		{
			DoVisitGroupNode: v.DoVisitGroupNode,
		},
		{
			DoVisitOneofNode: v.DoVisitOneofNode,
		},
		{
			DoVisitMapTypeNode: v.DoVisitMapTypeNode,
		},
		{
			DoVisitMapFieldNode: v.DoVisitMapFieldNode,
		},
		{
			DoVisitFileNode: v.DoVisitFileNode,
		},
		{
			DoVisitSyntaxNode: v.DoVisitSyntaxNode,
		},
		{
			DoVisitImportNode: v.DoVisitImportNode,
		},
		{
			DoVisitPackageNode: v.DoVisitPackageNode,
		},
		{
			DoVisitIdentValueNode: v.DoVisitIdentValueNode,
		},
		{
			DoVisitIdentNode: v.DoVisitIdentNode,
		},
		{
			DoVisitCompoundIdentNode: v.DoVisitCompoundIdentNode,
		},
		{
			DoVisitKeywordNode: v.DoVisitKeywordNode,
		},
		{
			DoVisitMessageDeclNode: v.DoVisitMessageDeclNode,
		},
		{
			DoVisitMessageNode: v.DoVisitMessageNode,
		},
		{
			DoVisitExtendNode: v.DoVisitExtendNode,
		},
		{
			DoVisitNode: v.DoVisitNode,
		},
		{
			DoVisitTerminalNode: v.DoVisitTerminalNode,
		},
		{
			DoVisitCompositeNode: v.DoVisitCompositeNode,
		},
		{
			DoVisitRuneNode: v.DoVisitRuneNode,
		},
		{
			DoVisitEmptyDeclNode: v.DoVisitEmptyDeclNode,
		},
		{
			DoVisitOptionNode: v.DoVisitOptionNode,
		},
		{
			DoVisitOptionNameNode: v.DoVisitOptionNameNode,
		},
		{
			DoVisitFieldReferenceNode: v.DoVisitFieldReferenceNode,
		},
		{
			DoVisitCompactOptionsNode: v.DoVisitCompactOptionsNode,
		},
		{
			DoVisitExtensionRangeNode: v.DoVisitExtensionRangeNode,
		},
		{
			DoVisitRangeNode: v.DoVisitRangeNode,
		},
		{
			DoVisitReservedNode: v.DoVisitReservedNode,
		},
		{
			DoVisitServiceNode: v.DoVisitServiceNode,
		},
		{
			DoVisitRPCNode: v.DoVisitRPCNode,
		},
		{
			DoVisitRPCTypeNode: v.DoVisitRPCTypeNode,
		},
		{
			DoVisitValueNode: v.DoVisitValueNode,
		},
		{
			DoVisitStringValueNode: v.DoVisitStringValueNode,
		},
		{
			DoVisitStringLiteralNode: v.DoVisitStringLiteralNode,
		},
		{
			DoVisitCompoundStringLiteralNode: v.DoVisitCompoundStringLiteralNode,
		},
		{
			DoVisitIntValueNode: v.DoVisitIntValueNode,
		},
		{
			DoVisitUintLiteralNode: v.DoVisitUintLiteralNode,
		},
		{
			DoVisitNegativeIntLiteralNode: v.DoVisitNegativeIntLiteralNode,
		},
		{
			DoVisitFloatValueNode: v.DoVisitFloatValueNode,
		},
		{
			DoVisitFloatLiteralNode: v.DoVisitFloatLiteralNode,
		},
		{
			DoVisitSpecialFloatLiteralNode: v.DoVisitSpecialFloatLiteralNode,
		},
		{
			DoVisitSignedFloatLiteralNode: v.DoVisitSignedFloatLiteralNode,
		},
		{
			DoVisitArrayLiteralNode: v.DoVisitArrayLiteralNode,
		},
		{
			DoVisitMessageLiteralNode: v.DoVisitMessageLiteralNode,
		},
		{
			DoVisitMessageFieldNode: v.DoVisitMessageFieldNode,
		},
	}
	return v, others
}

func TestVisitorAll(t *testing.T) {
	t.Parallel()
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
			"*GroupNode", "FieldDeclNode", "CompositeNode", "Node",
		},
		(*OneofNode)(nil): {
			"*OneofNode", "CompositeNode", "Node",
		},
		(*MapTypeNode)(nil): {
			"*MapTypeNode", "CompositeNode", "Node",
		},
		(*MapFieldNode)(nil): {
			"*MapFieldNode", "FieldDeclNode", "CompositeNode", "Node",
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

	for n := range testCases {
		expectedCalls := testCases[n]
		t.Run(fmt.Sprintf("%T", n), func(t *testing.T) {
			t.Parallel()
			var call string
			v, all := testVisitors(&call)
			_ = Visit(n, v)
			assert.Equal(t, expectedCalls[0], call)
			var allCalls []string
			for _, v := range all {
				call = ""
				_ = Visit(n, v)
				if call != "" {
					allCalls = append(allCalls, call)
				}
			}
			sort.Strings(allCalls)
			sort.Strings(expectedCalls)
			assert.Equal(t, expectedCalls, allCalls)
		})
	}
}

func TestVisitorPriorityOrder(t *testing.T) {
	t.Parallel()
	// This tests a handful of cases, concrete types that implement numerous interfaces,
	// and verifies that the preferred function on the visitor is called when present.

	t.Run("StringLiteralNode", func(t *testing.T) {
		t.Parallel()
		var call string
		v, _ := testVisitors(&call)
		n := (*StringLiteralNode)(nil)

		v.DoVisitStringLiteralNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "StringValueNode", call)
		call = ""
		v.DoVisitStringValueNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "ValueNode", call)
		call = ""
		v.DoVisitValueNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "TerminalNode", call)
		call = ""
		v.DoVisitTerminalNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "Node", call)
		call = ""
		v.DoVisitNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "", call)
	})
	t.Run("CompoundStringLiteralNode", func(t *testing.T) {
		t.Parallel()
		var call string
		v, _ := testVisitors(&call)
		n := (*CompoundStringLiteralNode)(nil)

		v.DoVisitCompoundStringLiteralNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "StringValueNode", call)
		call = ""
		v.DoVisitStringValueNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "ValueNode", call)
		call = ""
		v.DoVisitValueNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "CompositeNode", call)
		call = ""
		v.DoVisitCompositeNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "Node", call)
		call = ""
		v.DoVisitNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "", call)
	})
	t.Run("UintLiteralNode", func(t *testing.T) {
		t.Parallel()
		var call string
		v, _ := testVisitors(&call)
		n := (*UintLiteralNode)(nil)

		v.DoVisitUintLiteralNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "IntValueNode", call)
		call = ""
		v.DoVisitIntValueNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "FloatValueNode", call)
		call = ""
		v.DoVisitFloatValueNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "ValueNode", call)
		call = ""
		v.DoVisitValueNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "TerminalNode", call)
		call = ""
		v.DoVisitTerminalNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "Node", call)
		call = ""
		v.DoVisitNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "", call)
	})
	t.Run("GroupNode", func(t *testing.T) {
		t.Parallel()
		var call string
		v, _ := testVisitors(&call)
		n := (*GroupNode)(nil)

		v.DoVisitGroupNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "FieldDeclNode", call)
		call = ""
		v.DoVisitFieldDeclNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "CompositeNode", call)
		call = ""
		v.DoVisitCompositeNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "Node", call)
		call = ""
		v.DoVisitNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "", call)
	})
	t.Run("MapFieldNode", func(t *testing.T) {
		t.Parallel()
		var call string
		v, _ := testVisitors(&call)
		n := (*MapFieldNode)(nil)

		v.DoVisitMapFieldNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "FieldDeclNode", call)
		call = ""
		v.DoVisitFieldDeclNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "CompositeNode", call)
		call = ""
		v.DoVisitCompositeNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "Node", call)
		call = ""
		v.DoVisitNode = nil
		_ = Visit(n, v)
		assert.Equal(t, "", call)
	})
}

func TestDoGenerate(t *testing.T) {
	t.Parallel()
	t.SkipNow()
	generateVisitors()
}

func generateVisitors() {
	// This is manually-curated list of all node types in this package
	// Not all of them are valid as visitor functions, since we intentionally
	// omit NoSourceNode, SyntheticMapFieldNode, the various *Element interfaces,
	// and all of the *DeclNode interfaces that have only one real impl.
	types := `
*EnumNode
EnumElement
EnumValueDeclNode
*EnumValueNode
FieldDeclNode
*FieldNode
*FieldLabel
*GroupNode
*OneofNode
OneofElement
*MapTypeNode
*MapFieldNode
*SyntheticMapField
FileDeclNode
*FileNode
FileElement
*SyntaxNode
*ImportNode
*PackageNode
IdentValueNode
*IdentNode
*CompoundIdentNode
*KeywordNode
MessageDeclNode
*MessageNode
MessageElement
*ExtendNode
ExtendElement
Node
TerminalNode
CompositeNode
*RuneNode
*EmptyDeclNode
OptionDeclNode
*OptionNode
*OptionNameNode
*FieldReferenceNode
*CompactOptionsNode
*ExtensionRangeNode
RangeDeclNode
*RangeNode
*ReservedNode
*ServiceNode
ServiceElement
RPCDeclNode
*RPCNode
RPCElement
*RPCTypeNode
ValueNode
StringValueNode
*StringLiteralNode
*CompoundStringLiteralNode
IntValueNode
*UintLiteralNode
*PositiveUintLiteralNode
*NegativeIntLiteralNode
FloatValueNode
*FloatLiteralNode
*SpecialFloatLiteralNode
*SignedFloatLiteralNode
*BoolLiteralNode
*ArrayLiteralNode
*MessageLiteralNode
*MessageFieldNode
`
	strs := strings.Split(types, "\n")
	fmt.Println(`func testVisitors(methodCalled *string) (*Visitor, []*Visitor) {`)
	fmt.Println(`	v := &SimpleVisitor{`)
	for _, str := range strs {
		if str == "" {
			continue
		}
		name := strings.TrimPrefix(str, "*")
		fmt.Printf(`		DoVisit%s: func(%s) error {`, name, str)
		fmt.Println()
		fmt.Printf(`			*methodCalled = "%s"`, str)
		fmt.Println()
		fmt.Println(`			return nil`)
		fmt.Println(`		},`)
	}
	fmt.Println(`	}`)
	fmt.Println(`	others := []*SimpleVisitor{`)
	for _, str := range strs {
		if str == "" {
			continue
		}
		name := strings.TrimPrefix(str, "*")
		fmt.Println(`		{`)
		fmt.Printf(`			DoVisit%s: v.DoVisit%s,`, name, name)
		fmt.Println()
		fmt.Println(`		},`)
	}
	fmt.Println(`	}`)
	fmt.Println(`}`)
}
