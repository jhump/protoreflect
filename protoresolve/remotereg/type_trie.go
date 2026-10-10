package remotereg

import (
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/typepb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// typeTrie is a prefix trie where each key component is part of a fully-qualified type name. So key components
// will either be package name components or element names.
type typeTrie struct {
	// successor key components
	children map[string]*typeTrie
	// if non-nil, the element whose fully-qualified name is the path from the trie root to this node
	typ proto.Message
}

// addType recursively adds an element to the trie.
func (t *typeTrie) addType(key string, typ proto.Message) {
	if key == "" {
		t.typ = typ
		return
	}
	if t.children == nil {
		t.children = map[string]*typeTrie{}
	}
	curr, rest, _ := strings.Cut(key, ".")
	child := t.children[curr]
	if child == nil {
		child = &typeTrie{}
		t.children[curr] = child
	}
	child.addType(rest, typ)
}

// typeToDescriptor converts this level of the trie into a message or enum
// descriptor proto, requiring that the element stored in t.typ is a *ptype.Type
// or *ptype.Enum. If t.typ is nil, a placeholder message (with no fields) is
// returned that contains the trie's children as nested message and/or enum
// types.
//
// If the value in t.typ is already a *descriptor.DescriptorProto or a
// *descriptor.EnumDescriptorProto then it is returned as is. This function
// should not be used in type tries that may have service descriptors. That will
// result in a panic.
func (t *typeTrie) typeToDescriptor(name string, res protoresolve.SerializationResolver) (*descriptorpb.DescriptorProto, *descriptorpb.EnumDescriptorProto) {
	switch typ := t.typ.(type) {
	case *descriptorpb.EnumDescriptorProto:
		return nil, typ
	case *typepb.Enum:
		return nil, createEnumDescriptor(typ, res)
	case *descriptorpb.DescriptorProto:
		return typ, nil
	default:
		var msg *descriptorpb.DescriptorProto
		if t.typ == nil {
			msg = createIntermediateMessageDescriptor(name)
		} else {
			msg = createMessageDescriptor(t.typ.(*typepb.Type), res)
		}
		// sort children for deterministic output
		var keys []string
		for k := range t.children {
			keys = append(keys, k)
		}
		for _, name := range keys {
			nested := t.children[name]
			chMsg, chEnum := nested.typeToDescriptor(name, res)
			if chMsg != nil {
				msg.NestedType = append(msg.NestedType, chMsg)
			}
			if chEnum != nil {
				msg.EnumType = append(msg.EnumType, chEnum)
			}
		}
		return msg, nil
	}
}

// rewriteDescriptor converts this level of the trie into a new descriptor
// proto, requiring that the element stored in t.type is already a service,
// message, or enum descriptor proto. If this trie has children then t.typ must
// be a message descriptor proto. The returned descriptor proto is the same as
// .type but with possibly new nested elements to represent this trie node's
// children.
func (t *typeTrie) rewriteDescriptor(name string) (proto.Message, error) {
	if len(t.children) == 0 && t.typ != nil {
		if mdp, ok := t.typ.(*descriptorpb.DescriptorProto); ok {
			if len(mdp.NestedType) == 0 && len(mdp.EnumType) == 0 {
				return mdp, nil
			}
			mdp = proto.Clone(mdp).(*descriptorpb.DescriptorProto)
			mdp.NestedType = nil
			mdp.EnumType = nil
			return mdp, nil
		}
		return t.typ, nil
	}
	var mdp *descriptorpb.DescriptorProto
	if t.typ == nil {
		mdp = createIntermediateMessageDescriptor(name)
	} else {
		mdp = t.typ.(*descriptorpb.DescriptorProto)
		mdp = proto.Clone(mdp).(*descriptorpb.DescriptorProto)
		mdp.NestedType = nil
		mdp.EnumType = nil
	}
	// sort children for deterministic output
	var keys []string
	for k := range t.children {
		keys = append(keys, k)
	}
	for _, n := range keys {
		ch := t.children[n]
		typ, err := ch.rewriteDescriptor(n)
		if err != nil {
			return nil, err
		}
		switch typ := typ.(type) {
		case (*descriptorpb.DescriptorProto):
			mdp.NestedType = append(mdp.NestedType, typ)
		case (*descriptorpb.EnumDescriptorProto):
			mdp.EnumType = append(mdp.EnumType, typ)
		default:
			// TODO: this should probably panic instead
			return nil, fmt.Errorf("invalid descriptor trie: message cannot have child of type %v", reflect.TypeOf(typ))
		}
	}
	return mdp, nil
}
