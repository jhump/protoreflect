package remotereg

import (
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func (dc *DescriptorConverter) options(options proto.Message) []*typepb.Option {
	if options == nil || !options.ProtoReflect().IsValid() {
		// Nil interface or typed-nil message.
		return nil
	}
	var opts []*typepb.Option
	options.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		o := dc.option(fd, val)
		if len(o) > 0 {
			opts = append(opts, o...)
		}
		return true
	})
	// Range above results in non-deterministic ordering of extensions.
	// So sort the options to make the results deterministic.
	sort.SliceStable(opts, func(i, j int) bool {
		iName := opts[i].Name
		jName := opts[j].Name
		iExt := strings.Contains(iName, ".")
		jExt := strings.Contains(jName, ".")
		// Normal options first, custom options/extensions last.
		if iExt != jExt {
			return !iExt
		}
		// Then order by name.
		return iName < jName
	})
	return opts
}

func (dc *DescriptorConverter) option(field protoreflect.FieldDescriptor, value protoreflect.Value) []*typepb.Option {
	switch {
	case field.IsList():
		listVal := value.List()
		opts := make([]*typepb.Option, 0, listVal.Len())
		for i, length := 0, listVal.Len(); i < length; i++ {
			if opt := dc.singleOption(field, listVal.Get(i)); opt != nil {
				opts = append(opts, opt)
			}
		}
		return opts
	case field.IsMap():
		mapVal := value.Map()
		opts := make([]*typepb.Option, 0, mapVal.Len())
		mapVal.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
			entry := dynamicpb.NewMessage(field.Message())
			entry.Set(field.MapKey(), k.Value())
			entry.Set(field.MapValue(), v)
			if opt := dc.singleOption(field, protoreflect.ValueOfMessage(entry)); opt != nil {
				opts = append(opts, opt)
			}
			return true
		})
		return opts
	default:
		if opt := dc.singleOption(field, value); opt != nil {
			return []*typepb.Option{opt}
		}
		return nil
	}
}

func (dc *DescriptorConverter) singleOption(field protoreflect.FieldDescriptor, value protoreflect.Value) *typepb.Option {
	pm := maybeWrap(field.Kind(), value)
	if pm == nil {
		return nil
	}
	var a anypb.Any
	if err := anypb.MarshalFrom(&a, pm, proto.MarshalOptions{}); err != nil {
		return nil
	}
	var name string
	if field.IsExtension() {
		name = string(field.FullName())
	} else {
		name = string(field.Name())
	}
	return &typepb.Option{
		Name:  name,
		Value: &a,
	}
}

func maybeWrap(k protoreflect.Kind, v protoreflect.Value) proto.Message {
	if !v.IsValid() {
		return nil
	}
	if k == protoreflect.MessageKind || k == protoreflect.GroupKind {
		return v.Message().Interface()
	}
	switch k {
	case protoreflect.BoolKind:
		return &wrapperspb.BoolValue{Value: v.Bool()}
	case protoreflect.BytesKind:
		return &wrapperspb.BytesValue{Value: v.Bytes()}
	case protoreflect.StringKind:
		return &wrapperspb.StringValue{Value: v.String()}
	case protoreflect.FloatKind:
		return &wrapperspb.FloatValue{Value: float32(v.Float())}
	case protoreflect.DoubleKind:
		return &wrapperspb.DoubleValue{Value: v.Float()}
	case protoreflect.EnumKind:
		return &wrapperspb.Int32Value{Value: int32(v.Enum())}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return &wrapperspb.Int32Value{Value: int32(v.Int())}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return &wrapperspb.Int64Value{Value: v.Int()}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return &wrapperspb.UInt32Value{Value: uint32(v.Uint())}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return &wrapperspb.UInt64Value{Value: v.Uint()}
	default:
		return nil
	}
}

