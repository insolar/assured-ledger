// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"errors"
	"reflect"
	"strings"
)

type StackTraceHolder interface {
	// Reason returns a reason for the stack trace. It can NOT be used like Unwrap() as it may return self.
	Reason() error
	StackTrace() StackTrace
	DeepestStackTrace() StackTrace
}

// StackOf goes throw error chain and looks for a target that is wrapped by WithStack()
// Returns (_, true) if the given target was found. Please note that stack trace can be nil for the given target.
func StackOf(errChain, target error) (StackTraceHolder, bool) {
	isComparable := target == nil || reflect.TypeOf(target).Comparable()

	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok {
			reason := sw.Reason()
			if isThis(isComparable, reason, target) {
				return sw, true
			}
		} else if isThis(isComparable, errChain, target) {
			return nil, true
		}
		errChain = errors.Unwrap(errChain)
	}

	return nil, false
}

func OutermostStack(errChain error) StackTraceHolder {
	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok && sw.StackTrace() != nil {
			return sw
		}
		errChain = errors.Unwrap(errChain)
	}
	return nil
}

func InnermostStack(errChain error) (sth StackTraceHolder) {
	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok && sw.StackTrace() != nil {
			sth = sw
		}
		errChain = errors.Unwrap(errChain)
	}
	return
}

func InnermostError(errChain error) (error, StackTraceHolder) {
	for errChain != nil {
		nextErr := errors.Unwrap(errChain)

		if sw, ok := errChain.(StackTraceHolder); ok && errors.Unwrap(nextErr) == nil {
			if e := sw.Reason(); e != nil {
				return e, sw
			}
		}
		if nextErr == nil {
			return errChain, nil
		}
		errChain = nextErr
	}
	return nil, nil
}

// Walks through the given error chain and call (fn) for each entry, does unwrapping for stack trace data.
func Walk(errChain error, fn func(error, StackTraceHolder) bool) bool {
	if fn == nil {
		panic(IllegalValue())
	}

	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok {
			if fn(sw.Reason(), sw) {
				return true
			}
			errChain = errors.Unwrap(errChain)
		} else if fn(errChain, nil) {
			return true
		}
		errChain = errors.Unwrap(errChain)
	}
	return false
}

// PrintTo calls Walk() and builds a full list of errors and corresponding stacks in the chain.
func PrintTo(errChain error, includeStack bool, b *strings.Builder) {
	Walk(errChain, func(err error, traceHolder StackTraceHolder) bool {
		var trace StackTrace
		if traceHolder != nil {
			trace = traceHolder.StackTrace()
		}
		switch {
		case err != nil:
			b.WriteString(err.Error())
			b.WriteByte('\n')
			if trace == nil || !includeStack {
				return false
			}
			b.WriteString(StackTracePrefix)
		case trace != nil && includeStack:
			b.WriteString("<nil>\n" + StackTracePrefix)
		default:
			b.WriteString("<nil>\n")
			return false
		}

		if err := trace.WriteStackTraceTo(b); err != nil {
			panic(err)
		}
		b.WriteByte('\n')
		return false
	})
}

// Print is a convenience wrapper for PrintTo()
func Print(errChain error) string {
	if errChain == nil {
		return ""
	}
	b := strings.Builder{}
	PrintTo(errChain, true, &b)
	return b.String()
}

// IsEqual does panic-safe comparison of error values. Incomparable values will always return false.
// It uses Is() on both sides
func IsEqual(err0, err1 error) bool {
	if err0 == nil || err1 == nil {
		return err0 == err1
	}
	if reflect.TypeOf(err1).Comparable() && err0 == err1 {
		return true
	}
	if x, ok := err0.(interface{ Is(error) bool }); ok && x.Is(err1) {
		return true
	}
	if x, ok := err1.(interface{ Is(error) bool }); ok && x.Is(err0) {
		return true
	}
	return false
}

func isThis(isComparable bool, err, target error) bool {
	if isComparable && err == target {
		return true
	}
	if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
		return true
	}
	return false
}
