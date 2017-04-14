package dynamic

// JSON marshalling and unmarshalling for dynamic messages

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
)

type MarshalJSONOptions struct {
	Indent       bool
	EmitDefaults bool
}

func (m *Message) MarshalJSONWithOptions(opts MarshalJSONOptions) ([]byte, error) {
	var b indentBuffer
	if !opts.Indent {
		b.indent = -1
	}
	b.comma = true
	if err := m.marshalJSON(&b, opts); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (m *Message) MarshalJSON() ([]byte, error) {
	b, err := m.MarshalJSONWithOptions(MarshalJSONOptions{})
	return b, err
}

func (m *Message) MarshalJSONIndent() ([]byte, error) {
	b, err := m.MarshalJSONWithOptions(MarshalJSONOptions{Indent: true})
	return b, err
}

func (m *Message) marshalJSON(b *indentBuffer, opts MarshalJSONOptions) error {
	err := b.WriteByte('{')
	if err != nil {
		return err
	}
	err = b.start()
	if err != nil {
		return err
	}

	emitDefaults := opts.EmitDefaults

	var tags []int
	if emitDefaults {
		tags = m.allKnownFieldTags()
	} else {
		tags = m.knownFieldTags()
	}

	first := true

	// first the known fields
	for _, tag := range tags {
		itag := int32(tag)
		fd := m.FindFieldDescriptor(itag)

		v, ok := m.values[itag]
		if !ok {
			v = fd.GetDefaultValue()
		}

		err := b.maybeNext(&first)
		if err != nil {
			return err
		}
		err = marshalKnownFieldJSON(b, fd, v, opts)
		if err != nil {
			return err
		}
	}

	err = b.end()
	if err != nil {
		return err
	}
	err = b.WriteByte('}')
	if err != nil {
		return err
	}

	return nil
}

func marshalKnownFieldJSON(b *indentBuffer, fd *desc.FieldDescriptor, v interface{}, opts MarshalJSONOptions) error {
	jsonName := fd.AsFieldDescriptorProto().GetJsonName()
	if jsonName == "" {
		jsonName = fd.GetName()
	}
	err := writeJsonString(b, jsonName)
	if err != nil {
		return err
	}
	err = b.sep()
	if err != nil {
		return err
	}

	if v == nil {
		_, err := b.WriteString("null")
		return err
	}

	if fd.IsMap() {
		err = b.WriteByte('{')
		if err != nil {
			return err
		}
		err = b.start()
		if err != nil {
			return err
		}

		md := fd.GetMessageType()
		kfd := md.FindFieldByNumber(1)
		vfd := md.FindFieldByNumber(2)

		mp := v.(map[interface{}]interface{})
		if sort_map_keys {
			keys := make([]interface{}, 0, len(mp))
			for k := range mp {
				keys = append(keys, k)
			}
			sort.Sort(sortable(keys))
			first := true
			for _, mk := range keys {
				mv := mp[mk]
				err := b.maybeNext(&first)
				if err != nil {
					return err
				}

				err = marshalKnownFieldMapEntryJSON(b, kfd, mk, vfd, mv, opts)
				if err != nil {
					return err
				}
			}
		} else {
			first := true
			for mk, mv := range mp {
				err := b.maybeNext(&first)
				if err != nil {
					return err
				}
				err = marshalKnownFieldMapEntryJSON(b, kfd, mk, vfd, mv, opts)
				if err != nil {
					return err
				}
			}
		}

		err = b.end()
		if err != nil {
			return err
		}
		return b.WriteByte('}')

	} else if fd.IsRepeated() {
		err = b.WriteByte('[')
		if err != nil {
			return err
		}
		err = b.start()
		if err != nil {
			return err
		}

		sl := v.([]interface{})
		first := true
		for _, slv := range sl {
			err := b.maybeNext(&first)
			if err != nil {
				return err
			}
			err = marshalKnownFieldValueJSON(b, fd, slv, opts)
			if err != nil {
				return err
			}
		}

		err = b.end()
		if err != nil {
			return err
		}
		return b.WriteByte(']')

	} else {
		return marshalKnownFieldValueJSON(b, fd, v, opts)
	}
}

func marshalKnownFieldMapEntryJSON(b *indentBuffer, kfd *desc.FieldDescriptor, mk interface{}, vfd *desc.FieldDescriptor, mv interface{}, opts MarshalJSONOptions) error {
	rk := reflect.ValueOf(mk)
	var strkey string
	switch rk.Kind() {
	case reflect.Bool:
		strkey = strconv.FormatBool(rk.Bool())
	case reflect.Int32, reflect.Int64:
		strkey = strconv.FormatInt(rk.Int(), 10)
	case reflect.Uint32, reflect.Uint64:
		strkey = strconv.FormatUint(rk.Uint(), 10)
	case reflect.String:
		strkey = rk.String()
	default:
		return fmt.Errorf("Invalid map key value: %v (%v)", mk, rk.Type())
	}
	err := writeString(b, strkey)
	if err != nil {
		return err
	}
	err = b.sep()
	if err != nil {
		return err
	}
	return marshalKnownFieldValueJSON(b, vfd, mv, opts)
}

func marshalKnownFieldValueJSON(b *indentBuffer, fd *desc.FieldDescriptor, v interface{}, opts MarshalJSONOptions) error {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int32, reflect.Int64:
		ed := fd.GetEnumType()
		if ed != nil {
			n := int32(rv.Int())
			vd := ed.FindValueByNumber(n)
			if vd == nil {
				_, err := b.WriteString(strconv.FormatInt(rv.Int(), 10))
				return err
			} else {
				return writeJsonString(b, vd.GetName())
			}
		} else {
			_, err := b.WriteString(strconv.FormatInt(rv.Int(), 10))
			return err
		}
	case reflect.Uint32, reflect.Uint64:
		_, err := b.WriteString(strconv.FormatUint(rv.Uint(), 10))
		return err
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		var str string
		if math.IsNaN(f) {
			str = "NaN"
		} else if math.IsInf(f, 1) {
			str = "Infinity"
		} else if math.IsInf(f, -1) {
			str = "-Infinity"
		} else {
			var bits int
			if rv.Kind() == reflect.Float32 {
				bits = 32
			} else {
				bits = 64
			}
			str = strconv.FormatFloat(rv.Float(), 'g', -1, bits)
		}
		_, err := b.WriteString(str)
		return err
	case reflect.Bool:
		_, err := b.WriteString(strconv.FormatBool(rv.Bool()))
		return err
	case reflect.Slice:
		bstr := base64.StdEncoding.EncodeToString(rv.Bytes())
		return writeJsonString(b, bstr)
	case reflect.String:
		return writeJsonString(b, rv.String())
	default:
		// must be a message
		if dm, ok := v.(*Message); ok {
			return dm.marshalJSON(b, opts)
		} else {
			var err error
			if b.indent == -1 {
				m := jsonpb.Marshaler{}
				err = m.Marshal(b, v.(proto.Message))
			} else if b.indent == 0 {
				m := jsonpb.Marshaler{Indent: "  "}
				err = m.Marshal(b, v.(proto.Message))
			} else {
				m := jsonpb.Marshaler{Indent: "  "}
				str, err := m.MarshalToString(v.(proto.Message))
				if err != nil {
					return err
				}
				indent := strings.Repeat("  ", b.indent)
				pos := 0
				// add indention prefix to each line
				for pos < len(str) {
					start := pos
					pos = strings.Index(str[pos:], "\n")
					if pos == -1 {
						pos = len(str)
					} else {
						pos++ // include newline
					}
					line := str[start:pos]
					_, err = b.WriteString(indent)
					if err != nil {
						return err
					}
					_, err = b.WriteString(line)
					if err != nil {
						return err
					}
				}
			}
			return err
		}
	}
}

