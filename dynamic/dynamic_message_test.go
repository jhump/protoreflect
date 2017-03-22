package dynamic

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/testutil"
)

var typeOfGenericSlice = reflect.TypeOf([]interface{}(nil))
var typeOfString = reflect.TypeOf("")
var typeOfGenericMap = reflect.TypeOf(map[interface{}]interface{}(nil))

func TestGetSetClearScalarFields(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("desc_test.UnaryFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]struct {
		input interface{}
		zero  interface{}
	} {
		reflect.Bool:    { input: true, zero: false },
		reflect.Int32:   { input: int32(-12), zero: int32(0) },
		reflect.Int64:   { input: int64(-1234), zero: int64(0) },
		reflect.Uint32:  { input: uint32(45), zero: uint32(0) },
		reflect.Uint64:  { input: uint64(4567), zero: uint64(0) },
		reflect.Float32: { input: float32(2.718), zero: float32(0) },
		reflect.Float64: { input: float64(3.14159), zero: float64(0) },
		reflect.String:  { input: "foobar", zero: "" },
		reflect.Slice:   { input: []byte("snafu"), zero: []byte(nil) },
	}

	cases := []struct {
		kind      reflect.Kind
		tagNumber int
		fieldName string
	} {
		{ kind: reflect.Int32, tagNumber: 1, fieldName: "i" },
		{ kind: reflect.Int64, tagNumber: 2, fieldName: "j" },
		{ kind: reflect.Int32, tagNumber: 3, fieldName: "k" },
		{ kind: reflect.Int64, tagNumber: 4, fieldName: "l" },
		{ kind: reflect.Uint32, tagNumber: 5, fieldName: "m" },
		{ kind: reflect.Uint64, tagNumber: 6, fieldName: "n" },
		{ kind: reflect.Uint32, tagNumber: 7, fieldName: "o" },
		{ kind: reflect.Uint64, tagNumber: 8, fieldName: "p" },
		{ kind: reflect.Int32, tagNumber: 9, fieldName: "q" },
		{ kind: reflect.Int64, tagNumber: 10, fieldName: "r" },
		{ kind: reflect.Float32, tagNumber: 11, fieldName: "s" },
		{ kind: reflect.Float64, tagNumber: 12, fieldName: "t" },
		{ kind: reflect.Slice, tagNumber: 13, fieldName: "u" },
		{ kind: reflect.String, tagNumber: 14, fieldName: "v" },
		{ kind: reflect.Bool, tagNumber: 15, fieldName: "w" },
	}

	for _, c := range cases {
		zero := inputs[c.kind].zero

		for k, i := range inputs {
			// First run the case using Try* methods

			v, err := dm.TryGetFieldByNumber(c.tagNumber)
			testutil.Ok(t, err)
			testutil.Eq(t, zero, v)

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
			testutil.Eq(t, zero, v)

			// Now we do it again using the non-Try* methods (e.g. the ones that panic)

			v = dm.GetFieldByNumber(c.tagNumber)
			testutil.Eq(t, zero, v)

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
			testutil.Eq(t, zero, v)
		}
	}
}

