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

package prototest

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// RequireSameLayout generates require assertions for ensuring that a and b have
// the same layout.
//
// This is useful for verifying that a type used for unsafe.Pointer shenanigans
// matches another.
//
// NOTE: This will currently recurse infinitely on a type such as
//
//	type T struct { p *T }
//
// This function is only intended for testing so actually making sure we don't
// hit that case is not currently necessary.
func RequireSameLayout(t *testing.T, a, b reflect.Type) {
	t.Helper()
	if a == b {
		return // No need to check further.
	}

	require.Equal(
		t, a.Kind(), b.Kind(),
		"mismatched kinds: %s is %s; %s is %s", a, a.Kind(), b, b.Kind())

	switch a.Kind() {
	case reflect.Struct:
		require.Equal(t, a.NumField(), b.NumField(),
			"mismatched field counts: %s has %d fields; %s has %d fields", a, a.NumField(), b, b.NumField())

		for i := range a.NumField() {
			RequireSameLayout(t, a.Field(i).Type, b.Field(i).Type)
		}

	case reflect.Slice, reflect.Chan, reflect.Pointer:
		RequireSameLayout(t, a.Elem(), b.Elem())

	case reflect.Array:
		RequireSameLayout(t, a.Elem(), b.Elem())
		require.Equal(t, a.Len(), b.Len(), "mismatched array lengths: %s != %s", a, b)

	case reflect.Map:
		RequireSameLayout(t, a.Key(), b.Key())
		RequireSameLayout(t, a.Elem(), b.Elem())

	case reflect.Interface:
		require.True(t, a.Implements(b), "mismatched interface types: %s != %s", a, b)
		require.True(t, b.Implements(a), "mismatched interface types: %s != %s", a, b)

	case reflect.Func:
		require.True(t, a.ConvertibleTo(b), "mismatched function types: %s != %s", a, b)
		require.True(t, b.ConvertibleTo(a), "mismatched function types: %s != %s", a, b)

	default:
		// The others are simple scalars, so same kind is sufficient.
	}
}
