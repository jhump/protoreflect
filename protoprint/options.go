package protoprint

import (
	"bytes"
	"fmt"
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/internal/fielddefault"
	"github.com/jhump/protoreflect/v2/protomessage"
)

func (p *Printer) printOptionsLong(
	opts []option,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	p.printOptions(opts, w, indent,
		func(i int32) protoreflect.SourceLocation {
			return sourceInfo.ByPath(append(path, i))
		},
		func(w *writer, indent int, opt option, _ bool) {
			p.indent(w, indent)
			_, _ = fmt.Fprint(w, "option ")
			p.printOption(reg, opt.name, opt.val, w, indent)
			_, _ = fmt.Fprint(w, ";")
		},
		false)
}

func (p *Printer) extractAndPrintOptionsShort(
	dsc any,
	optsMsg proto.Message,
	reg *protoregistry.Types,
	optsTag int32,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	d, ok := dsc.(protoreflect.Descriptor)
	if !ok {
		d = dsc.(extensionRangeMarker).owner
	}
	opts := p.extractOptions(d, reg, optsMsg)
	p.printOptionsShort(dsc, opts, optsTag, reg, w, sourceInfo, path, indent)
}

func (p *Printer) printOptionsShort(
	dsc any,
	opts map[protoreflect.FieldNumber][]option,
	optsTag int32,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	elements := elementAddrs{dsc: dsc, opts: opts}
	elements.addrs = optionsAsElementAddrs(optsTag, 0, opts)
	if len(elements.addrs) == 0 {
		return
	}
	p.sort(elements, sourceInfo, path)

	// we render expanded form if there are many options
	count := 0
	for _, addr := range elements.addrs {
		opts := elements.at(addr).([]option)
		count += len(opts)
	}
	threshold := p.ShortOptionsExpansionThresholdCount
	if threshold <= 0 {
		threshold = 3
	}

	if count > threshold {
		p.printOptionElementsShort(elements, reg, w, sourceInfo, path, indent, true)
	} else {
		var tmp bytes.Buffer
		tmpW := *w
		tmpW.Writer = &tmp
		p.printOptionElementsShort(elements, reg, &tmpW, sourceInfo, path, indent, false)
		threshold := p.ShortOptionsExpansionThresholdLength
		if threshold <= 0 {
			threshold = 50
		}
		// we subtract 3 so we don't consider the leading " [" and trailing "]"
		if tmp.Len()-3 > threshold {
			p.printOptionElementsShort(elements, reg, w, sourceInfo, path, indent, true)
		} else {
			// not too long: commit what we rendered
			b := tmp.Bytes()
			if w.space && len(b) > 0 && b[0] == ' ' {
				// don't write extra space
				b = b[1:]
			}
			_, _ = w.Write(b)
			w.newline = tmpW.newline
			w.space = tmpW.space
		}
	}
}

func (p *Printer) printOptionElementsShort(
	addrs elementAddrs,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
	expand bool,
) {
	if expand {
		_, _ = fmt.Fprintln(w, "[")
		indent++
	} else {
		_, _ = fmt.Fprint(w, "[")
	}
	for i, addr := range addrs.addrs {
		opts := addrs.at(addr).([]option)
		var childPath []int32
		if addr.elementIndex < 0 {
			// pseudo-option
			childPath = append(path, int32(-addr.elementIndex))
		} else {
			childPath = append(path, addr.elementType, int32(addr.elementIndex))
		}
		optIndent := indent
		if !expand {
			optIndent = inline(indent)
		}
		p.printOptions(opts, w, optIndent,
			func(i int32) protoreflect.SourceLocation {
				p := childPath
				if addr.elementIndex >= 0 {
					p = append(p, i)
				}
				return sourceInfo.ByPath(p)
			},
			func(w *writer, indent int, opt option, more bool) {
				if expand {
					p.indent(w, indent)
				}
				p.printOption(reg, opt.name, opt.val, w, indent)
				if more {
					if expand {
						_, _ = fmt.Fprintln(w, ",")
					} else {
						_, _ = fmt.Fprint(w, ", ")
					}
				}
			},
			i < len(addrs.addrs)-1)
	}
	if expand {
		p.indent(w, indent-1)
	}
	_, _ = fmt.Fprint(w, "] ")
}

