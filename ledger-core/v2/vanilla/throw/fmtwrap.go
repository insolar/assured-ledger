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

// SetFormatter sets a formatter to be applied by New and NewMsg
func SetFormatter(fn FormatterFunc) {
	if fn == nil {
		panic(IllegalValue())
	}
	panicFmtFn.Store(fn)
}

// New creates an error with the given description. Log structs can be used
func New(description interface{}) error {
	if description == nil {
		return nil
	}
	if fmtFn := GetFormatter(); fmtFn != nil {
		return _wrap(fmtFn("", description))
	}
	if err, ok := description.(error); ok {
		return err
	}
	return _wrapDesc(description)
}

// NewMsg creates an error with the given message text and description. Log structs can be used
func NewMsg(msg string, description interface{}) error {
	if description == nil && msg == "" {
		return nil
	}
	if fmtFn := GetFormatter(); fmtFn != nil {
		return _wrap(fmtFn(msg, description))
	}
	s := ""
	if vv, ok := description.(logStringer); ok {
		s = vv.LogString()
	} else {
		s = defaultFmt(description, false)
	}
	return fmtWrap{msg: msg, extra: description, useExtra: msg != s}
}

func NewFmt(msg string, errorDescription interface{}, fmtFn FormatterFunc) error {
	if fmtFn == nil {
		panic(IllegalValue())
	}
	if errorDescription == nil && msg == "" {
		return nil
	}
	return _wrap(fmtFn(msg, errorDescription))
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

func UnwrapExtraInfo(err interface{}) (string, interface{}, bool) {
	if e, ok := err.(interface{ ExtraInfo() (string, interface{}) }); ok {
		s, ei := e.ExtraInfo()
		return s, ei, true
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
