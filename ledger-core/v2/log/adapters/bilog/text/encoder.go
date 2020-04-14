// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package text

import (
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/args"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/json"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/msgencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

func EncoderManager() msgencoder.EncoderFactory {
	return encoderMgr
}

var encoderMgr = encoderManager{}

type encoderManager struct{}

func (m encoderManager) CreateMetricWriter(downstream io.Writer, fieldName string, reportFn logcommon.DurationReportFunc) (io.Writer, error) {
	w := &json.MetricTimeWriter{
		Writer:   downstream,
		Eol:      []byte(tail),
		ReportFn: reportFn,
		AppendFn: nil}
	if fieldName != "" {
		w.AppendFn = func(dst []byte, d time.Duration) []byte {
			dst = AppendKey(dst, fieldName)
			return json.AppendOptQuotedString(dst, args.DurationFixedLen(d, 7))
		}
	}
	return w, nil
}

func (encoderManager) CreateEncoder(config logfmt.MsgFormatConfig) msgencoder.Encoder {
	return textEncoder{config.Sformat, config.Sformatf, config.TimeFmt}
}

var _ msgencoder.Encoder = textEncoder{}

type textEncoder struct {
	sformat  logfmt.FormatFunc
	sformatf logfmt.FormatfFunc
	timeFmt  string
}

func (p textEncoder) PrepareBuffer(dst []byte, _ string, level log.Level) []byte {
	return append(dst, encodeLevel(level)...)
}

func encodeLevel(level log.Level) string {
	switch level {
	case log.DebugLevel:
		return "DBG"
	case log.InfoLevel:
		return "INF"
	case log.WarnLevel:
		return "WRN"
	case log.ErrorLevel:
		return "ERR"
	case log.FatalLevel:
		return "FTL"
	case log.PanicLevel:
		return "PNC"
	default:
		return "UNK"
	}
}

const tail = "\n"

func (p textEncoder) FinalizeBuffer(dst []byte, metricTime time.Time) []byte {
	return json.AppendMetricTimeMark(append(dst, tail...), metricTime)
}

func AppendKey(dst []byte, key string) []byte {
	if len(dst) != 0 {
		dst = append(dst, ' ')
	}
	return append(json.AppendOptQuotedString(dst, key), '=')
}

func (p textEncoder) appendStrf(dst []byte, f string, a ...interface{}) []byte {
	return json.AppendOptQuotedString(dst, p.sformatf(f, a...))
}

func (p textEncoder) AppendIntField(dst []byte, key string, v int64, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}
	return strconv.AppendInt(dst, v, 10)
}

func (p textEncoder) AppendUintField(dst []byte, key string, v uint64, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	switch {
	case fFmt.Kind == reflect.Uintptr:
		if fFmt.HasFmt {
			return p.appendStrf(dst, fFmt.Fmt, uintptr(v))
		}
		return append(dst, p.sformat(uintptr(v))...)
	case fFmt.HasFmt:
		return p.appendStrf(dst, fFmt.Fmt, v)
	default:
		return strconv.AppendUint(dst, v, 10)
	}
}

func (p textEncoder) AppendBoolField(dst []byte, key string, v bool, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}
	return strconv.AppendBool(dst, v)
}

func (p textEncoder) AppendFloatField(dst []byte, key string, v float64, fFmt logfmt.LogFieldFormat) []byte {
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
	return json.AppendFloat(dst, v, bits)
}

func (p textEncoder) AppendComplexField(dst []byte, key string, v complex128, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}

	bits := 64
	if fFmt.Kind == reflect.Complex64 {
		bits = 32
	}
	dst = append(dst, '(')
	dst = json.AppendFloat(dst, real(v), bits)
	if imag(v) >= 0 {
		dst = append(dst, '+')
	}
	dst = json.AppendFloat(dst, imag(v), bits)
	return append(dst, `i)`...)
}

func (p textEncoder) AppendStrField(dst []byte, key string, v string, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}
	return json.AppendOptQuotedString(dst, v)
}

func (p textEncoder) AppendIntfField(dst []byte, key string, v interface{}, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return p.appendStrf(dst, fFmt.Fmt, v)
	}
	return json.AppendOptQuotedString(dst, p.sformat(v))
}

func (p textEncoder) AppendRawJSONField(dst []byte, key string, v interface{}, fFmt logfmt.LogFieldFormat) []byte {
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

	return json.AppendMarshalled(dst, v)
}

func (p textEncoder) AppendTimeField(dst []byte, key string, v time.Time, fFmt logfmt.LogFieldFormat) []byte {
	dst = AppendKey(dst, key)
	if fFmt.HasFmt {
		return json.AppendOptQuotedString(dst, v.Format(fFmt.Fmt))
	}

	return json.AppendOptQuotedString(dst, v.Format(p.timeFmt))
}