func writeJsonString(b *indentBuffer, s string) error {
	if sbytes, err := json.Marshal(s); err != nil {
		return err
	} else {
		_, err := b.Write(sbytes)
		return err
	}
}

func (m *Message) UnmarshalJSON(js []byte) error {
	m.Reset()
	if err := m.UnmarshalMergeJSON(js); err != nil {
		return err
	}
	return m.Validate()
}

func (m *Message) UnmarshalMergeJSON(js []byte) error {
	r := &jsReader{dec: json.NewDecoder(bytes.NewReader(js))}
	r.dec.UseNumber()
	err := m.unmarshalJson(r)
	if err != nil {
		return err
	}
	if t, err := r.poll(); err != io.EOF {
		b, _ := ioutil.ReadAll(r.dec.Buffered())
		s := fmt.Sprintf("%v%s", t, string(b))
		return fmt.Errorf("Superfluous data found after JSON object: %q", s)
	}
	return nil
}

func (m *Message) unmarshalJson(r *jsReader) error {
	t, err := r.peek()
	if err != nil {
		return err
	}
	if t == nil {
		// if json is simply "null" we do nothing
		r.poll()
		return nil
	}

	if err := r.beginObject(); err != nil {
		return err
	}

	for r.hasNext() {
		f, err := r.nextObjectKey()
		if err != nil {
			return err
		}
		fd := m.FindFieldDescriptorByName(f)
		if fd == nil {
			r.skip()
			continue
		}
		v, err := unmarshalJsField(fd, r, m.er)
		if err != nil {
			return err
		}
		if v != nil {
			m.internalSetField(fd, v)
		} else if m.values != nil {
			delete(m.values, fd.GetNumber())
		}
	}

	if err := r.endObject(); err != nil {
		return err
	}

	return nil
}

