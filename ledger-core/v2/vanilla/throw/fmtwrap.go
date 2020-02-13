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

type FormatterFunc func(string, interface{}) (string, interface{})

func GetFormatter() FormatterFunc {
	return panicFmtFn.Load().(FormatterFunc)
}

func SetFormatter(fn FormatterFunc) {
	if fn == nil {
		panic(IllegalValue())
	}
	panicFmtFn.Store(fn)
}

func WrapFmt(msg string, errorDescription interface{}, fmtFn FormatterFunc) error {
	if errorDescription == nil {
		return nil
	}
	if fmtFn == nil {
		panic(IllegalValue())
	}
	return _wrapFmt(fmtFn(msg, errorDescription))
}

func Wrap(errorDescription interface{}) error {
	return WrapMsg("", errorDescription)
}

func WrapMsg(msg string, errorDescription interface{}) error {
	if errorDescription == nil {
		return nil
	}
	if fmtFn := GetFormatter(); fmtFn != nil {
		return _wrapFmt(fmtFn(msg, errorDescription))
	}
	if msg == "" {
		if err, ok := errorDescription.(error); ok {
			return err
		}
	}

	s := defaultFmt(errorDescription, msg == "")
	switch {
	case s == "":
		s = msg
	case msg != "":
		s = msg + "\t" + s
	}
	return _wrapFmt(s, errorDescription)
}

func wrap(msg string, details interface{}) fmtWrap {
	if fmtFn := GetFormatter(); fmtFn != nil {
		msg, details = fmtFn(msg, details)
	} else {
		if err, ok := details.(error); ok {
			msg = err.Error()
		} else {
			msg = defaultFmt(details, msg == "")
		}
	}
	return _wrapFmt(msg, details)
}

func _wrapFmt(msg string, extra interface{}) fmtWrap {
	return fmtWrap{msg: msg, extra: extra}
}

func UnwrapExtraInfo(err error) (string, interface{}, bool) {
	switch vv := err.(type) {
	case fmtWrap:
		return vv.msg, vv.extra, true
	case panicWrap:
		return vv.msg, vv.extra, true
	}
	return "", nil, false
}

type logStringer interface{ LogString() string }

func defaultFmt(v interface{}, force bool) string {
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
		if !force {
			return ""
		}
		s := fmt.Sprint(vv)
		if len(s) == 0 {
			s = fmt.Sprintf("%T(%[1]p)", vv)
		}
		return s
	}
	return "<nil>"
}
