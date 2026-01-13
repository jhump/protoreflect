// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package unsafex contains extensions to Go's package unsafe.
//
// Importing this package should be treated as equivalent to importing unsafe.
package unsafex

import (
	"fmt"
	"sync"
	"unsafe"
)

// NoCopy can be embedded in a type to trigger go vet's no copy lint.
type NoCopy struct {
	_ [0]sync.Mutex
}

// Int is a constraint for any integer type.
type Int interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~int |
		~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint |
		~uintptr
}

// Size is like [unsafe.Sizeof], but it is a generic function and it returns
// an int instead of a uintptr (Go does not have types so large they would
// overflow an int).
func Size[T any]() int {
	var v T
	return int(unsafe.Sizeof(v))
}

// Add is like [unsafe.Add], but it operates on a typed pointer and scales the
// offset by that type's size, similar to pointer arithmetic in Rust or C.
//
// This function has the same safety caveats as [unsafe.Add].
//
//go:nosplit
func Add[P ~*E, E any, I Int](p P, idx I) P {
	raw := unsafe.Pointer(p)
	raw = unsafe.Add(raw, int(idx)*Size[E]())
	return P(raw)
}

// Bitcast bit-casts a value of type From to a value of type To.
//
// This operation is very dangerous, because it can be used to break package
// export barriers, read uninitialized memory, and forge pointers in violation
// of [unsafe.Pointer]'s contract, resulting in memory errors in the GC.
//
// Panics if To and From have different sizes.
//
//go:nosplit
func Bitcast[To, From any](v From) To {
	// This function is correctly compiled down to a mov, as seen here:
	// https://godbolt.org/z/qvndcYYba
	//
	// With redundant code removed, stenciling Bitcast[float64, int64] produces
	// (as seen in the above Godbolt):
	//
	//   TEXT    unsafex.Bitcast[float64,int64]
	//   MOVQ    32(R14), R12
	//   TESTQ   R12, R12
	//   JNE     morestack
	//   XCHGL   AX, AX
	//   MOVQ    AX, X0
	//   RET

	// This check is necessary because casting a smaller type into a larger
	// type will result in reading uninitialized memory, especially in the
	// presence of inlining that causes &aligned below to point into the heap.
	// The equivalent functions in Rust and C++ perform this check statically,
	// because it is so important.
	if Size[To]() != Size[From]() {
		// This check will always be inlined away, because Bitcast is
		// manifestly inline-able.
		//
		// NOTE: This could potentially be replaced with a link error, by making
		// this call a function with no body (and then not defining that
		// function in a .s file; although, note we do need an empty.s to
		// silence a compiler error in that case).
		panic(badBitcast[To, From]{})
	}

	// To avoid an unaligned load below, we copy From into a struct aligned to
	// To's alignment. Consider the following situation: we call
	// Bitcast[int32, [4]byte]. There is no guarantee that &v will be aligned
	// to the four byte boundary required for int32, and thus casting it to *To
	// may result in an unaligned load.
	//
	// As seen in the Godbolt above, for cases where the alignment change
	// is redundant, this gets optimized away.
	aligned := struct {
		_ [0]To
		v From
	}{v: v}

	return *(*To)(unsafe.Pointer(&aligned.v))
}

type badBitcast[To, From any] struct{}

func (badBitcast[To, From]) Error() string {
	var to To
	var from From
	return fmt.Sprintf(
		"unsafex: %T and %T are of unequal size (%d != %d)",
		to, from,
		Size[To](), Size[From](),
	)
}

// StringAlias returns a string that aliases a slice. This is useful for
// situations where we're allocating a string on the stack, or where we have
// a slice that will never be written to and we want to interpret as a string
// without a copy.
//
// data must not be written to: for the lifetime of the returned string (that
// is, until its final use in the program upon which a finalizer set on it could
// run), it must be treated as if goroutines are concurrently reading from it:
// data must not be mutated in any way.
//
//go:nosplit
func StringAlias[S ~[]E, E any](data S) string {
	return unsafe.String(
		Bitcast[*byte](unsafe.SliceData(data)),
		len(data)*Size[E](),
	)
}

// BytesAlias is the inverse of [StringAlias].
//
// The same caveats apply as with [StringAlias] around mutating `data`.
//
//go:nosplit
func BytesAlias[S ~[]B, B ~byte](data string) []B {
	return unsafe.Slice(
		Bitcast[*B](unsafe.StringData(data)),
		len(data),
	)
}

// NoEscape makes a copy of a pointer which is hidden from the compiler's
// escape analysis. If the return value of this function appears to escape, the
// compiler will *not* treat its input as escaping, potentially allowing stack
// promotion of values which escape. This function is essentially the opposite
// of runtime.KeepAlive.
//
// This is only safe when the return value does not actually escape to the heap,
// but only appears to, such as by being passed to a virtual call which does not
// actually result in a heap escape.
//
//go:nosplit
func NoEscape[P ~*E, E any](ptr P) P {
	p := unsafe.Pointer(ptr)
	// Xoring the address with zero is a reliable way to hide a pointer from
	// the compiler.
	p = unsafe.Pointer(uintptr(p) ^ 0) //nolint:staticcheck
	return P(p)
}