func (p *Printer) printOptions(
	opts []option,
	w *writer,
	indent int,
	siFetch func(i int32) protoreflect.SourceLocation,
	fn func(w *writer, indent int, opt option, more bool),
	haveMore bool,
) {
	for i, opt := range opts {
		more := haveMore
		if !more {
			more = i < len(opts)-1
		}
		si := siFetch(int32(i))
		p.printElement(false, si, w, indent, func(w *writer) {
			fn(w, indent, opt, more)
		})
	}
}

func sortKeys(m protoreflect.Map) []protoreflect.MapKey {
	res := make([]protoreflect.MapKey, m.Len())
	i := 0
	m.Range(func(k protoreflect.MapKey, _ protoreflect.Value) bool {
		res[i] = k
		i++
		return true
	})
	sort.Slice(res, func(i, j int) bool {
		switch i := res[i].Interface().(type) {
		case int32:
			return i < int32(res[j].Int())
		case uint32:
			return i < uint32(res[j].Uint())
		case int64:
			return i < res[j].Int()
		case uint64:
			return i < res[j].Uint()
		case string:
			return i < res[j].String()
		case bool:
			return !i && res[j].Bool()
		default:
			panic(fmt.Sprintf("invalid type for map key: %T", i))
		}
	})
	return res
}

func (p *Printer) printOption(reg *protoregistry.Types, name string, optVal any, w *writer, indent int) {
	_, _ = fmt.Fprintf(w, "%s = ", name)

	switch optVal := optVal.(type) {
	case int32, uint32, int64, uint64:
		_, _ = fmt.Fprintf(w, "%d", optVal)
	case float32:
		_, _ = fmt.Fprint(w, fielddefault.FormatFloat(float64(optVal), 32))
	case float64:
		_, _ = fmt.Fprint(w, fielddefault.FormatFloat(optVal, 64))
	case string:
		_, _ = fmt.Fprintf(w, "%s", quotedString(optVal))
	case []byte:
		_, _ = fmt.Fprintf(w, "%s", quotedBytes(string(optVal)))
	case bool:
		_, _ = fmt.Fprintf(w, "%v", optVal)
	case ident:
		_, _ = fmt.Fprintf(w, "%s", optVal)
	case messageVal:
		threshold := p.MessageLiteralExpansionThresholdLength
		if threshold == 0 {
			threshold = 50
		}
		var buf bytes.Buffer
		p.printMessageLiteralToBufferMaybeCompact(&buf, optVal.msg.ProtoReflect(), reg, optVal.pkg, optVal.scope, threshold, indent)
		_, _ = w.Write(buf.Bytes())

	default:
		panic(fmt.Sprintf("unknown type of value %T for field %s", optVal, name))
	}
}

func (p *Printer) extractOptions(dsc protoreflect.Descriptor, reg *protoregistry.Types, opts proto.Message) map[protoreflect.FieldNumber][]option {
	// The options belong to the descriptor, so we must not modify them.
	opts = proto.CloneOf(opts)
	protomessage.ReparseUnrecognized(opts, reg)

	pkg := dsc.ParentFile().Package()
	var scope protoreflect.FullName
	isMessage := false
	if _, ok := dsc.(protoreflect.FileDescriptor); ok {
		scope = pkg
	} else {
		_, isMessage = dsc.(protoreflect.MessageDescriptor)
		scope = dsc.FullName()
	}

	ref := opts.ProtoReflect()

	options := map[protoreflect.FieldNumber][]option{}
	ref.Range(func(fld protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		var name string
		if fld.IsExtension() {
			var n string
			if isMessage {
				n = p.qualifyMessageOptionName(pkg, scope, fld.FullName())
			} else {
				n = p.qualifyName(pkg, scope, fld.FullName())
			}
			name = fmt.Sprintf("(%s)", n)
		} else {
			name = string(fld.Name())
		}
		opts := valueToOptions(fld, name, val.Interface())
		if len(opts) > 0 {
			for i := range opts {
				if msg, ok := opts[i].val.(proto.Message); ok {
					opts[i].val = messageVal{pkg: pkg, scope: scope, msg: msg}
				}
			}
			options[fld.Number()] = opts
		}
		return true
	})
	return options
}

