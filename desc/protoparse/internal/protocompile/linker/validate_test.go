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

package linker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalEnumName(t *testing.T) {
	t.Parallel()
	testCases := map[string]string{
		"FOO_BAR___foo_bar_baz":     "FooBarBaz",
		"foo__bar__baz":             "Baz",
		"_foo_bar_":                 "FooBar",
		"__F_O_O_B_A_R_FOO_BAR_BAZ": "FooBarBaz",
		"FooBar_FooBarBaz":          "Foobarbaz",
		"FOOBAR_BAZ":                "Baz",
		"BAZ":                       "Baz",
		"B_A_Z":                     "BAZ",
		"___fu_bar_baz__":           "FuBarBaz",
		"foobarbaz":                 "Baz",
		"FOOBARFOOBARBAZ":           "Foobarbaz",
	}
	const enumName = "FooBar"
	for k, v := range testCases {
		name := canonicalEnumValueName(k, enumName)
		assert.Equalf(t, name, v, "enum value name %v (in enum %s) resulted in wrong canonical name", k, enumName)
	}
}
