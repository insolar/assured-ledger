//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package throw

import (
	"errors"
	"reflect"
	"strings"
)

// StackOf goes throw error chain and looks for a target that is wrapped by WithStack()
// Returns (_, true) if the given target was found. Please note that stack trace can be nil for the given target.
func StackOf(errChain, target error) (StackTrace, bool) {
	isComparable := target == nil || reflect.TypeOf(target).Comparable()

	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok {
			nextErr := sw.Unwrap()
			if equalErr(isComparable, nextErr, target) {
				return sw.StackTrace(), true
			}
			errChain = nextErr
			continue
		}
		if equalErr(isComparable, errChain, target) {
			return nil, true
		}
		errChain = errors.Unwrap(errChain)
	}

	return nil, false
}

func Outermost(errChain error) (error, StackTrace) {
	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok {
			errChain = sw.Unwrap()
			if st := sw.StackTrace(); st != nil {
				return errChain, st
			}
			continue
		}
		errChain = errors.Unwrap(errChain)
	}
	return nil, nil
}

func Innermost(errChain error) (err error, lst StackTrace) {
	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok {
			errChain = sw.Unwrap()
			if st := sw.StackTrace(); st != nil && errChain != nil {
				err = errChain
				lst = st
			}
			continue
		}
		errChain = errors.Unwrap(errChain)
	}
	return
}

// Walks through the given error chain and call (fn) for each entry, does unwrapping for stack trace data.
func Walk(errChain error, fn func(error, StackTrace) bool) bool {
	if fn == nil {
		panic(IllegalValue())
	}

	for errChain != nil {
		if sw, ok := errChain.(StackTraceHolder); ok {
			nextErr := sw.Unwrap()
			if fn(nextErr, sw.StackTrace()) {
				return true
			}
			errChain = nextErr
			continue
		}

		if fn(errChain, nil) {
			return true
		}
		errChain = errors.Unwrap(errChain)
	}
	return false
}

const StackTracePrefix = "Stack trace: "

// PrintTo calls Walk() and builds a full list of errors and corresponding stacks in the chain.
func PrintTo(errChain error, includeStack bool, b *strings.Builder) {
	Walk(errChain, func(err error, trace StackTrace) bool {
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

// Equal does panic-safe comparison of error values. Incomparable values will always return false.
// It does NOT use Is()
func Equal(err0, err1 error) bool {
	if err0 == nil || err1 == nil {
		return err0 == err1
	}
	return equalErr(reflect.TypeOf(err1).Comparable(), err0, err1)
}

func equalErr(isComparable bool, err0, err1 error) bool {
	if isComparable && err0 == err1 {
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
