//
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
//

package json

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/bilogencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

func EncoderManager() bilogencoder.EncoderFactory {
	return encoderMgr
}

var encoderMgr = encoderManager{}

type encoderManager struct{}

func (encoderManager) CreateEncoder(config logfmt.MsgFormatConfig) bilogencoder.Encoder {
	return jsonEncoder{config.Sformatf, config.TimeFmt}
}

var _ bilogencoder.Encoder = jsonEncoder{}

type jsonEncoder struct {
	sformatf logfmt.FormatfFunc
	timeFmt  string
	//	metricMode logcommon.LogMetricsMode
}

func (p jsonEncoder) PrepareBuffer(dst *[]byte, key string, level log.Level) {
	buf := append(*dst, `{`...)
	buf = AppendKey(buf, key)

	levelStr := ""
	if level.IsValid() {
		levelStr = level.String()
	}
	*dst = AppendString(*dst, levelStr)
}

func (p jsonEncoder) FinalizeBuffer(dst *[]byte, reportedAt time.Time) {
	*dst = append(*dst, `}\n`...)
}

func (p jsonEncoder) appendKey(dst *[]byte, key string) {
	*dst = AppendKey(*dst, key)
}

func (p jsonEncoder) appendStrf(dst *[]byte, f string, a ...interface{}) {
	*dst = AppendString(*dst, p.sformatf(f, a...))
}

func (p jsonEncoder) AppendIntField(dst *[]byte, key string, v int64, fFmt logfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		*dst = strconv.AppendInt(*dst, v, 10)
	}
}

func (p jsonEncoder) AppendUintField(dst *[]byte, key string, v uint64, fFmt logfmt.LogFieldFormat) {
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

func (p jsonEncoder) AppendBoolField(dst *[]byte, key string, v bool, fFmt logfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		*dst = strconv.AppendBool(*dst, v)
	}
}

func (p jsonEncoder) AppendFloatField(dst *[]byte, key string, v float64, fFmt logfmt.LogFieldFormat) {
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
		*dst = AppendFloat(*dst, v, bits)
	}
}

func (p jsonEncoder) AppendComplexField(dst *[]byte, key string, v complex128, fFmt logfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		bits := 64
		if fFmt.Kind == reflect.Complex64 {
			bits = 32
		}
		*dst = append(*dst, '[')
		*dst = AppendFloat(*dst, real(v), bits)
		*dst = append(*dst, ',')
		*dst = AppendFloat(*dst, imag(v), bits)
		*dst = append(*dst, ']')
	}
}

func (p jsonEncoder) AppendStrField(dst *[]byte, key string, v string, fFmt logfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		*dst = AppendString(*dst, v)
	}
}

func (p jsonEncoder) AppendIntfField(dst *[]byte, key string, v interface{}, fFmt logfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		p.appendStrf(dst, fFmt.Fmt, v)
	} else {
		marshaled, err := json.Marshal(v)
		if err != nil {
			p.appendStrf(dst, "marshaling error: %v", err)
		} else {
			*dst = append(*dst, marshaled...)
		}
	}
}

func (p jsonEncoder) AppendRawJSONField(dst *[]byte, key string, v interface{}, fFmt logfmt.LogFieldFormat) {
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
			marshaled, err := json.Marshal(vv)
			if err != nil {
				p.appendStrf(dst, "marshaling error: %v", err)
			} else {
				*dst = append(*dst, marshaled...)
			}
		}
	}
}

func (p jsonEncoder) AppendTimeField(dst *[]byte, key string, v time.Time, fFmt logfmt.LogFieldFormat) {
	p.appendKey(dst, key)
	if fFmt.HasFmt {
		*dst = append(*dst, v.Format(fFmt.Fmt)...)
	} else {
		*dst = append(*dst, v.Format(p.timeFmt)...)
	}
}