func valueToOptions(fld protoreflect.FieldDescriptor, name string, val any) []option {
	switch val := val.(type) {
	case protoreflect.List:
		if fld.Number() == internal.UninterpretedOptionsTag {
			// we handle uninterpreted options differently
			uninterp := make([]*descriptorpb.UninterpretedOption, 0, val.Len())
			for i := 0; i < val.Len(); i++ {
				uo := toUninterpretedOption(val.Get(i).Message().Interface())
				if uo != nil {
					uninterp = append(uninterp, uo)
				}
			}
			return uninterpretedToOptions(uninterp)
		}
		opts := make([]option, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			elem := valueForOption(fld, val.Get(i).Interface())
			if elem != nil {
				opts = append(opts, option{name: name, val: elem})
			}
		}
		return opts
	case protoreflect.Map:
		opts := make([]option, 0, val.Len())
		for _, k := range sortKeys(val) {
			v := val.Get(k)
			vf := fld.MapValue()
			if vf.Kind() == protoreflect.EnumKind {
				if vf.Enum().Values().ByNumber(v.Enum()) == nil {
					// have to skip unknown enum values :(
					continue
				}
			}
			entry := dynamicpb.NewMessage(fld.Message())
			entry.Set(fld.Message().Fields().ByNumber(1), k.Value())
			entry.Set(fld.Message().Fields().ByNumber(2), v)
			opts = append(opts, option{name: name, val: entry})
		}
		return opts
	default:
		v := valueForOption(fld, val)
		if v == nil {
			return nil
		}
		return []option{{name: name, val: v}}
	}
}

func valueForOption(fld protoreflect.FieldDescriptor, val any) any {
	switch val := val.(type) {
	case protoreflect.EnumNumber:
		ev := fld.Enum().Values().ByNumber(val)
		if ev == nil {
			// if enum val is unknown, we'll return nil and have to skip it :(
			return nil
		}
		return ident(ev.Name())
	case protoreflect.Message:
		return val.Interface()
	default:
		return val
	}
}

func toUninterpretedOption(message proto.Message) *descriptorpb.UninterpretedOption {
	if uo, ok := message.(*descriptorpb.UninterpretedOption); ok {
		return uo
	}
	// marshal and unmarshal to convert; if we fail to convert, skip it
	var uo descriptorpb.UninterpretedOption
	data, err := proto.Marshal(message)
	if err != nil {
		return nil
	}
	if proto.Unmarshal(data, &uo) != nil {
		return nil
	}
	return &uo
}

func uninterpretedToOptions(uninterp []*descriptorpb.UninterpretedOption) []option {
	opts := make([]option, len(uninterp))
	for i, unint := range uninterp {
		var buf bytes.Buffer
		for ni, n := range unint.Name {
			if ni > 0 {
				buf.WriteByte('.')
			}
			if n.GetIsExtension() {
				_, _ = fmt.Fprintf(&buf, "(%s)", n.GetNamePart())
			} else {
				buf.WriteString(n.GetNamePart())
			}
		}

		var v any
		switch {
		case unint.IdentifierValue != nil:
			v = ident(unint.GetIdentifierValue())
		case unint.StringValue != nil:
			v = string(unint.GetStringValue())
		case unint.DoubleValue != nil:
			v = unint.GetDoubleValue()
		case unint.PositiveIntValue != nil:
			v = unint.GetPositiveIntValue()
		case unint.NegativeIntValue != nil:
			v = unint.GetNegativeIntValue()
		case unint.AggregateValue != nil:
			v = ident("{ " + unint.GetAggregateValue() + " }")
		}

		opts[i] = option{name: buf.String(), val: v}
	}
	return opts
}

func optionsAsElementAddrs(optionsTag int32, order int, opts map[protoreflect.FieldNumber][]option) []elementAddr {
	optAddrs := make([]elementAddr, 0, len(opts))
	for tag := range opts {
		optAddrs = append(optAddrs, elementAddr{elementType: optionsTag, elementIndex: int(tag), order: order})
	}
	// We want stable output. So, if the printer can't sort these a better way,
	// they'll at least be in a deterministic order (by name).
	sort.Sort(optionsByName{addrs: optAddrs, opts: opts})
	return optAddrs
}
