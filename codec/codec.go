// Package codec contains a reader/write type that assists with encoding
// and decoding protobuf's binary representation.
//
// The code in this package is mostly a fork of proto.Buffer but provides
// additional API to make it more useful to code that needs to dynamically
// process or produce the protobuf binary format.
package codec

import (
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc"
)

// ErrOverflow is returned when an integer is too large to be represented.
var ErrOverflow = errors.New("proto: integer overflow")

// Buffer is a reader and a writer that wraps a slice of bytes and also
// provides API for decoding and encoding the protobuf binary format.
type Buffer struct {
	buf   []byte
	index int
}

// NewBuffer creates a new buffer with the given slice of bytes as the
// buffer's initial contents.
func NewBuffer(buf []byte) *Buffer {
	return &Buffer{buf: buf}
}

// Reset resets this buffer back to empty. Any subsequent writes/encodes
// to the buffer will allocate a new backing slice of bytes.
func (cb *Buffer) Reset() {
	cb.buf = []byte(nil)
	cb.index = 0
}

// Bytes returns the slice of bytes remaining in the buffer. Note that
// this does not perform a copy: if the contents of the returned slice
// are modified, the modifications will be visible to subsequent reads
// via the buffer.
func (cb *Buffer) Bytes() []byte {
	return cb.buf[cb.index:]
}

// String returns the remaining bytes in the buffer as a string.
func (cb *Buffer) String() string {
	return string(cb.Bytes())
}

// EOF returns true if there are no more bytes remaining to read.
func (cb *Buffer) EOF() bool {
	return cb.index >= len(cb.buf)
}

// Skip attempts to skip the given number of bytes in the input. If
// the input has fewer bytes than the given count, false is returned
// and the buffer is unchanged. Otherwise, the given number of bytes
// are skipped and true is returned.
func (cb *Buffer) Skip(count int) error {
	if count < 0 {
		return fmt.Errorf("proto: bad byte length %d", count)
	}
	newIndex := cb.index + count
	if newIndex < cb.index || newIndex > len(cb.buf) {
		return io.ErrUnexpectedEOF
	}
	cb.index = newIndex
	return nil
}

// Len returns the remaining number of bytes in the buffer.
func (cb *Buffer) Len() int {
	return len(cb.buf) - cb.index
}

// Read implements the io.Reader interface. If there are no bytes
// remaining in the buffer, it will return 0, io.EOF. Otherwise,
// it reads max(len(dest), cb.Len()) bytes from input and copies
// them into dest. It returns the number of bytes copied and a nil
// error in this case.
func (cb *Buffer) Read(dest []byte) (int, error) {
	if cb.index == len(cb.buf) {
		return 0, io.EOF
	}
	copied := copy(dest, cb.buf[cb.index:])
	cb.index += copied
	return copied, nil
}

var _ io.Reader = (*Buffer)(nil)

func (cb *Buffer) decodeVarintSlow() (x uint64, err error) {
	i := cb.index
	l := len(cb.buf)

	for shift := uint(0); shift < 64; shift += 7 {
		if i >= l {
			err = io.ErrUnexpectedEOF
			return
		}
		b := cb.buf[i]
		i++
		x |= (uint64(b) & 0x7F) << shift
		if b < 0x80 {
			cb.index = i
			return
		}
	}

	// The number is too large to represent in a 64-bit value.
	err = ErrOverflow
	return
}

// DecodeVarint reads a varint-encoded integer from the Buffer.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
func (cb *Buffer) DecodeVarint() (uint64, error) {
	i := cb.index
	buf := cb.buf

	if i >= len(buf) {
		return 0, io.ErrUnexpectedEOF
	} else if buf[i] < 0x80 {
		cb.index++
		return uint64(buf[i]), nil
	} else if len(buf)-i < 10 {
		return cb.decodeVarintSlow()
	}

	var b uint64
	// we already checked the first byte
	x := uint64(buf[i]) - 0x80
	i++

	b = uint64(buf[i])
	i++
	x += b << 7
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 7

	b = uint64(buf[i])
	i++
	x += b << 14
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 14

	b = uint64(buf[i])
	i++
	x += b << 21
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 21

	b = uint64(buf[i])
	i++
	x += b << 28
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 28

	b = uint64(buf[i])
	i++
	x += b << 35
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 35

	b = uint64(buf[i])
	i++
	x += b << 42
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 42

	b = uint64(buf[i])
	i++
	x += b << 49
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 49

	b = uint64(buf[i])
	i++
	x += b << 56
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 56

	b = uint64(buf[i])
	i++
	x += b << 63
	if b&0x80 == 0 {
		goto done
	}
	// x -= 0x80 << 63 // Always zero.

	return 0, ErrOverflow

done:
	cb.index = i
	return x, nil
}

