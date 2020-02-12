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
	"fmt"
	"math"
	"sync/atomic"
)

const NoStackTrace int = math.MaxInt32

var panicFmtFn atomic.Value

func GetPanicFmt() func(interface{}) string {
	return panicFmtFn.Load().(func(interface{}) string)
}

func SetPanicFmt(fn func(interface{}) string) {
	if fn == nil {
		panic(IllegalValue())
	}
	panicFmtFn.Store(fn)
}

func WrapPanic(recovered interface{}) error {
	return WrapPanicExt(recovered, 2) // Wrap*() + defer
}

func WrapPanicNoStack(recovered interface{}) error {
	return WrapPanicExt(recovered, NoStackTrace)
}

func WrapPanicExt(recovered interface{}, skipFrames int) error {
	return WrapPanicFmt(recovered, skipFrames, GetPanicFmt())
}

func WrapPanicFmt(recovered interface{}, skipFrames int, strFn func(interface{}) string) error {
	if recovered == nil {
		return nil
	}

	if skipFrames >= NoStackTrace {
		if pw, ok := recovered.(panicWrap); ok {
			if strFn != nil {
				pw.strFn = strFn
			}
			return pw
		}
		return panicWrap{recovered: recovered, strFn: strFn}
	}

	st := CaptureStack(skipFrames + 1)
	if pw, ok := recovered.(panicWrap); ok {
		if equalStackTrace(pw.StackTrace(), st) {
			if strFn != nil {
				pw.strFn = strFn
			}
			return pw
		}
	}

	return panicWrap{st: st, recovered: recovered, strFn: strFn}
}

func IsPanicWrap(err error) bool {
	_, ok := err.(panicWrap)
	return ok
}

type ErrorWithStackTrace interface {
	error
	StackTraceHolder
}

func InnermostPanicWithStack(recovered interface{}) ErrorWithStackTrace {
	switch vv := recovered.(type) {
	case error:
		return innermostWithStack(vv)
	case StackTraceHolder:
		st := vv.StackTrace()
		if st == nil {
			return nil
		}
		if err := innermostWithStack(vv.Unwrap()); err != nil {
			return err
		}
		return panicWrap{st: st, recovered: recovered}

	default:
		return nil
	}
}

func innermostWithStack(errChain error) ErrorWithStackTrace {
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
	strFn      func(interface{}) string
}

func (v panicWrap) StackTrace() StackTrace {
	return v.st
}

func (v panicWrap) Unwrap() error {
	if err, ok := v.recovered.(error); ok {
		return err
	}
	return nil
}

func (v panicWrap) Error() string {
	if v.strFn != nil {
		return v.strFn(v.recovered)
	}

	switch vv := v.recovered.(type) {
	case error:
		return vv.Error()
	case fmt.Stringer:
		return vv.String()
	case string:
		return vv
	case *string:
		if vv != nil {
			return *vv
		}
		return "<nil>"
	default:
		return fmt.Sprint(vv)
	}
}

func (v panicWrap) String() string {
	if v.st == nil {
		return v.Error()
	}
	return v.Error() + "\n" + StackTracePrefix + v.st.StackTraceAsText()
}
