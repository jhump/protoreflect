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
	// Value returns a Go representation of the value. For scalars, this
	// will be a string, int64, uint64, float64, or bool. This could also
	// be an Identifier (e.g. IdentValueNodes). It can also be a composite
	// literal:
	//   * For array literals, the type returned will be []ValueNode
	//   * For message literals, the type returned will be []*MessageFieldNode
	Value() interface{}
}

var _ ValueNode = (*IdentNode)(nil)
var _ ValueNode = (*CompoundIdentNode)(nil)
var _ ValueNode = (*StringLiteralNode)(nil)
var _ ValueNode = (*CompoundStringLiteralNode)(nil)
var _ ValueNode = (*UintLiteralNode)(nil)
var _ ValueNode = (*PositiveUintLiteralNode)(nil)
var _ ValueNode = (*NegativeIntLiteralNode)(nil)
var _ ValueNode = (*FloatLiteralNode)(nil)
var _ ValueNode = (*SpecialFloatLiteralNode)(nil)
var _ ValueNode = (*SignedFloatLiteralNode)(nil)
var _ ValueNode = (*BoolLiteralNode)(nil)
var _ ValueNode = (*ArrayLiteralNode)(nil)
var _ ValueNode = (*MessageLiteralNode)(nil)

// StringValueNode is an AST node that represents a string literal.
// Such a node can be a single literal (*StringLiteralNode) or a
// concatenation of multiple literals (*CompoundStringLiteralNode).
type StringValueNode interface {
	ValueNode
	AsString() string
}

var _ StringValueNode = (*StringLiteralNode)(nil)
var _ StringValueNode = (*CompoundStringLiteralNode)(nil)

type StringLiteralNode struct {
	terminalNode
	// Val is the actual string value that the literal indicates.
	Val string
	// the text as it appeared in the source file, including
	// quotation marks and interior escape sequences.
	rawText string
}

func NewStringLiteralNode(val, rawText string, info TokenInfo) *StringLiteralNode {
	return &StringLiteralNode{
		terminalNode: info.asTerminalNode(),
		Val:          val,
		rawText:      rawText,
	}
}

func (n *StringLiteralNode) RawText() string {
	return n.rawText
}

func (n *StringLiteralNode) Value() interface{} {
	return n.AsString()
}

func (n *StringLiteralNode) AsString() string {
	return n.Val
}

type CompoundStringLiteralNode struct {
	compositeNode
	Val string
}

func NewCompoundLiteralStringNode(components ...*StringLiteralNode) *CompoundStringLiteralNode {
	children := make([]Node, len(components))
	var b strings.Builder
	for i, comp := range components {
		children[i] = comp
		b.WriteString(comp.Val)
	}
	return &CompoundStringLiteralNode{
		compositeNode: compositeNode{
			children: children,
		},
		Val: b.String(),
	}
}

func (n *CompoundStringLiteralNode) Value() interface{} {
	return n.AsString()
}

func (n *CompoundStringLiteralNode) AsString() string {
	return n.Val
}

// IntValueNode is an AST node that represents an integer literal. If
// an integer literal is too large for an int64 (or uint64 for
// positive literals), it is represented instead by a FloatValueNode.
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
	// Val is the numeric value indicated by the literal
	Val uint64
	// rawText is the actual token text as it appeared in the source file.
	rawText string
}

func NewUintLiteralNode(val uint64, rawText string, info TokenInfo) *UintLiteralNode {
	return &UintLiteralNode{
		terminalNode: info.asTerminalNode(),
		Val:          val,
		rawText:      rawText,
	}
}

func (n *UintLiteralNode) RawText() string {
	return n.rawText
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

// FloatValueNode is an AST node that represents a numeric literal with
// a floating point, in scientific notation, or too large to fit in an
// int64 or uint64.
type FloatValueNode interface {
	ValueNode
	AsFloat() float64
}

var _ FloatValueNode = (*FloatLiteralNode)(nil)
var _ FloatValueNode = (*SpecialFloatLiteralNode)(nil)
var _ FloatValueNode = (*UintLiteralNode)(nil)

type FloatLiteralNode struct {
	terminalNode
	// Val is the numeric value indicated by the literal
	Val float64
	// rawText is the actual token text as it appeared in the source file,
	// which could be in scientific notation.
	rawText string
}

func NewFloatLiteralNode(val float64, rawText string, info TokenInfo) *FloatLiteralNode {
	return &FloatLiteralNode{
		terminalNode: info.asTerminalNode(),
		Val:          val,
		rawText:      rawText,
	}
}

func (n *FloatLiteralNode) RawText() string {
	return n.rawText
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

func NewSignedFloatLiteralNode(sign *RuneNode, f FloatValueNode) *SignedFloatLiteralNode {
	children := []Node{sign, f}
	val := f.AsFloat()
	if sign.Rune == '-' {
		val = -val
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

type ArrayLiteralNode struct {
	compositeNode
	OpenBracket  *RuneNode
	Elements     []ValueNode
	Commas       []*RuneNode
	CloseBracket *RuneNode
}

func NewArrayLiteralNode(open *RuneNode, vals []ValueNode, commas []*RuneNode, close *RuneNode) *ArrayLiteralNode {
	children := make([]Node, len(vals)*2+1)
	children = append(children, open)
	for i, val := range vals {
		if i > 0 {
			children = append(children, commas[i-1])
		}
		children = append(children, val)
	}
	children = append(children, close)

	return &ArrayLiteralNode{
		compositeNode: compositeNode{
			children: children,
		},
		OpenBracket:  open,
		Elements:     vals,
		Commas:       commas,
		CloseBracket: close,
	}
}

func (n *ArrayLiteralNode) Value() interface{} {
	return n.Elements
}

type MessageLiteralNode struct {
	compositeNode
	Open     *RuneNode
	Elements []*MessageFieldNode
	Close    *RuneNode
}

func NewMessageLiteralNode(open *RuneNode, vals []*MessageFieldNode, close *RuneNode) *MessageLiteralNode {
	children := make([]Node, len(vals)+2)
	children = append(children, open)
	for _, val := range vals {
		children = append(children, val)
	}
	children = append(children, close)

	return &MessageLiteralNode{
		compositeNode: compositeNode{
			children: children,
		},
		Open:     open,
		Elements: vals,
		Close:    close,
	}
}

func (n *MessageLiteralNode) Value() interface{} {
	return n.Elements
}

type MessageFieldNode struct {
	compositeNode
	Name  *FieldReferenceNode
	Sep   *RuneNode
	Val   ValueNode
	Final *RuneNode
}

func NewMessageFieldNode(name *FieldReferenceNode, sep *RuneNode, val ValueNode, final *RuneNode) *MessageFieldNode {
	numChildren := 2
	if sep != nil {
		numChildren++
	}
	if final != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	children = append(children, name)
	if sep != nil {
		children = append(children, sep)
	}
	children = append(children, val)
	if final != nil {
		children = append(children, final)
	}

	return &MessageFieldNode{
		compositeNode: compositeNode{
			children: children,
		},
		Name:  name,
		Sep:   sep,
		Val:   val,
		Final: final,
	}
}
