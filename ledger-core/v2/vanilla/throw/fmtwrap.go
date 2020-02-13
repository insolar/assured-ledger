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
	"fmt"
	"sync/atomic"
)

var panicFmtFn atomic.Value

type FormatterFunc func(interface{}) (string, interface{})

func GetFormatter() FormatterFunc {
	return panicFmtFn.Load().(FormatterFunc)
}

func SetFormatter(fn FormatterFunc) {
	if fn == nil {
		panic(IllegalValue())
	}
	panicFmtFn.Store(fn)
}

func Wrap(errorDescription interface{}) error {
	if errorDescription == nil {
		return nil
	}
	fmtFn := GetFormatter()
	if fmtFn != nil {
		return _wrapFmt(fmtFn(errorDescription))
	}
	if err, ok := errorDescription.(error); ok {
		return err
	}
	msg := defaultFmt(errorDescription)
	if _, ok := errorDescription.(logStringer); !ok {
		errorDescription = nil
	}
	return _wrapFmt(msg, errorDescription)
}

func WrapFmt(errorDescription interface{}, fmtFn FormatterFunc) error {
	if errorDescription == nil {
		return nil
	}
	if fmtFn == nil {
		panic(IllegalValue())
	}
	return _wrapFmt(fmtFn(errorDescription))
}

func _wrapFmt(msg string, extra interface{}) errWithFmt {
	return errWithFmt{msg: msg, extra: extra}
}

func UnwrapExtraInfo(err error) (string, interface{}, bool) {
	switch vv := err.(type) {
	case errWithFmt:
		return vv.msg, vv.extra, true
	case panicWrap:
		return vv.msg, vv.extra, true
	}
	return "", nil, false
}

type logStringer interface{ LogString() string }

func defaultFmt(v interface{}) string {
	switch vv := v.(type) {
	case fmt.Stringer:
		return vv.String()
	case error:
		return vv.Error()
	case string:
		return vv
	case *string:
		if vv != nil {
			return *vv
		}
	case nil:
		//
	default:
		s := fmt.Sprint(vv)
		if len(s) == 0 {
			s = fmt.Sprintf("%T(%[1]p)", vv)
		}
		return s
	}
	return "<nil>"
}

type errWithFmt struct {
	_logignore struct{} // will be ignored by struct-logger
	msg        string
	extra      interface{}
}

func (v errWithFmt) LogString() string {
	s := ""
	switch vv := v.extra.(type) {
	case nil:
		return v.msg
	case logStringer:
		s = vv.LogString()
	default:
		s = fmt.Sprint(vv)
	}
	if v.msg == "" {
		return s
	}
	return v.msg + " " + s
}

func (v errWithFmt) Error() string {
	return v.msg
}

func (v errWithFmt) ExtraInfo() interface{} {
	return v.extra
}
