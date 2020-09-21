package protoparse

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"github.com/golang/protobuf/proto"
	protov2 "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/desc/internal"
	"github.com/jhump/protoreflect/desc/protoparse/ast"
)

func (l *linker) interpretFileOptions(r *parseResult) error {
	fd := r.fd
	prefix := fd.GetPackage()
	if prefix != "" {
		prefix += "."
	}
	opts := fd.GetOptions()
	if opts != nil {
		if len(opts.UninterpretedOption) > 0 {
			if remain, err := l.interpretOptions(r, fd.GetName(), fd, opts, opts.UninterpretedOption); err != nil {
				return err
			} else {
				opts.UninterpretedOption = remain
			}
		}
	}
	for _, md := range fd.GetMessageType() {
		fqn := prefix + md.GetName()
		if err := l.interpretMessageOptions(r, fqn, md); err != nil {
			return err
		}
	}
	for _, fld := range fd.GetExtension() {
		fqn := prefix + fld.GetName()
		if err := l.interpretFieldOptions(r, fqn, fld); err != nil {
			return err
		}
	}
	for _, ed := range fd.GetEnumType() {
		fqn := prefix + ed.GetName()
		if err := l.interpretEnumOptions(r, fqn, ed); err != nil {
			return err
		}
	}
	for _, sd := range fd.GetService() {
		fqn := prefix + sd.GetName()
		opts := sd.GetOptions()
		if len(opts.GetUninterpretedOption()) > 0 {
			if remain, err := l.interpretOptions(r, fqn, sd, opts, opts.UninterpretedOption); err != nil {
				return err
			} else {
				opts.UninterpretedOption = remain
			}
		}
		for _, mtd := range sd.GetMethod() {
			mtdFqn := fqn + "." + mtd.GetName()
			opts := mtd.GetOptions()
			if len(opts.GetUninterpretedOption()) > 0 {
				if remain, err := l.interpretOptions(r, mtdFqn, mtd, opts, opts.UninterpretedOption); err != nil {
					return err
				} else {
					opts.UninterpretedOption = remain
				}
			}
		}
	}
	return nil
}

func (l *linker) interpretMessageOptions(r *parseResult, fqn string, md *descriptorpb.DescriptorProto) error {
	opts := md.GetOptions()
	if opts != nil {
		if len(opts.UninterpretedOption) > 0 {
			if remain, err := l.interpretOptions(r, fqn, md, opts, opts.UninterpretedOption); err != nil {
				return err
			} else {
				opts.UninterpretedOption = remain
			}
		}
	}
	for _, fld := range md.GetField() {
		fldFqn := fqn + "." + fld.GetName()
		if err := l.interpretFieldOptions(r, fldFqn, fld); err != nil {
			return err
		}
	}
	for _, ood := range md.GetOneofDecl() {
		oodFqn := fqn + "." + ood.GetName()
		opts := ood.GetOptions()
		if len(opts.GetUninterpretedOption()) > 0 {
			if remain, err := l.interpretOptions(r, oodFqn, ood, opts, opts.UninterpretedOption); err != nil {
				return err
			} else {
				opts.UninterpretedOption = remain
			}
		}
	}
	for _, fld := range md.GetExtension() {
		fldFqn := fqn + "." + fld.GetName()
		if err := l.interpretFieldOptions(r, fldFqn, fld); err != nil {
			return err
		}
	}
	for _, er := range md.GetExtensionRange() {
		erFqn := fmt.Sprintf("%s.%d-%d", fqn, er.GetStart(), er.GetEnd())
		opts := er.GetOptions()
		if len(opts.GetUninterpretedOption()) > 0 {
			if remain, err := l.interpretOptions(r, erFqn, er, opts, opts.UninterpretedOption); err != nil {
				return err
			} else {
				opts.UninterpretedOption = remain
			}
		}
	}
	for _, nmd := range md.GetNestedType() {
		nmdFqn := fqn + "." + nmd.GetName()
		if err := l.interpretMessageOptions(r, nmdFqn, nmd); err != nil {
			return err
		}
	}
	for _, ed := range md.GetEnumType() {
		edFqn := fqn + "." + ed.GetName()
		if err := l.interpretEnumOptions(r, edFqn, ed); err != nil {
			return err
		}
	}
	return nil
}