// DecodeTagAndWireType decodes a field tag and wire type from input.
// This reads a varint and then extracts the two fields from the varint
// value read.
func (cb *Buffer) DecodeTagAndWireType() (tag int32, wireType int8, err error) {
	var v uint64
	v, err = cb.DecodeVarint()
	if err != nil {
		return
	}
	// low 7 bits is wire type
	wireType = int8(v & 7)
	// rest is int32 tag number
	v = v >> 3
	if v > math.MaxInt32 {
		err = fmt.Errorf("tag number out of range: %d", v)
		return
	}
	tag = int32(v)
	return
}

// DecodeFixed64 reads a 64-bit integer from the Buffer.
// This is the format for the
// fixed64, sfixed64, and double protocol buffer types.
func (cb *Buffer) DecodeFixed64() (x uint64, err error) {
	// x, err already 0
	i := cb.index + 8
	if i < 0 || i > len(cb.buf) {
		err = io.ErrUnexpectedEOF
		return
	}
	cb.index = i

	x = uint64(cb.buf[i-8])
	x |= uint64(cb.buf[i-7]) << 8
	x |= uint64(cb.buf[i-6]) << 16
	x |= uint64(cb.buf[i-5]) << 24
	x |= uint64(cb.buf[i-4]) << 32
	x |= uint64(cb.buf[i-3]) << 40
	x |= uint64(cb.buf[i-2]) << 48
	x |= uint64(cb.buf[i-1]) << 56
	return
}

// DecodeFixed32 reads a 32-bit integer from the Buffer.
// This is the format for the
// fixed32, sfixed32, and float protocol buffer types.
func (cb *Buffer) DecodeFixed32() (x uint64, err error) {
	// x, err already 0
	i := cb.index + 4
	if i < 0 || i > len(cb.buf) {
		err = io.ErrUnexpectedEOF
		return
	}
	cb.index = i

	x = uint64(cb.buf[i-4])
	x |= uint64(cb.buf[i-3]) << 8
	x |= uint64(cb.buf[i-2]) << 16
	x |= uint64(cb.buf[i-1]) << 24
	return
}

// DecodeZigZag32 decodes a signed 32-bit integer from the given
// zig-zag encoded value.
func DecodeZigZag32(v uint64) int32 {
	return int32((uint32(v) >> 1) ^ uint32((int32(v&1)<<31)>>31))
}

// DecodeZigZag64 decodes a signed 64-bit integer from the given
// zig-zag encoded value.
func DecodeZigZag64(v uint64) int64 {
	return int64((v >> 1) ^ uint64((int64(v&1)<<63)>>63))
}

// DecodeRawBytes reads a count-delimited byte buffer from the Buffer.
// This is the format used for the bytes protocol buffer
// type and for embedded messages.
func (cb *Buffer) DecodeRawBytes(alloc bool) (buf []byte, err error) {
	n, err := cb.DecodeVarint()
	if err != nil {
		return nil, err
	}

	nb := int(n)
	if nb < 0 {
		return nil, fmt.Errorf("proto: bad byte length %d", nb)
	}
	end := cb.index + nb
	if end < cb.index || end > len(cb.buf) {
		return nil, io.ErrUnexpectedEOF
	}

	if !alloc {
		buf = cb.buf[cb.index:end]
		cb.index = end
		return
	}

	buf = make([]byte, nb)
	copy(buf, cb.buf[cb.index:])
	cb.index = end
	return
}

// ReadGroup reads the input until a "group end" tag is found
// and returns the data up to that point. Subsequent reads from
// the buffer will read data after the group end tag. If alloc
// is true, the data is copied to a new slice before being returned.
// Otherwise, the returned slice is a view into the buffer's
// underlying byte slice.
//
// This function correctly handles nested groups: if a "group start"
// tag is found, then that group's end tag will be included in the
// returned data.
func (cb *Buffer) ReadGroup(alloc bool) ([]byte, error) {
	var groupEnd, dataEnd int
	groupEnd, dataEnd, err := cb.findGroupEnd()
	if err != nil {
		return nil, err
	}
	if !alloc {
		return cb.buf[cb.index:dataEnd], nil
	}
	results := make([]byte, dataEnd-cb.index)
	copy(results, cb.buf[cb.index:])
	cb.index = groupEnd
	return results, nil
}

