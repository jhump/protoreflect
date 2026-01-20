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

package cases_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/internal/cases"
)

func TestCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		str                        string
		snake, enum, camel, pascal string
		naiveCamel, naivePascal    string
	}{
		{str: ""},
		{str: "_"},
		{str: "__"},

		{
			str:   "foo",
			snake: "foo", enum: "FOO",
			camel: "foo", pascal: "Foo",
			naiveCamel: "foo", naivePascal: "Foo",
		},

		{
			str:   "FOO4",
			snake: "foo4", enum: "FOO4",
			camel: "foo4", pascal: "Foo4",
			naiveCamel: "FOO4", naivePascal: "FOO4",
		},

		{
			str:   "_foo",
			snake: "foo", enum: "FOO",
			camel: "foo", pascal: "Foo",
			naiveCamel: "Foo", naivePascal: "Foo",
		},
		{
			str:   "foo_",
			snake: "foo", enum: "FOO",
			camel: "foo", pascal: "Foo",
			naiveCamel: "foo", naivePascal: "Foo",
		},
		{
			str:   "foo_bar",
			snake: "foo_bar", enum: "FOO_BAR",
			camel: "fooBar", pascal: "FooBar",
			naiveCamel: "fooBar", naivePascal: "FooBar",
		},
		{
			str:   "foo__bar",
			snake: "foo_bar", enum: "FOO_BAR",
			camel: "fooBar", pascal: "FooBar",
			naiveCamel: "fooBar", naivePascal: "FooBar",
		},
		{
			str:   "_foo_bar",
			snake: "foo_bar", enum: "FOO_BAR",
			camel: "fooBar", pascal: "FooBar",
			naiveCamel: "FooBar", naivePascal: "FooBar",
		},
		{
			str:   "FOO_BAR",
			snake: "foo_bar", enum: "FOO_BAR",
			camel: "fooBar", pascal: "FooBar",
			naiveCamel: "FOOBAR", naivePascal: "FOOBAR",
		},

		{
			str:   "fooBar",
			snake: "foo_bar", enum: "FOO_BAR",
			camel: "fooBar", pascal: "FooBar",
			naiveCamel: "fooBar", naivePascal: "FooBar",
		},
		{
			str:   "foo_Bar",
			snake: "foo_bar", enum: "FOO_BAR",
			camel: "fooBar", pascal: "FooBar",
			naiveCamel: "fooBar", naivePascal: "FooBar",
		},
		{
			str:   "FOOBar",
			snake: "foo_bar", enum: "FOO_BAR",
			camel: "fooBar", pascal: "FooBar",
			naiveCamel: "FOOBar", naivePascal: "FOOBar",
		},
	}

	for _, test := range tests {
		t.Run(test.str, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.snake, cases.Snake.Convert(test.str))
			assert.Equal(t, test.enum, cases.Enum.Convert(test.str))
			assert.Equal(t, test.camel, cases.Camel.Convert(test.str))
			assert.Equal(t, test.pascal, cases.Pascal.Convert(test.str))

			assert.Equal(t, test.naiveCamel, cases.Converter{Case: cases.Camel, NaiveSplit: true, NoLowercase: true}.Convert(test.str))
			assert.Equal(t, test.naivePascal, cases.Converter{Case: cases.Pascal, NaiveSplit: true, NoLowercase: true}.Convert(test.str))
		})
	}
}