func (l *linker) interpretFieldOptions(r *parseResult, fqn string, fld *descriptorpb.FieldDescriptorProto) error {
	opts := fld.GetOptions()
	if len(opts.GetUninterpretedOption()) > 0 {
		uo := opts.UninterpretedOption
		scope := fmt.Sprintf("field %s", fqn)

		// process json_name pseudo-option
		if index, err := findOption(r, scope, uo, "json_name"); err != nil && !r.lenient {
			return err
		} else if index >= 0 {
			opt := uo[index]
			optNode := r.getOptionNode(opt)

			// attribute source code info
			if on, ok := optNode.(*ast.OptionNode); ok {
				r.interpretedOptions[on] = []int32{-1, internal.Field_jsonNameTag}
			}
			uo = removeOption(uo, index)
			if opt.StringValue == nil {
				if err := r.errs.handleErrorWithPos(optNode.GetValue().Start(), "%s: expecting string value for json_name option", scope); err != nil {
					return err
				}
			} else {
				fld.JsonName = proto.String(string(opt.StringValue))
			}
		}

		// and process default pseudo-option
		if index, err := l.processDefaultOption(r, scope, fqn, fld, uo); err != nil && !r.lenient {
			return err
		} else if index >= 0 {
			// attribute source code info
			optNode := r.getOptionNode(uo[index])
			if on, ok := optNode.(*ast.OptionNode); ok {
				r.interpretedOptions[on] = []int32{-1, internal.Field_defaultTag}
			}
			uo = removeOption(uo, index)
		}

		if len(uo) == 0 {
			// no real options, only pseudo-options above? clear out options
			fld.Options = nil
		} else if remain, err := l.interpretOptions(r, fqn, fld, opts, uo); err != nil {
			return err
		} else {
			opts.UninterpretedOption = remain
		}
	}
	return nil
}

func (l *linker) processDefaultOption(res *parseResult, scope string, fqn string, fld *descriptorpb.FieldDescriptorProto, uos []*descriptorpb.UninterpretedOption) (defaultIndex int, err error) {
	found, err := findOption(res, scope, uos, "default")
	if err != nil || found == -1 {
		return -1, err
	}
	opt := uos[found]
	optNode := res.getOptionNode(opt)
	if fld.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return -1, res.errs.handleErrorWithPos(optNode.GetName().Start(), "%s: default value cannot be set because field is repeated", scope)
	}
	if fld.GetType() == descriptorpb.FieldDescriptorProto_TYPE_GROUP || fld.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		return -1, res.errs.handleErrorWithPos(optNode.GetName().Start(), "%s: default value cannot be set because field is a message", scope)
	}
	val := optNode.GetValue()
	if _, ok := val.(*ast.MessageLiteralNode); ok {
		return -1, res.errs.handleErrorWithPos(val.Start(), "%s: default value cannot be a message", scope)
	}
	mc := &messageContext{
		res:         res,
		file:        res.fd,
		elementName: fqn,
		elementType: descriptorType(fld),
		option:      opt,
	}
	var v interface{}
	if fld.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM {
		ed := l.findEnumType(res.fd, fld.GetTypeName())
		ev, err := l.enumFieldValue(mc, ed, val)
		if err != nil {
			return -1, res.errs.handleError(err)
		}
		v = string(ev.Name())
	} else {
		v, err = l.scalarFieldValue(mc, fld.GetType(), val)
		if err != nil {
			return -1, res.errs.handleError(err)
		}
	}
	if str, ok := v.(string); ok {
		fld.DefaultValue = proto.String(str)
	} else if b, ok := v.([]byte); ok {
		fld.DefaultValue = proto.String(encodeDefaultBytes(b))
	} else {
		var flt float64
		var ok bool
		if flt, ok = v.(float64); !ok {
			var flt32 float32
			if flt32, ok = v.(float32); ok {
				flt = float64(flt32)
			}
		}
		if ok {
			if math.IsInf(flt, 1) {
				fld.DefaultValue = proto.String("inf")
			} else if ok && math.IsInf(flt, -1) {
				fld.DefaultValue = proto.String("-inf")
			} else if ok && math.IsNaN(flt) {
				fld.DefaultValue = proto.String("nan")
			} else {
				fld.DefaultValue = proto.String(fmt.Sprintf("%v", v))
			}
		} else {
			fld.DefaultValue = proto.String(fmt.Sprintf("%v", v))
		}
	}
	return found, nil
}