func unmarshalJsField(fd *desc.FieldDescriptor, r *jsReader, er *ExtensionRegistry) (interface{}, error) {
	t, err := r.peek()
	if err != nil {
		return nil, err
	}
	if t == nil {
		// if value is null, just return nil
		r.poll()
		return nil, nil
	}

	if t == json.Delim('{') && fd.IsMap() {
		entryType := fd.GetMessageType()
		keyType := entryType.FindFieldByNumber(1)
		valueType := entryType.FindFieldByNumber(2)
		mp := map[interface{}]interface{}{}

		// TODO: if there are just two map keys "key" and "value" and they have the right type of values,
		// treat this JSON object as a single map entry message. (In keeping with support of map fields as
		// if they were normal repeated field of entry messages as well as supporting a transition from
		// optional to repeated...)

		if err := r.beginObject(); err != nil {
			return nil, err
		}
		for r.hasNext() {
			kk, err := unmarshalJsFieldElement(keyType, r, er)
			if err != nil {
				return nil, err
			}
			vv, err := unmarshalJsFieldElement(valueType, r, er)
			if err != nil {
				return nil, err
			}
			mp[kk] = vv
		}
		if err := r.endObject(); err != nil {
			return nil, err
		}

		return mp, nil
	} else if t == json.Delim('[') {
		// We support parsing an array, even if field is not repeated, to mimic support in proto
		// binary wire format that supports changing an optional field to repeated and vice versa.
		// If the field is not repeated, we only keep the last value in the array.

		if err := r.beginArray(); err != nil {
			return nil, err
		}
		var sl []interface{}
		var v interface{}
		for r.hasNext() {
			var err error
			v, err = unmarshalJsFieldElement(fd, r, er)
			if err != nil {
				return nil, err
			}
			if fd.IsRepeated() && v != nil {
				sl = append(sl, v)
			}
		}
		if err := r.endArray(); err != nil {
			return nil, err
		}
		if fd.IsMap() {
			mp := map[interface{}]interface{}{}
			for _, m := range sl {
				msg := m.(*Message)
				kk, err := msg.TryGetFieldByNumber(1)
				if err != nil {
					return nil, err
				}
				vv, err := msg.TryGetFieldByNumber(2)
				if err != nil {
					return nil, err
				}
				mp[kk] = vv
			}
			return mp, nil
		} else if fd.IsRepeated() {
			return sl, nil
		} else {
			return v, nil
		}
	} else {
		// We support parsing a singular value, even if field is repeated, to mimic support in proto
		// binary wire format that supports changing an optional field to repeated and vice versa.
		// If the field is repeated, we store value as singleton slice of that one value.

		v, err := unmarshalJsFieldElement(fd, r, er)
		if err != nil {
			return nil, err
		}
		if v == nil {
			return nil, nil
		}
		if fd.IsRepeated() {
			return []interface{}{v}, nil
		} else {
			return v, nil
		}
	}
}

func unmarshalJsFieldElement(fd *desc.FieldDescriptor, r *jsReader, er *ExtensionRegistry) (interface{}, error) {
	t, err := r.peek()
	if err != nil {
		return nil, err
	}
	if t == nil {
		// if value is null, just return nil
		r.poll()
		return nil, nil
	}

	switch fd.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE,
		descriptor.FieldDescriptorProto_TYPE_GROUP:
		m := NewMessageWithExtensionRegistry(fd.GetMessageType(), er)
		if err := m.unmarshalJson(r); err != nil {
			return nil, err
		} else {
			return m, nil
		}

	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		if e, err := r.nextNumber(); err != nil {
			return nil, err
		} else {
			// value could be string or number
			if i, err := e.Int64(); err != nil {
				// number cannot be parsed, so see if it's an enum value name
				vd := fd.GetEnumType().FindValueByName(string(e))
				if vd != nil {
					return vd.GetNumber(), nil
				} else {
					// could not find it!
					return nil, fmt.Errorf("Enum %q does not have value named %q", fd.GetEnumType().GetFullyQualifiedName(), e)
				}
			} else if i > math.MaxInt32 || i < math.MinInt32 {
				return nil, NumericOverflowError
			} else {
				return int32(i), err
			}
		}

	case descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		if i, err := r.nextInt(); err != nil {
			return nil, err
		} else if i > math.MaxInt32 || i < math.MinInt32 {
			return nil, NumericOverflowError
		} else {
			return int32(i), err
		}

	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_SINT64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return r.nextInt()

	case descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_FIXED32:
		if i, err := r.nextUint(); err != nil {
			return nil, err
		} else if i > math.MaxUint32 {
			return nil, NumericOverflowError
		} else {
			return uint32(i), err
		}

	case descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_FIXED64:
		return r.nextUint()

	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		if str, ok := t.(string); ok {
			if str == "true" {
				r.poll() // consume token
				return true, err
			} else if str == "false" {
				r.poll() // consume token
				return false, err
			}
		}
		return r.nextBool()

	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		if f, err := r.nextFloat(); err != nil {
			return nil, err
		} else {
			return float32(f), nil
		}

	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return r.nextFloat()

	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return r.nextBytes()

	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return r.nextString()

	default:
		return nil, fmt.Errorf("Unknown field type: %v", fd.GetType())
	}
}

