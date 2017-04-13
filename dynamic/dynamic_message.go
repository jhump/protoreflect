package dynamic

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
)

var UnknownTagNumberError = errors.New("Unknown tag number")
var FieldIsNotMapError = errors.New("Field is not a map type")
var FieldIsNotRepeatedError = errors.New("Field is not repeated")
var IndexOutOfRangeError = errors.New("Index is out of range")
var NumericOverflowError = errors.New("Numeric value is out of range")

var typeOfProtoMessage = reflect.TypeOf((*proto.Message)(nil)).Elem()
var typeOfDynamicMessage = reflect.TypeOf((*Message)(nil)).Elem()
var typeOfBytes = reflect.TypeOf(([]byte)(nil))

var varintTypes = map[descriptor.FieldDescriptorProto_Type]bool{}
var fixed32Types = map[descriptor.FieldDescriptorProto_Type]bool{}
var fixed64Types = map[descriptor.FieldDescriptorProto_Type]bool{}

func init() {
	varintTypes[descriptor.FieldDescriptorProto_TYPE_BOOL] = true
	varintTypes[descriptor.FieldDescriptorProto_TYPE_INT32] = true
	varintTypes[descriptor.FieldDescriptorProto_TYPE_INT64] = true
	varintTypes[descriptor.FieldDescriptorProto_TYPE_UINT32] = true
	varintTypes[descriptor.FieldDescriptorProto_TYPE_UINT64] = true
	varintTypes[descriptor.FieldDescriptorProto_TYPE_SINT32] = true
	varintTypes[descriptor.FieldDescriptorProto_TYPE_SINT64] = true
	varintTypes[descriptor.FieldDescriptorProto_TYPE_ENUM] = true

	fixed32Types[descriptor.FieldDescriptorProto_TYPE_FIXED32] = true
	fixed32Types[descriptor.FieldDescriptorProto_TYPE_SFIXED32] = true
	fixed32Types[descriptor.FieldDescriptorProto_TYPE_FLOAT] = true

	fixed64Types[descriptor.FieldDescriptorProto_TYPE_FIXED64] = true
	fixed64Types[descriptor.FieldDescriptorProto_TYPE_SFIXED64] = true
	fixed64Types[descriptor.FieldDescriptorProto_TYPE_DOUBLE] = true
}

type Message struct {
	md            *desc.MessageDescriptor
	er            *ExtensionRegistry
	extraFields   map[int32]*desc.FieldDescriptor
	values        map[int32]interface{}
	unknownFields map[int32][]UnknownField
}

type UnknownField struct {
	// Encoding indicates how the unknown field was encoded on the wire. If it
	// is proto.WireBytes or proto.WireGroupStart then Contents will be set to
	// the raw bytes. If it is proto.WireTypeFixed32 then the data is in the least
	// significant 32 bits of Value. Otherwise, the data is in all 64 bits of
	// Value.
	Encoding int8
	Contents []byte
	Value    uint64
}

func NewMessageWithExtensions(md *desc.MessageDescriptor, exts ...*desc.FieldDescriptor) (*Message, error) {
	if len(exts) == 0 {
		return NewMessage(md), nil
	}

	if !md.IsExtendable() {
		return nil, fmt.Errorf("Given message is not extendable: %s", md.GetFullyQualifiedName())
	}

	for _, fd := range exts {
		if !fd.IsExtension() {
			return nil, fmt.Errorf("Given field is not an extension: %s", fd.GetFullyQualifiedName())
		}
		if fd.GetOwner().GetFullyQualifiedName() != md.GetFullyQualifiedName() {
			return nil, fmt.Errorf("Given field, %s, extends wrong message type: %s; expecting: %s", fd.GetFullyQualifiedName(), fd.GetOwner().GetFullyQualifiedName(), md.GetFullyQualifiedName())
		}
		if !md.IsExtension(fd.GetNumber()) {
			return nil, fmt.Errorf("Given field, %s, has a tag number, %d, that is outside extendable range of %s: %v", fd.GetFullyQualifiedName(), fd.GetNumber(), fd.GetOwner().GetFullyQualifiedName(), fd.GetOwner().GetExtensionRanges())
		}
	}

	er := &ExtensionRegistry{}
	er.AddExtension(exts...)

	return NewMessageWithExtensionRegistry(md, er), nil
}

func NewMessage(md *desc.MessageDescriptor) *Message {
	return NewMessageWithExtensionRegistry(md, nil)
}

func NewMessageWithExtensionRegistry(md *desc.MessageDescriptor, er *ExtensionRegistry) *Message {
	return &Message{
		md: md,
		er: er,
	}
}

func (m *Message) GetMessageDescriptor() *desc.MessageDescriptor {
	return m.md
}

func (m *Message) GetKnownFields() []*desc.FieldDescriptor {
	if len(m.extraFields) == 0 {
		return m.md.GetFields()
	}
	flds := make([]*desc.FieldDescriptor, len(m.md.GetFields())+len(m.extraFields))
	copy(flds, m.md.GetFields())
	for _, fld := range m.extraFields {
		flds = append(flds, fld)
	}
	return flds
}

func (m *Message) GetUnknownFields() []int32 {
	flds := make([]int32, 0, len(m.unknownFields))
	for tag := range m.unknownFields {
		flds = append(flds, tag)
	}
	return flds
}

func (m *Message) Descriptor() ([]byte, []int) {
	// get encoded file descriptor
	b, err := proto.Marshal(m.md.GetFile().AsProto())
	if err != nil {
		panic(fmt.Sprintf("Failed to get encoded descriptor for %s", m.md.GetFile().GetName()))
	}
	var zippedBytes bytes.Buffer
	w := gzip.NewWriter(&zippedBytes)
	_, err = w.Write(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to get encoded descriptor for %s", m.md.GetFile().GetName()))
	}

	// and path to message
	path := []int{}
	var d desc.Descriptor
	name := m.md.GetFullyQualifiedName()
	for d = m.md; d != nil; d = d.GetParent() {
		found := false
		switch d := d.(type) {
		case (*desc.FileDescriptor):
			for i, md := range d.GetMessageTypes() {
				if md.GetFullyQualifiedName() == name {
					found = true
					path = append(path, i)
				}
			}
		case (*desc.MessageDescriptor):
			for i, md := range d.GetNestedMessageTypes() {
				if md.GetFullyQualifiedName() == name {
					found = true
					path = append(path, i)
				}
			}
		}
		if !found {
			panic(fmt.Sprintf("Failed to get descriptor path for %s", m.md.GetFullyQualifiedName()))
		}
	}
	// reverse the path
	i := 0
	j := len(path)
	for i < j {
		path[i], path[j] = path[j], path[i]
		i++
		j--
	}

	return zippedBytes.Bytes(), path
}

