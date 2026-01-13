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

package protocompile

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/reporter"
)

func TestErrorReporting(t *testing.T) {
	t.Parallel()
	tooManyErrors := errors.New("too many errors")
	limitedErrReporter := func(limit int, count *int) reporter.ErrorReporter {
		return func(err reporter.ErrorWithPos) error {
			fmt.Printf("* error reported: %v\n", err)
			*count++
			if *count > limit {
				return tooManyErrors
			}
			return nil
		}
	}
	trackingReporter := func(errs *[]reporter.ErrorWithPos, count *int) reporter.ErrorReporter {
		return func(err reporter.ErrorWithPos) error {
			fmt.Printf("* error reported: %v\n", err)
			*count++
			*errs = append(*errs, err)
			return nil
		}
	}
	fail := errors.New("failure")
	failFastReporter := func(count *int) reporter.ErrorReporter {
		return func(err reporter.ErrorWithPos) error {
			fmt.Printf("* error reported: %v\n", err)
			*count++
			return fail
		}
	}

	testCases := []struct {
		fileNames    []string
		files        map[string]string
		expectedErrs [][]string
	}{
		{
			// multiple syntax errors
			fileNames: []string{"test.proto"},
			files: map[string]string{
				"test.proto": `
					syntax = "proto";
					package foo

					enum State { A = 0; B = 1; C; D }
					message Foo {
						foo = 1;
					}
					`,
			},
			expectedErrs: [][]string{
				{
					"test.proto:5:41: syntax error: expecting ';'",
					"test.proto:5:69: syntax error: unexpected ';', expecting '='",
					"test.proto:7:53: syntax error: unexpected '='"},
			},
		},
		{
			// multiple validation errors
			fileNames: []string{"test.proto"},
			files: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo {
						string foo = 0;
					}
					enum State { C = 0; }
					enum Bar {
						BAZ = 1;
						BUZZ = 1;
					}
					`,
			},
			expectedErrs: [][]string{
				{
					"test.proto:4:62: tag number 0 must be greater than zero",
					"test.proto:8:55: enum Bar: proto3 requires that first value of enum have numeric value zero",
					"test.proto:9:56: enum Bar: values BAZ and BUZZ both have the same numeric value 1; use allow_alias option if intentional",
				},
			},
		},
		{
			// multiple link errors
			fileNames: []string{"test.proto"},
			files: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo {
						string foo = 1;
					}
					enum Bar {
						BAZ = 0;
						BAZ = 2;
					}
					service Bar {
						rpc Foobar (Foo) returns (Foo);
						rpc Foobar (Frob) returns (Nitz);
					}
					`,
			},
			expectedErrs: [][]string{
				{
					"test.proto:8:49: symbol \"BAZ\" already defined at test.proto:7:49; protobuf uses C++ scoping rules for enum values, so they exist in the scope enclosing the enum",
					"test.proto:10:49: symbol \"Bar\" already defined at test.proto:6:46",
					"test.proto:12:53: symbol \"Bar.Foobar\" already defined at test.proto:11:53",
				},
			},
		},
		{
			// syntax errors across multiple files
			fileNames: []string{"test1.proto", "test2.proto"},
			files: map[string]string{
				"test1.proto": `
					syntax = "proto3";
					import "test2.proto";
					message Foo {
						string foo = -1;
					}
					service Bar {
						rpc Foo (Foo);
					}
					`,
				"test2.proto": `
					syntax = "proto3";
					message Baz {
						required string foo = 1;
					}
					service Service {
						Foo; Bar; Baz;
					}
					`,
			},
			expectedErrs: [][]string{
				{
					"*", // errors can be in different order than below (due to concurrency)
					"test1.proto:5:62: syntax error: unexpected '-', expecting int literal",
					"test1.proto:8:62: syntax error: unexpected ';', expecting \"returns\"",
					"test2.proto:7:49: syntax error: unexpected identifier, expecting \"option\" or \"rpc\" or '}'",
				},
			},
		},
		{
			// link errors across multiple files
			fileNames: []string{"test1.proto", "test2.proto"},
			files: map[string]string{
				"test1.proto": `
					syntax = "proto3";
					message Foo {
						string foo = 1;
					}
					service Bar {
						rpc Frob (Empty) returns (Nitz);
					}
					`,
				"test2.proto": `
					syntax = "proto3";
					message Empty {}
					enum Bar {
						BAZ = 0;
					}
					service Foo {
						rpc DoSomething (Empty) returns (Empty);
					}
					`,
			},
			// because files are compiled concurrently, the order of processing can
			// impact the actual errors reported
			expectedErrs: [][]string{
				{
					// if test2.proto processed first
					"test1.proto:3:49: symbol \"Foo\" already defined at test2.proto:7:49",
					"test1.proto:6:49: symbol \"Bar\" already defined at test2.proto:4:46",
				},
				{
					"*", // errors can be in different order than below (due to concurrency)
					"test1.proto:7:59: method Bar.Frob: unknown request type Empty",
					"test1.proto:7:75: method Bar.Frob: unknown response type Nitz",
					"test2.proto:4:46: symbol \"Bar\" already defined at test1.proto:6:49",
					"test2.proto:7:49: symbol \"Foo\" already defined at test1.proto:3:49",
				},
			},
		},
	}

	ctx := t.Context()
	for i, tc := range testCases {
		fmt.Printf("---- case #%d: tracking ----\n", i+1)
		compiler := Compiler{
			Resolver: &SourceResolver{Accessor: SourceAccessorFromMap(tc.files)},
		}

		var reported []reporter.ErrorWithPos
		count := 0
		compiler.Reporter = reporter.NewReporter(trackingReporter(&reported, &count), nil)
		_, err := compiler.Compile(ctx, tc.fileNames...)
		reportedMsgs := make([]string, len(reported))
		for j := range reported {
			reportedMsgs[j] = reported[j].Error()
		}
		t.Logf("case #%d: got %d errors:\n\t%s", i+1, len(reported), strings.Join(reportedMsgs, "\n\t"))

		// returns sentinel, but all actual errors in reported
		assert.Equal(t, reporter.ErrInvalidSource, err, "case #%d: parse should have failed with invalid source error", i+1)
		var match []string
		for _, errs := range tc.expectedErrs {
			actualErrs := reportedMsgs
			if errs[0] == "*" {
				// errors could be reported in any order due to goroutine execution
				// interleaving, so compare sorted
				errs = errs[1:]
				actualErrs = make([]string, len(reportedMsgs))
				copy(actualErrs, reportedMsgs)
				sort.Strings(actualErrs)
			}
			if reflect.DeepEqual(errs, actualErrs) {
				match = errs
				break
			}
		}
		assert.NotNil(t, match, "case #%d: reported errors do not match expected", i+1)

		fmt.Printf("---- case #%d: fail fast ----\n", i+1)
		count = 0
		compiler.Reporter = reporter.NewReporter(failFastReporter(&count), nil)
		_, err = compiler.Compile(ctx, tc.fileNames...)
		assert.Equal(t, fail, err, "case #%d: parse should have failed fast", i+1)
		assert.Equal(t, 1, count, "case #%d: parse should have called reporter only once", i+1)

		fmt.Printf("---- case #%d: error limit ----\n", i+1)
		count = 0
		compiler.Reporter = reporter.NewReporter(limitedErrReporter(2, &count), nil)
		_, err = compiler.Compile(ctx, tc.fileNames...)
		if count > 2 {
			assert.Equal(t, tooManyErrors, err, "case #%d: parse should have failed with too many errors", i+1)
			assert.Equal(t, 3, count, "case #%d: parse should have called reporter 3 times", i+1)
			// this should only be possible if one of the errors scenarios expects >2 errors
			maxErrs := 0
			for _, errs := range tc.expectedErrs {
				if len(errs) > maxErrs {
					maxErrs = len(errs)
				}
			}
			assert.Greater(t, maxErrs, 2, "case #%d: should not have called reporter so many times (%d), max expected errors only %d", i+1, count, maxErrs)
		} else {
			// less than threshold means reporter always returned nil,
			// so parse returns ErrInvalidSource sentinel
			assert.Equal(t, reporter.ErrInvalidSource, err, "case #%d: parse should have failed with invalid source error", i+1)
			// the number of errors reported should match some error scenario
			okay := false
			for _, errs := range tc.expectedErrs {
				if len(errs) == count {
					okay = true
					break
				}
			}
			assert.True(t, okay, "case #%d: parse called reporter unexpected number of times (%d)", i+1, count)
		}
	}
}