// SkipGroup is like ReadGroup, except that it discards the
// data and just advances the buffer to point to the input
// right *after* the "group end" tag.
func (cb *Buffer) SkipGroup() error {
	groupEnd, _, err := cb.findGroupEnd()
	if err != nil {
		return err
	}
	cb.index = groupEnd
	return nil
}

func (cb *Buffer) findGroupEnd() (groupEnd int, dataEnd int, err error) {
	bs := cb.buf
	start := cb.index
	defer func() {
		cb.index = start
	}()
	for {
		fieldStart := cb.index
		// read a field tag
		_, wireType, err := cb.DecodeTagAndWireType()
		if err != nil {
			return 0, 0, err
		}
		// skip past the field's data
		switch wireType {
		case proto.WireFixed32:
			if err := cb.Skip(4); err != nil {
				return 0, 0, err
			}
		case proto.WireFixed64:
			if err := cb.Skip(8); err != nil {
				return 0, 0, err
			}
		case proto.WireVarint:
			// skip varint by finding last byte (has high bit unset)
			i := cb.index
			limit := i + 10 // varint cannot be >10 bytes
			for {
				if i >= limit {
					return 0, 0, ErrOverflow
				}
				if i >= len(bs) {
					return 0, 0, io.ErrUnexpectedEOF
				}
				if bs[i]&0x80 == 0 {
					break
				}
				i++
			}
			// TODO: This would only overflow if buffer length was MaxInt and we
			// read the last byte. This is not a real/feasible concern on 64-bit
			// systems. Something to worry about for 32-bit systems? Do we care?
			cb.index++
		case proto.WireBytes:
			l, err := cb.DecodeVarint()
			if err != nil {
				return 0, 0, err
			}
			if err := cb.Skip(int(l)); err != nil {
				return 0, 0, err
			}
		case proto.WireStartGroup:
			if err := cb.SkipGroup(); err != nil {
				return 0, 0, err
			}
		case proto.WireEndGroup:
			return cb.index, fieldStart, nil
		default:
			return 0, 0, proto.ErrInternalBadWireType
		}
	}
}

// Write implements the io.Writer interface. It always returns
// len(data), nil.
func (cb *Buffer) Write(data []byte) (int, error) {
	cb.buf = append(cb.buf, data...)
	return len(data), nil
}

var _ io.Writer = (*Buffer)(nil)

// EncodeVarint writes a varint-encoded integer to the Buffer.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
func (cb *Buffer) EncodeVarint(x uint64) error {
	for x >= 1<<7 {
		cb.buf = append(cb.buf, uint8(x&0x7f|0x80))
		x >>= 7
	}
	cb.buf = append(cb.buf, uint8(x))
	return nil
}

// EncodeTagAndWireType encodes the given field tag and wire type to the
// buffer. This combines the two values and then writes them as a varint.
func (cb *Buffer) EncodeTagAndWireType(tag int32, wireType int8) error {
	v := uint64((int64(tag) << 3) | int64(wireType))
	return cb.EncodeVarint(v)
}

// EncodeFixed64 writes a 64-bit integer to the Buffer.
// This is the format for the
// fixed64, sfixed64, and double protocol buffer types.
func (cb *Buffer) EncodeFixed64(x uint64) error {
	cb.buf = append(cb.buf,
		uint8(x),
		uint8(x>>8),
		uint8(x>>16),
		uint8(x>>24),
		uint8(x>>32),
		uint8(x>>40),
		uint8(x>>48),
		uint8(x>>56))
	return nil
}

// EncodeFixed32 writes a 32-bit integer to the Buffer.
// This is the format for the
// fixed32, sfixed32, and float protocol buffer types.
func (cb *Buffer) EncodeFixed32(x uint64) error {
	cb.buf = append(cb.buf,
		uint8(x),
		uint8(x>>8),
		uint8(x>>16),
		uint8(x>>24))
	return nil
}

// EncodeZigZag64 does zig-zag encoding to convert the given
// signed 64-bit integer into a form that can be expressed
// efficiently as a varint, even for negative values.
func EncodeZigZag64(v int64) uint64 {
	return (uint64(v) << 1) ^ uint64(v>>63)
}

// EncodeZigZag32 does zig-zag encoding to convert the given
// signed 32-bit integer into a form that can be expressed
// efficiently as a varint, even for negative values.
func EncodeZigZag32(v int32) uint64 {
	return uint64((uint32(v) << 1) ^ uint32((v >> 31)))
}