func (m *Message) FindFieldDescriptor(tagNumber int32) *desc.FieldDescriptor {
	fd := m.md.FindFieldByNumber(tagNumber)
	if fd != nil {
		return fd
	}
	fd = m.er.FindExtension(m.md.GetFullyQualifiedName(), tagNumber)
	if fd != nil {
		return fd
	}
	return m.extraFields[tagNumber]
}

func (m *Message) FindFieldDescriptorByName(name string) *desc.FieldDescriptor {
	if name == "" {
		return nil
	}
	fd := m.md.FindFieldByName(name)
	if fd != nil {
		return fd
	}
	mustBeExt := false
	if name[0] == '(' {
		if name[len(name)-1] != ')' {
			// malformed name
			return nil
		}
		mustBeExt = true
		name = name[1 : len(name)-1]
	} else if name[0] == '[' {
		if name[len(name)-1] != ']' {
			// malformed name
			return nil
		}
		name = name[1 : len(name)-1]
	}
	fd = m.er.FindExtensionByName(m.md.GetFullyQualifiedName(), name)
	if fd != nil {
		return fd
	}
	for _, fd := range m.extraFields {
		if fd.IsExtension() && name == fd.GetFullyQualifiedName() {
			return fd
		} else if !mustBeExt && !fd.IsExtension() && name == fd.GetName() {
			return fd
		}
	}
	return nil
}

func (m *Message) checkField(fd *desc.FieldDescriptor) error {
	if fd.GetOwner().GetFullyQualifiedName() != m.md.GetFullyQualifiedName() {
		return fmt.Errorf("Given field, %s, is for wrong message type: %s; expecting %s", fd.GetName(), fd.GetOwner().GetFullyQualifiedName(), m.md.GetFullyQualifiedName())
	}
	if fd.IsExtension() && !m.md.IsExtension(fd.GetNumber()) {
		return fmt.Errorf("Given field, %s, is an extension but is not in message extension range: %v", fd.GetFullyQualifiedName(), m.md.GetExtensionRanges())
	}
	return nil
}

func (m *Message) GetField(fd *desc.FieldDescriptor) interface{} {
	if v, err := m.TryGetField(fd); err != nil {
		panic(err.Error())
	} else {
		return v
	}
}

func (m *Message) TryGetField(fd *desc.FieldDescriptor) (interface{}, error) {
	if err := m.checkField(fd); err != nil {
		return nil, err
	}
	return m.getField(fd)
}

func (m *Message) GetFieldByNumber(tagNumber int) interface{} {
	if v, err := m.TryGetFieldByNumber(tagNumber); err != nil {
		panic(err.Error())
	} else {
		return v
	}
}

func (m *Message) TryGetFieldByNumber(tagNumber int) (interface{}, error) {
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return nil, UnknownTagNumberError
	}
	return m.getField(fd)
}

func (m *Message) getField(fd *desc.FieldDescriptor) (interface{}, error) {
	return m.doGetField(fd, false)
}

func (m *Message) doGetField(fd *desc.FieldDescriptor, nilIfAbsent bool) (interface{}, error) {
	res := m.values[fd.GetNumber()]
	if res == nil {
		var err error
		if res, err = m.parseUnknownField(fd); err != nil {
			return nil, err
		} else if res != nil {
			return res, nil
		} else if nilIfAbsent {
			return nil, nil
		} else {
			def := fd.GetDefaultValue()
			if def != nil {
				return def, nil
			}
			// GetDefaultValue only returns nil for message types
			md := fd.GetMessageType()
			if md.IsProto3() {
				return (*Message)(nil), nil
			} else {
				// for proto2, return default instance of message
				return NewMessageWithExtensionRegistry(md, m.er), nil
			}
		}
	}
	rt := reflect.TypeOf(res)
	if rt.Kind() == reflect.Map {
		// make defensive copies to prevent caller from storing illegal keys and values
		m := res.(map[interface{}]interface{})
		res := map[interface{}]interface{}{}
		for k, v := range m {
			res[k] = v
		}
		return res, nil
	} else if rt.Kind() == reflect.Slice && rt != typeOfBytes {
		// make defensive copies to prevent caller from storing illegal elements
		sl := res.([]interface{})
		res := make([]interface{}, len(sl))
		for i, e := range sl {
			res[i] = e
		}
		return res, nil
	}
	return res, nil
}