func TestGetSetClearRepeatedFields(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("desc_test.RepeatedFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]interface{} {
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

	sliceKinds := []func(interface{}) interface{} {
		// index 0 will not work since it doesn't return a slice
		func(v interface{}) interface{} {
			return v
		},
		func(v interface{}) interface{} {
			// generic slice
			return []interface{} { v, v, v }
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
	} {
		{ kind: reflect.Int32, tagNumber: 1, fieldName: "i" },
		{ kind: reflect.Int64, tagNumber: 2, fieldName: "j" },
		{ kind: reflect.Int32, tagNumber: 3, fieldName: "k" },
		{ kind: reflect.Int64, tagNumber: 4, fieldName: "l" },
		{ kind: reflect.Uint32, tagNumber: 5, fieldName: "m" },
		{ kind: reflect.Uint64, tagNumber: 6, fieldName: "n" },
		{ kind: reflect.Uint32, tagNumber: 7, fieldName: "o" },
		{ kind: reflect.Uint64, tagNumber: 8, fieldName: "p" },
		{ kind: reflect.Int32, tagNumber: 9, fieldName: "q" },
		{ kind: reflect.Int64, tagNumber: 10, fieldName: "r" },
		{ kind: reflect.Float32, tagNumber: 11, fieldName: "s" },
		{ kind: reflect.Float64, tagNumber: 12, fieldName: "t" },
		{ kind: reflect.Slice, tagNumber: 13, fieldName: "u" },
		{ kind: reflect.String, tagNumber: 14, fieldName: "v" },
		{ kind: reflect.Bool, tagNumber: 15, fieldName: "w" },
	}

	zero := reflect.Zero(typeOfGenericSlice).Interface()

	for _, c := range cases {
		for k, i := range inputs {
			for j, sk := range sliceKinds {
				// First run the case using Try* methods

				v, err := dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				input := sk(i)
				err = dm.TrySetFieldByNumber(c.tagNumber, input)
				if j != 0 && k == c.kind && err != nil {
					t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
				} else if (j == 0 || k != c.kind) && err == nil {
					t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
				} else if j != 0 && k == c.kind {
					// make sure value stuck
					v, err = dm.TryGetFieldByNumber(c.tagNumber)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericSlice, reflect.TypeOf(v))
					testutil.Eq(t, input, v)
				}

				err = dm.TryClearFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)

				v, err = dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				// Now we do it again using the non-Try* methods (e.g. the ones that panic)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)

				err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, input) })
				if j != 0 && k == c.kind && err != nil {
					t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
				} else if (j == 0 || k != c.kind) && err == nil {
					t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
				} else if j != 0 && k == c.kind {
					// make sure value stuck
					v = dm.GetFieldByNumber(c.tagNumber)

					testutil.Eq(t, typeOfGenericSlice, reflect.TypeOf(v))
					testutil.Eq(t, input, v)
				}

				dm.ClearFieldByNumber(c.tagNumber)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func TestGetSetClearMapFields_KeyTypes(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("desc_test.MapKeyFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]interface{} {
		reflect.Bool:    true,
		reflect.Int32:   int32(-12),
		reflect.Int64:   int64(-1234),
		reflect.Uint32:  uint32(45),
		reflect.Uint64:  uint64(4567),
		reflect.String:  "foobar",
	}

	mapKinds := []func(interface{}) interface{} {
		// index 0 will not work since it doesn't return a map
		func(v interface{}) interface{} {
			return v
		},
		func(v interface{}) interface{} {
			// generic slice
			return map[interface{}]interface{} { v: "foo" }
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
	} {
		{ kind: reflect.Int32, tagNumber: 1, fieldName: "i" },
		{ kind: reflect.Int64, tagNumber: 2, fieldName: "j" },
		{ kind: reflect.Int32, tagNumber: 3, fieldName: "k" },
		{ kind: reflect.Int64, tagNumber: 4, fieldName: "l" },
		{ kind: reflect.Uint32, tagNumber: 5, fieldName: "m" },
		{ kind: reflect.Uint64, tagNumber: 6, fieldName: "n" },
		{ kind: reflect.Uint32, tagNumber: 7, fieldName: "o" },
		{ kind: reflect.Uint64, tagNumber: 8, fieldName: "p" },
		{ kind: reflect.Int32, tagNumber: 9, fieldName: "q" },
		{ kind: reflect.Int64, tagNumber: 10, fieldName: "r" },
		{ kind: reflect.String, tagNumber: 11, fieldName: "s" },
		{ kind: reflect.Bool, tagNumber: 12, fieldName: "t" },
	}

	zero := reflect.Zero(typeOfGenericMap).Interface()

	for _, c := range cases {
		for k, i := range inputs {
			for j, mk := range mapKinds {
				// First run the case using Try* methods

				v, err := dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				input := mk(i)
				err = dm.TrySetFieldByNumber(c.tagNumber, input)
				if j != 0 && k == c.kind && err != nil {
					t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
				} else if (j == 0 || k != c.kind) && err == nil {
					t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
				} else if j != 0 && k == c.kind {
					// make sure value stuck
					v, err = dm.TryGetFieldByNumber(c.tagNumber)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, input, v)
				}

				err = dm.TryClearFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)

				v, err = dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				// Now we do it again using the non-Try* methods (e.g. the ones that panic)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)

				err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, input) })
				if j != 0 && k == c.kind && err != nil {
					t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
				} else if (j == 0 || k != c.kind) && err == nil {
					t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
				} else if j != 0 && k == c.kind {
					// make sure value stuck
					v = dm.GetFieldByNumber(c.tagNumber)

					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, input, v)
				}

				dm.ClearFieldByNumber(c.tagNumber)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func TestGetSetClearMapFields_ValueTypes(t *testing.T) {
	fd, err := desc.LoadFileDescriptor("desc_test_field_types.proto")
	testutil.Ok(t, err)
	md := fd.FindSymbol("desc_test.MapValFields").(*desc.MessageDescriptor)
	dm := NewMessage(md)

	inputs := map[reflect.Kind]interface{} {
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

	mapKinds := []func(interface{}) interface{} {
		// index 0 will not work since it doesn't return a map
		func(v interface{}) interface{} {
			return v
		},
		func(v interface{}) interface{} {
			// generic slice
			return map[interface{}]interface{} { "foo": v, "bar": v, "baz": v }
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
	} {
		{ kind: reflect.Int32, tagNumber: 1, fieldName: "i" },
		{ kind: reflect.Int64, tagNumber: 2, fieldName: "j" },
		{ kind: reflect.Int32, tagNumber: 3, fieldName: "k" },
		{ kind: reflect.Int64, tagNumber: 4, fieldName: "l" },
		{ kind: reflect.Uint32, tagNumber: 5, fieldName: "m" },
		{ kind: reflect.Uint64, tagNumber: 6, fieldName: "n" },
		{ kind: reflect.Uint32, tagNumber: 7, fieldName: "o" },
		{ kind: reflect.Uint64, tagNumber: 8, fieldName: "p" },
		{ kind: reflect.Int32, tagNumber: 9, fieldName: "q" },
		{ kind: reflect.Int64, tagNumber: 10, fieldName: "r" },
		{ kind: reflect.Float32, tagNumber: 11, fieldName: "s" },
		{ kind: reflect.Float64, tagNumber: 12, fieldName: "t" },
		{ kind: reflect.Slice, tagNumber: 13, fieldName: "u" },
		{ kind: reflect.String, tagNumber: 14, fieldName: "v" },
		{ kind: reflect.Bool, tagNumber: 15, fieldName: "w" },
	}

	zero := reflect.Zero(typeOfGenericMap).Interface()

	for _, c := range cases {
		for k, i := range inputs {
			for j, mk := range mapKinds {
				// First run the case using Try* methods

				v, err := dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				input := mk(i)
				err = dm.TrySetFieldByNumber(c.tagNumber, input)
				if j != 0 && k == c.kind && err != nil {
					t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
				} else if (j == 0 || k != c.kind) && err == nil {
					t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
				} else if j != 0 && k == c.kind {
					// make sure value stuck
					v, err = dm.TryGetFieldByNumber(c.tagNumber)
					testutil.Ok(t, err)
					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, input, v)
				}

				err = dm.TryClearFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)

				v, err = dm.TryGetFieldByNumber(c.tagNumber)
				testutil.Ok(t, err)
				testutil.Eq(t, zero, v)

				// Now we do it again using the non-Try* methods (e.g. the ones that panic)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)

				err = catchPanic(func() { dm.SetFieldByNumber(c.tagNumber, input) })
				if j != 0 && k == c.kind && err != nil {
					t.Errorf("Not expecting an error assigning a %v to a %v (%v): %s", k, c.kind, c, err.Error())
				} else if (j == 0 || k != c.kind) && err == nil {
					t.Errorf("Expecting an error assigning a %v to a %v", k, c.kind)
				} else if j != 0 && k == c.kind {
					// make sure value stuck
					v = dm.GetFieldByNumber(c.tagNumber)

					testutil.Eq(t, typeOfGenericMap, reflect.TypeOf(v))
					testutil.Eq(t, input, v)
				}

				dm.ClearFieldByNumber(c.tagNumber)

				v = dm.GetFieldByNumber(c.tagNumber)
				testutil.Eq(t, zero, v)
			}
		}
	}
}

func TestGetSetOneOfFields(t *testing.T) {
	// TODO
}

func TestGetSetAtIndexAddRepeatedFields(t *testing.T) {
	// TODO
}

func TestGetPutDeleteMapFields(t *testing.T) {
	// TODO
}

func TestMapFields_AsIfRepeatedFieldOfEntries(t *testing.T) {
	// TODO
}

func TestMergeInto(t *testing.T) {
	// TODO
}

func TestMergeFrom(t *testing.T) {
	// TODO
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