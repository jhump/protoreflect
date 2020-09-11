package ast

import "fmt"

type ExtensionRangeNode struct {
	compositeNode
	Keyword *KeywordNode
	Ranges  []*RangeNode
	// Commas represent the separating ',' characters between ranges. The
	// length of this slice must be exactly len(Ranges)-1, each item in Ranges
	// having a corresponding item in this slice *except the last* (since a
	// trailing comma is not allowed).
	Commas    []*RuneNode
	Options   *CompactOptionsNode
	Semicolon *RuneNode
}

func (e *ExtensionRangeNode) msgElement() {}

func NewExtensionRangeNode(keyword *KeywordNode, ranges []*RangeNode, commas []*RuneNode, opts *CompactOptionsNode, semicolon *RuneNode) *ExtensionRangeNode {
	if keyword == nil {
		panic("keyword is nil")
	}
	if semicolon == nil {
		panic("semicolon is nil")
	}
	if len(ranges) == 0 {
		panic("must have at least one range")
	}
	if len(commas) != len(ranges)-1 {
		panic(fmt.Sprintf("%d ranges requires %d commas, not %d", len(ranges), len(ranges)-1, len(commas)))
	}
	numChildren := len(ranges)*2 + 1
	if opts != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	children = append(children, keyword)
	for i, rng := range ranges {
		if i > 0 {
			if commas[i-1] == nil {
				panic(fmt.Sprintf("commas[%d] is nil", i-1))
			}
			children = append(children, commas[i-1])
		}
		if rng == nil {
			panic(fmt.Sprintf("ranges[%d] is nil", i))
		}
		children = append(children, rng)
	}
	if opts != nil {
		children = append(children, opts)
	}
	children = append(children, semicolon)
	return &ExtensionRangeNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:   keyword,
		Ranges:    ranges,
		Commas:    commas,
		Options:   opts,
		Semicolon: semicolon,
	}
}

type RangeDeclNode interface {
	Node
	RangeStart() Node
	RangeEnd() Node
}

var _ RangeDeclNode = (*RangeNode)(nil)
var _ RangeDeclNode = NoSourceNode{}

type RangeNode struct {
	compositeNode
	StartVal IntValueNode
	To       *KeywordNode
	EndVal   IntValueNode
	Max      *KeywordNode
}

func NewRangeNode(start IntValueNode, to *KeywordNode, end IntValueNode, max *KeywordNode) *RangeNode {
	if start == nil {
		panic("start is nil")
	}
	numChildren := 1
	if to != nil {
		if end == nil && max == nil {
			panic("to is not nil, but end and max both are")
		}
		if end != nil && max != nil {
			panic("end and max cannot be both non-nil")
		}
		numChildren = 3
	} else {
		if end != nil {
			panic("to is nil, but end is not")
		}
		if max != nil {
			panic("to is nil, but max is not")
		}
	}
	children := make([]Node, 0, numChildren)
	children = append(children, start)
	if to != nil {
		children = append(children, to)
		if end != nil {
			children = append(children, end)
		} else {
			children = append(children, max)
		}
	}
	return &RangeNode{
		compositeNode: compositeNode{
			children: children,
		},
		StartVal: start,
		To:       to,
		EndVal:   end,
		Max:      max,
	}
}

func (n *RangeNode) RangeStart() Node {
	return n.StartVal
}

func (n *RangeNode) RangeEnd() Node {
	if n.Max != nil {
		return n.Max
	}
	if n.EndVal != nil {
		return n.EndVal
	}
	return n.StartVal
}

func (n *RangeNode) StartValue() interface{} {
	return n.StartVal.Value()
}

func (n *RangeNode) StartValueAsInt32(min, max int32) (int32, bool) {
	return AsInt32(n.StartVal, min, max)
}

func (n *RangeNode) EndValue() interface{} {
	if n.EndVal == nil {
		return nil
	}
	return n.EndVal.Value()
}

func (n *RangeNode) EndValueAsInt32(min, max int32) (int32, bool) {
	if n.Max != nil {
		return max, true
	}
	if n.EndVal == nil {
		return n.StartValueAsInt32(min, max)
	}
	return AsInt32(n.EndVal, min, max)
}

type ReservedNode struct {
	compositeNode
	Keyword *KeywordNode
	// If non-empty, this node represents reserved ranges and Names will be empty.
	Ranges []*RangeNode
	// If non-empty, this node represents reserved names and Ranges will be empty.
	Names []StringValueNode
	// Commas represent the separating ',' characters between options. The
	// length of this slice must be exactly len(Ranges)-1 or len(Names)-1, depending
	// on whether this node represents reserved ranges or reserved names. Each item
	// in Ranges or Names has a corresponding item in this slice *except the last*
	// (since a trailing comma is not allowed).
	Commas    []*RuneNode
	Semicolon *RuneNode
}

func (*ReservedNode) msgElement()  {}
func (*ReservedNode) enumElement() {}

func NewReservedRangesNode(keyword *KeywordNode, ranges []*RangeNode, commas []*RuneNode, semicolon *RuneNode) *ReservedNode {
	if keyword == nil {
		panic("keyword is nil")
	}
	if semicolon == nil {
		panic("semicolon is nil")
	}
	if len(ranges) == 0 {
		panic("must have at least one range")
	}
	if len(commas) != len(ranges)-1 {
		panic(fmt.Sprintf("%d ranges requires %d commas, not %d", len(ranges), len(ranges)-1, len(commas)))
	}
	children := make([]Node, 0, len(ranges)*2+1)
	children = append(children, keyword)
	for i, rng := range ranges {
		if i > 0 {
			if commas[i-1] == nil {
				panic(fmt.Sprintf("commas[%d] is nil", i-1))
			}
			children = append(children, commas[i-1])
		}
		if rng == nil {
			panic(fmt.Sprintf("ranges[%d] is nil", i))
		}
		children = append(children, rng)
	}
	children = append(children, semicolon)
	return &ReservedNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:   keyword,
		Ranges:    ranges,
		Commas:    commas,
		Semicolon: semicolon,
	}
}

func NewReservedNamesNode(keyword *KeywordNode, names []StringValueNode, commas []*RuneNode, semicolon *RuneNode) *ReservedNode {
	if keyword == nil {
		panic("keyword is nil")
	}
	if semicolon == nil {
		panic("semicolon is nil")
	}
	if len(names) == 0 {
		panic("must have at least one name")
	}
	if len(commas) != len(names)-1 {
		panic(fmt.Sprintf("%d names requires %d commas, not %d", len(names), len(names)-1, len(commas)))
	}
	children := make([]Node, 0, len(names)*2+1)
	children = append(children, keyword)
	for i, name := range names {
		if i > 0 {
			if commas[i-1] == nil {
				panic(fmt.Sprintf("commas[%d] is nil", i-1))
			}
			children = append(children, commas[i-1])
		}
		if name == nil {
			panic(fmt.Sprintf("names[%d] is nil", i))
		}
		children = append(children, name)
	}
	children = append(children, semicolon)
	return &ReservedNode{
		compositeNode: compositeNode{
			children: children,
		},
		Keyword:   keyword,
		Names:     names,
		Commas:    commas,
		Semicolon: semicolon,
	}
}