func (m *Message) SetField(fd *desc.FieldDescriptor, val interface{}) {
	if err := m.TrySetField(fd, val); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TrySetField(fd *desc.FieldDescriptor, val interface{}) error {
	if err := m.checkField(fd); err != nil {
		return err
	}
	return m.setField(fd, val)
}

func (m *Message) SetFieldByNumber(tagNumber int, val interface{}) {
	if err := m.TrySetFieldByNumber(tagNumber, val); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TrySetFieldByNumber(tagNumber int, val interface{}) error {
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return UnknownTagNumberError
	}
	return m.setField(fd, val)
}

func (m *Message) setField(fd *desc.FieldDescriptor, val interface{}) error {
	var err error
	if val, err = validFieldValue(fd, val); err != nil {
		return err
	}
	m.internalSetField(fd, val)
	return nil
}

func (m *Message) internalSetField(fd *desc.FieldDescriptor, val interface{}) {
	if m.md.IsProto3() {
		// proto3 considers fields that are set to their zero value as unset
		if fd.IsRepeated() {
			// can't use == comparison below for map and slices, so just test length
			// (zero length is same as default)
			if reflect.ValueOf(val).Len() == 0 {
				return
			}
		} else if fd.GetDefaultValue() == val {
			return
		}
	}
	if m.values == nil {
		m.values = map[int32]interface{}{}
	}
	m.values[fd.GetNumber()] = val
	// if this field is part of a one-of, make sure all other one-of choices are cleared
	od := fd.GetOneOf()
	if od != nil {
		for _, other := range od.GetChoices() {
			if other.GetNumber() != fd.GetNumber() {
				delete(m.values, other.GetNumber())
			}
		}
	}
	// also clear any unknown fields
	if m.unknownFields != nil {
		delete(m.unknownFields, fd.GetNumber())
	}
	// and add this field if it was previously unknown
	if fd := m.FindFieldDescriptor(fd.GetNumber()); fd == nil {
		m.addField(fd)
	}
}

func (m *Message) addField(fd *desc.FieldDescriptor) {
	if m.extraFields == nil {
		m.extraFields = map[int32]*desc.FieldDescriptor{}
	}
	m.extraFields[fd.GetNumber()] = fd
}

func (m *Message) ClearField(fd *desc.FieldDescriptor) {
	if err := m.TryClearField(fd); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TryClearField(fd *desc.FieldDescriptor) error {
	if err := m.checkField(fd); err != nil {
		return err
	}
	m.clearField(fd)
	return nil
}

func (m *Message) ClearFieldByNumber(tagNumber int) {
	if err := m.TryClearFieldByNumber(tagNumber); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TryClearFieldByNumber(tagNumber int) error {
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return UnknownTagNumberError
	}
	m.clearField(fd)
	return nil
}

func (m *Message) clearField(fd *desc.FieldDescriptor) {
	if m.values != nil {
		delete(m.values, fd.GetNumber())
	}
}

func (m *Message) GetOneOfField(od *desc.OneOfDescriptor) (*desc.FieldDescriptor, interface{}) {
	if fd, val, err := m.TryGetOneOfField(od); err != nil {
		panic(err.Error())
	} else {
		return fd, val
	}
}

func (m *Message) TryGetOneOfField(od *desc.OneOfDescriptor) (*desc.FieldDescriptor, interface{}, error) {
	if od.GetOwner().GetFullyQualifiedName() != m.md.GetFullyQualifiedName() {
		return nil, nil, fmt.Errorf("Given one-of, %s, is for wrong message type: %s; expecting %s", od.GetName(), od.GetOwner().GetFullyQualifiedName(), m.md.GetFullyQualifiedName())
	}
	for _, fd := range od.GetChoices() {
		val, err := m.doGetField(fd, true)
		if err != nil {
			return nil, nil, err
		}
		if val != nil {
			return fd, val, nil
		}
	}
	return nil, nil, nil
}

func (m *Message) GetMapField(fd *desc.FieldDescriptor, key interface{}) interface{} {
	if v, err := m.TryGetMapField(fd, key); err != nil {
		panic(err.Error())
	} else {
		return v
	}
}

func (m *Message) TryGetMapField(fd *desc.FieldDescriptor, key interface{}) (interface{}, error) {
	if err := m.checkField(fd); err != nil {
		return nil, err
	}
	return m.getMapField(fd, key)
}

func (m *Message) GetMapFieldByNumber(tagNumber int, key interface{}) interface{} {
	if v, err := m.TryGetMapFieldByNumber(tagNumber, key); err != nil {
		panic(err.Error())
	} else {
		return v
	}
}

func (m *Message) TryGetMapFieldByNumber(tagNumber int, key interface{}) (interface{}, error) {
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return nil, UnknownTagNumberError
	}
	return m.getMapField(fd, key)
}

func (m *Message) getMapField(fd *desc.FieldDescriptor, key interface{}) (interface{}, error) {
	if !fd.IsMap() {
		return nil, FieldIsNotMapError
	}
	kfd := fd.GetMessageType().GetFields()[0]
	ki, err := validElementFieldValue(kfd, key)
	if err != nil {
		return nil, err
	}
	mp := m.values[fd.GetNumber()]
	if mp == nil {
		if mp, err = m.parseUnknownField(fd); err != nil {
			return nil, err
		} else if mp == nil {
			return nil, nil
		}
	}
	return mp.(map[interface{}]interface{})[ki], nil
}

func (m *Message) PutMapField(fd *desc.FieldDescriptor, key interface{}, val interface{}) {
	if err := m.TryPutMapField(fd, key, val); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TryPutMapField(fd *desc.FieldDescriptor, key interface{}, val interface{}) error {
	if err := m.checkField(fd); err != nil {
		return err
	}
	return m.putMapField(fd, key, val)
}

func (m *Message) PutMapFieldByNumber(tagNumber int, key interface{}, val interface{}) {
	if err := m.TryPutMapFieldByNumber(tagNumber, key, val); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TryPutMapFieldByNumber(tagNumber int, key interface{}, val interface{}) error {
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return UnknownTagNumberError
	}
	return m.putMapField(fd, key, val)
}

func (m *Message) putMapField(fd *desc.FieldDescriptor, key interface{}, val interface{}) error {
	if !fd.IsMap() {
		return FieldIsNotMapError
	}
	kfd := fd.GetMessageType().GetFields()[0]
	ki, err := validElementFieldValue(kfd, key)
	if err != nil {
		return err
	}
	vfd := fd.GetMessageType().GetFields()[1]
	vi, err := validElementFieldValue(vfd, val)
	if err != nil {
		return err
	}
	mp := m.values[fd.GetNumber()]
	if mp == nil {
		if mp, err = m.parseUnknownField(fd); err != nil {
			return err
		} else if mp == nil {
			mp = map[interface{}]interface{}{}
			m.internalSetField(fd, mp)
		}
	}
	mp.(map[interface{}]interface{})[ki] = vi
	return nil
}

func (m *Message) RemoveMapField(fd *desc.FieldDescriptor, key interface{}) {
	if err := m.TryRemoveMapField(fd, key); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TryRemoveMapField(fd *desc.FieldDescriptor, key interface{}) error {
	if err := m.checkField(fd); err != nil {
		return err
	}
	return m.removeMapField(fd, key)
}

func (m *Message) RemoveMapFieldByNumber(tagNumber int, key interface{}) {
	if err := m.TryRemoveMapFieldByNumber(tagNumber, key); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TryRemoveMapFieldByNumber(tagNumber int, key interface{}) error {
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return UnknownTagNumberError
	}
	return m.removeMapField(fd, key)
}

func (m *Message) removeMapField(fd *desc.FieldDescriptor, key interface{}) error {
	if !fd.IsMap() {
		return FieldIsNotMapError
	}
	kfd := fd.GetMessageType().GetFields()[0]
	ki, err := validElementFieldValue(kfd, key)
	if err != nil {
		return err
	}
	mp := m.values[fd.GetNumber()]
	if mp == nil {
		if mp, err = m.parseUnknownField(fd); err != nil {
			return err
		} else if mp == nil {
			return nil
		}
	}
	res := mp.(map[interface{}]interface{})
	delete(res, ki)
	if len(res) == 0 {
		delete(m.values, fd.GetNumber())
	}
	return nil
}

func (m *Message) GetRepeatedField(fd *desc.FieldDescriptor, index int) interface{} {
	if v, err := m.TryGetRepeatedField(fd, index); err != nil {
		panic(err.Error())
	} else {
		return v
	}
}

func (m *Message) TryGetRepeatedField(fd *desc.FieldDescriptor, index int) (interface{}, error) {
	if index < 0 {
		return nil, IndexOutOfRangeError
	}
	if err := m.checkField(fd); err != nil {
		return nil, err
	}
	return m.getRepeatedField(fd, index)
}

func (m *Message) GetRepeatedFieldByNumber(tagNumber int, index int) interface{} {
	if v, err := m.TryGetRepeatedFieldByNumber(tagNumber, index); err != nil {
		panic(err.Error())
	} else {
		return v
	}
}

func (m *Message) TryGetRepeatedFieldByNumber(tagNumber int, index int) (interface{}, error) {
	if index < 0 {
		return nil, IndexOutOfRangeError
	}
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return nil, UnknownTagNumberError
	}
	return m.getRepeatedField(fd, index)
}

func (m *Message) getRepeatedField(fd *desc.FieldDescriptor, index int) (interface{}, error) {
	if !fd.IsRepeated() {
		return nil, FieldIsNotRepeatedError
	}
	sl := m.values[fd.GetNumber()]
	if sl == nil {
		var err error
		if sl, err = m.parseUnknownField(fd); err != nil {
			return nil, err
		} else if sl == nil {
			return nil, IndexOutOfRangeError
		}
	}
	res := sl.([]interface{})
	if index >= len(res) {
		return nil, IndexOutOfRangeError
	}
	return res[index], nil
}

func (m *Message) AddRepeatedField(fd *desc.FieldDescriptor, val interface{}) {
	if err := m.TryAddRepeatedField(fd, val); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TryAddRepeatedField(fd *desc.FieldDescriptor, val interface{}) error {
	if err := m.checkField(fd); err != nil {
		return err
	}
	return m.addRepeatedField(fd, val)
}

func (m *Message) AddRepeatedFieldByNumber(tagNumber int, val interface{}) {
	if err := m.TryAddRepeatedFieldByNumber(tagNumber, val); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TryAddRepeatedFieldByNumber(tagNumber int, val interface{}) error {
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return UnknownTagNumberError
	}
	return m.addRepeatedField(fd, val)
}

func (m *Message) addRepeatedField(fd *desc.FieldDescriptor, val interface{}) error {
	if !fd.IsRepeated() {
		return FieldIsNotRepeatedError
	}
	val, err := validElementFieldValue(fd, val)
	if err != nil {
		return err
	}

	if fd.IsMap() {
		// We're lenient. Just as we allow setting a map field to a slice of entry messages, we also allow
		// adding entries one at a time (as if the field were a normal repeated field).
		msg := val.(proto.Message)
		dm, err := asDynamicMessage(msg, fd.GetMessageType())
		if err != nil {
			return err
		}
		k, err := dm.TryGetFieldByNumber(1)
		if err != nil {
			return err
		}
		v, err := dm.TryGetFieldByNumber(2)
		if err != nil {
			return err
		}
		return m.putMapField(fd, k, v)
	}

	sl := m.values[fd.GetNumber()]
	if sl == nil {
		if sl, err = m.parseUnknownField(fd); err != nil {
			return err
		} else if sl == nil {
			sl = []interface{}{}
		}
	}
	res := sl.([]interface{})
	res = append(res, val)
	m.internalSetField(fd, res)
	return nil
}

func (m *Message) SetRepeatedField(fd *desc.FieldDescriptor, index int, val interface{}) {
	if err := m.TrySetRepeatedField(fd, index, val); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TrySetRepeatedField(fd *desc.FieldDescriptor, index int, val interface{}) error {
	if index < 0 {
		return IndexOutOfRangeError
	}
	if err := m.checkField(fd); err != nil {
		return err
	}
	return m.setRepeatedField(fd, index, val)
}

func (m *Message) SetRepeatedFieldByNumber(tagNumber int, index int, val interface{}) {
	if err := m.TrySetRepeatedFieldByNumber(tagNumber, index, val); err != nil {
		panic(err.Error())
	}
}

func (m *Message) TrySetRepeatedFieldByNumber(tagNumber int, index int, val interface{}) error {
	if index < 0 {
		return IndexOutOfRangeError
	}
	fd := m.FindFieldDescriptor(int32(tagNumber))
	if fd == nil {
		return UnknownTagNumberError
	}
	return m.setRepeatedField(fd, index, val)
}

func (m *Message) setRepeatedField(fd *desc.FieldDescriptor, index int, val interface{}) error {
	if fd.IsMap() || !fd.IsRepeated() {
		return FieldIsNotRepeatedError
	}
	val, err := validElementFieldValue(fd, val)
	if err != nil {
		return err
	}
	sl := m.values[fd.GetNumber()]
	if sl == nil {
		if sl, err = m.parseUnknownField(fd); err != nil {
			return err
		} else if sl == nil {
			return IndexOutOfRangeError
		}
	}
	res := sl.([]interface{})
	if index >= len(res) {
		return IndexOutOfRangeError
	}
	res[index] = val
	return nil
}

func (m *Message) GetUnknownField(tagNumber int32) []UnknownField {
	if u, ok := m.unknownFields[tagNumber]; ok {
		return u
	} else {
		return nil
	}
}

func (m *Message) parseUnknownField(fd *desc.FieldDescriptor) (interface{}, error) {
	unks, ok := m.unknownFields[fd.GetNumber()]
	if !ok {
		return nil, nil
	}
	var v interface{}
	var sl []interface{}
	var mp map[interface{}]interface{}
	if fd.IsMap() {
		mp = map[interface{}]interface{}{}
	}
	var err error
	for _, unk := range unks {
		var val interface{}
		if unk.Encoding == proto.WireBytes || unk.Encoding == proto.WireStartGroup {
			val, err = unmarshalLengthDelimitedField(fd, unk.Contents, m.er)
		} else {
			val, err = unmarshalSimpleField(fd, unk.Value)
		}
		if err != nil {
			return nil, err
		}
		if fd.IsMap() {
			newEntry := val.(*Message)
			kk, err := newEntry.TryGetFieldByNumber(1)
			if err != nil {
				return nil, err
			}
			vv, err := newEntry.TryGetFieldByNumber(2)
			if err != nil {
				return nil, err
			}
			mp[kk] = vv
			v = mp
		} else if fd.IsRepeated() {
			t := reflect.TypeOf(val)
			if t.Kind() == reflect.Slice && t != typeOfBytes {
				// append slices if we unmarshalled a packed repeated field
				newVals := val.([]interface{})
				sl = append(sl, newVals...)
			} else {
				sl = append(sl, val)
			}
			v = sl
		} else {
			v = val
		}
	}
	m.internalSetField(fd, v)
	m.addField(fd)
	return v, nil
}

func validFieldValue(fd *desc.FieldDescriptor, val interface{}) (interface{}, error) {
	v := reflect.ValueOf(val)
	if fd.IsMap() && v.Kind() == reflect.Map {
		// make a defensive copy while we check the contents
		// (also converts to map[interface{}]interface{} if it's some other type)
		keyField := fd.GetMessageType().GetFields()[0]
		valField := fd.GetMessageType().GetFields()[1]
		m := map[interface{}]interface{}{}
		for _, k := range v.MapKeys() {
			v := v.MapIndex(k)
			kk, err := validFieldValue(keyField, k.Interface())
			if err != nil {
				return nil, err
			}
			vv, err := validFieldValue(valField, v.Interface())
			if err != nil {
				return nil, err
			}
			m[kk] = vv
		}
		return m, nil
	}

	if fd.IsRepeated() { // this will also catch map fields where given value was not a map
		if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
			if fd.IsMap() {
				return nil, fmt.Errorf("Value for map field must be a map; instead was %v", v.Type())
			} else {
				return nil, fmt.Errorf("Value for repeated field must be a slice; instead was %v", v.Type())
			}
		}

		if fd.IsMap() {
			// value should be a slice of entry messages that we need convert into a map[interface{}]interface{}
			m := map[interface{}]interface{}{}
			for i := 0; i < v.Len(); i++ {
				e := v.Index(i).Interface()
				msg, ok := e.(proto.Message)
				if !ok {
					return nil, fmt.Errorf("Contents of slice/array for a map field must be map entry messages; instead was %v", v.Index(i).Type())
				}
				dm, err := asDynamicMessage(msg, fd.GetMessageType())
				if err != nil {
					return nil, err
				}
				k, err := dm.TryGetFieldByNumber(1)
				if err != nil {
					return nil, err
				}
				v, err := dm.TryGetFieldByNumber(2)
				if err != nil {
					return nil, err
				}
				m[k] = v
			}
			return m, nil
		}

		// make a defensive copy while checking contents (also converts to []interface{})
		s := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			e := v.Index(i).Interface()
			e, err := validElementFieldValue(fd, e)
			if err != nil {
				return nil, err
			}
			s[i] = e
		}

		return s, nil
	}

	return validElementFieldValue(fd, val)
}

func asDynamicMessage(m proto.Message, md *desc.MessageDescriptor) (*Message, error) {
	if dm, ok := m.(*Message); ok {
		return dm, nil
	}
	dm := NewMessage(md)
	if err := dm.mergeFrom(m); err != nil {
		return nil, err
	}
	return dm, nil
}

func validElementFieldValue(fd *desc.FieldDescriptor, val interface{}) (interface{}, error) {
	v := reflect.ValueOf(val)
	t := fd.GetType()
	typeName := strings.ToLower(t.String())
	switch t {
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_ENUM:
		return toInt32(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_SINT64:
		return toInt64(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_UINT32:
		return toUint32(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_UINT64:
		return toUint64(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return toFloat32(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return toFloat64(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return toBool(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return toBytes(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return toString(elem(v), typeName, fd.GetFullyQualifiedName())

	case descriptor.FieldDescriptorProto_TYPE_MESSAGE,
		descriptor.FieldDescriptorProto_TYPE_GROUP:
		m, err := asMessage(v, fd.GetFullyQualifiedName())
		// check that message is correct type
		if err != nil {
			return nil, err
		}
		var msgType string
		if dm, ok := m.(*Message); ok {
			msgType = dm.GetMessageDescriptor().GetFullyQualifiedName()
		} else {
			msgType = proto.MessageName(m)
		}
		if msgType != fd.GetMessageType().GetFullyQualifiedName() {
			return nil, fmt.Errorf("message field %s requires value of type %s; received %s", fd.GetFullyQualifiedName(), fd.GetMessageType().GetFullyQualifiedName(), msgType)
		}
		return m, nil

	default:
		return nil, fmt.Errorf("unable to handle unrecognized field type: %v", fd.GetType())
	}
}

func elem(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		return v.Elem()
	}
	return v
}

func toInt32(v reflect.Value, what string, fieldName string) (int32, error) {
	if v.Kind() == reflect.Int32 {
		return int32(v.Int()), nil
	}
	return 0, fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func toUint32(v reflect.Value, what string, fieldName string) (uint32, error) {
	if v.Kind() == reflect.Uint32 {
		return uint32(v.Uint()), nil
	}
	return 0, fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func toFloat32(v reflect.Value, what string, fieldName string) (float32, error) {
	if v.Kind() == reflect.Float32 {
		return float32(v.Float()), nil
	}
	return 0, fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func toInt64(v reflect.Value, what string, fieldName string) (int64, error) {
	if v.Kind() == reflect.Int64 {
		return v.Int(), nil
	}
	return 0, fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func toUint64(v reflect.Value, what string, fieldName string) (uint64, error) {
	if v.Kind() == reflect.Uint64 {
		return v.Uint(), nil
	}
	return 0, fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func toFloat64(v reflect.Value, what string, fieldName string) (float64, error) {
	if v.Kind() == reflect.Float64 {
		return v.Float(), nil
	}
	return 0, fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func toBool(v reflect.Value, what string, fieldName string) (bool, error) {
	if v.Kind() == reflect.Bool {
		return v.Bool(), nil
	}
	return false, fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func toBytes(v reflect.Value, what string, fieldName string) ([]byte, error) {
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		return v.Bytes(), nil
	}
	return nil, fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func toString(v reflect.Value, what string, fieldName string) (string, error) {
	if v.Kind() == reflect.String {
		return v.String(), nil
	}
	return "", fmt.Errorf("%s field %s is not compatible with value of type %v", what, fieldName, v.Type())
}

func asMessage(v reflect.Value, fieldName string) (proto.Message, error) {
	t := v.Type()
	// we need a pointer to a struct that implements proto.Message
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct || !t.Implements(typeOfProtoMessage) {
		return nil, fmt.Errorf("message field %s requires is not compatible with value of type %v", fieldName, v.Type())
	}
	return v.Interface().(proto.Message), nil
}

func (m *Message) Reset() {
	m.values = nil
	m.extraFields = nil
	m.unknownFields = nil
}

func (m *Message) String() string {
	b, err := m.MarshalText()
	if err != nil {
		panic(fmt.Sprintf("Failed to create string representation of message: %s", err.Error()))
	}
	return string(b)
}

func (m *Message) ProtoMessage() {
}

func (m *Message) ConvertTo(target proto.Message) error {
	if err := m.checkType(target); err != nil {
		return err
	}

	target.Reset()
	return m.mergeInto(target)
}

func (m *Message) ConvertFrom(target proto.Message) error {
	if err := m.checkType(target); err != nil {
		return err
	}

	m.Reset()
	return m.mergeFrom(target)
}

func (m *Message) MergeInto(target proto.Message) error {
	if err := m.checkType(target); err != nil {
		return err
	}
	return m.mergeInto(target)
}

func (m *Message) MergeFrom(target proto.Message) error {
	if err := m.checkType(target); err != nil {
		return err
	}
	return m.mergeFrom(target)
}

func (m *Message) checkType(target proto.Message) error {
	if dm, ok := target.(*Message); ok && dm.md.GetFullyQualifiedName() != m.md.GetFullyQualifiedName() {
		return fmt.Errorf("Given message has wrong type: %q; expecting %q", dm.md.GetFullyQualifiedName(), m.md.GetFullyQualifiedName())
	}

	msgName := proto.MessageName(target)
	if msgName != m.md.GetFullyQualifiedName() {
		return fmt.Errorf("Given message has wrong type: %q; expecting %q", msgName, m.md.GetFullyQualifiedName())
	}
	return nil
}

func (m *Message) mergeInto(pm proto.Message) error {
	if dm, ok := pm.(*Message); ok {
		return dm.mergeFrom(m)
	}

	target := reflect.ValueOf(pm)

	// track tags for which the dynamic message has data but the given
	// message doesn't know about it
	u := target.FieldByName("XXX_unrecognized")
	var unknownTags map[int32]struct{}
	if u.IsValid() && u.Type() == typeOfBytes {
		unknownTags = map[int32]struct{}{}
		for tag := range m.values {
			unknownTags[tag] = struct{}{}
		}
	}

	// check that we can successfully do the merge
	structProps := proto.GetProperties(reflect.TypeOf(pm).Elem())
	for _, prop := range structProps.Prop {
		if prop.Tag == 0 {
			continue // one-of or special field (such as XXX_unrecognized, etc.)
		}
		tag := int32(prop.Tag)
		v, ok := m.values[tag]
		if !ok {
			continue
		}
		if unknownTags != nil {
			delete(unknownTags, tag)
		}
		f := target.FieldByName(prop.Name)
		ft := f.Type()
		val := reflect.ValueOf(v)
		if !canConvert(val, ft) {
			return fmt.Errorf("Cannot convert %v to %v", val.Type(), ft)
		}
	}
	// check one-of fields
	for _, oop := range structProps.OneofTypes {
		prop := oop.Prop
		tag := int32(prop.Tag)
		v, ok := m.values[tag]
		if !ok {
			continue
		}
		if unknownTags != nil {
			delete(unknownTags, tag)
		}
		stf, ok := oop.Type.Elem().FieldByName(prop.Name)
		if !ok {
			return fmt.Errorf("One-of field indicates struct field name %s, but type %v has no such field", prop.Name, oop.Type.Elem())
		}
		ft := stf.Type
		val := reflect.ValueOf(v)
		if !canConvert(val, ft) {
			return fmt.Errorf("Cannot convert %v to %v", val.Type(), ft)
		}
	}
	// and check extensions, too
	for tag, ext := range proto.RegisteredExtensions(pm) {
		v, ok := m.values[tag]
		if !ok {
			continue
		}
		if unknownTags != nil {
			delete(unknownTags, tag)
		}
		ft := reflect.TypeOf(ext.ExtensionType)
		val := reflect.ValueOf(v)
		if !canConvert(val, ft) {
			return fmt.Errorf("Cannot convert %v to %v", val.Type(), ft)
		}
	}

	// now actually perform the merge
	for _, prop := range structProps.Prop {
		v, ok := m.values[int32(prop.Tag)]
		if !ok {
			continue
		}
		f := target.FieldByName(prop.Name)
		convert(reflect.ValueOf(v), f)
	}
	// merge one-ofs
	for _, oop := range structProps.OneofTypes {
		prop := oop.Prop
		tag := int32(prop.Tag)
		v, ok := m.values[tag]
		if !ok {
			continue
		}
		oov := reflect.New(oop.Type.Elem())
		f := oov.FieldByName(prop.Name)
		convert(reflect.ValueOf(v), f)
		target.Field(oop.Field).Set(oov)
	}
	// merge extensions, too
	for tag, ext := range proto.RegisteredExtensions(pm) {
		v, ok := m.values[tag]
		if !ok {
			continue
		}
		e := reflect.New(reflect.TypeOf(ext.ExtensionType)).Elem()
		convert(reflect.ValueOf(v), e)
		if err := proto.SetExtension(pm, ext, e.Interface()); err != nil {
			// shouldn't happen since we already checked that the extension type was compatible above
			return err
		}
	}

	// if we have fields that the given message doesn't know about, add to its unknown fields
	if len(unknownTags) > 0 {
		ub := u.Interface().([]byte)
		var b codedBuffer
		for tag := range unknownTags {
			fd := m.FindFieldDescriptor(tag)
			if err := marshalField(tag, fd, m.values[tag], &b); err != nil {
				return err
			}
		}
		ub = append(ub, b.buf...)
		u.Set(reflect.ValueOf(ub))
	}

	// finally, convey unknown fields into the given message by letting it unmarshal them
	// (this will append to its unknown fields if not known; if somehow the given message recognizes
	// a field even though the dynamic message did not, it will get correctly unmarshalled)
	if unknownTags != nil && len(m.unknownFields) > 0 {
		var b codedBuffer
		m.marshalUnknownFields(&b)
		proto.UnmarshalMerge(b.buf, pm)
	}

	return nil
}

func canConvert(src reflect.Value, target reflect.Type) bool {
	srcType := src.Type()
	if target == srcType {
		return true
	} else if target.Kind() == reflect.Ptr && target.Elem() == srcType {
		return true
	} else if target.Kind() == reflect.Slice {
		if srcType.Kind() != reflect.Slice {
			return false
		}
		et := target.Elem()
		for i := 0; i < src.Len(); i++ {
			if !canConvert(src.Index(i), et) {
				return false
			}
		}
		return true
	} else if target.Kind() == reflect.Map {
		if srcType.Kind() != reflect.Map {
			return false
		}
		kt := target.Key()
		vt := target.Elem()
		for _, k := range src.MapKeys() {
			if !canConvert(k, kt) {
				return false
			}
			if !canConvert(src.MapIndex(k), vt) {
				return false
			}
		}
		return true
	} else if srcType == typeOfDynamicMessage && target.Implements(typeOfProtoMessage) {
		z := reflect.Zero(target).Interface()
		msgType := proto.MessageName(z.(proto.Message))
		return msgType == src.Interface().(*Message).GetMessageDescriptor().GetFullyQualifiedName()
	} else {
		return false
	}
}

func convert(src, target reflect.Value) {
	srcType := src.Type()
	targetType := target.Type()
	if targetType == srcType {
		target.Set(src)
	} else if targetType.Kind() == reflect.Ptr && targetType.Elem() == srcType {
		target.Set(src.Addr())
	} else if targetType.Kind() == reflect.Slice {
		l := target.Len()
		newL := l + src.Len()
		if target.Cap() < newL {
			// expand capacity of the slice and copy
			newSl := reflect.MakeSlice(targetType.Elem(), newL, newL)
			for i := 0; i < target.Len(); i++ {
				newSl.Index(i).Set(target.Index(i))
			}
			target.Set(newSl)
		} else {
			target.SetLen(newL)
		}
		for i := 0; i < src.Len(); i++ {
			convert(src.Index(i), target.Index(l+i))
		}
	} else if targetType.Kind() == reflect.Map {
		tkt := targetType.Key()
		tvt := targetType.Elem()
		for _, k := range src.MapKeys() {
			v := src.MapIndex(k)
			skt := k.Type()
			svt := v.Type()
			var nk, nv reflect.Value
			if tkt == skt {
				nk = k
			} else if tkt.Kind() == reflect.Ptr && tkt.Elem() == skt {
				nk = k.Addr()
			} else {
				nk = reflect.New(tkt).Elem()
				convert(k, nk)
			}
			if tvt == svt {
				nv = v
			} else if tvt.Kind() == reflect.Ptr && tvt.Elem() == svt {
				nv = v.Addr()
			} else {
				nv = reflect.New(tvt).Elem()
				convert(v, nv)
			}
			target.SetMapIndex(nk, nv)
		}
	} else if srcType == typeOfDynamicMessage && targetType.Implements(typeOfProtoMessage) {
		dm := src.Interface().(*Message)
		m := target.Interface().(proto.Message)
		dm.ConvertTo(m)
	} else {
		panic(fmt.Sprintf("Cannot convert %v to %v", srcType, targetType))
	}
}

func (m *Message) mergeFrom(pm proto.Message) error {
	if dm, ok := pm.(*Message); ok {
		// if given message is also a dynamic message, we merge differently
		for tag, v := range dm.values {
			fd := m.FindFieldDescriptor(tag)
			if fd == nil {
				fd = dm.FindFieldDescriptor(tag)
			}
			if fd.IsMap() {
				existing := m.values[tag]
				if existing == nil {
					m.internalSetField(fd, v)
				} else {
					emap := existing.(map[interface{}]interface{})
					nmap := v.(map[interface{}]interface{})
					for kk, vv := range nmap {
						emap[kk] = vv
					}
				}
			} else if fd.IsRepeated() {
				existing := m.values[tag]
				if existing == nil {
					m.internalSetField(fd, v)
				} else {
					esl := existing.([]interface{})
					nsl := v.([]interface{})
					esl = append(esl, nsl...)
					m.values[tag] = esl
				}
			} else {
				m.internalSetField(fd, v)
			}
		}
		return nil
	}

	// check that we can successfully do the merge
	src := reflect.ValueOf(pm).Elem()
	values := map[*desc.FieldDescriptor]interface{}{}
	var extraFields []*desc.FieldDescriptor
	props := proto.GetProperties(reflect.TypeOf(pm).Elem())
	if props == nil {
		return fmt.Errorf("Could not determine message properties to merge for %v", reflect.TypeOf(pm).Elem())
	}

	// regular fields
	for _, prop := range props.Prop {
		if prop.Tag == 0 {
			continue // one-of or special field (such as XXX_unrecognized, etc.)
		}
		rv := src.FieldByName(prop.Name)
		var keyProps, valProps *proto.Properties
		if rv.Kind() == reflect.Map {
			sfd, ok := src.Type().Elem().FieldByName(prop.Name)
			if !ok {
				panic(fmt.Sprintf("Could not get field %q for type %v", prop.Name, src.Type().Elem()))
			}
			keyProps = &proto.Properties{}
			valProps = &proto.Properties{}
			// NB: Stinks to have re-parse these every time. Would be nicer if proto.Properties exported
			// its mkeyprop and mvalprop fields!
			keyProps.Parse(sfd.Tag.Get("protobuf_key"))
			valProps.Parse(sfd.Tag.Get("protobuf_val"))
		}
		v := rv.Interface()

		fd := m.FindFieldDescriptor(int32(prop.Tag))
		if fd == nil {
			// Our descriptor has different fields than this message object. So
			// try to reflect on the message object's fields.
			md, err := desc.LoadMessageDescriptorForMessage(pm)
			if err != nil {
				return err
			}
			fd = md.FindFieldByNumber(int32(prop.Tag))
			if fd == nil {
				return fmt.Errorf("Message descriptor %q did not contain field for tag %d (%q)", md.GetFullyQualifiedName(), prop.Tag, prop.Name)
			} else {
				extraFields = append(extraFields, fd)
			}
		}
		if v, err := validFieldValue(fd, v); err != nil {
			return err
		} else {
			values[fd] = v
		}
	}

	// one-of fields
	for _, oop := range props.OneofTypes {
		oov := src.Field(oop.Field)
		if oov.Type() != oop.Type {
			// this field is unset (in other words, one-of message field is not currently set to this option)
			continue
		}
		prop := oop.Prop
		v := oov.FieldByName(prop.Name).Interface()
		fd := m.FindFieldDescriptor(int32(prop.Tag))
		if fd == nil {
			// Our descriptor has different fields than this message object. So
			// try to reflect on the message object's fields.
			md, err := desc.LoadMessageDescriptorForMessage(pm)
			if err != nil {
				return err
			}
			fd = md.FindFieldByNumber(int32(prop.Tag))
			if fd == nil {
				return fmt.Errorf("Message descriptor %q did not contain field for tag %d (%q in one-of %q)", md.GetFullyQualifiedName(), prop.Tag, prop.Name, src.Type().Field(oop.Field).Name)
			} else {
				extraFields = append(extraFields, fd)
			}
		}
		if v, err := validFieldValue(fd, v); err != nil {
			return err
		} else {
			values[fd] = v
		}
	}

	// extension fields
	rexts := proto.RegisteredExtensions(pm)
	if len(rexts) > 0 {
		for tag, ed := range rexts {
			if proto.HasExtension(pm, ed) {
				v, _ := proto.GetExtension(pm, ed)
				if v == nil {
					continue
				}
				var prop proto.Properties
				prop.Parse(ed.Tag)
				fd := m.er.FindExtension(m.md.GetFullyQualifiedName(), tag)
				if fd == nil {
					var err error
					if fd, err = asFieldDescriptor(ed); err != nil {
						return err
					}
					extraFields = append(extraFields, fd)
				}
				if v, err := validFieldValue(fd, v); err != nil {
					return err
				} else {
					values[fd] = v
				}
			}
		}
	}

	// now actually perform the merge
	for fd, v := range values {
		m.internalSetField(fd, v)
	}
	for _, f := range extraFields {
		m.extraFields[f.GetNumber()] = f
	}

	u := src.FieldByName("XXX_unrecognized")
	if u.IsValid() && u.Type() == typeOfBytes {
		// ignore any error returned: pulling in unknown fields is best-effort
		m.UnmarshalMerge(u.Interface().([]byte))
	}
	return nil
}

// Validate checks that all required fields are present. It returns an error if any are absent.
func (m *Message) Validate() error {
	missingFields := []string{}
	for _, fd := range m.md.GetFields() {
		if fd.IsRequired() {
			if _, ok := m.values[fd.GetNumber()]; !ok {
				missingFields = append(missingFields, fd.GetName())
			}
		}
	}
	if len(missingFields) == 0 {
		return nil
	}
	return fmt.Errorf("Some required fields missing: %v", strings.Join(missingFields, ", "))
}

// knownFieldTags return tags of present and recognized fields, in sorted order.
func (m *Message) knownFieldTags() []int {
	if len(m.values) == 0 {
		return []int(nil)
	}

	keys := make([]int, len(m.values))
	i := 0
	for k := range m.values {
		keys[i] = int(k)
		i++
	}

	sort.Ints(keys)
	return keys
}

// allKnownFieldTags return tags of present and recognized fields, including those that are unset, in sorted order.
func (m *Message) allKnownFieldTags() []int {
	fds := m.md.GetFields()
	keys := make([]int, len(fds))

	i := 0
	for _, fd := range fds {
		keys[i] = int(fd.GetNumber())
		i++
	}

	for _, fd := range m.extraFields {
		if !fd.IsExtension() {
			keys = append(keys, int(fd.GetNumber()))
		}
	}

	sort.Ints(keys)
	return keys
}

// unknownFieldTags return tags of present but unrecognized fields, in sorted order.
func (m *Message) unknownFieldTags() []int {
	if len(m.unknownFields) == 0 {
		return []int(nil)
	}
	keys := make([]int, len(m.unknownFields))
	i := 0
	for k := range m.unknownFields {
		keys[i] = int(k)
		i++
	}
	sort.Ints(keys)
	return keys
}
