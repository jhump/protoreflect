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
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bufbuild/protocompile/internal/cases"
)

func TestWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		str  string
		want []string
	}{
		{str: ""},
		{str: "_"},
		{str: "__"},

		{str: "foo", want: []string{"foo"}},
		{str: "_foo", want: []string{"foo"}},
		{str: "foo_", want: []string{"foo"}},
		{str: "foo_bar", want: []string{"foo", "bar"}},
		{str: "foo__bar", want: []string{"foo", "bar"}},

		{str: "fooBar", want: []string{"foo", "Bar"}},
		{str: "foo_Bar", want: []string{"foo", "Bar"}},
		{str: "FOOBar", want: []string{"FOO", "Bar"}},
		{str: "FooX", want: []string{"Foo", "X"}},
		{str: "FOO", want: []string{"FOO"}},
		{str: "FooBARBaz", want: []string{"FooBAR", "Baz"}},
	}

	for _, test := range tests {
		t.Run(test.str, func(t *testing.T) {
			t.Parallel()

			got := slices.Collect(cases.Words(test.str))
			assert.Equal(t, test.want, got)
		})
	}
}
