package ast

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
	numChildren := len(ranges)*2 + 1
	if opts != nil {
		numChildren++
	}
	children := make([]Node, 0, numChildren)
	children = append(children, keyword)
	for i, rng := range ranges {
		if i > 0 {
			children = append(children, commas[i-1])
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
	Start IntValueNode
	To    *IdentNode
	End   IntValueNode
	Max   *KeywordNode
}

func NewRangeNode(start IntValueNode, to *IdentNode, end IntValueNode, max *KeywordNode) *RangeNode {
	numChildren := 1
	if to != nil {
		numChildren = 3
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
		Start: start,
		To:    to,
		End:   end,
		Max:   max,
	}
}

func (n *RangeNode) RangeStart() Node {
	return n.Start
}

func (n *RangeNode) RangeEnd() Node {
	if n.End == nil {
		return n.Start
	}
	return n.End
}

func (n *RangeNode) StartValue() interface{} {
	return n.Start.Value()
}

func (n *RangeNode) StartValueAsInt32(min, max int32) (int32, bool) {
	return AsInt32(n.Start, min, max)
}

func (n *RangeNode) EndValue() interface{} {
	if n.End == nil {
		return nil
	}
	return n.End.Value()
}

func (n *RangeNode) EndValueAsInt32(min, max int32) (int32, bool) {
	if n.Max != nil {
		return max, true
	}
	if n.End == nil {
		return n.StartValueAsInt32(min, max)
	}
	return AsInt32(n.End, min, max)
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
	children := make([]Node, 0, len(ranges)*2+1)
	children = append(children, keyword)
	for i, rng := range ranges {
		if i > 0 {
			children = append(children, commas[i-1])
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
	children := make([]Node, 0, len(names)*2+1)
	children = append(children, keyword)
	for i, name := range names {
		if i > 0 {
			children = append(children, commas[i-1])
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