// EncodeRawBytes writes a count-delimited byte buffer to the Buffer.
// This is the format used for the bytes protocol buffer
// type and for embedded messages.
func (cb *Buffer) EncodeRawBytes(b []byte) error {
	if err := cb.EncodeVarint(uint64(len(b))); err != nil {
		return err
	}
	cb.buf = append(cb.buf, b...)
	return nil
}

func (cb *Buffer) EncodeMessage(pm proto.Message) error {
	return cb.encodeMessage(pm, false)
}

func (cb *Buffer) EncodeMessageDeterministic(pm proto.Message) error {
	return cb.encodeMessage(pm, true)
}

func (cb *Buffer) encodeMessage(pm proto.Message, deterministic bool) error {
	bytes, err := proto.Marshal(pm)
	if err != nil {
		return err
	}
	return cb.EncodeRawBytes(bytes)
}

func (cb *Buffer) EncodeFieldValue(fd *desc.FieldDescriptor, val interface{}) error {
	return cb.encodeField(fd, val, false)
}

func (cb *Buffer) EncodeFieldValueDeterministic(fd *desc.FieldDescriptor, val interface{}) error {
	return cb.encodeField(fd, val, true)
}

func (b *Buffer) encodeField(fd *desc.FieldDescriptor, val interface{}, deterministic bool) error {
	if fd.IsMap() {
		mp := val.(map[interface{}]interface{})
		entryType := fd.GetMessageType()
		keyType := entryType.FindFieldByNumber(1)
		valType := entryType.FindFieldByNumber(2)
		var entryBuffer Buffer
		if deterministic {
			keys := make([]interface{}, 0, len(mp))
			for k := range mp {
				keys = append(keys, k)
			}
			sort.Sort(sortable(keys))
			for _, k := range keys {
				v := mp[k]
				entryBuffer.Reset()
				if err := entryBuffer.encodeFieldElement(keyType, k, deterministic); err != nil {
					return err
				}
				if err := entryBuffer.encodeFieldElement(valType, v, deterministic); err != nil {
					return err
				}
				if err := b.EncodeTagAndWireType(fd.GetNumber(), proto.WireBytes); err != nil {
					return err
				}
				if err := b.EncodeRawBytes(entryBuffer.Bytes()); err != nil {
					return err
				}
			}
		} else {
			for k, v := range mp {
				entryBuffer.Reset()
				if err := entryBuffer.encodeFieldElement(keyType, k, deterministic); err != nil {
					return err
				}
				if err := entryBuffer.encodeFieldElement(valType, v, deterministic); err != nil {
					return err
				}
				if err := b.EncodeTagAndWireType(fd.GetNumber(), proto.WireBytes); err != nil {
					return err
				}
				if err := b.EncodeRawBytes(entryBuffer.Bytes()); err != nil {
					return err
				}
			}
		}
		return nil
	} else if fd.IsRepeated() {
		sl := val.([]interface{})
		wt, err := getWireType(fd.GetType())
		if err != nil {
			return err
		}
		if isPacked(fd) && len(sl) > 1 &&
			(wt == proto.WireVarint || wt == proto.WireFixed32 || wt == proto.WireFixed64) {
			// packed repeated field
			var packedBuffer Buffer
			for _, v := range sl {
				if err := packedBuffer.encodeFieldValue(fd, v, deterministic); err != nil {
					return err
				}
			}
			if err := b.EncodeTagAndWireType(fd.GetNumber(), proto.WireBytes); err != nil {
				return err
			}
			return b.EncodeRawBytes(packedBuffer.Bytes())
		} else {
			// non-packed repeated field
			for _, v := range sl {
				if err := b.encodeFieldElement(fd, v, deterministic); err != nil {
					return err
				}
			}
			return nil
		}
	} else {
		return b.encodeFieldElement(fd, val, deterministic)
	}
}

func isPacked(fd *desc.FieldDescriptor) bool {
	opts := fd.AsFieldDescriptorProto().GetOptions()
	// if set, use that value
	if opts != nil && opts.Packed != nil {
		return opts.GetPacked()
	}
	// if unset: proto2 defaults to false, proto3 to true
	return fd.GetFile().IsProto3()
}

// sortable is used to sort map keys. Values will be integers (int32, int64, uint32, and uint64),
// bools, or strings.
type sortable []interface{}

func (s sortable) Len() int {
	return len(s)
}

