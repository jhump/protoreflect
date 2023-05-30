package protoresolve

import (
	"reflect"
	"testing"
	"unsafe"
)

func BenchmarkReflectArray_3(b *testing.B) {
	bench(b, 3, reflectArray)
}

func BenchmarkReflectArray_10(b *testing.B) {
	bench(b, 10, reflectArray)
}

func BenchmarkReflectArray_20(b *testing.B) {
	bench(b, 20, reflectArray)
}

func BenchmarkUnsafeArray_3(b *testing.B) {
	bench(b, 3, unsafeArray)
}

func BenchmarkUnsafeArray_10(b *testing.B) {
	bench(b, 10, unsafeArray)
}

func BenchmarkUnsafeArray_20(b *testing.B) {
	bench(b, 20, unsafeArray)
}

func BenchmarkIfaceList_3(b *testing.B) {
	bench(b, 3, ifaceList)
}

func BenchmarkIfaceList_10(b *testing.B) {
	bench(b, 10, ifaceList)
}

func BenchmarkIfaceList_20(b *testing.B) {
	bench(b, 20, ifaceList)
}

func BenchmarkString_3(b *testing.B) {
	bench(b, 3, toString)
}

func BenchmarkString_10(b *testing.B) {
	bench(b, 10, toString)
}

func BenchmarkString_20(b *testing.B) {
	bench(b, 20, toString)
}

func BenchmarkArrayOrString_3(b *testing.B) {
	bench(b, 3, arrayOrString)
}

func BenchmarkArrayOrString_10(b *testing.B) {
	bench(b, 10, arrayOrString)
}

func BenchmarkArrayOrString_20(b *testing.B) {
	bench(b, 20, arrayOrString)
}

func bench[T any](b *testing.B, length int, fn func([]int32) T) {
	path := make([]int32, length)
	for i := range path {
		path[i] = int32(i + 3)
	}
	for i := 0; i < b.N; i++ {
		var v any
		v = fn(path)
		_ = v
		//runtime.KeepAlive(t)
	}
}

type rec struct {
	s string
	a [16]byte
}

func arrayOrString(path []int32) rec {
	var r rec
	if len(path) < 16 {
		for i, p := range path {
			if p < 0 || p > 255 {
				return rec{s: toString(path)}
			}
			r.a[i] = byte(p)
		}
		r.a[15] = byte(len(path))
	}
	return rec{s: toString(path)}
}

func toString(path []int32) string {
	b := make([]byte, len(path)*4)
	j := 0
	for _, s := range path {
		b[j] = byte(s)
		b[j+1] = byte(s >> 8)
		b[j+2] = byte(s >> 16)
		b[j+3] = byte(s >> 24)
		j += 4
	}
	return string(b)
}

func reflectArray(path []int32) any {
	rv := reflect.ValueOf(path)
	arrayType := reflect.ArrayOf(rv.Len(), rv.Type().Elem())
	array := reflect.New(arrayType).Elem()
	reflect.Copy(array, rv)
	return array.Interface()
}

var pathElementType = reflect.TypeOf(int32(0))

func unsafeArray(path []int32) any {
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(reflect.ValueOf(&path).Pointer()))
	array := reflect.NewAt(reflect.ArrayOf(hdr.Len, pathElementType), unsafe.Pointer(hdr.Data))
	return array.Elem().Interface()
}

type list struct {
	i    int32
	next any
}

func ifaceList(path []int32) list {
	var h list
	for _, p := range path {
		h = list{i: p, next: h}
	}
	return h
}
