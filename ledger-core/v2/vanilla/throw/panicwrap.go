// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"errors"
	"math"
)

const NoStackTrace int = math.MaxInt32

// PanicHolder is a marker interface for a recovered panic
type PanicHolder interface {
	StackTraceHolder
	Recovered() interface{}
}

const recoverSkipFrames = 2 // defer() + runtime.panic()

// WrapPanic returns an error for the given recovered panic with a stack trace.
// The error is also marked as recovered panic. Will return nil for a nil value
// NB! Should be used inside a defer.
func WrapPanic(recovered interface{}) error {
	return WrapPanicExt(recovered, recoverSkipFrames+1) // WrapPanic() + defer
}

// WrapPanicNoStack returns an error for the given recovered panic without a stack trace.
// The error is also marked as recovered panic. Will return nil for a nil value
// NB! Should be used inside a defer.
func WrapPanicNoStack(recovered interface{}) error {
	return WrapPanicExt(recovered, NoStackTrace)
}

// WrapPanicNoStack returns an error for the given recovered panic with or without a stack trace.
// When (skipFrames) = NoStackTrace then no stack trace is included.
// The error is also marked as recovered panic. Will return nil for a nil value
func WrapPanicExt(recovered interface{}, skipFrames int) error {
	if recovered == nil {
		return nil
	}

	var st StackTrace
	if skipFrames < NoStackTrace {
		st = CaptureStack(skipFrames + 1)
	}

	var stDeepest StackTrace
	var stDeepMod DeepestStackMode

	switch vv := recovered.(type) {
	case panicWrap:
		if st == nil {
			return vv
		}
		switch CompareStackTrace(vv.st, st) {
		case EqualStack, SupersetStack:
			return vv
		}
	case stackWrap:
		stDeepest, stDeepMod = reuseSupersetTrace(st, vv.stDeepest)
	case fmtWrap:
		return panicWrap{st: st, stDeepest: stDeepest, fmtWrap: vv}
	case error:
		sth := OutermostStack(vv)
		if sth != nil {
			stDeep, _ := sth.DeepestStackTrace()
			stDeepest, stDeepMod = reuseSupersetTrace(st, stDeep)
		}
	}
	return panicWrap{st, recovered, stDeepest, wrapInternal(recovered), stDeepMod}
}

// InnermostPanicWithStack returns the most distant panic holder from the chain.
// The value can be either error or PanicHolder. Otherwise will return nil.
func InnermostPanicWithStack(errorOrHolder interface{}) PanicHolder {
	switch vv := errorOrHolder.(type) {
	case PanicHolder:
		st := vv.ShallowStackTrace()
		if st != nil {
			return vv
		}
		return innermostWithStack(vv.Cause())
	case error:
		return innermostWithStack(vv)
	default:
		return nil
	}
}

func innermostWithStack(errChain error) PanicHolder {
	for errChain != nil {
		if sw, ok := errChain.(panicWrap); ok {
			nextErr := sw.Unwrap()
			if sw.ShallowStackTrace() != nil && nextErr != nil {
				return sw
			}
			errChain = nextErr
			continue
		}
		errChain = errors.Unwrap(errChain)
	}
	return nil
}
