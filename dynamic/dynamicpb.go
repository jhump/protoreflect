package dynamic

import (
	protov2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ToDynamicPB returns a copy of this message as a *dynamicpb.Message, which
// is the dynamic message implementation in the google.golang.org/protobuf
// ("v2") API. The returned message can be used with packages in that API that
// do not support this message type, such as protojson and prototext.
//
// Any extensions present in this message, or in messages nested inside it,
// are also present in the returned message. Unknown fields are preserved, but
// note that protojson ignores unknown fields when marshaling.
//
// To use protojson with a message that contains a google.protobuf.Any whose
// value is a dynamic message type, the protojson options must be given a
// resolver that can find that type, such as a *dynamicpb.Types.
//
// To convert a *dynamicpb.Message back to a dynamic message, use ConvertFrom
// or MergeFrom.
func (m *Message) ToDynamicPB() (*dynamicpb.Message, error) {
	dpb := dynamicpb.NewMessage(m.md.UnwrapMessage())
	if err := m.mergeIntoDynamicPB(dpb); err != nil {
		return nil, err
	}
	return dpb, nil
}

func (m *Message) mergeIntoDynamicPB(dpb *dynamicpb.Message) error {
	// Marshaling first also verifies that the message is not too deeply
	// nested (and has no cycles), which makes it safe to then recursively
	// visit it to find extensions.
	data, err := m.Marshal()
	if err != nil {
		return err
	}
	// The extensions must be resolvable when unmarshaling, or else they would
	// be treated as unknown fields. We can't rely on the global registry since
	// the descriptors for dynamic messages are usually not linked into the
	// program.
	var extensions protoregistry.Types
	if err := m.addExtensionTypes(&extensions); err != nil {
		return err
	}
	return protov2.UnmarshalOptions{
		Merge:        true,
		AllowPartial: true,
		Resolver:     extensionResolver{types: &extensions},
	}.Unmarshal(data, dpb)
}

func (m *Message) mergeFromDynamicPB(dpb *dynamicpb.Message) error {
	data, err := protov2.MarshalOptions{AllowPartial: true}.Marshal(dpb)
	if err != nil {
		return err
	}
	return m.UnmarshalMerge(data)
}

// addExtensionTypes adds types for all extensions present in m, and in
// messages nested inside m, to the given registry.
func (m *Message) addExtensionTypes(types *protoregistry.Types) error {
	for tag, v := range m.values {
		fd := m.FindFieldDescriptor(tag)
		if fd.IsExtension() {
			xd := fd.UnwrapField()
			_, errByName := types.FindExtensionByName(xd.FullName())
			_, errByNumber := types.FindExtensionByNumber(xd.ContainingMessage().FullName(), xd.Number())
			if errByName == protoregistry.NotFound && errByNumber == protoregistry.NotFound {
				if err := types.RegisterExtension(dynamicpb.NewExtensionType(xd)); err != nil {
					return err
				}
			}
		}
		if fd.GetMessageType() == nil {
			continue
		}
		switch v := v.(type) {
		case *Message:
			if err := v.addExtensionTypes(types); err != nil {
				return err
			}
		case []interface{}:
			for _, e := range v {
				if dm, ok := e.(*Message); ok {
					if err := dm.addExtensionTypes(types); err != nil {
						return err
					}
				}
			}
		case map[interface{}]interface{}:
			for _, e := range v {
				if dm, ok := e.(*Message); ok {
					if err := dm.addExtensionTypes(types); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// extensionResolver resolves extensions using the given types, falling back
// to the global registry. The fallback is for extensions in nested messages
// that are generated types instead of dynamic messages.
type extensionResolver struct {
	types *protoregistry.Types
}

func (r extensionResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if xt, err := r.types.FindExtensionByName(field); err == nil {
		return xt, nil
	}
	return protoregistry.GlobalTypes.FindExtensionByName(field)
}

func (r extensionResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	if xt, err := r.types.FindExtensionByNumber(message, field); err == nil {
		return xt, nil
	}
	return protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
}
