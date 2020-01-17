///
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
///

package text

import (
	std_json "encoding/json"
	"reflect"
	"strconv"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/json"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/bilogencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logmsgfmt"
)

func EncoderManager() bilogencoder.EncoderFactory {
	return encoderMgr
}

var encoderMgr = encoderManager{}

type encoderManager struct{}

func (encoderManager) CreateEncoder(config logmsgfmt.MsgFormatConfig) bilogencoder.Encoder {
	return textEncoder{config.Sformatf, config.TimeFmt}
}

var _ bilogencoder.Encoder = textEncoder{}

type textEncoder struct {
	sformatf logmsgfmt.FormatfFunc
	timeFmt  string
}

func (p textEncoder) PrepareBuffer(dst *[]byte, _ string, level log.LogLevel) {
	*dst = append(*dst, encodeLevel(level)...)
}

func encodeLevel(level log.LogLevel) string {
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

func (p textEncoder) FinalizeBuffer(dst *[]byte, reportedAt time.Time) {
	*dst = append(*dst, `\n`...)
}

func (p textEncoder) appendKey(dst *[]byte, key string) {
	b := *dst
	if len(b) != 0 {
		b = append(b, ' ')
	}
	//b = append(json.AppendString(b, key), '=')
	b = append(append(b, key...), '=')
	*dst = b
}

func (p textEncoder) appendStrf(dst *[]byte, f string, a ...interface{}) {
	*dst = json.AppendString(*dst, p.sformatf(f, a...))
}

func (p textEncoder) AppendIntField(dst *[]byte, key string, v int64, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		*dst = strconv.AppendInt(*dst, v, 10)
	}
}

func (p textEncoder) AppendUintField(dst *[]byte, key string, v uint64, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	switch {
	case fFmt.Kind == reflect.Uintptr:
		if !fFmt.HasFmt {
			fFmt.Fmt = "%v"
		}
		p.appendStrf(dst, fFmt.Fmt, uintptr(v))
	case fFmt.HasFmt:
		p.appendStrf(dst, fFmt.Fmt, v)
	default:
		*dst = strconv.AppendUint(*dst, uint64(v), 10)
	}
}

func (p textEncoder) AppendBoolField(dst *[]byte, key string, v bool, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		*dst = strconv.AppendBool(*dst, v)
	}
}

func (p textEncoder) AppendFloatField(dst *[]byte, key string, v float64, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		if fFmt.Kind == reflect.Float32 {
			p.appendStrf(dst, fFmt.Fmt, float32(v))
		} else {
			p.appendStrf(dst, fFmt.Fmt, v)
		}
	} else {
		bits := 64
		if fFmt.Kind == reflect.Float32 {
			bits = 32
		}
		*dst = json.AppendFloat(*dst, v, bits)
	}
}

func (p textEncoder) AppendComplexField(dst *[]byte, key string, v complex128, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		bits := 64
		if fFmt.Kind == reflect.Complex64 {
			bits = 32
		}
		*dst = append(*dst, '[')
		*dst = json.AppendFloat(*dst, real(v), bits)
		*dst = append(*dst, ',')
		*dst = json.AppendFloat(*dst, imag(v), bits)
		*dst = append(*dst, ']')
	}
}

func (p textEncoder) AppendStrField(dst *[]byte, key string, v string, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		*dst = json.AppendString(*dst, v)
	}
}

func (p textEncoder) AppendIntfField(dst *[]byte, key string, v interface{}, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		p.appendStrf(dst, "%v", v)
	}
}

func (p textEncoder) AppendRawJSONField(dst *[]byte, key string, v interface{}, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		*dst = append(*dst, p.sformatf(fFmt.Fmt, v)...)
	} else {
		switch vv := v.(type) {
		case string:
			*dst = append(*dst, vv...)
		case []byte:
			*dst = append(*dst, vv...)
		default:
			marshaled, err := std_json.Marshal(vv)
			if err != nil {
				p.appendStrf(dst, "marshaling error: %v", err)
			} else {
				*dst = append(*dst, marshaled...)
			}
		}
	}
}

func (p textEncoder) AppendTimeField(dst *[]byte, key string, v time.Time, fFmt logmsgfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		*dst = append(*dst, v.Format(fFmt.Fmt)...)
	} else {
		*dst = append(*dst, v.Format(p.timeFmt)...)
	}
}
