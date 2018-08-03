package dynamic

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

var typeOfGenericSlice = reflect.TypeOf([]interface{}(nil))
var typeOfString = reflect.TypeOf("")
var typeOfGenericMap = reflect.TypeOf(map[interface{}]interface{}(nil))

func canAssign(target, source reflect.Kind) bool {
	if target == reflect.Int64 && (source == reflect.Int32 || source == reflect.Int) {
		return true
	}
	if target == reflect.Uint64 && (source == reflect.Uint32 || source == reflect.Uint) {
		return true
	}
	if target == reflect.Float64 && source == reflect.Float32 {
		return true
	}
	return target == source
}

func TestGetSetClearScalarFields(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.UnaryFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]struct {
		input interface{}
		zero  interface{}
	}{
		reflect.Bool:    {input: true, zero: false},
		reflect.Int32:   {input: int32(-12), zero: int32(0)},
		reflect.Int64:   {input: int64(-1234), zero: int64(0)},
		reflect.Uint32:  {input: uint32(45), zero: uint32(0)},
		reflect.Uint64:  {input: uint64(4567), zero: uint64(0)},
		reflect.Float32: {input: float32(2.718), zero: float32(0)},
		reflect.Float64: {input: float64(3.14159), zero: float64(0)},
		reflect.String:  {input: "foobar", zero: ""},
		reflect.Slice:   {input: []byte("snafu"), zero: []byte(nil)},
	}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Int32, tagNumber: 1, fieldName: "i"},
		{kind: reflect.Int64, tagNumber: 2, fieldName: "j"},
		{kind: reflect.Int32, tagNumber: 3, fieldName: "k"},
		{kind: reflect.Int64, tagNumber: 4, fieldName: "l"},
		{kind: reflect.Uint32, tagNumber: 5, fieldName: "m"},
		{kind: reflect.Uint64, tagNumber: 6, fieldName: "n"},
		{kind: reflect.Uint32, tagNumber: 7, fieldName: "o"},
		{kind: reflect.Uint64, tagNumber: 8, fieldName: "p"},
		{kind: reflect.Int32, tagNumber: 9, fieldName: "q"},
		{kind: reflect.Int64, tagNumber: 10, fieldName: "r"},
		{kind: reflect.Float32, tagNumber: 11, fieldName: "s"},
		{kind: reflect.Float64, tagNumber: 12, fieldName: "t"},
		{kind: reflect.Slice, tagNumber: 13, fieldName: "u"},
		{kind: reflect.String, tagNumber: 14, fieldName: "v"},
		{kind: reflect.Bool, tagNumber: 15, fieldName: "w"},
	}

	for idx, c := range cases {
		zero := inputs[c.kind].zero

		for k, i := range inputs {
			allowed := canAssign(c.kind, k)

			// First run the case using Try* methods

			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			v, err := dm.TryGetFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			testutil.Eq(t, zero, v)
			v, err = dm.TryGetFieldByName(c.fieldName)
			testutil.Ok(t, err)
			testutil.Eq(t, zero, v)

			_, err = dm.TryGetRepeatedFieldByNumber(c.tagNumber, 0)
			testutil.Eq(t, FieldIsNotRepeatedError, err)
			_, err = dm.TryGetRepeatedFieldByName(c.fieldName, 0)
			testutil.Eq(t, FieldIsNotRepeatedError, err)

			err = dm.TrySetRepeatedFieldByNumber(c.tagNumber, 0, i.input)
			testutil.Eq(t, FieldIsNotRepeatedError, err)
			err = dm.TrySetRepeatedFieldByName(c.fieldName, 0, i.input)
			testutil.Eq(t, FieldIsNotRepeatedError, err)

			_, err = dm.TryGetMapFieldByNumber(c.tagNumber, "foo")
			testutil.Eq(t, FieldIsNotMapError, err)
			_, err = dm.TryGetMapFieldByName(c.fieldName, "foo")
			testutil.Eq(t, FieldIsNotMapError, err)

			err = dm.TryPutMapFieldByNumber(c.tagNumber, "foo", i.input)
			testutil.Eq(t, FieldIsNotMapError, err)
			err = dm.TryPutMapFieldByName(c.fieldName, "foo", i.input)
			testutil.Eq(t, FieldIsNotMapError, err)

			err = dm.TryRemoveMapFieldByNumber(c.tagNumber, "foo")
			testutil.Eq(t, FieldIsNotMapError, err)
			err = dm.TryRemoveMapFieldByName(c.fieldName, "foo")
			testutil.Eq(t, FieldIsNotMapError, err)

			err = dm.TrySetFieldByNumber(c.tagNumber, i.input)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err = dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}
			err = dm.TrySetFieldByName(c.fieldName, i.input)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err = dm.TryGetFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = dm.TryClearFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))
			err = dm.TryClearFieldByName(c.fieldName)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

			v, err = dm.TryGetFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			testutil.Eq(t, zero, v)
			v, err = dm.TryGetFieldByName(c.fieldName)
			testutil.Ok(t, err)
			testutil.Eq(t, zero, v)

			// Now we do it again using the non-Try* methods (e.g. the ones that panic)

			v = dm.GetFieldByNumber(c.tagNumber)
			testutil.Eq(t, zero, v)
			v = dm.GetFieldByName(c.fieldName)
			testutil.Eq(t, zero, v)

			err = catchPanic(func() { dm.GetRepeatedFieldByNumber(c.tagNumber, 0) })
			testutil.Eq(t, FieldIsNotRepeatedError.Error(), err.(panicError).panic)
			err = catchPanic(func() { dm.GetRepeatedFieldByName(c.fieldName, 0) })
			testutil.Eq(t, FieldIsNotRepeatedError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.SetRepeatedFieldByNumber(c.tagNumber, 0, i.input) })
			testutil.Eq(t, FieldIsNotRepeatedError.Error(), err.(panicError).panic)
			err = catchPanic(func() { dm.SetRepeatedFieldByName(c.fieldName, 0, i.input) })
			testutil.Eq(t, FieldIsNotRepeatedError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.GetMapFieldByNumber(c.tagNumber, "foo") })
			testutil.Eq(t, FieldIsNotMapError.Error(), err.(panicError).panic)
			err = catchPanic(func() { dm.GetMapFieldByName(c.fieldName, "foo") })
			testutil.Eq(t, FieldIsNotMapError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.PutMapFieldByNumber(c.tagNumber, "foo", i.input) })
			testutil.Eq(t, FieldIsNotMapError.Error(), err.(panicError).panic)
			err = catchPanic(func() { dm.PutMapFieldByName(c.fieldName, "foo", i.input) })
			testutil.Eq(t, FieldIsNotMapError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.RemoveMapFieldByNumber(c.tagNumber, "foo") })
			testutil.Eq(t, FieldIsNotMapError.Error(), err.(panicError).panic)
			err = catchPanic(func() { dm.RemoveMapFieldByName(c.fieldName, "foo") })
			testutil.Eq(t, FieldIsNotMapError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, i.input) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, coerce(i.input, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}
			dm.ClearFieldByNumber(c.tagNumber)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			err = catchPanic(func() { dm.SetFieldByName(c.fieldName, i.input) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v = dm.GetFieldByName(c.fieldName)
				testutil.Eq(t, coerce(i.input, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}
			dm.ClearFieldByName(c.fieldName)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

			v = dm.GetFieldByNumber(c.tagNumber)
			testutil.Eq(t, zero, v)
			v = dm.GetFieldByName(c.fieldName)
			testutil.Eq(t, zero, v)
		}
	}
}

