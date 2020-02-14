// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
