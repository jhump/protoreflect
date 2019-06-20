package protoparse

import (
	"errors"
	"testing"
)

func TestErrorReporting(t *testing.T) {
	tooManyErrors := errors.New("too many errors")
	limitedErrReporter := func(limit int) ErrorReporter {
		numErrs := 0
		return func(err ErrorWithPos) error {
			numErrs++
			if numErrs > limit {
				return tooManyErrors
			}
			return nil
		}
	}
	trackingReporter := func(errs *[]ErrorWithPos) ErrorReporter {
		return func(err ErrorWithPos) error {
			*errs = append(*errs, err)
			return nil
		}
	}
	fail := errors.New("failure!")
	failFastReporter := ErrorReporter(func(err ErrorWithPos) error {
		return fail
	})

	testCases := []struct {
		name         string
		fileNames    []string
		files        map[string]string
		expectedErrs []string
	}{
		{},
	}

	for _, tc := range testCases {
		var p Parser
		p.ErrorReporter = failFastReporter
		//TODO
		p.ErrorReporter = limitedErrReporter(5)
		//TODO
		var reported []ErrorWithPos
		p.ErrorReporter = trackingReporter(&reported)
		//TODO
		_ = tc

	}
}
