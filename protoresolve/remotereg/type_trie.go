package remotereg

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
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

// forEachType calls fn for this node and every node below it in the trie that
// has an element, along with the node's fully-qualified name and its element.
// Nodes without an element are skipped, but their children are still visited.
// The given name is the fully-qualified name of this node (empty for the trie
// root).
func (t *typeTrie) forEachType(name string, fn func(name string, typ proto.Message)) {
	if t.typ != nil {
		fn(name, t.typ)
	}
	for key, child := range t.children {
		child.forEachType(childName(name, key), fn)
	}
}

// forEachMessage calls fn for this node and every node below it in the trie
// that will become a message, along with the node's fully-qualified name and
// whether it will be a placeholder. A node becomes a message if its element is
// a message, or a placeholder message if it has no element. Nodes whose
// elements are enums are skipped, along with their children (since enums
// cannot have children). The given name is the fully-qualified name of this
// node (empty for the trie root).
func (t *typeTrie) forEachMessage(name string, fn func(name string, placeholder bool)) {
	switch t.typ.(type) {
	case nil:
		if name != "" { // the trie root is not a message
			fn(name, true)
		}
	case *typepb.Type, *descriptorpb.DescriptorProto:
		fn(name, false)
	default:
		return
	}
	for key, child := range t.children {
		child.forEachMessage(childName(name, key), fn)
	}
}

// childName returns the fully-qualified name of the child with the given key,
// of the node with the given fully-qualified name (empty for the trie root).
func childName(name, key string) string {
	if name == "" {
		return key
	}
	return name + "." + key
}

// typeToDescriptor converts this level of the trie into a message or enum
// descriptor proto, requiring that the element stored in t.typ is a *ptype.Type
// or *ptype.Enum, or already a message or enum descriptor proto. If t.typ is
// nil, a placeholder message (with no fields) is returned. The trie's children
// are added to the returned message as nested types (see addChildren).
//
// If the value in t.typ is already a *descriptorpb.EnumDescriptorProto then it
// is returned as is. If it is a *descriptorpb.DescriptorProto, it is returned
// as is unless the trie has children, in which case a copy is returned.
func (t *typeTrie) typeToDescriptor(name string, res protoresolve.SerializationResolver) (proto.Message, error) {
	var msg *descriptorpb.DescriptorProto
	switch typ := t.typ.(type) {
	case *descriptorpb.EnumDescriptorProto:
		return typ, nil
	case *typepb.Enum:
		return createEnumDescriptor(typ, res), nil
	case *descriptorpb.DescriptorProto:
		if len(t.children) == 0 {
			return typ, nil
		}
		msg = proto.Clone(typ).(*descriptorpb.DescriptorProto)
	case *typepb.Type:
		msg = createMessageDescriptor(typ, res)
	case nil:
		msg = createIntermediateMessageDescriptor(name)
	default:
		return nil, fmt.Errorf("invalid descriptor trie: unexpected element of type %v", reflect.TypeOf(typ))
	}
	err := t.addChildren(msg, func(child *typeTrie, childName string) (proto.Message, error) {
		return child.typeToDescriptor(childName, res)
	})
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// addChildren converts the trie's children to descriptor protos, using the
// given function, and adds them to msg as nested types. If msg already has a
// nested type with the same name as a child, the child replaces it. But if the
// child is just a placeholder (i.e. has no element), its own children are
// instead added to the existing nested type.
func (t *typeTrie) addChildren(msg *descriptorpb.DescriptorProto, convert func(child *typeTrie, name string) (proto.Message, error)) error {
	// sort children for deterministic output
	for _, name := range slices.Sorted(maps.Keys(t.children)) {
		child := t.children[name]
		if child.typ == nil {
			idx := slices.IndexFunc(msg.NestedType, func(existing *descriptorpb.DescriptorProto) bool {
				return existing.GetName() == name
			})
			if idx != -1 {
				existing := proto.Clone(msg.NestedType[idx]).(*descriptorpb.DescriptorProto)
				if err := child.addChildren(existing, convert); err != nil {
					return err
				}
				msg.NestedType[idx] = existing
				continue
			}
		}
		converted, err := convert(child, name)
		if err != nil {
			return err
		}
		switch converted := converted.(type) {
		case *descriptorpb.DescriptorProto:
			msg.NestedType = replaceOrAppend(msg.NestedType, converted)
		case *descriptorpb.EnumDescriptorProto:
			msg.EnumType = replaceOrAppend(msg.EnumType, converted)
		default:
			return fmt.Errorf("invalid descriptor trie: message cannot have child of type %v", reflect.TypeOf(converted))
		}
	}
	return nil
}

// replaceOrAppend replaces the element in elements with the same name as elem.
// If there is no such element, it appends elem.
func replaceOrAppend[T interface{ GetName() string }](elements []T, elem T) []T {
	idx := slices.IndexFunc(elements, func(existing T) bool {
		return existing.GetName() == elem.GetName()
	})
	if idx == -1 {
		return append(elements, elem)
	}
	elements[idx] = elem
	return elements
}

// rewriteDescriptor converts this level of the trie into a new descriptor
// proto, requiring that the element stored in t.typ is already a service,
// message, or enum descriptor proto. If this trie has children then t.typ must
// be a message descriptor proto (or nil, for a placeholder message). The
// returned descriptor proto is the same as t.typ, but with the trie's children
// added as nested types (see addChildren). The message's existing nested types
// are retained.
func (t *typeTrie) rewriteDescriptor(name string) (proto.Message, error) {
	if len(t.children) == 0 && t.typ != nil {
		return t.typ, nil
	}
	var mdp *descriptorpb.DescriptorProto
	switch typ := t.typ.(type) {
	case nil:
		mdp = createIntermediateMessageDescriptor(name)
	case *descriptorpb.DescriptorProto:
		mdp = proto.Clone(typ).(*descriptorpb.DescriptorProto)
	default:
		return nil, fmt.Errorf("invalid descriptor trie: %v cannot have children", reflect.TypeOf(typ))
	}
	if err := t.addChildren(mdp, (*typeTrie).rewriteDescriptor); err != nil {
		return nil, err
	}
	return mdp, nil
}
