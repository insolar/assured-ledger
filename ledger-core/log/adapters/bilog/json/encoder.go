// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package json

import (
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/args"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/msgencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

func EncoderManager() msgencoder.EncoderFactory {
	return encoderMgr
}

var encoderMgr = encoderManager{}

type encoderManager struct{}

func (m encoderManager) CreateMetricWriter(downstream io.Writer, fieldName string, reportFn logcommon.DurationReportFunc) (io.Writer, error) {
	w := &MetricTimeWriter{downstream, []byte(tail), reportFn, nil}
	if fieldName != "" {
		w.AppendFn = func(dst []byte, d time.Duration) []byte {
			dst = AppendKey(dst, fieldName)
			return AppendString(dst, args.DurationFixedLen(d, 7))
		}
	}
	return w, nil
}

func (encoderManager) CreateEncoder(config logfmt.MsgFormatConfig) msgencoder.Encoder {
	return jsonEncoder{config.Sformat, config.Sformatf, config.TimeFmt}
}

var _ msgencoder.Encoder = jsonEncoder{}

type jsonEncoder struct {
	sformat  logfmt.FormatFunc
	sformatf logfmt.FormatfFunc
	timeFmt  string
}

func (p jsonEncoder) PrepareBuffer(dst []byte, key string, level log.Level) []byte {
	dst = append(dst, `{`...)
	dst = AppendKey(dst, key)

	levelStr := ""
	if level.IsValid() {
		levelStr = level.String()
	}
	return AppendString(dst, levelStr)
}

const tail = "}\n"

func (p jsonEncoder) FinalizeBuffer(dst []byte, metricTime time.Time) []byte {
	return AppendMetricTimeMark(append(dst, tail...), metricTime)
}

func (p jsonEncoder) appendStrf(dst []byte, f string, a ...interface{}) []byte {
	return AppendString(dst, p.sformatf(f, a...))
}

func (p jsonEncoder) AppendIntField(dst []byte, key string, v int64, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}
	return strconv.AppendInt(dst, v, 10)
}

func (p jsonEncoder) AppendUintField(dst []byte, key string, v uint64, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)

	switch {
	case fFmt.Kind == reflect.Uintptr:
		if !fFmt.HasFmt {
			return append(dst, p.sformat(uintptr(v))...)
		}
		return p.appendStrf(dst, fFmt.Fmt, uintptr(v))
	case fFmt.HasFmt:
		return p.appendStrf(dst, fFmt.Fmt, v)
	default:
		return strconv.AppendUint(dst, v, 10)
	}
}

func (p jsonEncoder) AppendBoolField(dst []byte, key string, v bool, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}
	return strconv.AppendBool(dst, v)
}

func (p jsonEncoder) AppendFloatField(dst []byte, key string, v float64, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		if fFmt.Kind == reflect.Float32 {
			return p.appendStrf(dst, fFmt.Fmt, float32(v))
		}
		return p.appendStrf(dst, fFmt.Fmt, v)
	}

	bits := 64
	if fFmt.Kind == reflect.Float32 {
		bits = 32
	}
	return AppendFloat(dst, v, bits)
}

func (p jsonEncoder) AppendComplexField(dst []byte, key string, v complex128, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}
	bits := 64
	if fFmt.Kind == reflect.Complex64 {
		bits = 32
	}
	dst = append(dst, '[')
	dst = AppendFloat(dst, real(v), bits)
	dst = append(dst, ',')
	dst = AppendFloat(dst, imag(v), bits)
	return append(dst, ']')
}

func (p jsonEncoder) AppendStrField(dst []byte, key string, v string, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}
	return AppendString(dst, v)
}

func (p jsonEncoder) AppendIntfField(dst []byte, key string, v interface{}, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}

	return AppendMarshalled(dst, v)
}

func (p jsonEncoder) AppendRawJSONField(dst []byte, key string, v interface{}, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return append(dst, p.sformatf(fFmt.Fmt, v)...)
	}

	switch vv := v.(type) {
	case string:
		return append(dst, vv...)
	case []byte:
		return append(dst, vv...)
	}

	return AppendMarshalled(dst, v)
}

func (p jsonEncoder) AppendTimeField(dst []byte, key string, v time.Time, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return AppendString(dst, v.Format(fFmt.Fmt))
	}

	return AppendString(dst, v.Format(p.timeFmt))
}