func encodeDefaultBytes(b []byte) string {
	var buf bytes.Buffer
	writeEscapedBytes(&buf, b)
	return buf.String()
}

func (l *linker) interpretEnumOptions(r *parseResult, fqn string, ed *descriptorpb.EnumDescriptorProto) error {
	opts := ed.GetOptions()
	if opts != nil {
		if len(opts.UninterpretedOption) > 0 {
			if remain, err := l.interpretOptions(r, fqn, ed, opts, opts.UninterpretedOption); err != nil {
				return err
			} else {
				opts.UninterpretedOption = remain
			}
		}
	}
	for _, evd := range ed.GetValue() {
		evdFqn := fqn + "." + evd.GetName()
		opts := evd.GetOptions()
		if len(opts.GetUninterpretedOption()) > 0 {
			if remain, err := l.interpretOptions(r, evdFqn, evd, opts, opts.UninterpretedOption); err != nil {
				return err
			} else {
				opts.UninterpretedOption = remain
			}
		}
	}
	return nil
}

func (l *linker) interpretOptions(res *parseResult, fqn string, element, opts proto.Message, uninterpreted []*descriptorpb.UninterpretedOption) ([]*descriptorpb.UninterpretedOption, error) {
	var md protoreflect.MessageDescriptor
	optsFqn := string(proto.MessageReflect(opts).Descriptor().FullName())
	// see if the parse included an override copy for these options
	for _, symbols := range l.descriptorPool {
		if _, ok := symbols[optsFqn]; ok {
			// it did! use that descriptor instead
			md = l.findMessageType(res.fd, optsFqn)
			if md != nil {
				break
			}
		}
	}
	var msg protoreflect.Message
	if md != nil {
		dm := dynamicpb.NewMessage(md)
		if err := cloneInto(dm, proto.MessageV2(opts)); err != nil {
			node := res.nodes[element]
			return nil, res.errs.handleError(ErrorWithSourcePos{Pos: node.Start(), Underlying: err})
		}
		msg = dm
	} else {
		msg = proto.MessageReflect(proto.Clone(opts))
	}

	mc := &messageContext{res: res, file: res.fd, elementName: fqn, elementType: descriptorType(element)}
	var remain []*descriptorpb.UninterpretedOption
	for _, uo := range uninterpreted {
		node := res.getOptionNode(uo)
		if !uo.Name[0].GetIsExtension() && uo.Name[0].GetNamePart() == "uninterpreted_option" {
			if res.lenient {
				remain = append(remain, uo)
				continue
			}
			// uninterpreted_option might be found reflectively, but is not actually valid for use
			if err := res.errs.handleErrorWithPos(node.GetName().Start(), "%vinvalid option 'uninterpreted_option'", mc); err != nil {
				return nil, err
			}
		}
		mc.option = uo
		path, err := l.interpretField(res, mc, element, msg, uo, 0, nil)
		if err != nil {
			if res.lenient {
				remain = append(remain, uo)
				continue
			}
			return nil, err
		}
		if optn, ok := node.(*ast.OptionNode); ok {
			res.interpretedOptions[optn] = path
		}
	}

	if res.lenient {
		// If we're lenient, then we don't want to clobber the passed in message
		// and leave it partially populated. So we convert into a copy first
		optsClone := proto.Clone(opts)
		if err := cloneInto(optsClone, msg.Interface()); err != nil {
			// TODO: do this in a more granular way, so we can convert individual
			// fields and leave bad ones uninterpreted instead of skipping all of
			// the work we've done so far.
			return uninterpreted, nil
		}
		// conversion from dynamic message above worked, so now
		// it is safe to overwrite the passed in message
		opts.Reset()
		proto.Merge(opts, optsClone)

		return remain, nil
	}

	if err := validateRecursive(msg, ""); err != nil {
		node := res.nodes[element]
		if err := res.errs.handleErrorWithPos(node.Start(), "error in %s options: %v", descriptorType(element), err); err != nil {
			return nil, err
		}
	}

	// now try to convert into the passed in message and fail if not successful
	if err := cloneInto(opts, msg.Interface()); err != nil {
		node := res.nodes[element]
		return nil, res.errs.handleError(ErrorWithSourcePos{Pos: node.Start(), Underlying: err})
	}

	return nil, nil
}

