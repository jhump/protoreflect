package ast

import (
	"math"
	"strings"
)

type ValueNode interface {
	Node
	Value() interface{}
}

var _ ValueNode = (*IdentNode)(nil)
var _ ValueNode = (*CompoundIdentNode)(nil)
var _ ValueNode = (*StringLiteralNode)(nil)
var _ ValueNode = (*CompoundStringNode)(nil)
var _ ValueNode = (*UintLiteralNode)(nil)
var _ ValueNode = (*CompoundUintNode)(nil)
var _ ValueNode = (*NegativeIntNode)(nil)
var _ ValueNode = (*FloatLiteralNode)(nil)
var _ ValueNode = (*SpecialFloatLiteralNode)(nil)
var _ ValueNode = (*CompoundFloatNode)(nil)
var _ ValueNode = (*BoolLiteralNode)(nil)
var _ ValueNode = (*SliceLiteralNode)(nil)
var _ ValueNode = (*AggregateLiteralNode)(nil)

type StringValueNode interface {
	ValueNode
	AsString() string
}

var _ StringValueNode = (*StringLiteralNode)(nil)
var _ StringValueNode = (*CompoundStringNode)(nil)

type StringLiteralNode struct {
	basicNode
	Val string
}

func NewStringLiteralNode(val string, info TokenInfo) *StringLiteralNode {
	return &StringLiteralNode{
		basicNode: basicNode{
			posRange: info.PosRange,
			leading:  info.LeadingComments,
			trailing: info.TrailingComments,
		},
		Val: val,
	}
}

func (n *StringLiteralNode) Value() interface{} {
	return n.AsString()
}

func (n *StringLiteralNode) AsString() string {
	return n.Val
}

type CompoundStringNode struct {
	basicCompositeNode
	Val string
}

func NewCompoundStringNode(components ...*StringLiteralNode) *CompoundStringNode {
	children := make([]Node, len(components))
	var b strings.Builder
	for i, comp := range components {
		children[i] = comp
		b.WriteString(comp.Val)
	}
	return &CompoundStringNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Val: b.String(),
	}
}

func (n *CompoundStringNode) Value() interface{} {
	return n.AsString()
}

func (n *CompoundStringNode) AsString() string {
	return n.Val
}

type IntValueNode interface {
	ValueNode
	AsInt64() (int64, bool)
	AsUint64() (uint64, bool)
}

func AsInt32(n IntValueNode, min, max int32) (int32, bool) {
	i, ok := n.AsInt64()
	if !ok {
		return 0, false
	}
	if i < int64(min) || i > int64(max) {
		return 0, false
	}
	return int32(i), true
}

var _ IntValueNode = (*UintLiteralNode)(nil)
var _ IntValueNode = (*CompoundUintNode)(nil)
var _ IntValueNode = (*NegativeIntNode)(nil)

type UintLiteralNode struct {
	basicNode
	Val uint64
}

func NewUintLiteralNode(val uint64, info TokenInfo) *UintLiteralNode {
	return &UintLiteralNode{
		basicNode: basicNode{
			posRange: info.PosRange,
			leading:  info.LeadingComments,
			trailing: info.TrailingComments,
		},
		Val: val,
	}
}

func (n *UintLiteralNode) Value() interface{} {
	return n.Val
}

func (n *UintLiteralNode) AsInt64() (int64, bool) {
	if n.Val > math.MaxInt64 {
		return 0, false
	}
	return int64(n.Val), true
}

func (n *UintLiteralNode) AsUint64() (uint64, bool) {
	return n.Val, true
}

func (n *UintLiteralNode) AsFloat() float64 {
	return float64(n.Val)
}

type CompoundUintNode struct {
	basicCompositeNode
	Sign *RuneNode
	Uint *UintLiteralNode
	Val  uint64
}

func NewCompoundUintNode(sign *RuneNode, i *UintLiteralNode) *CompoundUintNode {
	children := []Node{sign, i}
	return &CompoundUintNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Sign: sign,
		Uint: i,
		Val:  i.Val,
	}
}

func (n *CompoundUintNode) Value() interface{} {
	return n.Val
}

func (n *CompoundUintNode) AsInt64() (int64, bool) {
	if n.Val > math.MaxInt64 {
		return 0, false
	}
	return int64(n.Val), true
}

func (n *CompoundUintNode) AsUint64() (uint64, bool) {
	return n.Val, true
}

type NegativeIntNode struct {
	basicCompositeNode
	Sign *RuneNode
	Uint *UintLiteralNode
	Val  int64
}

func NewNegativeIntNode(sign *RuneNode, i *UintLiteralNode) *NegativeIntNode {
	children := []Node{sign, i}
	return &NegativeIntNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Sign: sign,
		Uint: i,
		Val:  -int64(i.Val),
	}
}

func (n *NegativeIntNode) Value() interface{} {
	return n.Val
}

