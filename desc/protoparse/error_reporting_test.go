package protoparse

import (
	"errors"
	"strings"
	"testing"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestErrorReporting(t *testing.T) {
	tooManyErrors := errors.New("too many errors")
	limitedErrReporter := func(limit int, count *int) ErrorReporter {
		return func(err ErrorWithPos) error {
			*count++
			if *count > limit {
				return tooManyErrors
			}
			return nil
		}
	}
	trackingReporter := func(errs *[]ErrorWithPos, count *int) ErrorReporter {
		return func(err ErrorWithPos) error {
			*count++
			*errs = append(*errs, err)
			return nil
		}
	}
	fail := errors.New("failure!")
	failFastReporter := func(count *int) ErrorReporter {
		return func(err ErrorWithPos) error {
			*count++
			return fail
		}
	}

	testCases := []struct {
		fileNames    []string
		files        map[string]string
		expectedErrs []string
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
			expectedErrs: []string{
				"test.proto:2:50: syntax value must be 'proto2' or 'proto3'",
				"test.proto:5:41: syntax error: unexpected \"enum\", expecting ';' or '.'",
				"test.proto:5:69: syntax error: unexpected ';', expecting '='",
				"test.proto:7:53: syntax error: unexpected '='",
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
					enum State { C }
					enum Bar {
						BAZ = 1;
						BUZZ = 1;
					}
					`,
			},
			expectedErrs: []string{
				"test.proto:4:62: tag number 0 must be greater than zero",
				"test.proto:6:56: syntax error: unexpected '}', expecting '='",
				"test.proto:8:55: enum Bar: proto3 requires that first value in enum have numeric value of 0",
				"test.proto:9:56: enum Bar: values BAZ and BUZZ both have the same numeric value 1; use allow_alias option if intentional",
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
						rpc Foo (Foo) returns (Foo);
						rpc Foo (Frob) returns (Nitz);
					}
					`,
			},
			expectedErrs: []string{
				"test.proto:8:49: duplicate symbol Bar.BAZ: already defined as enum value",
				"test.proto:10:41: duplicate symbol Bar: already defined as enum",
				"test.proto:12:49: duplicate symbol Bar.Foo: already defined as method",
				"test.proto:12:58: method Bar.Foo: unknown request type Frob",
				"test.proto:12:73: method Bar.Foo: unknown response type Nitz",
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
			expectedErrs: []string{
				"test1.proto:5:62: syntax error: unexpected '-', expecting int literal",
				"test1.proto:8:62: syntax error: unexpected ';', expecting \"returns\"",
				"test2.proto:7:49: syntax error: unexpected identifier, expecting \"option\" or \"rpc\" or ';' or '}'",
				"test2.proto:4:49: field Baz.foo: field has label LABEL_REQUIRED, but proto3 must omit labels other than 'repeated'",
			},
		},
		{
			// link errors across multiple files
			fileNames: []string{"test1.proto", "test2.proto"},
			files: map[string]string{
				"test1.proto": `
					syntax = "proto3";
					import "test2.proto";
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
						rpc DoSomething (Empty) returns (Foo);
					}
					`,
			},
			expectedErrs: []string{
				"test2.proto:4:41: duplicate symbol Bar: already defined as service in \"test1.proto\"",
				"test2.proto:7:41: duplicate symbol Foo: already defined as message in \"test1.proto\"",
				"test1.proto:8:75: method Bar.Frob: unknown response type Nitz",
				"test2.proto:8:82: method Foo.DoSomething: invalid response type: Foo is a service, not a message",
			},
		},
	}

	for i, tc := range testCases {
		var p Parser
		p.Accessor = FileContentsFromMap(tc.files)

		var reported []ErrorWithPos
		count := 0
		p.ErrorReporter = trackingReporter(&reported, &count)
		_, err := p.ParseFiles(tc.fileNames...)
		reportedMsgs := make([]string, len(reported))
		for j := range reported {
			reportedMsgs[j] = reported[j].Error()
		}
		t.Logf("case #%d: got %d errors:\n\t%s", i+1, len(reported), strings.Join(reportedMsgs, "\n\t"))

		// returns sentinel, but all actual errors in reported
		testutil.Eq(t, ErrInvalidSource, err, "case #%d: parse should have failed with invalid source error", i+1)
		testutil.Eq(t, len(tc.expectedErrs), count, "case #%d: parse should have called reporter %d times", i+1, len(tc.expectedErrs))
		testutil.Eq(t, len(tc.expectedErrs), len(reported), "case #%d: wrong number of errors reported", i+1)
		for j := range tc.expectedErrs {
			testutil.Eq(t, tc.expectedErrs[j], reported[j].Error(), "case #%d: parse error[%d] have %q; instead got %q", i+1, j, tc.expectedErrs[j], reported[j].Error())
			split := strings.SplitN(tc.expectedErrs[j], ":", 4)
			testutil.Eq(t, 4, len(split), "case #%d: expected %q [%d] to contain at least 4 elements split by :", i+1, tc.expectedErrs[j], j)
			testutil.Eq(t, split[3], " "+reported[j].Unwrap().Error(), "case #%d: parse error underlying[%d] have %q; instead got %q", i+1, j, split[3], reported[j].Unwrap().Error())
		}

		count = 0
		p.ErrorReporter = failFastReporter(&count)
		_, err = p.ParseFiles(tc.fileNames...)
		testutil.Eq(t, fail, err, "case #%d: parse should have failed fast", i+1)
		testutil.Eq(t, 1, count, "case #%d: parse should have called reporter only once", i+1)

		count = 0
		p.ErrorReporter = limitedErrReporter(3, &count)
		_, err = p.ParseFiles(tc.fileNames...)
		if len(tc.expectedErrs) > 3 {
			testutil.Eq(t, tooManyErrors, err, "case #%d: parse should have failed with too many errors", i+1)
			testutil.Eq(t, 4, count, "case #%d: parse should have called reporter 4 times", i+1)
		} else {
			// less than threshold means reporter always returned nil,
			// so parse returns ErrInvalidSource sentinel
			testutil.Eq(t, ErrInvalidSource, err, "case #%d: parse should have failed with invalid source error", i+1)
			testutil.Eq(t, len(tc.expectedErrs), count, "case #%d: parse should have called reporter %d times", i+1, len(tc.expectedErrs))
		}
	}
}