func cloneInto(dest proto.Message, src protov2.Message) error {
	dest.Reset()
	destV2 := proto.MessageV2(dest)
	if destV2.ProtoReflect().Descriptor() == src.ProtoReflect().Descriptor() {
		protov2.Merge(destV2, src)
		if err := protov2.CheckInitialized(destV2); err != nil {
			return err
		}
		return nil
	}
	// different descriptors means we must serialize
	// and de-serialize in order to merge values
	data, err := protov2.Marshal(src)
	if err != nil {
		return err
	}
	return protov2.Unmarshal(data, destV2)
}

func validateRecursive(msg protoreflect.Message, prefix string) error {
	flds := msg.Descriptor().Fields()
	var missingFields []string
	for i := 0; i < flds.Len(); i++ {
		fld := flds.Get(i)
		if fld.Cardinality() == protoreflect.Required && !msg.Has(fld) {
			missingFields = append(missingFields, fmt.Sprintf("%s%s", prefix, fld.Name()))
		}
	}
	if len(missingFields) > 0 {
		return fmt.Errorf("some required fields missing: %v", strings.Join(missingFields, ", "))
	}

	var err error
	msg.Range(func(fld protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		if fld.IsMap() {
			md := fld.MapValue().Message()
			if md != nil {
				val.Map().Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
					chprefix := fmt.Sprintf("%s%s[%v].", prefix, fieldName(fld), k)
					err = validateRecursive(v.Message(), chprefix)
					if err != nil {
						return false
					}
					return true
				})
				if err != nil {
					return false
				}
			}
		} else {
			md := fld.Message()
			if md != nil {
				if fld.IsList() {
					sl := val.List()
					for i := 0; i < sl.Len(); i++ {
						v := sl.Get(i)
						chprefix := fmt.Sprintf("%s%s[%d].", prefix, fieldName(fld), i)
						err = validateRecursive(v.Message(), chprefix)
						if err != nil {
							return false
						}
					}
				} else {
					chprefix := fmt.Sprintf("%s%s.", prefix, fieldName(fld))
					err = validateRecursive(val.Message(), chprefix)
					if err != nil {
						return false
					}
				}
			}
		}
		return true
	})
	return err
}

func (l *linker) interpretField(res *parseResult, mc *messageContext, element proto.Message, msg protoreflect.Message, opt *descriptorpb.UninterpretedOption, nameIndex int, pathPrefix []int32) (path []int32, err error) {
	var fld protoreflect.FieldDescriptor
	nm := opt.GetName()[nameIndex]
	node := res.getOptionNamePartNode(nm)
	if nm.GetIsExtension() {
		extName := nm.GetNamePart()
		if extName[0] == '.' {
			extName = extName[1:] /* skip leading dot */
		}
		fld = l.findExtension(res.fd, extName)
		if fld == nil {
			return nil, res.errs.handleErrorWithPos(node.Start(),
				"%vunrecognized extension %s of %s",
				mc, extName, msg.Descriptor().FullName())
		}
		if fld.ContainingMessage().FullName() != msg.Descriptor().FullName() {
			return nil, res.errs.handleErrorWithPos(node.Start(),
				"%vextension %s should extend %s but instead extends %s",
				mc, extName, msg.Descriptor().FullName(), fld.ContainingMessage().FullName())
		}
	} else {
		fld = msg.Descriptor().Fields().ByName(protoreflect.Name(nm.GetNamePart()))
		if fld == nil {
			return nil, res.errs.handleErrorWithPos(node.Start(),
				"%vfield %s of %s does not exist",
				mc, nm.GetNamePart(), msg.Descriptor().FullName())
		}
	}

	path = append(pathPrefix, int32(fld.Number()))

	if len(opt.GetName()) > nameIndex+1 {
		nextnm := opt.GetName()[nameIndex+1]
		nextnode := res.getOptionNamePartNode(nextnm)
		k := fld.Kind()
		if k != protoreflect.MessageKind && k != protoreflect.GroupKind {
			return nil, res.errs.handleErrorWithPos(nextnode.Start(),
				"%vcannot set field %s because %s is not a message",
				mc, nextnm.GetNamePart(), nm.GetNamePart())
		}
		if fld.Cardinality() == protoreflect.Repeated {
			return nil, res.errs.handleErrorWithPos(nextnode.Start(),
				"%vcannot set field %s because %s is repeated (must use an aggregate)",
				mc, nextnm.GetNamePart(), nm.GetNamePart())
		}
		var fdm protoreflect.Message
		if msg.Has(fld) {
			v := msg.Mutable(fld)
			fdm = v.Message()
		} else {
			fdm = dynamicpb.NewMessage(fld.Message())
			msg.Set(fld, protoreflect.ValueOfMessage(fdm))
		}
		// recurse to set next part of name
		return l.interpretField(res, mc, element, fdm, opt, nameIndex+1, path)
	}

	optNode := res.getOptionNode(opt)
	if err := l.setOptionField(res, mc, msg, fld, node, optNode.GetValue()); err != nil {
		return nil, res.errs.handleError(err)
	}
	if fld.IsMap() {
		path = append(path, int32(msg.Get(fld).Map().Len())-1)
	} else if fld.IsList() {
		path = append(path, int32(msg.Get(fld).List().Len())-1)
	}
	return path, nil
}

