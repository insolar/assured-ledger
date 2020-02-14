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
		case EqualTrace, SupersetTrace:
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
