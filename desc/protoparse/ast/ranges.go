package ast

type ExtensionRangeNode struct {
	compositeNode
	Keyword   *KeywordNode
	Ranges    []*RangeNode
	Commas    []*RuneNode
	Options   *CompactOptionsNode
	Semicolon *RuneNode
}

func (e *ExtensionRangeNode) msgElement() {}

type RangeNode struct {
	compositeNode
	StartNode IntValueNode
	To        *IdentNode
	EndNode   IntValueNode
	Max       *KeywordNode
}

func (n *RangeNode) RangeStart() Node {
	return n.StartNode
}

func (n *RangeNode) RangeEnd() Node {
	if n.EndNode == nil {
		return n.StartNode
	}
	return n.EndNode
}

func (n *RangeNode) StartValue() interface{} {
	return n.StartNode.(IntValueNode).Value()
}

func (n *RangeNode) StartValueAsInt32(min, max int32) (int32, bool) {
	return AsInt32(n.StartNode.(IntValueNode), min, max)
}

func (n *RangeNode) EndValue() interface{} {
	l, ok := n.EndNode.(IntValueNode)
	if !ok {
		return nil
	}
	return l.Value()
}

func (n *RangeNode) EndValueAsInt32(min, max int32) (int32, bool) {
	if n.Max != nil {
		return max, true
	}
	if n.EndNode == nil {
		return n.StartValueAsInt32(min, max)
	}
	return AsInt32(n.EndNode.(IntValueNode), min, max)
}

type ReservedNode struct {
	compositeNode
	Keyword   *KeywordNode
	Ranges    []*RangeNode
	Names     []StringValueNode
	Commas    []*RuneNode
	Semicolon *RuneNode
}

func (*ReservedNode) msgElement()  {}
func (*ReservedNode) enumElement() {}