func (l *linker) setOptionField(res *parseResult, mc *messageContext, msg protoreflect.Message, fld protoreflect.FieldDescriptor, name ast.Node, val ast.ValueNode) error {
	v := val.Value()
	if sl, ok := v.([]ast.ValueNode); ok {
		// handle slices a little differently than the others
		if fld.Cardinality() != protoreflect.Repeated {
			return errorWithPos(val.Start(), "%vvalue is an array but field is not repeated", mc)
		}
		origPath := mc.optAggPath
		defer func() {
			mc.optAggPath = origPath
		}()
		for index, item := range sl {
			mc.optAggPath = fmt.Sprintf("%s[%d]", origPath, index)
			value, err := l.fieldValue(res, mc, fld, item)
			if err != nil {
				return err
			}
			if fld.IsMap() {
				entry := value.Message()
				key := entry.Get(fld.MapKey()).MapKey()
				val := entry.Get(fld.MapValue())
				if dm, ok := val.Interface().(*dynamicpb.Message); ok && (dm == nil || !dm.IsValid()) {
					val = protoreflect.ValueOfMessage(dynamicpb.NewMessage(fld.MapValue().Message()))
				}
				msg.Mutable(fld).Map().Set(key, val)
			} else {
				msg.Mutable(fld).List().Append(value)
			}
		}
		return nil
	}

	value, err := l.fieldValue(res, mc, fld, val)
	if err != nil {
		return err
	}
	if fld.IsMap() {
		entry := value.Message()
		key := entry.Get(fld.MapKey()).MapKey()
		val := entry.Get(fld.MapValue())
		if dm, ok := val.Interface().(*dynamicpb.Message); ok && (dm == nil || !dm.IsValid()) {
			val = protoreflect.ValueOfMessage(dynamicpb.NewMessage(fld.MapValue().Message()))
		}
		msg.Mutable(fld).Map().Set(key, val)
	} else if fld.IsList() {
		msg.Mutable(fld).List().Append(value)
	} else {
		if msg.Has(fld) {
			return errorWithPos(name.Start(), "%vnon-repeated option field %s already set", mc, fieldName(fld))
		}
		msg.Set(fld, value)
	}

	return nil
}

func findOption(res *parseResult, scope string, opts []*descriptorpb.UninterpretedOption, name string) (int, error) {
	found := -1
	for i, opt := range opts {
		if len(opt.Name) != 1 {
			continue
		}
		if opt.Name[0].GetIsExtension() || opt.Name[0].GetNamePart() != name {
			continue
		}
		if found >= 0 {
			optNode := res.getOptionNode(opt)
			return -1, res.errs.handleErrorWithPos(optNode.GetName().Start(), "%s: option %s cannot be defined more than once", scope, name)
		}
		found = i
	}
	return found, nil
}

func removeOption(uo []*descriptorpb.UninterpretedOption, indexToRemove int) []*descriptorpb.UninterpretedOption {
	if indexToRemove == 0 {
		return uo[1:]
	} else if indexToRemove == len(uo)-1 {
		return uo[:len(uo)-1]
	} else {
		return append(uo[:indexToRemove], uo[indexToRemove+1:]...)
	}
}

