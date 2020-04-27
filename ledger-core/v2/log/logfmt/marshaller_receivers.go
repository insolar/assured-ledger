// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/reflectkit"
)

var _ reflectkit.TypedReceiver = fieldFmtReceiver{}

type fieldFmtReceiver struct {
	w LogObjectWriter
	*logField
}

func (f fieldFmtReceiver) def(t reflect.Kind) bool {
	if f.receiver.fmtTag == fmtTagText {
		f.w.AddStrField(f.receiver.key, f.receiver.fmtStr, LogFieldFormat{Kind: t})
		return true
	}
	return false
}

func (f fieldFmtReceiver) fmt(t reflect.Kind) LogFieldFormat {
	return LogFieldFormat{Fmt: f.receiver.fmtStr, Kind: t, HasFmt: f.receiver.fmtTag.HasFmt()}
}

func (f fieldFmtReceiver) ReceiveBool(t reflect.Kind, v bool) {
	switch {
	case f.def(t):
		return
	case f.receiver.fmtTag.IsRaw():
		f.w.AddRawJSONField(f.receiver.key, v, f.fmt(t))
	default:
		f.w.AddBoolField(f.receiver.key, v, f.fmt(t))
	}
}

func (f fieldFmtReceiver) ReceiveInt(t reflect.Kind, v int64) {
	switch {
	case f.def(t):
		return
	case f.receiver.fmtTag.IsRaw():
		f.w.AddRawJSONField(f.receiver.key, v, f.fmt(t))
	default:
		f.w.AddIntField(f.receiver.key, v, f.fmt(t))
	}
}

func (f fieldFmtReceiver) ReceiveUint(t reflect.Kind, v uint64) {
	switch {
	case f.def(t):
		return
	case f.receiver.fmtTag.IsRaw():
		f.w.AddRawJSONField(f.receiver.key, v, f.fmt(t))
	default:
		f.w.AddUintField(f.receiver.key, v, f.fmt(t))
	}
}

func (f fieldFmtReceiver) ReceiveFloat(t reflect.Kind, v float64) {
	switch {
	case f.def(t):
		return
	case f.receiver.fmtTag.IsRaw():
		f.w.AddRawJSONField(f.receiver.key, v, f.fmt(t))
	default:
		f.w.AddFloatField(f.receiver.key, v, f.fmt(t))
	}
}

func (f fieldFmtReceiver) ReceiveComplex(t reflect.Kind, v complex128) {
	switch {
	case f.def(t):
		return
	case f.receiver.fmtTag.IsRaw():
		f.w.AddRawJSONField(f.receiver.key, v, f.fmt(t))
	default:
		f.w.AddComplexField(f.receiver.key, v, f.fmt(t))
	}
}

func (f fieldFmtReceiver) ReceiveString(t reflect.Kind, v string) {
	switch {
	case f.def(t):
		return
	case f.receiver.fmtTag.IsRaw():
		f.w.AddRawJSONField(f.receiver.key, v, f.fmt(t))
	default:
		f.w.AddStrField(f.receiver.key, v, f.fmt(t))
	}
}

func (f fieldFmtReceiver) ReceiveZero(t reflect.Kind) {
	f.def(t)
}

func (f fieldFmtReceiver) ReceiveNil(t reflect.Kind) {
	switch {
	case f.def(t) || f.receiver.fmtTag.IsOpt():
		return
	case f.receiver.fmtTag.IsRaw():
		f.w.AddRawJSONField(f.receiver.key, nil, f.fmt(t))
	default:
		f.w.AddIntfField(f.receiver.key, nil, f.fmt(t))
	}
}

func (f fieldFmtReceiver) ReceiveIface(t reflect.Kind, v interface{}) {
	switch {
	case f.def(t):
		return
	case f.receiver.fmtTag.IsRaw():
		f.w.AddRawJSONField(f.receiver.key, v, f.fmt(t))
	default:
		f.w.AddIntfField(f.receiver.key, v, f.fmt(t))
	}
}

func (f fieldFmtReceiver) ReceiveElse(t reflect.Kind, v interface{}, isZero bool) {
	if f.def(t) || f.receiver.fmtTag.IsOpt() && isZero {
		return
	}

	if t == reflect.Func {
		fn := runtime.FuncForPC(reflectkit.CodeOf(v))
		s := "<nil>"
		if fn != nil {
			s = fn.Name()
			s = s[strings.LastIndex(s, "/")+1:]
		}
		if !f.receiver.fmtTag.IsRaw() {
			f.w.AddStrField(f.receiver.key, s, f.fmt(t))
			return
		}
	}

	if f.receiver.fmtTag.IsRaw() {
		f.w.AddRawJSONField(f.receiver.key, v, f.fmt(t))
	} else {
		f.w.AddIntfField(f.receiver.key, v, f.fmt(t))
	}
}

type stringCapturer struct {
	v string
	*MsgFormatConfig
}

func (p *stringCapturer) set(v interface{}, fmt LogFieldFormat) {
	if fmt.HasFmt {
		p.v = p.Sformatf(fmt.Fmt, v)
	} else {
		p.v = p.Sformat(v)
	}
}

func (p *stringCapturer) AddComplexField(_ string, v complex128, fmt LogFieldFormat) {
	p.set(v, fmt)
}

func (p *stringCapturer) AddRawJSONField(_ string, v interface{}, fmt LogFieldFormat) {
	p.set(v, fmt)
}

func (p *stringCapturer) AddIntField(_ string, v int64, fmt LogFieldFormat) {
	p.set(v, fmt)
}

func (p *stringCapturer) AddUintField(_ string, v uint64, fmt LogFieldFormat) {
	if fmt.Kind == reflect.Uintptr {
		p.set(uintptr(v), fmt)
	} else {
		p.set(v, fmt)
	}
}

func (p *stringCapturer) AddBoolField(_ string, v bool, fmt LogFieldFormat) {
	p.set(v, fmt)
}

func (p *stringCapturer) AddFloatField(_ string, v float64, fmt LogFieldFormat) {
	p.set(v, fmt)
}

func (p *stringCapturer) AddStrField(_ string, v string, fmt LogFieldFormat) {
	if fmt.HasFmt {
		p.set(v, fmt)
	} else {
		p.v = v
	}
}

func (p *stringCapturer) AddIntfField(_ string, v interface{}, fmt LogFieldFormat) {
	p.set(v, fmt)
}

func (p *stringCapturer) AddTimeField(key string, v time.Time, fFmt LogFieldFormat) {
	if fFmt.HasFmt {
		p.v = v.Format(fFmt.Fmt)
	} else {
		p.v = v.Format(p.TimeFmt)
	}
}

func (p *stringCapturer) AddErrorField(string, throw.StackTrace, throw.Severity, bool) {
	// ignore
}