func TestWarningReporting(t *testing.T) {
	t.Parallel()
	type msg struct {
		pos  ast.SourcePos
		text string
	}

	testCases := []struct {
		name            string
		sources         map[string]string
		expectedNotices []string
	}{
		{
			name: "syntax proto2",
			sources: map[string]string{
				"test.proto": `syntax = "proto2"; message Foo {}`,
			},
		},
		{
			name: "syntax proto3",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; message Foo {}`,
			},
		},
		{
			name: "no syntax",
			sources: map[string]string{
				"test.proto": `message Foo {}`,
			},
			expectedNotices: []string{
				"test.proto:1:1: no syntax specified; defaulting to proto2 syntax",
			},
		},
		{
			name: "used import",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import "foo.proto"; message Foo { Bar bar = 1; }`,
				"foo.proto":  `syntax = "proto3"; message Bar { string name = 1; }`,
			},
		},
		{
			name: "used public import",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import "foo.proto"; message Foo { Bar bar = 1; }`,
				// we're only asking to compile test.proto, so we won't report unused import for baz.proto
				"foo.proto": `syntax = "proto3"; import public "bar.proto"; import "baz.proto";`,
				"bar.proto": `syntax = "proto3"; message Bar { string name = 1; }`,
				"baz.proto": `syntax = "proto3"; message Baz { }`,
			},
		},
		{
			name: "used nested public import",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import "foo.proto"; message Foo { Bar bar = 1; }`,
				"foo.proto":  `syntax = "proto3"; import public "baz.proto";`,
				"baz.proto":  `syntax = "proto3"; import public "bar.proto";`,
				"bar.proto":  `syntax = "proto3"; message Bar { string name = 1; }`,
			},
		},
		{
			name: "unused import",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import "foo.proto"; message Foo { string name = 1; }`,
				"foo.proto":  `syntax = "proto3"; message Bar { string name = 1; }`,
			},
			expectedNotices: []string{
				`test.proto:1:20: import "foo.proto" not used`,
			},
		},
		{
			name: "multiple unused imports",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import "foo.proto"; import "bar.proto"; import "baz.proto"; message Test { Bar bar = 1; }`,
				"foo.proto":  `syntax = "proto3"; message Foo {};`,
				"bar.proto":  `syntax = "proto3"; message Bar {};`,
				"baz.proto":  `syntax = "proto3"; message Baz {};`,
			},
			expectedNotices: []string{
				`test.proto:1:20: import "foo.proto" not used`,
				`test.proto:1:60: import "baz.proto" not used`,
			},
		},
		{
			name: "unused public import is not reported",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import public "foo.proto"; message Foo { }`,
				"foo.proto":  `syntax = "proto3"; message Bar { string name = 1; }`,
			},
		},
		{
			name: "unused descriptor.proto import",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import "google/protobuf/descriptor.proto"; message Foo { }`,
			},
			expectedNotices: []string{
				`test.proto:1:20: import "google/protobuf/descriptor.proto" not used`,
			},
		},
		{
			name: "explicitly used descriptor.proto import",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import "google/protobuf/descriptor.proto"; extend google.protobuf.MessageOptions { string foobar = 33333; }`,
			},
		},
		{
			// having options implicitly uses decriptor.proto
			name: "implicitly used descriptor.proto import",
			sources: map[string]string{
				"test.proto": `syntax = "proto3"; import "google/protobuf/descriptor.proto"; message Foo { option deprecated = true; }`,
			},
		},
		{
			// makes sure we can use a given descriptor.proto to override non-custom options
			name: "implicitly used descriptor.proto import with new option",
			sources: map[string]string{
				"test.proto":                       `syntax = "proto3"; import "google/protobuf/descriptor.proto"; message Foo { option foobar = 123; }`,
				"google/protobuf/descriptor.proto": `syntax = "proto2"; package google.protobuf; message MessageOptions { optional fixed32 foobar = 99; }`,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var msgs []msg
			rep := func(warn reporter.ErrorWithPos) {
				msgs = append(msgs, msg{
					pos: warn.GetPosition(), text: warn.Unwrap().Error(),
				})
			}
			compiler := Compiler{
				Resolver: WithStandardImports(&SourceResolver{Accessor: SourceAccessorFromMap(testCase.sources)}),
				Reporter: reporter.NewReporter(nil, rep),
			}
			_, err := compiler.Compile(ctx, "test.proto")
			require.NoError(t, err)

			var actualNotices []string
			if len(msgs) > 0 {
				actualNotices = make([]string, len(msgs))
				for j, msg := range msgs {
					actualNotices[j] = fmt.Sprintf("%s: %s", msg.pos, msg.text)
				}
			}
			assert.Equal(t, testCase.expectedNotices, actualNotices)
		})
	}
}
