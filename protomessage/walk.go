package protomessage

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/internal"
)

// Walk traverses the given root messages, iterating through its fields and
// through all values in maps and lists, calling the given action for all
// message values encountered. The given action is called for root first
// before being called for any contained message values.
//
// The path provided to the callback is the sequence of field numbers,
// list indices, and map keys that identifies the location of the given
// message. It is empty when called for the root message. The types of
// values in the slice will be protoreflect.FieldNumber, int (an index
// into a list field), or protoreflect.MapKey (indicating which entry
// in a map field).
//
// If the callback returns false, the traversal is terminated and the
// callback will not be invoked again.
func Walk(root protoreflect.Message, action func(path []any, val protoreflect.Message) bool) {
	walk(root, make([]any, 0, 8), action)
}

func walk(root protoreflect.Message, path []any, action func(path []any, val protoreflect.Message) bool) bool {
	ok := action(path, root)
	root.Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		path = append(path, field.Number())
		switch {
		case field.IsList() && internal.IsMessageKind(field.Kind()):
			listVal := val.List()
			for i, length := 0, listVal.Len(); i < length; i++ {
				path = append(path, i)
				ok = walk(listVal.Get(i).Message(), path, action)
				path = path[:len(path)-1] // pop index
				if !ok {
					break
				}
			}
		case field.IsMap() && internal.IsMessageKind(field.MapValue().Kind()):
			mapVal := val.Map()
			mapVal.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
				path = append(path, key)
				ok = walk(val.Message(), path, action)
				path = path[:len(path)-1] // pop entry key
				return ok
			})
		case !field.IsMap() && internal.IsMessageKind(field.Kind()):
			ok = walk(val.Message(), path, action)
		}
		path = path[:len(path)-1] // pop field number
		return ok
	})
	return ok
}