func processOptions(options []*typepb.Option, optsMsg protoreflect.Message, res protoresolve.SerializationResolver) {
	// these are created "best effort" so entries which are unresolvable
	// (or seemingly invalid) are simply ignored...
	optsDesc := optsMsg.Descriptor()
	fields := optsDesc.Fields()
	for _, o := range options {
		field := fields.ByName(protoreflect.Name(o.Name))
		if field == nil {
			// must be an extension
			extType, err := res.FindExtensionByName(protoreflect.FullName(o.Name))
			if err != nil {
				continue
			}
			field = extType.TypeDescriptor()
			if field.ContainingMessage() != optsDesc {
				continue
			}
		}
		msgValue := newMessageValueForField(optsMsg, field)
		if msgValue == nil {
			continue
		}
		if protoresolve.TypeNameFromURL(o.Value.TypeUrl) != msgValue.Descriptor().FullName() {
			continue
		}
		err := o.Value.UnmarshalTo(msgValue.Interface())
		if err != nil {
			// can't interpret value? skip it
			continue
		}

		if field.IsMap() {
			// Value is a dynamic message representing entry type. So unpack it.
			k := msgValue.Get(field.MapKey()).MapKey()
			v := msgValue.Get(field.MapValue())
			optsMsg.Mutable(field).Map().Set(k, v)
			continue
		}

		var fv protoreflect.Value
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			fv = unwrap(msgValue.Interface(), field.Kind() == protoreflect.EnumKind)
			if !fv.IsValid() {
				// It should not be possible to get here...
				continue
			}
		} else {
			fv = protoreflect.ValueOfMessage(msgValue)
		}
		if field.IsList() {
			optsMsg.Mutable(field).List().Append(fv)
		} else {
			optsMsg.Set(field, fv)
		}
	}
}

func newMessageValueForField(msg protoreflect.Message, field protoreflect.FieldDescriptor) protoreflect.Message {
	switch {
	case field.IsList() && internal.IsMessageKind(field.Kind()):
		return msg.NewField(field).List().NewElement().Message()
	case field.IsMap():
		// For maps, create a dynamic message representing the map entry
		return dynamicpb.NewMessage(field.Message())
	case internal.IsMessageKind(field.Kind()):
		return msg.NewField(field).Message()
	default:
		switch field.Kind() {
		case protoreflect.BoolKind:
			return (&wrapperspb.BoolValue{}).ProtoReflect()
		case protoreflect.FloatKind:
			return (&wrapperspb.FloatValue{}).ProtoReflect()
		case protoreflect.DoubleKind:
			return (&wrapperspb.DoubleValue{}).ProtoReflect()
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.EnumKind:
			return (&wrapperspb.Int32Value{}).ProtoReflect()
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			return (&wrapperspb.Int64Value{}).ProtoReflect()
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			return (&wrapperspb.UInt32Value{}).ProtoReflect()
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			return (&wrapperspb.UInt64Value{}).ProtoReflect()
		case protoreflect.BytesKind:
			return (&wrapperspb.BytesValue{}).ProtoReflect()
		case protoreflect.StringKind:
			return (&wrapperspb.StringValue{}).ProtoReflect()
		}
	}
	return nil
}

func unwrap(msg proto.Message, isEnum bool) protoreflect.Value {
	switch m := msg.(type) {
	case *wrapperspb.BoolValue:
		return protoreflect.ValueOfBool(m.Value)
	case *wrapperspb.FloatValue:
		return protoreflect.ValueOfFloat32(m.Value)
	case *wrapperspb.DoubleValue:
		return protoreflect.ValueOfFloat64(m.Value)
	case *wrapperspb.Int32Value:
		if isEnum {
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(m.Value))
		}
		return protoreflect.ValueOfInt32(m.Value)
	case *wrapperspb.Int64Value:
		return protoreflect.ValueOfInt64(m.Value)
	case *wrapperspb.UInt32Value:
		return protoreflect.ValueOfUint32(m.Value)
	case *wrapperspb.UInt64Value:
		return protoreflect.ValueOfUint64(m.Value)
	case *wrapperspb.BytesValue:
		return protoreflect.ValueOfBytes(m.Value)
	case *wrapperspb.StringValue:
		return protoreflect.ValueOfString(m.Value)
	default:
		return protoreflect.Value{}
	}
}
