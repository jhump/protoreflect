package ast

import (
	"math"
	"strings"
)

// ValueNode is an AST node that represents a literal value.
//
// It also includes references (e.g. IdentifierValueNode), which can be
// used as values in some contexts, such as describing the default value
// for a field, which can refer to an enum value.
type ValueNode interface {
	Node
	Value() interface{}
}

var _ ValueNode = (*IdentNode)(nil)
var _ ValueNode = (*CompoundIdentNode)(nil)
var _ ValueNode = (*StringLiteralNode)(nil)
var _ ValueNode = (*CompoundStringListeralNode)(nil)
var _ ValueNode = (*UintLiteralNode)(nil)
var _ ValueNode = (*PositiveUintLiteralNode)(nil)
var _ ValueNode = (*NegativeIntLiteralNode)(nil)
var _ ValueNode = (*FloatLiteralNode)(nil)
var _ ValueNode = (*SpecialFloatLiteralNode)(nil)
var _ ValueNode = (*SignedFloatLiteralNode)(nil)
var _ ValueNode = (*BoolLiteralNode)(nil)
var _ ValueNode = (*SliceLiteralNode)(nil)
var _ ValueNode = (*AggregateLiteralNode)(nil)

type StringValueNode interface {
	ValueNode
	AsString() string
}

var _ StringValueNode = (*StringLiteralNode)(nil)
var _ StringValueNode = (*CompoundStringListeralNode)(nil)

type StringLiteralNode struct {
	terminalNode
	Val string
}

func NewStringLiteralNode(val string, info TokenInfo) *StringLiteralNode {
	return &StringLiteralNode{
		terminalNode: terminalNode{
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

type CompoundStringListeralNode struct {
	compositeNode
	Val string
}

func NewCompoundLiteralStringNode(components ...*StringLiteralNode) *CompoundStringListeralNode {
	children := make([]Node, len(components))
	var b strings.Builder
	for i, comp := range components {
		children[i] = comp
		b.WriteString(comp.Val)
	}
	return &CompoundStringListeralNode{
		compositeNode: compositeNode{
			children: children,
		},
		Val: b.String(),
	}
}

func (n *CompoundStringListeralNode) Value() interface{} {
	return n.AsString()
}

func (n *CompoundStringListeralNode) AsString() string {
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
var _ IntValueNode = (*PositiveUintLiteralNode)(nil)
var _ IntValueNode = (*NegativeIntLiteralNode)(nil)

type UintLiteralNode struct {
	terminalNode
	Val uint64
}

func NewUintLiteralNode(val uint64, info TokenInfo) *UintLiteralNode {
	return &UintLiteralNode{
		terminalNode: terminalNode{
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

type PositiveUintLiteralNode struct {
	compositeNode
	Plus *RuneNode
	Uint *UintLiteralNode
	Val  uint64
}

func NewPositiveUintLiteralNode(sign *RuneNode, i *UintLiteralNode) *PositiveUintLiteralNode {
	children := []Node{sign, i}
	return &PositiveUintLiteralNode{
		compositeNode: compositeNode{
			children: children,
		},
		Plus: sign,
		Uint: i,
		Val:  i.Val,
	}
}

func (n *PositiveUintLiteralNode) Value() interface{} {
	return n.Val
}

func (n *PositiveUintLiteralNode) AsInt64() (int64, bool) {
	if n.Val > math.MaxInt64 {
		return 0, false
	}
	return int64(n.Val), true
}

func (n *PositiveUintLiteralNode) AsUint64() (uint64, bool) {
	return n.Val, true
}

type NegativeIntLiteralNode struct {
	compositeNode
	Minus *RuneNode
	Uint  *UintLiteralNode
	Val   int64
}

func NewNegativeIntLiteralNode(sign *RuneNode, i *UintLiteralNode) *NegativeIntLiteralNode {
	children := []Node{sign, i}
	return &NegativeIntLiteralNode{
		compositeNode: compositeNode{
			children: children,
		},
		Minus: sign,
		Uint:  i,
		Val:   -int64(i.Val),
	}
}

func (n *NegativeIntLiteralNode) Value() interface{} {
	return n.Val
}

func (n *NegativeIntLiteralNode) AsInt64() (int64, bool) {
	return n.Val, true
}

func (n *NegativeIntLiteralNode) AsUint64() (uint64, bool) {
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
	terminalNode
	Val float64
}

func NewFloatLiteralNode(val float64, info TokenInfo) *FloatLiteralNode {
	return &FloatLiteralNode{
		terminalNode: terminalNode{
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
	*KeywordNode
	Val float64
}

func NewSpecialFloatLiteralNode(name *KeywordNode) *SpecialFloatLiteralNode {
	var f float64
	if name.Val == "inf" {
		f = math.Inf(1)
	} else {
		f = math.NaN()
	}
	return &SpecialFloatLiteralNode{
		KeywordNode: name,
		Val:         f,
	}
}

func (n *SpecialFloatLiteralNode) Value() interface{} {
	return n.AsFloat()
}

func (n *SpecialFloatLiteralNode) AsFloat() float64 {
	return n.Val
}

type SignedFloatLiteralNode struct {
	compositeNode
	Sign  *RuneNode
	Float FloatValueNode
	Val   float64
}

func NewSignedFloatLiteralNode(sign *RuneNode, f *FloatLiteralNode) *SignedFloatLiteralNode {
	children := []Node{sign, f}
	val := f.Val
	if sign.Rune == '-' {
		val = -f.Val
	}
	return &SignedFloatLiteralNode{
		compositeNode: compositeNode{
			children: children,
		},
		Sign:  sign,
		Float: f,
		Val:   val,
	}
}

func (n *SignedFloatLiteralNode) Value() interface{} {
	return n.Val
}

type BoolLiteralNode struct {
	*KeywordNode
	Val bool
}

func NewBoolLiteralNode(name *KeywordNode) *BoolLiteralNode {
	return &BoolLiteralNode{
		KeywordNode: name,
		Val:         name.Val == "true",
	}
}

func (n *BoolLiteralNode) Value() interface{} {
	return n.Val
}

type SliceLiteralNode struct {
	compositeNode
	OpenBracket  *RuneNode
	Elements     []ValueNode
	Commas       []*RuneNode
	CloseBracket *RuneNode
}

func NewSliceLiteralNode(open *RuneNode, vals []ValueNode, commas []*RuneNode, close *RuneNode) *SliceLiteralNode {
	children := make([]Node, len(vals)*2 + 1)
	children = append(children, open)
	for i, val := range vals {
		if i > 0 {
			children = append(children, commas[i-1])
		}
		children = append(children, val)
	}
	children = append(children, close)

	return &SliceLiteralNode{
		compositeNode: compositeNode{
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
	compositeNode
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
		compositeNode: compositeNode{
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
	compositeNode
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
		compositeNode: compositeNode{
			children: children,
		},
		Name: name,
		Sep:  sep,
		Val:  val,
		End:  end,
	}
}
