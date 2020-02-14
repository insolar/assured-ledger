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
	"reflect"
	"sync/atomic"
)

var panicFmtFn atomic.Value

type FormatterFunc func(string, interface{}) (msg string, extra interface{}, includeExtra bool)

func GetFormatter() FormatterFunc {
	if vv, ok := panicFmtFn.Load().(FormatterFunc); ok {
		return vv
	}
	return nil
}

func SetFormatter(fn FormatterFunc) {
	if fn == nil {
		panic(IllegalValue())
	}
	panicFmtFn.Store(fn)
}

func WrapFmt(msg string, errorDescription interface{}, fmtFn FormatterFunc) error {
	if fmtFn == nil {
		panic(IllegalValue())
	}
	if errorDescription == nil && msg == "" {
		return nil
	}
	return _wrap(fmtFn(msg, errorDescription))
}

func Wrap(errorDescription interface{}) error {
	if errorDescription == nil {
		return nil
	}
	if fmtFn := GetFormatter(); fmtFn != nil {
		return _wrap(fmtFn("", errorDescription))
	}
	if err, ok := errorDescription.(error); ok {
		return err
	}
	return _wrapDesc(errorDescription)
}

func WrapMsg(msg string, errorDescription interface{}) error {
	if errorDescription == nil && msg == "" {
		return nil
	}
	if fmtFn := GetFormatter(); fmtFn != nil {
		return _wrap(fmtFn(msg, errorDescription))
	}
	s := ""
	if vv, ok := errorDescription.(logStringer); ok {
		s = vv.LogString()
	} else {
		s = defaultFmt(errorDescription, false)
	}
	return fmtWrap{msg: msg, extra: errorDescription, useExtra: msg != s}
}

func wrapInternal(errorDescription interface{}) fmtWrap {
	if fmtFn := GetFormatter(); fmtFn != nil {
		return _wrap(fmtFn("", errorDescription))
	}
	return _wrapDesc(errorDescription)
}

func _wrapDesc(errorDescription interface{}) fmtWrap {
	if vv, ok := errorDescription.(logStringer); ok {
		msg := vv.LogString()
		return fmtWrap{msg: msg, extra: errorDescription}
	}
	msg := defaultFmt(errorDescription, true)
	return fmtWrap{msg: msg, extra: errorDescription}
}

func _wrap(msg string, extra interface{}, useExtra bool) fmtWrap {
	return fmtWrap{msg: msg, extra: extra, useExtra: useExtra}
}

func UnwrapExtraInfo(err error) (interface{}, bool) {
	if e, ok := err.(interface{ ExtraInfo() interface{} }); ok {
		return e.ExtraInfo(), true
	}
	return nil, false
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
		s := ""
		if t := reflect.TypeOf(v); t.Kind() == reflect.Struct && t.PkgPath() == "" {
			return fmt.Sprintf("%+v", vv)
		} else if force {
			s = fmt.Sprint(vv)
		} else {
			return ""
		}
		if len(s) == 0 {
			s = fmt.Sprintf("%T(%[1]p)", vv)
		}
		return s
	}
	return "<nil>"
}
