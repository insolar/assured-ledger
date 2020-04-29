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

type DeepestStackMode uint8

const (
	_ DeepestStackMode = iota
	InheritedTrace
	SupersededTrace
)

type StackTraceHolder interface {
	// Cause returns a cause for this stack trace. It can NOT be used like Unwrap() as it may return self.
	Cause() error
	// ShallowStackTrace returns a stack trace registered for this cause
	ShallowStackTrace() StackTrace
	// DeepestStackTrace returns this or the deepest stack trace from cause's error chain that embeds ShallowStackTrace.
	// Bool value is true when the stack trace was inherited from the cause.
	DeepestStackTrace() (StackTrace, DeepestStackMode)
}

// StackOf returns a target-matched entry with a wrapping StackTraceHolder (optional)
// When (target) is nil - returns the last error of the chain
// nolint
func StackOf(errChain, target error) (error, StackTraceHolder, bool) {
	isComparable := target == nil || reflect.TypeOf(target).Comparable()

	for errChain != nil {
		nextErr := errors.Unwrap(errChain)
		if sw, ok := errChain.(StackTraceHolder); ok {
			if target == nil && nextErr == nil || isThis(isComparable, sw.Cause(), target) {
				return errChain, sw, true
			}
		} else if target == nil && nextErr == nil || isThis(isComparable, errChain, target) {
			return errChain, nil, true
		}
		errChain = nextErr
	}

	return nil, nil, false
}

// NearestStackOf returns a target-matched entry with a last encountered StackTraceHolder with a either shallow or deep non-nil stack (optional)
// When (target) is nil - returns the last error of the chain
// When (target) is not found, returns only the last StackTraceHolder.
// nolint
func NearestStackOf(errChain, target error) (error, StackTraceHolder, bool) {
	isComparable := target == nil || reflect.TypeOf(target).Comparable()

	var sth StackTraceHolder
	for errChain != nil {
		nextErr := errors.Unwrap(errChain)
		if sw, ok := errChain.(StackTraceHolder); ok {
			if st, _ := sw.DeepestStackTrace(); st != nil {
				sth = sw
			}
			if target == nil && nextErr == nil || isThis(isComparable, sw.Cause(), target) {
				return errChain, sth, true
			}
		} else if target == nil && nextErr == nil || isThis(isComparable, errChain, target) {
			return errChain, sth, true
		}
		errChain = nextErr
	}

	return nil, sth, false
}

func DeepestStackTraceOf(errChain error) StackTrace {
	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok {
			if st, _ := sw.DeepestStackTrace(); st != nil {
				return st
			}
		}
		errChain = errors.Unwrap(errChain)
	}
	return nil
}

func ErrorWithStack(errChain error) string {
	return JoinStackText(errChain.Error(), DeepestStackTraceOf(errChain))
}

func ErrorWithTopOrFullStack(errChain error) (string, StackTrace) {
	st := DeepestStackTraceOf(errChain)
	if st == nil || st.IsFullStack() {
		return errChain.Error(), st
	}
	return JoinStackText(errChain.Error(), st), nil
}

func ErrorWithTopOrMinimizedStack(errChain error, boundPackage string, includeBoundary bool) (string, StackTrace) {
	st := DeepestStackTraceOf(errChain)
	switch {
	case st == nil:
		return errChain.Error(), nil
	case !st.IsFullStack():
		return JoinStackText(errChain.Error(), st), nil
	}
	if boundPackage != "" {
		st = MinimizeStackTrace(st, boundPackage, includeBoundary)
	}
	min := ExtractStackTop(st, 0)
	return JoinStackText(errChain.Error(), min), st
}

// OutermostStack returns the closest StackTraceHolder with non-nil ShallowStackTrace from errChain
func OutermostStack(errChain error) StackTraceHolder {
	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok && sw.ShallowStackTrace() != nil {
			return sw
		}
		errChain = errors.Unwrap(errChain)
	}
	return nil
}

// InnermostStack returns the most distant StackTraceHolder with non-nil ShallowStackTrace from errChain
func InnermostStack(errChain error) (sth StackTraceHolder) {
	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok && sw.ShallowStackTrace() != nil {
			sth = sw
		}
		errChain = errors.Unwrap(errChain)
	}
	return
}

// InnermostFullStack returns the most distant StackTraceHolder with non-nil ShallowStackTrace and IsFullStack from errChain
func InnermostFullStack(errChain error) (sth StackTraceHolder) {
	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok && sw.ShallowStackTrace() != nil && sw.ShallowStackTrace().IsFullStack() {
			sth = sw
		}
		errChain = errors.Unwrap(errChain)
	}
	return
}

// Walks through the given error chain and call (fn) for each entry, does unwrapping for stack trace data.
func Walk(errChain error, fn func(error, StackTraceHolder) bool) bool {
	if fn == nil {
		panic(IllegalValue())
	}

	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok {
			if fn(sw.Cause(), sw) {
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
			trace = traceHolder.ShallowStackTrace()
		}
		switch {
		case err != nil:
			b.WriteString(err.Error())
			b.WriteByte('\n')
			if trace == nil || !includeStack {
				return false
			}
			b.WriteString(stackTracePrintPrefix)
		case trace != nil && includeStack:
			b.WriteString("<nil>\n" + stackTracePrintPrefix)
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

type iser interface {
	Is(error) bool
}

func isThis(isComparable bool, err, target error) bool {
	if isComparable && err == target {
		return true
	}
	if x, ok := err.(iser); ok && x.Is(target) {
		return true
	}
	return false
}