type messageContext struct {
	res         *parseResult
	file        *descriptorpb.FileDescriptorProto
	elementType string
	elementName string
	option      *descriptorpb.UninterpretedOption
	optAggPath  string
}

func (c *messageContext) String() string {
	var ctx bytes.Buffer
	if c.elementType != "file" {
		_, _ = fmt.Fprintf(&ctx, "%s %s: ", c.elementType, c.elementName)
	}
	if c.option != nil && c.option.Name != nil {
		ctx.WriteString("option ")
		writeOptionName(&ctx, c.option.Name)
		if c.res.nodes == nil {
			// if we have no source position info, try to provide as much context
			// as possible (if nodes != nil, we don't need this because any errors
			// will actually have file and line numbers)
			if c.optAggPath != "" {
				_, _ = fmt.Fprintf(&ctx, " at %s", c.optAggPath)
			}
		}
		ctx.WriteString(": ")
	}
	return ctx.String()
}

func writeOptionName(buf *bytes.Buffer, parts []*descriptorpb.UninterpretedOption_NamePart) {
	first := true
	for _, p := range parts {
		if first {
			first = false
		} else {
			buf.WriteByte('.')
		}
		nm := p.GetNamePart()
		if nm[0] == '.' {
			// skip leading dot
			nm = nm[1:]
		}
		if p.GetIsExtension() {
			buf.WriteByte('(')
			buf.WriteString(nm)
			buf.WriteByte(')')
		} else {
			buf.WriteString(nm)
		}
	}
}

func fieldName(fld protoreflect.FieldDescriptor) string {
	if fld.IsExtension() {
		return fmt.Sprintf("(%s)", fld.FullName())
	} else {
		return string(fld.Name())
	}
}

func valueKind(val interface{}) string {
	switch val := val.(type) {
	case ast.Identifier:
		return "identifier"
	case bool:
		return "bool"
	case int64:
		if val < 0 {
			return "negative integer"
		}
		return "integer"
	case uint64:
		return "integer"
	case float64:
		return "double"
	case string, []byte:
		return "string"
	case []*ast.MessageFieldNode:
		return "message"
	case []ast.ValueNode:
		return "array"
	default:
		return fmt.Sprintf("%T", val)
	}
}

func (l *linker) fieldValue(res *parseResult, mc *messageContext, fld protoreflect.FieldDescriptor, val ast.ValueNode) (protoreflect.Value, error) {
	k := fld.Kind()
	switch k {
	case protoreflect.EnumKind:
		evd, err := l.enumFieldValue(mc, fld.Enum(), val)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfEnum(evd.Number()), nil

	case protoreflect.MessageKind, protoreflect.GroupKind:
		v := val.Value()
		if aggs, ok := v.([]*ast.MessageFieldNode); ok {
			fmd := fld.Message()
			fdm := dynamicpb.NewMessage(fmd)
			origPath := mc.optAggPath
			defer func() {
				mc.optAggPath = origPath
			}()
			for _, a := range aggs {
				if origPath == "" {
					mc.optAggPath = a.Name.Value()
				} else {
					mc.optAggPath = origPath + "." + a.Name.Value()
				}
				var ffld protoreflect.FieldDescriptor
				if a.Name.IsExtension() {
					n := string(a.Name.Name.AsIdentifier())
					ffld = l.findExtension(res.fd, n)
					if ffld == nil {
						// may need to qualify with package name
						pkg := mc.file.GetPackage()
						if pkg != "" {
							ffld = l.findExtension(res.fd, pkg+"."+n)
						}
					}
				} else {
					ffld = fmd.Fields().ByName(protoreflect.Name(a.Name.Value()))
				}
				if ffld == nil {
					return protoreflect.Value{}, errorWithPos(val.Start(), "%vfield %s not found", mc, string(a.Name.Name.AsIdentifier()))
				}
				if err := l.setOptionField(res, mc, fdm, ffld, a.Name, a.Val); err != nil {
					return protoreflect.Value{}, err
				}
			}
			return protoreflect.ValueOfMessage(fdm), nil
		}
		return protoreflect.Value{}, errorWithPos(val.Start(), "%vexpecting message, got %s", mc, valueKind(v))

	default:
		v, err := l.scalarFieldValue(mc, descriptorpb.FieldDescriptorProto_Type(k), val)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOf(v), nil
	}
}

