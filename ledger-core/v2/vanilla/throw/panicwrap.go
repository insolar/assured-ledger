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

type PanicHolder interface {
	StackTraceHolder
	Recovered() interface{}
}

const recoverSkipFrames = 2 // defer() + runtime.panic()

func WrapPanic(recovered interface{}) error {
	return WrapPanicExt(recovered, recoverSkipFrames+1) // WrapPanic() + defer
}

func WrapPanicNoStack(recovered interface{}) error {
	return WrapPanicExt(recovered, NoStackTrace)
}

func WrapPanicExt(recovered interface{}, skipFrames int) error {
	if recovered == nil {
		return nil
	}

	var st StackTrace
	if skipFrames < NoStackTrace {
		st = CaptureStack(skipFrames + 1)
	}
	stDeepest := st

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
		stDeepest = reuseSupersetTrace(stDeepest, vv.stDeepest)
	case fmtWrap:
		return panicWrap{st: st, stDeepest: stDeepest, fmtWrap: vv}
	case error:
		sth := OutermostStack(vv)
		if sth != nil {
			stDeepest = reuseSupersetTrace(stDeepest, sth.DeepestStackTrace())
		}
	}
	return panicWrap{st: st, stDeepest: stDeepest, recovered: recovered, fmtWrap: wrapInternal(recovered)}
}

func UnwrapPanic(err error) (interface{}, StackTrace, bool) {
	if vv, ok := err.(panicWrap); ok {
		return vv.recovered, vv.st, true
	}
	return err, nil, false
}

func InnermostPanicWithStack(recovered interface{}) PanicHolder {
	switch vv := recovered.(type) {
	case error:
		return innermostWithStack(vv)
	case PanicHolder:
		st := vv.StackTrace()
		if st != nil {
			return vv
		}
		return innermostWithStack(vv.Reason())
	default:
		return nil
	}
}

func innermostWithStack(errChain error) PanicHolder {
	for errChain != nil {
		if sw, ok := errChain.(panicWrap); ok {
			nextErr := sw.Unwrap()
			if sw.StackTrace() != nil && nextErr != nil {
				return sw
			}
			errChain = nextErr
			continue
		}
		errChain = errors.Unwrap(errChain)
	}
	return nil
}