func TestGetSetClearRepeatedFields(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.RepeatedFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]interface{}{
		reflect.Bool:    true,
		reflect.Int32:   int32(-12),
		reflect.Int64:   int64(-1234),
		reflect.Uint32:  uint32(45),
		reflect.Uint64:  uint64(4567),
		reflect.Float32: float32(2.718),
		reflect.Float64: float64(3.14159),
		reflect.String:  "foobar",
		reflect.Slice:   []byte("snafu"),
	}

	sliceKinds := []func(interface{}) interface{}{
		// index 0 will not work since it doesn't return a slice
		func(v interface{}) interface{} {
			return v
		},
		func(v interface{}) interface{} {
			// generic slice
			return []interface{}{v, v, v}
		},
		func(v interface{}) interface{} {
			// slice element type is the same as value type
			sl := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(v)), 3, 3)
			val := reflect.ValueOf(v)
			sl.Index(0).Set(val)
			sl.Index(1).Set(val)
			sl.Index(2).Set(val)
			return sl.Interface()
		},
	}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Int32, tagNumber: 1, fieldName: "i"},
		{kind: reflect.Int64, tagNumber: 2, fieldName: "j"},
		{kind: reflect.Int32, tagNumber: 3, fieldName: "k"},
		{kind: reflect.Int64, tagNumber: 4, fieldName: "l"},
		{kind: reflect.Uint32, tagNumber: 5, fieldName: "m"},
		{kind: reflect.Uint64, tagNumber: 6, fieldName: "n"},
		{kind: reflect.Uint32, tagNumber: 7, fieldName: "o"},
		{kind: reflect.Uint64, tagNumber: 8, fieldName: "p"},
		{kind: reflect.Int32, tagNumber: 9, fieldName: "q"},
		{kind: reflect.Int64, tagNumber: 10, fieldName: "r"},
		{kind: reflect.Float32, tagNumber: 11, fieldName: "s"},
		{kind: reflect.Float64, tagNumber: 12, fieldName: "t"},
		{kind: reflect.Slice, tagNumber: 13, fieldName: "u"},
		{kind: reflect.String, tagNumber: 14, fieldName: "v"},
		{kind: reflect.Bool, tagNumber: 15, fieldName: "w"},
	}

	zero := reflect.Zero(typeOfGenericSlice).Interface()

	for idx, c := range cases {
		for k, i := range inputs {
			allowed := canAssign(c.kind, k)
			for j, sk := range sliceKinds {

				// First run the case using Try* methods

				testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))
				testutil.Require(t, !dm.HasFieldName(c.fieldName))

				v, err := dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)
				v, err = dm.TryGetFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				input := sk(i)
				err = dm.TrySetFieldByNumber(c.tagNumber, input)
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v, err = dm.TryGetFieldByNumber(c.tagNumber)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericSlice, reflect.TypeOf(v))
					testutil.Eq(t, coerceSlice(input, c.kind), v)
					testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
				}
				err = dm.TryClearFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

				err = dm.TrySetFieldByName(c.fieldName, input)
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v, err = dm.TryGetFieldByName(c.fieldName)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericSlice, reflect.TypeOf(v))
					testutil.Eq(t, coerceSlice(input, c.kind), v)
					testutil.Require(t, dm.HasFieldName(c.fieldName))
				}
				err = dm.TryClearFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Require(t, !dm.HasFieldName(c.fieldName))

				v, err = dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)
				v, err = dm.TryGetFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)
				// Now we do it again using the non-Try* methods (e.g. the ones that panic)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)
				v = dm.GetFieldByName(c.fieldName)
				testutil.Eq(t, zero, v)

				err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, input) })
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v = dm.GetFieldByNumber(c.tagNumber)
					testutil.Eq(t, typeOfGenericSlice, reflect.TypeOf(v))
					testutil.Eq(t, coerceSlice(input, c.kind), v)
					testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
				}
				dm.ClearFieldByNumber(c.tagNumber)
				testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

				err = catchPanic(func() { dm.SetFieldByName(c.fieldName, input) })
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v = dm.GetFieldByName(c.fieldName)
					testutil.Eq(t, typeOfGenericSlice, reflect.TypeOf(v))
					testutil.Eq(t, coerceSlice(input, c.kind), v)
					testutil.Require(t, dm.HasFieldName(c.fieldName))
				}
				dm.ClearFieldByName(c.fieldName)
				testutil.Require(t, !dm.HasFieldName(c.fieldName))

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)
				v = dm.GetFieldByName(c.fieldName)
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func TestGetSetAtIndexAddRepeatedFields(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.RepeatedFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]struct {
		input1 interface{}
		input2 interface{}
		zero   interface{}
	}{
		reflect.Bool:    {input1: true, input2: false, zero: false},
		reflect.Int32:   {input1: int32(-12), input2: int32(42), zero: int32(0)},
		reflect.Int64:   {input1: int64(-1234), input2: int64(424242), zero: int64(0)},
		reflect.Uint32:  {input1: uint32(45), input2: uint32(42), zero: uint32(0)},
		reflect.Uint64:  {input1: uint64(4567), input2: uint64(424242), zero: uint64(0)},
		reflect.Float32: {input1: float32(2.718), input2: float32(-3.14159), zero: float32(0)},
		reflect.Float64: {input1: float64(3.14159), input2: float64(-2.718), zero: float64(0)},
		reflect.String:  {input1: "foobar", input2: "snafu", zero: ""},
		reflect.Slice:   {input1: []byte("snafu"), input2: []byte("foobar"), zero: []byte(nil)},
	}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Int32, tagNumber: 1, fieldName: "i"},
		{kind: reflect.Int64, tagNumber: 2, fieldName: "j"},
		{kind: reflect.Int32, tagNumber: 3, fieldName: "k"},
		{kind: reflect.Int64, tagNumber: 4, fieldName: "l"},
		{kind: reflect.Uint32, tagNumber: 5, fieldName: "m"},
		{kind: reflect.Uint64, tagNumber: 6, fieldName: "n"},
		{kind: reflect.Uint32, tagNumber: 7, fieldName: "o"},
		{kind: reflect.Uint64, tagNumber: 8, fieldName: "p"},
		{kind: reflect.Int32, tagNumber: 9, fieldName: "q"},
		{kind: reflect.Int64, tagNumber: 10, fieldName: "r"},
		{kind: reflect.Float32, tagNumber: 11, fieldName: "s"},
		{kind: reflect.Float64, tagNumber: 12, fieldName: "t"},
		{kind: reflect.Slice, tagNumber: 13, fieldName: "u"},
		{kind: reflect.String, tagNumber: 14, fieldName: "v"},
		{kind: reflect.Bool, tagNumber: 15, fieldName: "w"},
	}

	for idx, c := range cases {
		zero := inputs[c.kind].zero

		for k, i := range inputs {
			allowed := canAssign(c.kind, k)

			// First run the case using Try* methods

			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			_, err := dm.TryGetRepeatedFieldByNumber(c.tagNumber, 0)
			testutil.Eq(t, IndexOutOfRangeError, err)
			_, err = dm.TryGetRepeatedFieldByName(c.fieldName, 0)
			testutil.Eq(t, IndexOutOfRangeError, err)

			err = dm.TryAddRepeatedFieldByNumber(c.tagNumber, i.input1)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetRepeatedFieldByNumber(c.tagNumber, 0)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = dm.TryAddRepeatedFieldByNumber(c.tagNumber, i.input2)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetRepeatedFieldByNumber(c.tagNumber, 1)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			var exp interface{}
			if !allowed {
				// adds above failed (because wrong kind), so go ahead and add
				// correct values so we can test Set* methods
				dm.AddRepeatedFieldByNumber(c.tagNumber, zero)
				exp = zero
				dm.AddRepeatedFieldByNumber(c.tagNumber, exp)
			} else {
				exp = coerce(i.input2, c.kind)
			}
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())

			err = dm.TrySetRepeatedFieldByNumber(c.tagNumber, 2, zero)
			testutil.Eq(t, IndexOutOfRangeError, err)
			err = dm.TrySetRepeatedFieldByNumber(c.tagNumber, 0, i.input2)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetRepeatedFieldByNumber(c.tagNumber, 0)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
				// and value at other index is unchanged
				v, err = dm.TryGetRepeatedFieldByNumber(c.tagNumber, 1)
				testutil.Ok(t, err)
				testutil.Eq(t, exp, v)
			}

			err = dm.TrySetRepeatedFieldByNumber(c.tagNumber, 1, i.input1)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetRepeatedFieldByNumber(c.tagNumber, 1)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = dm.TryClearFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			// Now we do it again for the ByName methods
			err = dm.TryAddRepeatedFieldByName(c.fieldName, i.input1)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetRepeatedFieldByName(c.fieldName, 0)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}
			err = dm.TryAddRepeatedFieldByName(c.fieldName, i.input2)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetRepeatedFieldByName(c.fieldName, 1)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			if !allowed {
				// adds above failed (because wrong kind), so go ahead and add
				// correct values so we can test Set* methods
				dm.AddRepeatedFieldByName(c.fieldName, zero)
				exp = zero
				dm.AddRepeatedFieldByName(c.fieldName, exp)
			} else {
				exp = coerce(i.input2, c.kind)
			}
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())

			err = dm.TrySetRepeatedFieldByName(c.fieldName, 2, zero)
			testutil.Eq(t, IndexOutOfRangeError, err)
			err = dm.TrySetRepeatedFieldByName(c.fieldName, 0, i.input2)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetRepeatedFieldByName(c.fieldName, 0)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
				// and value at other index is unchanged
				v, err = dm.TryGetRepeatedFieldByName(c.fieldName, 1)
				testutil.Ok(t, err)
				testutil.Eq(t, exp, v)
			}

			err = dm.TrySetRepeatedFieldByName(c.fieldName, 1, i.input1)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetRepeatedFieldByName(c.fieldName, 1)
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = dm.TryClearFieldByName(c.fieldName)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

			// Now we do it again using the non-Try* methods (e.g. the ones that panic)

			err = catchPanic(func() { dm.GetRepeatedFieldByNumber(c.tagNumber, 0) })
			testutil.Require(t, err != nil)
			testutil.Eq(t, IndexOutOfRangeError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.AddRepeatedFieldByNumber(c.tagNumber, i.input1) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetRepeatedFieldByNumber(c.tagNumber, 0)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = catchPanic(func() { dm.AddRepeatedFieldByNumber(c.tagNumber, i.input2) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetRepeatedFieldByNumber(c.tagNumber, 1)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			if !allowed {
				// adds above failed (because wrong kind), so go ahead and add
				// correct values so we can test Set* methods
				dm.AddRepeatedFieldByNumber(c.tagNumber, zero)
				dm.AddRepeatedFieldByNumber(c.tagNumber, exp)
			}
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())

			err = catchPanic(func() { dm.SetRepeatedFieldByNumber(c.tagNumber, 2, zero) })
			testutil.Require(t, err != nil)
			testutil.Eq(t, IndexOutOfRangeError.Error(), err.(panicError).panic)
			err = catchPanic(func() { dm.SetRepeatedFieldByNumber(c.tagNumber, 0, i.input2) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetRepeatedFieldByNumber(c.tagNumber, 0)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
				// and value at other index is unchanged
				v = dm.GetRepeatedFieldByNumber(c.tagNumber, 1)
				testutil.Eq(t, exp, v)
			}

			err = catchPanic(func() { dm.SetRepeatedFieldByNumber(c.tagNumber, 1, i.input1) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetRepeatedFieldByNumber(c.tagNumber, 1)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			dm.ClearFieldByNumber(c.tagNumber)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			// Again - Non Try methds by Name
			err = catchPanic(func() { dm.GetRepeatedFieldByName(c.fieldName, 0) })
			testutil.Require(t, err != nil)
			testutil.Eq(t, IndexOutOfRangeError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.AddRepeatedFieldByName(c.fieldName, i.input1) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetRepeatedFieldByName(c.fieldName, 0)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = catchPanic(func() { dm.AddRepeatedFieldByName(c.fieldName, i.input2) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetRepeatedFieldByName(c.fieldName, 1)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			if !allowed {
				// adds above failed (because wrong kind), so go ahead and add
				// correct values so we can test Set* methods
				dm.AddRepeatedFieldByName(c.fieldName, zero)
				dm.AddRepeatedFieldByName(c.fieldName, exp)
			}
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())

			err = catchPanic(func() { dm.SetRepeatedFieldByName(c.fieldName, 2, zero) })
			testutil.Require(t, err != nil)
			testutil.Eq(t, IndexOutOfRangeError.Error(), err.(panicError).panic)
			err = catchPanic(func() { dm.SetRepeatedFieldByName(c.fieldName, 0, i.input2) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetRepeatedFieldByName(c.fieldName, 0)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
				// and value at other index is unchanged
				v = dm.GetRepeatedFieldByName(c.fieldName, 1)
				testutil.Eq(t, exp, v)
			}

			err = catchPanic(func() { dm.SetRepeatedFieldByName(c.fieldName, 1, i.input1) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetRepeatedFieldByName(c.fieldName, 1)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			dm.ClearFieldByName(c.fieldName)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))
		}
	}
}

func TestGetSetClearMapFields_KeyTypes(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.MapKeyFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]interface{}{
		reflect.Bool:   true,
		reflect.Int32:  int32(-12),
		reflect.Int64:  int64(-1234),
		reflect.Uint32: uint32(45),
		reflect.Uint64: uint64(4567),
		reflect.String: "foobar",
	}

	mapKinds := []func(interface{}) interface{}{
		// index 0 will not work since it doesn't return a map
		func(v interface{}) interface{} {
			return v
		},
		func(v interface{}) interface{} {
			// generic map
			return map[interface{}]interface{}{v: "foo"}
		},
		func(v interface{}) interface{} {
			// specific key and value types
			mp := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(v), typeOfString))
			val := reflect.ValueOf(v)
			mp.SetMapIndex(val, reflect.ValueOf("foo"))
			return mp.Interface()
		},
	}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Int32, tagNumber: 1, fieldName: "i"},
		{kind: reflect.Int64, tagNumber: 2, fieldName: "j"},
		{kind: reflect.Int32, tagNumber: 3, fieldName: "k"},
		{kind: reflect.Int64, tagNumber: 4, fieldName: "l"},
		{kind: reflect.Uint32, tagNumber: 5, fieldName: "m"},
		{kind: reflect.Uint64, tagNumber: 6, fieldName: "n"},
		{kind: reflect.Uint32, tagNumber: 7, fieldName: "o"},
		{kind: reflect.Uint64, tagNumber: 8, fieldName: "p"},
		{kind: reflect.Int32, tagNumber: 9, fieldName: "q"},
		{kind: reflect.Int64, tagNumber: 10, fieldName: "r"},
		{kind: reflect.String, tagNumber: 11, fieldName: "s"},
		{kind: reflect.Bool, tagNumber: 12, fieldName: "t"},
	}

	zero := reflect.Zero(typeOfGenericMap).Interface()

	for idx, c := range cases {
		for k, i := range inputs {
			allowed := canAssign(c.kind, k)
			for j, mk := range mapKinds {
				// First run the case using Try* methods

				testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

				v, err := dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)
				v, err = dm.TryGetFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				input := mk(i)
				err = dm.TrySetFieldByNumber(c.tagNumber, input)
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v, err = dm.TryGetFieldByNumber(c.tagNumber)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, coerceMapKeys(input, c.kind), v)
					testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
				}
				err = dm.TryClearFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))
				v, err = dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				err = dm.TrySetFieldByName(c.fieldName, input)
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v, err = dm.TryGetFieldByName(c.fieldName)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, coerceMapKeys(input, c.kind), v)
					testutil.Require(t, dm.HasFieldName(c.fieldName))
				}
				err = dm.TryClearFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Require(t, !dm.HasFieldName(c.fieldName))
				v, err = dm.TryGetFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				// Now we do it again using the non-Try* methods (e.g. the ones that panic)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)
				v = dm.GetFieldByName(c.fieldName)
				testutil.Eq(t, zero, v)

				err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, input) })
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v = dm.GetFieldByNumber(c.tagNumber)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, coerceMapKeys(input, c.kind), v)
					testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
				}
				dm.ClearFieldByNumber(c.tagNumber)
				testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))
				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)

				err = catchPanic(func() { dm.SetFieldByName(c.fieldName, input) })
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v = dm.GetFieldByNumber(c.tagNumber)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, coerceMapKeys(input, c.kind), v)
					testutil.Require(t, dm.HasFieldName(c.fieldName))
				}
				dm.ClearFieldByName(c.fieldName)
				testutil.Require(t, !dm.HasFieldName(c.fieldName))
				v = dm.GetFieldByName(c.fieldName)
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func TestGetSetClearMapFields_ValueTypes(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.MapValFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]interface{}{
		reflect.Bool:    true,
		reflect.Int32:   int32(-12),
		reflect.Int64:   int64(-1234),
		reflect.Uint32:  uint32(45),
		reflect.Uint64:  uint64(4567),
		reflect.Float32: float32(2.718),
		reflect.Float64: float64(3.14159),
		reflect.String:  "foobar",
		reflect.Slice:   []byte("snafu"),
	}

	mapKinds := []func(interface{}) interface{}{
		// index 0 will not work since it doesn't return a map
		func(v interface{}) interface{} {
			return v
		},
		func(v interface{}) interface{} {
			// generic slice
			return map[interface{}]interface{}{"foo": v, "bar": v, "baz": v}
		},
		func(v interface{}) interface{} {
			// specific key and value types
			mp := reflect.MakeMap(reflect.MapOf(typeOfString, reflect.TypeOf(v)))
			val := reflect.ValueOf(v)
			mp.SetMapIndex(reflect.ValueOf("foo"), val)
			mp.SetMapIndex(reflect.ValueOf("bar"), val)
			mp.SetMapIndex(reflect.ValueOf("baz"), val)
			return mp.Interface()
		},
	}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Int32, tagNumber: 1, fieldName: "i"},
		{kind: reflect.Int64, tagNumber: 2, fieldName: "j"},
		{kind: reflect.Int32, tagNumber: 3, fieldName: "k"},
		{kind: reflect.Int64, tagNumber: 4, fieldName: "l"},
		{kind: reflect.Uint32, tagNumber: 5, fieldName: "m"},
		{kind: reflect.Uint64, tagNumber: 6, fieldName: "n"},
		{kind: reflect.Uint32, tagNumber: 7, fieldName: "o"},
		{kind: reflect.Uint64, tagNumber: 8, fieldName: "p"},
		{kind: reflect.Int32, tagNumber: 9, fieldName: "q"},
		{kind: reflect.Int64, tagNumber: 10, fieldName: "r"},
		{kind: reflect.Float32, tagNumber: 11, fieldName: "s"},
		{kind: reflect.Float64, tagNumber: 12, fieldName: "t"},
		{kind: reflect.Slice, tagNumber: 13, fieldName: "u"},
		{kind: reflect.String, tagNumber: 14, fieldName: "v"},
		{kind: reflect.Bool, tagNumber: 15, fieldName: "w"},
	}

	zero := reflect.Zero(typeOfGenericMap).Interface()

	for idx, c := range cases {
		for k, i := range inputs {
			allowed := canAssign(c.kind, k)
			for j, mk := range mapKinds {
				// First run the case using Try* methods

				v, err := dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)
				v, err = dm.TryGetFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				input := mk(i)
				err = dm.TrySetFieldByNumber(c.tagNumber, input)
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v, err = dm.TryGetFieldByNumber(c.tagNumber)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, coerceMapVals(input, c.kind), v)
				}

				err = dm.TryClearFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)

				v, err = dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				err = dm.TrySetFieldByName(c.fieldName, input)
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v, err = dm.TryGetFieldByName(c.fieldName)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, coerceMapVals(input, c.kind), v)
				}

				err = dm.TryClearFieldByName(c.fieldName)
				testutil.Ok(t, err)

				v, err = dm.TryGetFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)
				// Now we do it again using the non-Try* methods (e.g. the ones that panic)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)
				v = dm.GetFieldByName(c.fieldName)
				testutil.Eq(t, zero, v)

				err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, input) })
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v = dm.GetFieldByNumber(c.tagNumber)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, coerceMapVals(input, c.kind), v)
				}
				dm.ClearFieldByNumber(c.tagNumber)
				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)

				err = catchPanic(func() { dm.SetFieldByName(c.fieldName, input) })
				if shouldTestValue(t, err, j != 0 && allowed, k, c.kind, idx) {
					// make sure value stuck
					v = dm.GetFieldByName(c.fieldName)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, coerceMapVals(input, c.kind), v)
				}
				dm.ClearFieldByName(c.fieldName)
				v = dm.GetFieldByName(c.fieldName)
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func TestGetPutDeleteMapFields(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.MapValFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]struct {
		input1 interface{}
		input2 interface{}
		zero   interface{}
	}{
		reflect.Bool:    {input1: true, input2: false, zero: false},
		reflect.Int32:   {input1: int32(-12), input2: int32(42), zero: int32(0)},
		reflect.Int64:   {input1: int64(-1234), input2: int64(424242), zero: int64(0)},
		reflect.Uint32:  {input1: uint32(45), input2: uint32(42), zero: uint32(0)},
		reflect.Uint64:  {input1: uint64(4567), input2: uint64(424242), zero: uint64(0)},
		reflect.Float32: {input1: float32(2.718), input2: float32(-3.14159), zero: float32(0)},
		reflect.Float64: {input1: float64(3.14159), input2: float64(-2.718), zero: float64(0)},
		reflect.String:  {input1: "foobar", input2: "snafu", zero: ""},
		reflect.Slice:   {input1: []byte("snafu"), input2: []byte("foobar"), zero: []byte(nil)},
	}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Int32, tagNumber: 1, fieldName: "i"},
		{kind: reflect.Int64, tagNumber: 2, fieldName: "j"},
		{kind: reflect.Int32, tagNumber: 3, fieldName: "k"},
		{kind: reflect.Int64, tagNumber: 4, fieldName: "l"},
		{kind: reflect.Uint32, tagNumber: 5, fieldName: "m"},
		{kind: reflect.Uint64, tagNumber: 6, fieldName: "n"},
		{kind: reflect.Uint32, tagNumber: 7, fieldName: "o"},
		{kind: reflect.Uint64, tagNumber: 8, fieldName: "p"},
		{kind: reflect.Int32, tagNumber: 9, fieldName: "q"},
		{kind: reflect.Int64, tagNumber: 10, fieldName: "r"},
		{kind: reflect.Float32, tagNumber: 11, fieldName: "s"},
		{kind: reflect.Float64, tagNumber: 12, fieldName: "t"},
		{kind: reflect.Slice, tagNumber: 13, fieldName: "u"},
		{kind: reflect.String, tagNumber: 14, fieldName: "v"},
		{kind: reflect.Bool, tagNumber: 15, fieldName: "w"},
	}

	for idx, c := range cases {
		zero := inputs[c.kind].zero

		for k, i := range inputs {
			allowed := canAssign(c.kind, k)

			// First run the case using Try* methods

			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			v, err := dm.TryGetMapFieldByNumber(c.tagNumber, "foo")
			testutil.Ok(t, err)
			testutil.Require(t, v == nil)

			err = dm.TryPutMapFieldByNumber(c.tagNumber, "foo", i.input1)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err = dm.TryGetMapFieldByNumber(c.tagNumber, "foo")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = dm.TryPutMapFieldByNumber(c.tagNumber, "bar", i.input2)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err = dm.TryGetMapFieldByNumber(c.tagNumber, "bar")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
			}

			var exp interface{}
			if !allowed {
				// adds above failed (because wrong kind), so go ahead and add
				// correct values so we can test Set* methods
				dm.PutMapFieldByNumber(c.tagNumber, "foo", zero)
				exp = zero
				dm.PutMapFieldByNumber(c.tagNumber, "bar", exp)
			} else {
				exp = coerce(i.input2, c.kind)
			}
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())

			// removing missing key is not an error
			err = dm.TryRemoveMapFieldByNumber(c.tagNumber, "baz")
			testutil.Ok(t, err)
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())

			err = dm.TryRemoveMapFieldByNumber(c.tagNumber, "foo")
			testutil.Ok(t, err)
			testutil.Eq(t, 1, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())
			// value has been deleted
			v, err = dm.TryGetMapFieldByNumber(c.tagNumber, "foo")
			testutil.Ok(t, err)
			testutil.Require(t, v == nil)
			// other key not affected
			v, err = dm.TryGetMapFieldByNumber(c.tagNumber, "bar")
			testutil.Ok(t, err)
			testutil.Eq(t, exp, v)

			err = dm.TryRemoveMapFieldByNumber(c.tagNumber, "bar")
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))
			testutil.Eq(t, 0, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())

			// Now run the try methods using ByName methods
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

			v, err = dm.TryGetMapFieldByName(c.fieldName, "foo")
			testutil.Ok(t, err)
			testutil.Require(t, v == nil)

			err = dm.TryPutMapFieldByName(c.fieldName, "foo", i.input1)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err = dm.TryGetMapFieldByName(c.fieldName, "foo")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = dm.TryPutMapFieldByName(c.fieldName, "bar", i.input2)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err = dm.TryGetMapFieldByName(c.fieldName, "bar")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
			}

			if !allowed {
				// adds above failed (because wrong kind), so go ahead and add
				// correct values so we can test Set* methods
				dm.PutMapFieldByName(c.fieldName, "foo", zero)
				exp = zero
				dm.PutMapFieldByName(c.fieldName, "bar", exp)
			} else {
				exp = coerce(i.input2, c.kind)
			}
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())

			// removing missing key is not an error
			err = dm.TryRemoveMapFieldByName(c.fieldName, "baz")
			testutil.Ok(t, err)
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())

			err = dm.TryRemoveMapFieldByName(c.fieldName, "foo")
			testutil.Ok(t, err)
			testutil.Eq(t, 1, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())
			// value has been deleted
			v, err = dm.TryGetMapFieldByName(c.fieldName, "foo")
			testutil.Ok(t, err)
			testutil.Require(t, v == nil)
			// other key not affected
			v, err = dm.TryGetMapFieldByName(c.fieldName, "bar")
			testutil.Ok(t, err)
			testutil.Eq(t, exp, v)

			err = dm.TryRemoveMapFieldByName(c.fieldName, "bar")
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))
			testutil.Eq(t, 0, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())

			// Now we do it again using the non-Try* methods (e.g. the ones that panic)

			v = dm.GetMapFieldByNumber(c.tagNumber, "foo")
			testutil.Require(t, v == nil)

			err = catchPanic(func() { dm.PutMapFieldByNumber(c.tagNumber, "foo", i.input1) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v = dm.GetMapFieldByNumber(c.tagNumber, "foo")
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = catchPanic(func() { dm.PutMapFieldByNumber(c.tagNumber, "bar", i.input2) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v = dm.GetMapFieldByNumber(c.tagNumber, "bar")
				testutil.Eq(t, coerce(i.input2, c.kind), v)
			}

			if !allowed {
				// adds above failed (because wrong kind), so go ahead and add
				// correct values so we can test Set* methods
				dm.PutMapFieldByNumber(c.tagNumber, "foo", zero)
				dm.PutMapFieldByNumber(c.tagNumber, "bar", exp)
			}
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())

			// removing missing key does not panic
			err = catchPanic(func() { dm.RemoveMapFieldByNumber(c.tagNumber, "baz") })
			testutil.Ok(t, err)
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())

			err = catchPanic(func() { dm.RemoveMapFieldByNumber(c.tagNumber, "foo") })
			testutil.Ok(t, err)
			testutil.Eq(t, 1, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())
			// value has been deleted
			v = dm.GetMapFieldByNumber(c.tagNumber, "foo")
			testutil.Require(t, v == nil)
			// other key not affected
			v = dm.GetMapFieldByNumber(c.tagNumber, "bar")
			testutil.Eq(t, exp, v)

			err = catchPanic(func() { dm.RemoveMapFieldByNumber(c.tagNumber, "bar") })
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))
			testutil.Eq(t, 0, reflect.ValueOf(dm.GetFieldByNumber(c.tagNumber)).Len())

			// Now run the non-try methods using byName methods
			v = dm.GetMapFieldByName(c.fieldName, "foo")
			testutil.Require(t, v == nil)

			err = catchPanic(func() { dm.PutMapFieldByName(c.fieldName, "foo", i.input1) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v = dm.GetMapFieldByName(c.fieldName, "foo")
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = catchPanic(func() { dm.PutMapFieldByName(c.fieldName, "bar", i.input2) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v = dm.GetMapFieldByName(c.fieldName, "bar")
				testutil.Eq(t, coerce(i.input2, c.kind), v)
			}

			if !allowed {
				// adds above failed (because wrong kind), so go ahead and add
				// correct values so we can test Set* methods
				dm.PutMapFieldByName(c.fieldName, "foo", zero)
				dm.PutMapFieldByName(c.fieldName, "bar", exp)
			}
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())

			// removing missing key does not panic
			err = catchPanic(func() { dm.RemoveMapFieldByName(c.fieldName, "baz") })
			testutil.Ok(t, err)
			testutil.Eq(t, 2, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())

			err = catchPanic(func() { dm.RemoveMapFieldByName(c.fieldName, "foo") })
			testutil.Ok(t, err)
			testutil.Eq(t, 1, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())
			// value has been deleted
			v = dm.GetMapFieldByName(c.fieldName, "foo")
			testutil.Require(t, v == nil)
			// other key not affected
			v = dm.GetMapFieldByName(c.fieldName, "bar")
			testutil.Eq(t, exp, v)

			err = catchPanic(func() { dm.RemoveMapFieldByName(c.fieldName, "bar") })
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))
			testutil.Eq(t, 0, reflect.ValueOf(dm.GetFieldByName(c.fieldName)).Len())
		}
	}
}