func (l *linker) enumFieldValue(mc *messageContext, ed protoreflect.EnumDescriptor, val ast.ValueNode) (protoreflect.EnumValueDescriptor, error) {
	v := val.Value()
	if id, ok := v.(ast.Identifier); ok {
		ev := ed.Values().ByName(protoreflect.Name(id))
		if ev == nil {
			return nil, errorWithPos(val.Start(), "%venum %s has no value named %s", mc, ed.FullName(), id)
		}
		return ev, nil
	}
	return nil, errorWithPos(val.Start(), "%vexpecting enum, got %s", mc, valueKind(v))
}

func (l *linker) scalarFieldValue(mc *messageContext, fldType descriptorpb.FieldDescriptorProto_Type, val ast.ValueNode) (interface{}, error) {
	v := val.Value()
	switch fldType {
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		if b, ok := v.(bool); ok {
			return b, nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting bool, got %s", mc, valueKind(v))
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		if str, ok := v.(string); ok {
			return []byte(str), nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting bytes, got %s", mc, valueKind(v))
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		if str, ok := v.(string); ok {
			return str, nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting string, got %s", mc, valueKind(v))
	case descriptorpb.FieldDescriptorProto_TYPE_INT32, descriptorpb.FieldDescriptorProto_TYPE_SINT32, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		if i, ok := v.(int64); ok {
			if i > math.MaxInt32 || i < math.MinInt32 {
				return nil, errorWithPos(val.Start(), "%vvalue %d is out of range for int32", mc, i)
			}
			return int32(i), nil
		}
		if ui, ok := v.(uint64); ok {
			if ui > math.MaxInt32 {
				return nil, errorWithPos(val.Start(), "%vvalue %d is out of range for int32", mc, ui)
			}
			return int32(ui), nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting int32, got %s", mc, valueKind(v))
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32, descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		if i, ok := v.(int64); ok {
			if i > math.MaxUint32 || i < 0 {
				return nil, errorWithPos(val.Start(), "%vvalue %d is out of range for uint32", mc, i)
			}
			return uint32(i), nil
		}
		if ui, ok := v.(uint64); ok {
			if ui > math.MaxUint32 {
				return nil, errorWithPos(val.Start(), "%vvalue %d is out of range for uint32", mc, ui)
			}
			return uint32(ui), nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting uint32, got %s", mc, valueKind(v))
	case descriptorpb.FieldDescriptorProto_TYPE_INT64, descriptorpb.FieldDescriptorProto_TYPE_SINT64, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		if i, ok := v.(int64); ok {
			return i, nil
		}
		if ui, ok := v.(uint64); ok {
			if ui > math.MaxInt64 {
				return nil, errorWithPos(val.Start(), "%vvalue %d is out of range for int64", mc, ui)
			}
			return int64(ui), nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting int64, got %s", mc, valueKind(v))
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64, descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		if i, ok := v.(int64); ok {
			if i < 0 {
				return nil, errorWithPos(val.Start(), "%vvalue %d is out of range for uint64", mc, i)
			}
			return uint64(i), nil
		}
		if ui, ok := v.(uint64); ok {
			return ui, nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting uint64, got %s", mc, valueKind(v))
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		if d, ok := v.(float64); ok {
			return d, nil
		}
		if i, ok := v.(int64); ok {
			return float64(i), nil
		}
		if u, ok := v.(uint64); ok {
			return float64(u), nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting double, got %s", mc, valueKind(v))
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		if d, ok := v.(float64); ok {
			if (d > math.MaxFloat32 || d < -math.MaxFloat32) && !math.IsInf(d, 1) && !math.IsInf(d, -1) && !math.IsNaN(d) {
				return nil, errorWithPos(val.Start(), "%vvalue %f is out of range for float", mc, d)
			}
			return float32(d), nil
		}
		if i, ok := v.(int64); ok {
			return float32(i), nil
		}
		if u, ok := v.(uint64); ok {
			return float32(u), nil
		}
		return nil, errorWithPos(val.Start(), "%vexpecting float, got %s", mc, valueKind(v))
	default:
		return nil, errorWithPos(val.Start(), "%vunrecognized field type: %s", mc, fldType)
	}
}