func (s sortable) Less(i, j int) bool {
	vi := s[i]
	vj := s[j]
	switch reflect.TypeOf(vi).Kind() {
	case reflect.Int32:
		return vi.(int32) < vj.(int32)
	case reflect.Int64:
		return vi.(int64) < vj.(int64)
	case reflect.Uint32:
		return vi.(uint32) < vj.(uint32)
	case reflect.Uint64:
		return vi.(uint64) < vj.(uint64)
	case reflect.String:
		return vi.(string) < vj.(string)
	case reflect.Bool:
		return !vi.(bool) && vj.(bool)
	default:
		panic(fmt.Sprintf("cannot compare keys of type %v", reflect.TypeOf(vi)))
	}
}

func (s sortable) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (b *Buffer) encodeFieldElement(fd *desc.FieldDescriptor, val interface{}, deterministic bool) error {
	wt, err := getWireType(fd.GetType())
	if err != nil {
		return err
	}
	if err := b.EncodeTagAndWireType(fd.GetNumber(), wt); err != nil {
		return err
	}
	if err := b.encodeFieldValue(fd, val, deterministic); err != nil {
		return err
	}
	if wt == proto.WireStartGroup {
		return b.EncodeTagAndWireType(fd.GetNumber(), proto.WireEndGroup)
	}
	return nil
}

type deterministicMarshaler interface {
	MarshalDeterministic() ([]byte, error)
}

func (b *Buffer) encodeFieldValue(fd *desc.FieldDescriptor, val interface{}, deterministic bool) error {
	switch fd.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		v := val.(bool)
		if v {
			return b.EncodeVarint(1)
		} else {
			return b.EncodeVarint(0)
		}

	case descriptor.FieldDescriptorProto_TYPE_ENUM,
		descriptor.FieldDescriptorProto_TYPE_INT32:
		v := val.(int32)
		return b.EncodeVarint(uint64(v))

	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		v := val.(int32)
		return b.EncodeFixed32(uint64(v))

	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		v := val.(int32)
		return b.EncodeVarint(EncodeZigZag32(v))

	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		v := val.(uint32)
		return b.EncodeVarint(uint64(v))

	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		v := val.(uint32)
		return b.EncodeFixed32(uint64(v))

	case descriptor.FieldDescriptorProto_TYPE_INT64:
		v := val.(int64)
		return b.EncodeVarint(uint64(v))

	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		v := val.(int64)
		return b.EncodeFixed64(uint64(v))

	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		v := val.(int64)
		return b.EncodeVarint(EncodeZigZag64(v))

	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		v := val.(uint64)
		return b.EncodeVarint(v)

	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		v := val.(uint64)
		return b.EncodeFixed64(v)

	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		v := val.(float64)
		return b.EncodeFixed64(math.Float64bits(v))

	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		v := val.(float32)
		return b.EncodeFixed32(uint64(math.Float32bits(v)))

	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		v := val.([]byte)
		return b.EncodeRawBytes(v)

	case descriptor.FieldDescriptorProto_TYPE_STRING:
		v := val.(string)
		return b.EncodeRawBytes(([]byte)(v))

	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		return b.encodeMessage(val.(proto.Message), deterministic)

	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		// just append the nested message to this buffer
		var bb []byte
		if dm, ok := val.(deterministicMarshaler); ok && deterministic {
			var err error
			bb, err = dm.MarshalDeterministic()
			if err != nil {
				return err
			}
		} else {
			var err error
			bb, err = proto.Marshal(val.(proto.Message))
			if err != nil {
				return err
			}
		}
		_, err := b.Write(bb)
		return err
		// whosoever writeth start-group tag (e.g. caller) is responsible for writing end-group tag

	default:
		return fmt.Errorf("unrecognized field type: %v", fd.GetType())
	}
}

func getWireType(t descriptor.FieldDescriptorProto_Type) (int8, error) {
	switch t {
	case descriptor.FieldDescriptorProto_TYPE_ENUM,
		descriptor.FieldDescriptorProto_TYPE_BOOL,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_SINT64,
		descriptor.FieldDescriptorProto_TYPE_UINT64:
		return proto.WireVarint, nil

	case descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32,
		descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return proto.WireFixed32, nil

	case descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return proto.WireFixed64, nil

	case descriptor.FieldDescriptorProto_TYPE_BYTES,
		descriptor.FieldDescriptorProto_TYPE_STRING,
		descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		return proto.WireBytes, nil

	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		return proto.WireStartGroup, nil

	default:
		return 0, proto.ErrInternalBadWireType
	}
}