func TestMapFields_AsIfRepeatedFieldOfEntries(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.MapValFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]struct {
		input1 interface{}
		input2 interface{}
	}{
		reflect.Bool:    {input1: true, input2: false},
		reflect.Int32:   {input1: int32(-12), input2: int32(42)},
		reflect.Int64:   {input1: int64(-1234), input2: int64(424242)},
		reflect.Uint32:  {input1: uint32(45), input2: uint32(42)},
		reflect.Uint64:  {input1: uint64(4567), input2: uint64(424242)},
		reflect.Float32: {input1: float32(2.718), input2: float32(-3.14159)},
		reflect.Float64: {input1: float64(3.14159), input2: float64(-2.718)},
		reflect.String:  {input1: "foobar", input2: "snafu"},
		reflect.Slice:   {input1: []byte("snafu"), input2: []byte("foobar")},
	}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Int32, tagNumber: 1, fieldName: "i"},
		{kind: reflect.Int64, tagNumber: 2, fieldName: "j"},
		{kind: reflect.Int32, tagNumber: 3, fieldName: "k"},
		{kind: reflect.Int64, tagNumber: 4, fieldName: "l"},
		{kind: reflect.Uint32, tagNumber: 5, fieldName: "m"},
		{kind: reflect.Uint64, tagNumber: 6, fieldName: "n"},
		{kind: reflect.Uint32, tagNumber: 7, fieldName: "o"},
		{kind: reflect.Uint64, tagNumber: 8, fieldName: "p"},
		{kind: reflect.Int32, tagNumber: 9, fieldName: "q"},
		{kind: reflect.Int64, tagNumber: 10, fieldName: "r"},
		{kind: reflect.Float32, tagNumber: 11, fieldName: "s"},
		{kind: reflect.Float64, tagNumber: 12, fieldName: "t"},
		{kind: reflect.Slice, tagNumber: 13, fieldName: "u"},
		{kind: reflect.String, tagNumber: 14, fieldName: "v"},
		{kind: reflect.Bool, tagNumber: 15, fieldName: "w"},
	}

	for idx, c := range cases {
		// instead of iterating through all of the possible input types, we
		// just grab a couple via index into cases (so we can easily use the
		// tagNumber to build an appropriate entry message)
		var i1, i2 int
		if idx == 0 {
			i1 = len(cases) - 1
		} else {
			i1 = idx - 1
		}
		if idx == len(cases)-1 {
			i2 = 0
		} else {
			i2 = idx + 1
		}

		for _, jdx := range []int{i1, idx, i2} {
			k := cases[jdx].kind
			i := inputs[k]

			mdEntry := md.FindFieldByNumber(int32(cases[jdx].tagNumber)).GetMessageType()
			input1 := NewMessage(mdEntry)
			input1.SetFieldByNumber(1, "foo")
			input1.SetFieldByNumber(2, i.input1)
			input2 := NewMessage(mdEntry)
			input2.SetFieldByNumber(1, "bar")
			input2.SetFieldByNumber(2, i.input2)

			// we don't use canAssign because even though type of c.kind might be assignable to k, the
			// map entry types are messages which are not assignable
			allowed := c.kind == k //canAssign(c.kind, k)

			// First run the case using Try* methods

			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			_, err := dm.TryGetRepeatedFieldByNumber(c.tagNumber, 0)
			testutil.Eq(t, FieldIsNotRepeatedError, err)

			err = dm.TryAddRepeatedFieldByNumber(c.tagNumber, input1)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetMapFieldByNumber(c.tagNumber, "foo")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = dm.TryAddRepeatedFieldByNumber(c.tagNumber, input2)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetMapFieldByNumber(c.tagNumber, "bar")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = dm.TrySetRepeatedFieldByNumber(c.tagNumber, 0, input2)
			testutil.Eq(t, FieldIsNotRepeatedError, err)

			err = dm.TryClearFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			err = dm.TrySetFieldByNumber(c.tagNumber, []interface{}{input1, input2})
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure values stuck
				v, err := dm.TryGetMapFieldByNumber(c.tagNumber, "foo")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				v, err = dm.TryGetMapFieldByNumber(c.tagNumber, "bar")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = dm.TryClearFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			// Now use the try ByName methods
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

			_, err = dm.TryGetRepeatedFieldByName(c.fieldName, 0)
			testutil.Eq(t, FieldIsNotRepeatedError, err)

			err = dm.TryAddRepeatedFieldByName(c.fieldName, input1)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetMapFieldByName(c.fieldName, "foo")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = dm.TryAddRepeatedFieldByName(c.fieldName, input2)
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v, err := dm.TryGetMapFieldByName(c.fieldName, "bar")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = dm.TrySetRepeatedFieldByName(c.fieldName, 0, input2)
			testutil.Eq(t, FieldIsNotRepeatedError, err)

			err = dm.TryClearFieldByName(c.fieldName)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

			err = dm.TrySetFieldByName(c.fieldName, []interface{}{input1, input2})
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure values stuck
				v, err := dm.TryGetMapFieldByName(c.fieldName, "foo")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				v, err = dm.TryGetMapFieldByName(c.fieldName, "bar")
				testutil.Ok(t, err)
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = dm.TryClearFieldByName(c.fieldName)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

			// Now we do it again using the non-Try* methods (e.g. the ones that panic)

			err = catchPanic(func() { dm.GetRepeatedFieldByNumber(c.tagNumber, 0) })
			testutil.Eq(t, FieldIsNotRepeatedError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.AddRepeatedFieldByNumber(c.tagNumber, input1) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetMapFieldByNumber(c.tagNumber, "foo")
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = catchPanic(func() { dm.AddRepeatedFieldByNumber(c.tagNumber, input2) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetMapFieldByNumber(c.tagNumber, "bar")
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			err = catchPanic(func() { dm.SetRepeatedFieldByNumber(c.tagNumber, 0, input2) })
			testutil.Eq(t, FieldIsNotRepeatedError.Error(), err.(panicError).panic)

			dm.ClearFieldByNumber(c.tagNumber)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, []interface{}{input1, input2}) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure values stuck
				v := dm.GetMapFieldByNumber(c.tagNumber, "foo")
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				v = dm.GetMapFieldByNumber(c.tagNumber, "bar")
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldNumber(c.tagNumber))
			}

			dm.ClearFieldByNumber(c.tagNumber)
			testutil.Require(t, !dm.HasFieldNumber(c.tagNumber))

			// Now the non-try ByName methods
			err = catchPanic(func() { dm.GetRepeatedFieldByName(c.fieldName, 0) })
			testutil.Eq(t, FieldIsNotRepeatedError.Error(), err.(panicError).panic)

			err = catchPanic(func() { dm.AddRepeatedFieldByName(c.fieldName, input1) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetMapFieldByName(c.fieldName, "foo")
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = catchPanic(func() { dm.AddRepeatedFieldByName(c.fieldName, input2) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure value stuck
				v := dm.GetMapFieldByName(c.fieldName, "bar")
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			err = catchPanic(func() { dm.SetRepeatedFieldByName(c.fieldName, 0, input2) })
			testutil.Eq(t, FieldIsNotRepeatedError.Error(), err.(panicError).panic)

			dm.ClearFieldByName(c.fieldName)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

			err = catchPanic(func() { dm.SetFieldByName(c.fieldName, []interface{}{input1, input2}) })
			if shouldTestValue(t, err, allowed, k, c.kind, idx) {
				// make sure values stuck
				v := dm.GetMapFieldByName(c.fieldName, "foo")
				testutil.Eq(t, coerce(i.input1, c.kind), v)
				v = dm.GetMapFieldByName(c.fieldName, "bar")
				testutil.Eq(t, coerce(i.input2, c.kind), v)
				testutil.Require(t, dm.HasFieldName(c.fieldName))
			}

			dm.ClearFieldByNumber(c.tagNumber)
			testutil.Require(t, !dm.HasFieldName(c.fieldName))

		}
	}
}

func shouldTestValue(t *testing.T, err error, expectOk bool, sk, tk reflect.Kind, idx int) bool {
	if expectOk && err != nil {
		t.Errorf("Not expecting an error assigning a %v to a %v (case #%d): %s", sk, tk, idx, err.Error())
		return false
	} else if !expectOk && err == nil {
		t.Errorf("Expecting an error assigning a %v to a %v (case #%d)", sk, tk, idx)
		return false
	} else {
		return expectOk
	}
}

func coerce(v interface{}, k reflect.Kind) interface{} {
	switch k {
	case reflect.Int64:
		return reflect.ValueOf(v).Int()
	case reflect.Uint64:
		return reflect.ValueOf(v).Uint()
	case reflect.Float64:
		return reflect.ValueOf(v).Float()
	default:
		return v
	}
}

func coerceSlice(v interface{}, k reflect.Kind) interface{} {
	switch k {
	case reflect.Int64:
		rv := reflect.ValueOf(v)
		sl := make([]int64, rv.Len())
		for i := range sl {
			sl[i] = reflect.ValueOf(rv.Index(i).Interface()).Int()
		}
		return sl
	case reflect.Uint64:
		rv := reflect.ValueOf(v)
		sl := make([]uint64, rv.Len())
		for i := range sl {
			sl[i] = reflect.ValueOf(rv.Index(i).Interface()).Uint()
		}
		return sl
	case reflect.Float64:
		rv := reflect.ValueOf(v)
		sl := make([]float64, rv.Len())
		for i := range sl {
			sl[i] = reflect.ValueOf(rv.Index(i).Interface()).Float()
		}
		return sl
	default:
		return v
	}
}

func coerceMapKeys(v interface{}, k reflect.Kind) interface{} {
	switch k {
	case reflect.Int64:
		rv := reflect.ValueOf(v)
		m := make(map[int64]interface{}, rv.Len())
		for _, key := range rv.MapKeys() {
			val := rv.MapIndex(key)
			m[reflect.ValueOf(key.Interface()).Int()] = val.Interface()
		}
		return m
	case reflect.Uint64:
		rv := reflect.ValueOf(v)
		m := make(map[uint64]interface{}, rv.Len())
		for _, key := range rv.MapKeys() {
			val := rv.MapIndex(key)
			m[reflect.ValueOf(key.Interface()).Uint()] = val.Interface()
		}
		return m
	// no case for Float64 because map keys can't be floats
	default:
		return v
	}
}

func coerceMapVals(v interface{}, k reflect.Kind) interface{} {
	switch k {
	case reflect.Int64:
		rv := reflect.ValueOf(v)
		m := make(map[interface{}]int64, rv.Len())
		for _, key := range rv.MapKeys() {
			val := rv.MapIndex(key)
			m[key.Interface()] = reflect.ValueOf(val.Interface()).Int()
		}
		return m
	case reflect.Uint64:
		rv := reflect.ValueOf(v)
		m := make(map[interface{}]uint64, rv.Len())
		for _, key := range rv.MapKeys() {
			val := rv.MapIndex(key)
			m[key.Interface()] = reflect.ValueOf(val.Interface()).Uint()
		}
		return m
	case reflect.Float64:
		rv := reflect.ValueOf(v)
		m := make(map[interface{}]float64, rv.Len())
		for _, key := range rv.MapKeys() {
			val := rv.MapIndex(key)
			m[key.Interface()] = reflect.ValueOf(val.Interface()).Float()
		}
		return m
	default:
		return v
	}
}

func TestGetSetExtensionFields(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.AnotherTestMessage").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]struct {
		input interface{}
		zero  interface{}
	}{
		reflect.Ptr: {
			input: &testprotos.TestMessage{Ne: []testprotos.TestMessage_NestedEnum{testprotos.TestMessage_VALUE1}},
			zero:  (*testprotos.TestMessage)(nil),
		},
		reflect.Int32:  {input: int32(-12), zero: int32(0)},
		reflect.Uint64: {input: uint64(4567), zero: uint64(0)},
		reflect.String: {input: "foobar", zero: ""},
		reflect.Slice:  {input: []bool{true, false, true, false, true}, zero: []bool(nil)}}

	cases := []struct {
		kind  reflect.Kind
		extfd *desc.FieldDescriptor
	}{
		{kind: reflect.Ptr, extfd: loadExtension(t, testprotos.E_Xtm)},
		{kind: reflect.Int32, extfd: loadExtension(t, testprotos.E_Xi)},
		{kind: reflect.Uint64, extfd: loadExtension(t, testprotos.E_Xui)},
		{kind: reflect.String, extfd: loadExtension(t, testprotos.E_Xs)},
		{kind: reflect.Slice, extfd: loadExtension(t, testprotos.E_TestMessage_NestedMessage_AnotherNestedMessage_Flags)},
	}

	for _, c := range cases {
		zero := inputs[c.kind].zero

		for k, i := range inputs {
			// First run the case using Try* methods

			testutil.Require(t, !dm.HasField(c.extfd))

			v, err := dm.TryGetField(c.extfd)
			testutil.Ok(t, err)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			err = dm.TrySetField(c.extfd, i.input)
			if k == c.kind && err != nil {
				t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
			} else if k != c.kind && err == nil {
				t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
			} else if k == c.kind {
				// make sure value stuck
				v, err = dm.TryGetField(c.extfd)
				testutil.Ok(t, err)
				testutil.Eq(t, i.input, v)
				testutil.Require(t, dm.HasField(c.extfd))
			}

			err = dm.TryClearField(c.extfd)
			testutil.Ok(t, err)
			testutil.Require(t, !dm.HasField(c.extfd))

			v, err = dm.TryGetField(c.extfd)
			testutil.Ok(t, err)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			// Now we do it again using the non-Try* methods (e.g. the ones that panic)

			v = dm.GetField(c.extfd)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			err = catchPanic(func() { dm.SetField(c.extfd, i.input) })
			if k == c.kind && err != nil {
				t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
			} else if k != c.kind && err == nil {
				t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
			} else if k == c.kind {
				// make sure value stuck
				v = dm.GetField(c.extfd)
				testutil.Eq(t, i.input, v)
				testutil.Require(t, dm.HasField(c.extfd))
			}

			dm.ClearField(c.extfd)
			testutil.Require(t, !dm.HasField(c.extfd))

			v = dm.GetField(c.extfd)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func TestGetSetExtensionFields_ByTagNumber(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.AnotherTestMessage").(*desc.MessageDescriptor)
	er := NewExtensionRegistryWithDefaults()
	dm := NewMessageFactoryWithExtensionRegistry(er).NewMessage(md).(*Message)

	inputs := map[reflect.Kind]struct {
		input interface{}
		zero  interface{}
	}{
		reflect.Ptr: {
			input: &testprotos.TestMessage{Ne: []testprotos.TestMessage_NestedEnum{testprotos.TestMessage_VALUE1}},
			zero:  (*testprotos.TestMessage)(nil),
		},
		reflect.Int32:  {input: int32(-12), zero: int32(0)},
		reflect.Uint64: {input: uint64(4567), zero: uint64(0)},
		reflect.String: {input: "foobar", zero: ""},
		reflect.Slice:  {input: []bool{true, false, true, false, true}, zero: []bool(nil)}}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Ptr, tagNumber: int(testprotos.E_Xtm.Field), fieldName: testprotos.E_Xtm.Name},
		{kind: reflect.Int32, tagNumber: int(testprotos.E_Xi.Field), fieldName: testprotos.E_Xi.Name},
		{kind: reflect.Uint64, tagNumber: int(testprotos.E_Xui.Field), fieldName: testprotos.E_Xui.Name},
		{kind: reflect.String, tagNumber: int(testprotos.E_Xs.Field), fieldName: testprotos.E_Xs.Name},
		{kind: reflect.Slice, tagNumber: int(testprotos.E_TestMessage_NestedMessage_AnotherNestedMessage_Flags.Field),
			fieldName: testprotos.E_TestMessage_NestedMessage_AnotherNestedMessage_Flags.Name},
	}

	for _, c := range cases {
		zero := inputs[c.kind].zero

		for k, i := range inputs {
			// First run the case using Try* methods

			v, err := dm.TryGetFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			err = dm.TrySetFieldByNumber(c.tagNumber, i.input)
			if k == c.kind && err != nil {
				t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
			} else if k != c.kind && err == nil {
				t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
			} else if k == c.kind {
				// make sure value stuck
				v, err = dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, i.input, v)
			}

			err = dm.TryClearFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)

			v, err = dm.TryGetFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			// Now we do it again using the non-Try* methods (e.g. the ones that panic)

			v = dm.GetFieldByNumber(c.tagNumber)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, i.input) })
			if k == c.kind && err != nil {
				t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
			} else if k != c.kind && err == nil {
				t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
			} else if k == c.kind {
				// make sure value stuck
				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, i.input, v)
			}

			dm.ClearFieldByNumber(c.tagNumber)

			v = dm.GetFieldByNumber(c.tagNumber)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func TestGetSetExtensionFields_ByName(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.AnotherTestMessage").(*desc.MessageDescriptor)
	er := NewExtensionRegistryWithDefaults()
	dm := NewMessageFactoryWithExtensionRegistry(er).NewMessage(md).(*Message)

	inputs := map[reflect.Kind]struct {
		input interface{}
		zero  interface{}
	}{
		reflect.Ptr: {
			input: &testprotos.TestMessage{Ne: []testprotos.TestMessage_NestedEnum{testprotos.TestMessage_VALUE1}},
			zero:  (*testprotos.TestMessage)(nil),
		},
		reflect.Int32:  {input: int32(-12), zero: int32(0)},
		reflect.Uint64: {input: uint64(4567), zero: uint64(0)},
		reflect.String: {input: "foobar", zero: ""},
		reflect.Slice:  {input: []bool{true, false, true, false, true}, zero: []bool(nil)}}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	}{
		{kind: reflect.Ptr, tagNumber: int(testprotos.E_Xtm.Field), fieldName: testprotos.E_Xtm.Name},
		{kind: reflect.Int32, tagNumber: int(testprotos.E_Xi.Field), fieldName: testprotos.E_Xi.Name},
		{kind: reflect.Uint64, tagNumber: int(testprotos.E_Xui.Field), fieldName: testprotos.E_Xui.Name},
		{kind: reflect.String, tagNumber: int(testprotos.E_Xs.Field), fieldName: testprotos.E_Xs.Name},
		{kind: reflect.Slice, tagNumber: int(testprotos.E_TestMessage_NestedMessage_AnotherNestedMessage_Flags.Field),
			fieldName: testprotos.E_TestMessage_NestedMessage_AnotherNestedMessage_Flags.Name},
	}

	for _, c := range cases {
		zero := inputs[c.kind].zero

		for k, i := range inputs {
			// First run the case using Try* methods

			v, err := dm.TryGetFieldByName(c.fieldName)
			testutil.Ok(t, err)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			err = dm.TrySetFieldByName(c.fieldName, i.input)
			if k == c.kind && err != nil {
				t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
			} else if k != c.kind && err == nil {
				t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
			} else if k == c.kind {
				// make sure value stuck
				v, err = dm.TryGetFieldByName(c.fieldName)
				testutil.Ok(t, err)
				testutil.Eq(t, i.input, v)
			}

			err = dm.TryClearFieldByName(c.fieldName)
			testutil.Ok(t, err)

			v, err = dm.TryGetFieldByName(c.fieldName)
			testutil.Ok(t, err)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			// Now we do it again using the non-Try* methods (e.g. the ones that panic)

			v = dm.GetFieldByName(c.fieldName)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}

			err = catchPanic(func() { dm.SetFieldByName(c.fieldName, i.input) })
			if k == c.kind && err != nil {
				t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
			} else if k != c.kind && err == nil {
				t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
			} else if k == c.kind {
				// make sure value stuck
				v = dm.GetFieldByName(c.fieldName)
				testutil.Eq(t, i.input, v)
			}

			dm.ClearFieldByName(c.fieldName)

			v = dm.GetFieldByName(c.fieldName)
			if c.kind == reflect.Ptr {
				testutil.Ceq(t, zero, v, eqm)
			} else {
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func loadExtension(t *testing.T, ed *proto.ExtensionDesc) *desc.FieldDescriptor {
	fd, err := desc.LoadFieldDescriptorForExtension(ed)
	testutil.Ok(t, err, "failed to load descriptor for extension %s (%d)", ed.Name, ed.Field)
	return fd
}

func TestGetSetOneOfFields(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test2.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("testprotos.Frobnitz").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	fdc1 := md.FindFieldByName("c1")
	fdc2 := md.FindFieldByName("c2")
	fdg1 := md.FindFieldByName("g1")
	fdg2 := md.FindFieldByName("g2")
	fdg3 := md.FindFieldByName("g3")

	oodc := md.GetFile().FindSymbol("testprotos.Frobnitz.abc").(*desc.OneOfDescriptor)
	oodg := md.GetFile().FindSymbol("testprotos.Frobnitz.def").(*desc.OneOfDescriptor)

	// nothing set
	fld, v := dm.GetOneOfField(oodc)
	testutil.Require(t, fld == nil && v == nil)
	fld, v = dm.GetOneOfField(oodg)
	testutil.Require(t, fld == nil && v == nil)

	nm := &testprotos.TestMessage_NestedMessage{}
	dm.SetField(fdc1, nm)
	fld, v = dm.GetOneOfField(oodc)
	testutil.Eq(t, fdc1, fld)
	testutil.Eq(t, nm, v)
	fld, v = dm.GetOneOfField(oodg) // other one-of untouched
	testutil.Require(t, fld == nil && v == nil)

	// setting c2 should unset field c1
	dm.SetField(fdc2, testprotos.TestMessage_VALUE1)
	fld, v = dm.GetOneOfField(oodc)
	testutil.Eq(t, fdc2, fld)
	testutil.Eq(t, int32(testprotos.TestMessage_VALUE1), v)
	testutil.Require(t, !dm.HasField(fdc1))

	// try other one-of, too
	dm.SetField(fdg1, int32(321))
	fld, v = dm.GetOneOfField(oodg)
	testutil.Eq(t, fdg1, fld)
	testutil.Eq(t, int32(321), v)
	fld, v = dm.GetOneOfField(oodc) // other one-of untouched
	testutil.Eq(t, fdc2, fld)
	testutil.Eq(t, int32(testprotos.TestMessage_VALUE1), v)

	// setting g2 should unset field g1
	dm.SetField(fdg2, int32(654))
	fld, v = dm.GetOneOfField(oodg)
	testutil.Eq(t, fdg2, fld)
	testutil.Eq(t, int32(654), v)
	testutil.Require(t, !dm.HasField(fdg1))

	// similar for g3
	dm.SetField(fdg3, uint32(987))
	fld, v = dm.GetOneOfField(oodg)
	testutil.Eq(t, fdg3, fld)
	testutil.Eq(t, uint32(987), v)
	testutil.Require(t, !dm.HasField(fdg1))
	testutil.Require(t, !dm.HasField(fdg2))

	// ensure clearing fields behaves as expected
	dm.ClearField(fdc2)
	fld, v = dm.GetOneOfField(oodc)
	testutil.Require(t, fld == nil && v == nil)

	dm.ClearField(fdg3)
	fld, v = dm.GetOneOfField(oodg)
	testutil.Require(t, fld == nil && v == nil)
}

func TestSetGetOneOfDefaultValue(t *testing.T) {
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.OneOfMessage)(nil))
	testutil.Ok(t, err)
	oneof := md.GetFile().FindSymbol("testprotos.OneOfMessage.value").(*desc.OneOfDescriptor)

	var testCases = []struct {
		fieldName  string
		fieldValue interface{}
	}{
		{"binary_value", []byte{}},
		{"string_value", ""},
		{"boolean_value", false},
		{"int_value", int32(0)},
		{"int64_value", int64(0)},
		{"double_value", float64(0)},
		{"float_value", float32(0)},
		{"msg_value", (*testprotos.OneOfMessage)(nil)},
	}

	for _, tc := range testCases {
		dm := NewMessage(md)
		field := md.FindFieldByName(tc.fieldName)
		dm.SetField(field, tc.fieldValue)

		// Ensure that the zero-value is set correctly
		oneOfField, resultValue := dm.GetOneOfField(oneof)
		testutil.Eq(t, tc.fieldValue, resultValue)
		testutil.Eq(t, field, oneOfField)

		// Ensure that clearing the field results in no field being set
		dm.ClearField(field)
		oneOfField, resultValue = dm.GetOneOfField(oneof)
		testutil.Eq(t, nil, resultValue)
		testutil.Eq(t, (*desc.FieldDescriptor)(nil), oneOfField)
	}
}

