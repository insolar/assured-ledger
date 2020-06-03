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

// SetFormatter sets a formatter to be applied by NewDescription and New
func SetFormatter(fn FormatterFunc) {
	if fn == nil {
		panic(IllegalValue())
	}
	panicFmtFn.Store(fn)
}

// NewDescription creates an error with the given description. Log structs can be used
func NewDescription(description interface{}) error {
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

// New creates an error with the given message text and description. Log structs can be used
func New(msg string, description ...interface{}) error {
	return Severe(0, msg, description...)
}

func Severe(severity Severity, msg string, description ...interface{}) error {
	if description == nil && msg == "" && severity == 0 {
		return nil
	}
	if fmtFn := GetFormatter(); fmtFn != nil {
		return _wrap(fmtFn(msg, description))
	}

	var err error
	if len(description) > 0 {
		d0 := description[0]
		s := ""
		if vv, ok := d0.(logStringer); ok {
			s = vv.LogString()
		} else {
			s = defaultFmt(d0, false)
		}
		err = fmtWrap{msg: msg, severity: severity, extra: d0, useExtra: msg != s}
	} else {
		err = fmtWrap{msg: msg, severity: severity}
	}

	for i := 1; i < len(description); i++ {
		if description[i] == nil {
			continue
		}
		err = WithDetails(err, description[i])
	}

	return err
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

type extraInfo interface {
	ExtraInfo() (string, Severity, interface{})
}

func UnwrapExtraInfo(err interface{}) (string, Severity, interface{}, bool) {
	if e, ok := err.(extraInfo); ok {
		st, sv, ei := e.ExtraInfo()
		return st, sv, ei, true
	}
	return "", 0, nil, false
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
		t := reflect.TypeOf(v)
		switch {
		case t.Kind() == reflect.Struct && t.PkgPath() == "":
			return fmt.Sprintf("%+v", vv)
		case force:
			s = fmt.Sprint(vv)
		default:
			return ""
		}
		if len(s) == 0 {
			s = fmt.Sprintf("%T(%[1]p)", vv)
		}
		return s
	}
	return "<nil>"
}
