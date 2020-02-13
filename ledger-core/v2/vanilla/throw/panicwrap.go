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

func WrapPanic(recovered interface{}) error {
	return WrapPanicExt(recovered, 2) // Wrap*() + defer
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

	switch vv := recovered.(type) {
	case panicWrap:
		if st == nil || equalStackTrace(vv.StackTrace(), st) {
			return vv
		}
	case errWithFmt:
		return panicWrap{errWithFmt: vv}
	}

	var (
		msg   string
		extra interface{}
	)

	if fmtFn := GetFormatter(); fmtFn != nil {
		msg, extra = fmtFn(recovered)
	} else {
		if err, ok := recovered.(error); ok {
			msg = err.Error()
		} else {
			msg = defaultFmt(recovered)
		}
		if _, ok := recovered.(logStringer); ok {
			extra = recovered
		}
	}
	return panicWrap{st: st, recovered: recovered, errWithFmt: _wrapFmt(msg, extra)}
}

func UnwrapPanic(err error) (interface{}, StackTrace, bool) {
	if vv, ok := err.(panicWrap); ok {
		return vv.recovered, vv.st, true
	}
	return err, nil, false
}

func InnermostPanicWithStack(recovered interface{}) StackTraceHolder {
	switch vv := recovered.(type) {
	case error:
		return innermostWithStack(vv)
	case StackTraceHolder:
		st := vv.StackTrace()
		if st != nil {
			return vv
		}
		return innermostWithStack(vv.Reason())
	default:
		return nil
	}
}

func innermostWithStack(errChain error) StackTraceHolder {
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

type panicWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	st         StackTrace
	recovered  interface{}
	errWithFmt
}

func (v panicWrap) Reason() error {
	return v.Unwrap()
}

func (v panicWrap) StackTrace() StackTrace {
	return v.st
}

func (v panicWrap) Recovered() interface{} {
	if v.recovered == nil {
		return v.errWithFmt
	}
	return v.recovered
}

func (v panicWrap) Unwrap() error {
	if err, ok := v.recovered.(error); ok {
		return err
	}
	return nil
}

func (v panicWrap) LogString() string {
	if v.st == nil {
		return v.errWithFmt.LogString()
	}
	return v.errWithFmt.LogString() + "\n" + StackTracePrefix + v.st.StackTraceAsText()
}