func TestForEachMapFieldEntry(t *testing.T) {
	data := map[string]float64{
		"pi":     3.14159,
		"e":      2.71828,
		"answer": 42,
	}
	msg := testprotos.MapValFields{
		T: data,
	}
	dm, err := AsDynamicMessage(&msg)
	testutil.Ok(t, err)

	count := 0
	entries := map[string]float64{}
	dm.ForEachMapFieldEntryByName("t", func(k, v interface{}) bool {
		count++
		entries[k.(string)] = v.(float64)
		return true
	})
	testutil.Eq(t, 3, count)
	testutil.Eq(t, data, entries)

	count = 0
	dm.ForEachMapFieldEntryByName("t", func(k, v interface{}) bool {
		count++
		// break immediately
		return false
	})
	// only saw first entry
	testutil.Eq(t, 1, count)
}

func TestSetIntroducesNewField(t *testing.T) {
	// TODO
}

func TestGetEnablesParsingUnknownField(t *testing.T) {
	// TODO
}

func TestSetClearsUnknownFields(t *testing.T) {
	// TODO
}

func TestMergeInto(t *testing.T) {
	// TODO
}

func TestMergeFrom(t *testing.T) {
	// TODO
}

func TestGetDescriptor(t *testing.T) {
	type generatedMessage interface {
		proto.Message
		Descriptor() ([]byte, []int)
	}

	testCases := []struct {
		name   string
		genMsg generatedMessage
	}{
		{name: "top-level message", genMsg: (*testprotos.TestMessage)(nil)},
		{name: "nested message", genMsg: (*testprotos.TestMessage_NestedMessage)(nil)},
		{name: "deeply nested message", genMsg: (*testprotos.TestMessage_NestedMessage_AnotherNestedMessage_YetAnotherNestedMessage)(nil)},
	}
	for _, testCase := range testCases {
		md, err := desc.LoadMessageDescriptorForMessage(testCase.genMsg)
		testutil.Ok(t, err, "%s: failed to load descriptor", testCase.name)

		dynMsg := NewMessage(md)
		actualBytes, actualPath := dynMsg.Descriptor()

		expectedBytes, expectedPath := testCase.genMsg.Descriptor()
		testutil.Eq(t, expectedPath, actualPath, "%s: descriptor paths are not the same", testCase.name)

		actualFd, err := internal.DecodeFileDescriptor("TestMessage", actualBytes)
		testutil.Ok(t, err, "%s: failed to decode descriptor from bytes", testCase.name)
		expectedFd, err := internal.DecodeFileDescriptor("TestMessage", expectedBytes)
		testutil.Ok(t, err, "%s: failed to decode descriptor from bytes", testCase.name)

		testutil.Ceq(t, expectedFd, actualFd, eqpm, "%s: descriptors do not match", testCase.name)
	}
}

type panicError struct {
	panic interface{}
}

func (e panicError) Error() string {
	return fmt.Sprintf("panic: %v", e.panic)
}

func catchPanic(action func()) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = panicError{e}
		}
	}()

	action()
	return nil
}