func (n *NegativeIntNode) AsInt64() (int64, bool) {
	return n.Val, true
}

func (n *NegativeIntNode) AsUint64() (uint64, bool) {
	if n.Val < 0 {
		return 0, false
	}
	return uint64(n.Val), true
}

type FloatValueNode interface {
	ValueNode
	AsFloat() float64
}

var _ FloatValueNode = (*FloatLiteralNode)(nil)
var _ FloatValueNode = (*SpecialFloatLiteralNode)(nil)
var _ FloatValueNode = (*UintLiteralNode)(nil)

type FloatLiteralNode struct {
	basicNode
	Val float64
}

func NewFloatLiteralNode(val float64, info TokenInfo) *FloatLiteralNode {
	return &FloatLiteralNode{
		basicNode: basicNode{
			posRange: info.PosRange,
			leading:  info.LeadingComments,
			trailing: info.TrailingComments,
		},
		Val: val,
	}
}

func (n *FloatLiteralNode) Value() interface{} {
	return n.AsFloat()
}

func (n *FloatLiteralNode) AsFloat() float64 {
	return n.Val
}

type SpecialFloatLiteralNode struct {
	*IdentNode
	Val float64
}

func NewSpecialFloatLiteralNode(name *IdentNode) *SpecialFloatLiteralNode {
	var f float64
	if name.Val == "inf" {
		f = math.Inf(1)
	} else {
		f = math.NaN()
	}
	return &SpecialFloatLiteralNode{
		IdentNode: name,
		Val:       f,
	}
}

func (n *SpecialFloatLiteralNode) Value() interface{} {
	return n.AsFloat()
}

func (n *SpecialFloatLiteralNode) AsFloat() float64 {
	return n.Val
}

type CompoundFloatNode struct {
	basicCompositeNode
	Sign  *RuneNode
	Float FloatValueNode
	Val   float64
}

func NewCompoundFloatNode(sign *RuneNode, f *FloatLiteralNode) *CompoundFloatNode {
	children := []Node{sign, f}
	val := f.Val
	if sign.Rune == '-' {
		val = -f.Val
	}
	return &CompoundFloatNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Sign:  sign,
		Float: f,
		Val:   val,
	}
}

func (n *CompoundFloatNode) Value() interface{} {
	return n.Val
}

type BoolLiteralNode struct {
	*IdentNode
	Val bool
}

func NewBoolLiteralNode(name *IdentNode) *BoolLiteralNode {
	return &BoolLiteralNode{
		IdentNode: name,
		Val:       name.Val == "true",
	}
}

func (n *BoolLiteralNode) Value() interface{} {
	return n.Val
}

type SliceLiteralNode struct {
	basicCompositeNode
	OpenBracket  *RuneNode
	Elements     []ValueNode
	Commas       []*RuneNode
	CloseBracket *RuneNode
}

func NewSliceLiteralNode(open *RuneNode, vals []ValueNode, commas []*RuneNode, close *RuneNode) *SliceLiteralNode {
	children := make([]Node, len(vals)*2 + 1)
	children = append(children, open)
	for i, val := range vals {
		children = append(children, val)
		if i > 0 {
			children = append(children, commas[i-1])
		}
	}
	children = append(children, close)

	return &SliceLiteralNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		OpenBracket:  open,
		Elements:     vals,
		Commas:       commas,
		CloseBracket: close,
	}
}

func (n *SliceLiteralNode) Value() interface{} {
	return n.Elements
}

type AggregateLiteralNode struct {
	basicCompositeNode
	Open     *RuneNode
	Elements []*AggregateEntryNode
	Close    *RuneNode
}

func NewAggregateLiteralNode(open *RuneNode, vals []*AggregateEntryNode, close *RuneNode) *AggregateLiteralNode {
	children := make([]Node, len(vals) + 2)
	children = append(children, open)
	for _, val := range vals {
		children = append(children, val)
	}
	children = append(children, close)

	return &AggregateLiteralNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Open:     open,
		Elements: vals,
		Close:    close,
	}
}

func (n *AggregateLiteralNode) Value() interface{} {
	return n.Elements
}

type AggregateEntryNode struct {
	basicCompositeNode
	Name *FieldReferenceNode
	Sep  *RuneNode
	Val  ValueNode
	End  *RuneNode
}

func NewAggregateEntryNode(name *FieldReferenceNode, sep *RuneNode, val ValueNode, end *RuneNode) *AggregateEntryNode {
	numChildren := 2
	if sep != nil {
		numChildren++
	}
	if end != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	children = append(children, name)
	if sep != nil {
		children = append(children, sep)
	}
	children = append(children, val)
	if end != nil {
		children = append(children, end)
	}

	return &AggregateEntryNode{
		basicCompositeNode: basicCompositeNode{
			children: children,
		},
		Name: name,
		Sep:  sep,
		Val:  val,
		End:  end,
	}
}