type jsReader struct {
	dec     *json.Decoder
	current json.Token
	peeked  bool
}

func (r *jsReader) hasNext() bool {
	return r.dec.More()
}

func (r *jsReader) peek() (json.Token, error) {
	if r.peeked {
		return r.current, nil
	}
	t, err := r.dec.Token()
	if err != nil {
		return nil, err
	}
	r.peeked = true
	r.current = t
	return t, nil
}

func (r *jsReader) poll() (json.Token, error) {
	if r.peeked {
		ret := r.current
		r.current = nil
		r.peeked = false
		return ret, nil
	}
	return r.dec.Token()
}

func (r *jsReader) beginObject() error {
	_, err := r.expect(func(t json.Token) bool { return t == json.Delim('{') }, nil, "start of JSON object: '{'")
	return err
}

func (r *jsReader) endObject() error {
	_, err := r.expect(func(t json.Token) bool { return t == json.Delim('}') }, nil, "end of JSON object: '}'")
	return err
}

func (r *jsReader) beginArray() error {
	_, err := r.expect(func(t json.Token) bool { return t == json.Delim('[') }, nil, "start of array: '['")
	return err
}

func (r *jsReader) endArray() error {
	_, err := r.expect(func(t json.Token) bool { return t == json.Delim(']') }, nil, "end of array: ']'")
	return err
}

func (r *jsReader) nextObjectKey() (string, error) {
	return r.nextString()
}

func (r *jsReader) nextString() (string, error) {
	t, err := r.expect(func(t json.Token) bool { _, ok := t.(string); return ok }, "", "string")
	if err != nil {
		return "", err
	}
	return t.(string), nil
}

func (r *jsReader) nextBytes() ([]byte, error) {
	str, err := r.nextString()
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(str)
}

func (r *jsReader) nextBool() (bool, error) {
	t, err := r.expect(func(t json.Token) bool { _, ok := t.(bool); return ok }, false, "boolean")
	if err != nil {
		return false, err
	}
	return t.(bool), nil
}

func (r *jsReader) nextInt() (int64, error) {
	n, err := r.nextNumber()
	if err != nil {
		return 0, err
	}
	return n.Int64()
}

func (r *jsReader) nextUint() (uint64, error) {
	n, err := r.nextNumber()
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(string(n), 10, 64)
}

func (r *jsReader) nextFloat() (float64, error) {
	n, err := r.nextNumber()
	if err != nil {
		return 0, err
	}
	return n.Float64()
}

func (r *jsReader) nextNumber() (json.Number, error) {
	t, err := r.expect(func(t json.Token) bool { return reflect.TypeOf(t).Kind() == reflect.String }, "0", "number")
	if err != nil {
		return "", err
	}
	switch t := t.(type) {
	case json.Number:
		return t, nil
	case string:
		return json.Number(t), nil
	}
	return "", fmt.Errorf("Expecting a number but got %v", t)
}

func (r *jsReader) skip() error {
	t, err := r.poll()
	if err != nil {
		return err
	}
	if t == json.Delim('[') {
		if err := r.skipArray(); err != nil {
			return err
		}
	} else if t == json.Delim('{') {
		if err := r.skipObject(); err != nil {
			return err
		}
	}
	return nil
}

func (r *jsReader) skipArray() error {
	for r.hasNext() {
		if err := r.skip(); err != nil {
			return err
		}
	}
	if err := r.endArray(); err != nil {
		return err
	}
	return nil
}

func (r *jsReader) skipObject() error {
	for r.hasNext() {
		// skip object key
		if err := r.skip(); err != nil {
			return err
		}
		// and value
		if err := r.skip(); err != nil {
			return err
		}
	}
	if err := r.endObject(); err != nil {
		return err
	}
	return nil
}

func (r *jsReader) expect(predicate func(json.Token) bool, ifNil interface{}, expected string) (interface{}, error) {
	t, err := r.poll()
	if err != nil {
		return nil, err
	}
	if t == nil && ifNil != nil {
		return ifNil, nil
	}
	if !predicate(t) {
		return t, fmt.Errorf("Bad input. Expecting %s. Instead got: %v.", expected, t)
	}
	return t, nil
}
