// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pbuf

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/msgencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

func EncoderManager() msgencoder.EncoderFactory {
	return encoderMgr
}

var encoderMgr = encoderManager{}

type encoderManager struct {
	//names map[string]int
}

func (encoderManager) CreateMetricWriter(downstream io.Writer, fieldName string, reportFn logcommon.DurationReportFunc) (io.Writer, error) {
	return MetricTimeWriter{downstream, reportFn}, nil
}

func (encoderManager) CreateEncoder(config logfmt.MsgFormatConfig) msgencoder.Encoder {
	return pbufEncoder{sformat: config.Sformat, sformatf: config.Sformatf}
}

var _ msgencoder.Encoder = pbufEncoder{}

type pbufEncoder struct {
	sformat  logfmt.FormatFunc
	sformatf logfmt.FormatfFunc

	writeMetricFieldName string

	names      map[string]int
	encodeType bool

	// formatting is always allowed for Str/Intf/RawJSON fields
	enableFmt bool
}

const LevelFieldID = 16
const TypeFieldID = 17
const KeyFieldID = 18
const ValueFieldID = 19

const ErrorFieldType = reflect.Kind(100)
const TimeFieldType = reflect.Kind(101)
const DurationFieldType = reflect.Kind(102)

var fieldLogEntry = protokit.WireBytes.Tag(20)

var fieldLevel = protokit.WireVarint.Tag(LevelFieldID)
var fieldType = protokit.WireVarint.Tag(TypeFieldID)
var fieldKeyID = protokit.WireVarint.Tag(KeyFieldID)
var fieldKeyName = protokit.WireBytes.Tag(KeyFieldID)

func (p pbufEncoder) appendKey(dst []byte, key string, fk reflect.Kind, wt protokit.WireType) (*encodeBuf, protokit.WireTag) {
	b := &encodeBuf{dst}
	if id, ok := p.names[key]; ok {
		if id > ValueFieldID && id <= protokit.MaxFieldID {
			return b, wt.Tag(id)
		}
		fieldKeyID.MustWrite(b, uint64(id))
	} else {
		if fk != reflect.Invalid && p.encodeType {
			fieldType.MustWrite(b, uint64(fk))
		}
		kb := []byte(key)
		fieldKeyName.MustWrite(b, uint64(len(kb)))
		if _, err := b.Write(kb); err != nil {
			panic(err)
		}
	}
	return b, wt.Tag(ValueFieldID)
}

const prependFieldSize = 2                                   // = fieldLogEntry.TagSize()
const prependSize = binary.MaxVarintLen32 + prependFieldSize // fieldLogEntry.TagSize()

func (p pbufEncoder) PrepareBuffer(dst []byte, _ string, level log.Level) []byte {
	dst = append(dst, make([]byte, prependSize)...) // preallocate space for record length
	b := &encodeBuf{dst}
	if !level.IsValid() {
		level = logcommon.NoLevel
	}
	fieldLevel.MustWrite(b, uint64(level))
	return b.dst
}

func (p pbufEncoder) FinalizeBuffer(dst []byte, metricTime time.Time) []byte {

	metricUinxTime := uint64(0)
	if !metricTime.IsZero() {
		metricUinxTime = uint64(metricTime.UnixNano())
		if len(p.writeMetricFieldName) > 0 {
			dst = p.appendUint64(dst, p.writeMetricFieldName, DurationFieldType, metricUinxTime)
		}
	}

	n := uint64(len(dst))
	headSize := protokit.SizeVarint64(n) + fieldLogEntry.TagSize()
	headStart := prependSize - headSize
	if headStart < 0 {
		panic("message is too long")
	}
	b := &encodeBuf{dst[headStart:headStart:prependSize]}
	fieldLogEntry.MustWrite(b, n)
	if len(b.dst) != headSize {
		panic("unexpected")
	}

	if !metricTime.IsZero() {
		if len(p.writeMetricFieldName) > 0 {
			dst = append(dst, metricTimeMarker)
		} else {
			pos := len(dst)
			dst = append(dst, make([]byte, metricTimeMarkFullSize)...)
			dst[pos] = metricTimeMarker
			binary.LittleEndian.PutUint64(dst[pos+1:], metricUinxTime)
		}
	}

	return dst[headStart:]
}

func (p pbufEncoder) appendUint64(dst []byte, key string, ft reflect.Kind, val uint64) []byte {
	b, wt := p.appendKey(dst, key, ft, protokit.WireFixed64)
	wt.MustWrite(b, val)
	return b.dst
}

func (p pbufEncoder) appendUint32(dst []byte, key string, ft reflect.Kind, val uint32) []byte {
	b, wt := p.appendKey(dst, key, ft, protokit.WireFixed32)
	wt.MustWrite(b, uint64(val))
	return b.dst
}

func (p pbufEncoder) appendVarint(dst []byte, key string, ft reflect.Kind, val uint64) []byte {
	b, wt := p.appendKey(dst, key, ft, protokit.WireVarint)
	wt.MustWrite(b, val)
	return b.dst
}

func (p pbufEncoder) appendStr(dst []byte, key string, ft reflect.Kind, val string) []byte {
	return p._appendBytes(dst, key, ft, []byte(val))
}

func (p pbufEncoder) appendStrf(dst []byte, key string, ft reflect.Kind, f string, a ...interface{}) []byte {
	return p.appendStr(dst, key, ft, p.sformatf(f, a...))
}

func (p pbufEncoder) appendFmt(dst []byte, key string, val interface{}, fFmt logfmt.LogFieldFormat) []byte {
	if !fFmt.HasFmt {
		panic("illegal value")
	}
	return p._appendBytes(dst, key, fFmt.Kind, []byte(p.sformatf(fFmt.Fmt, val)))
}

func (p pbufEncoder) appendBytes(dst []byte, key string, val []byte) []byte {
	return p._appendBytes(dst, key, reflect.Slice, val)
}

func (p pbufEncoder) _appendBytes(dst []byte, key string, ft reflect.Kind, val []byte) []byte {
	b, wt := p.appendKey(dst, key, ft, protokit.WireBytes)
	wt.MustWrite(b, uint64(len(val)))
	if _, err := b.Write(val); err != nil {
		panic(err)
	}
	return b.dst
}

func (p pbufEncoder) AppendIntField(dst []byte, key string, v int64, fFmt logfmt.LogFieldFormat) []byte {
	if fFmt.HasFmt && p.enableFmt {
		return p.appendFmt(dst, key, v, fFmt)
	}
	return p.appendVarint(dst, key, fFmt.Kind, uint64(v))
}

func (p pbufEncoder) AppendUintField(dst []byte, key string, v uint64, fFmt logfmt.LogFieldFormat) []byte {
	switch {
	case fFmt.HasFmt && p.enableFmt:
		if fFmt.Kind == reflect.Uintptr {
			return p.appendFmt(dst, key, uintptr(v), fFmt)
		}
		return p.appendFmt(dst, key, v, fFmt)

	case fFmt.Kind == reflect.Uint || fFmt.Kind >= reflect.Uint64:
		return p.appendUint64(dst, key, fFmt.Kind, v)
	default:
		return p.appendUint32(dst, key, fFmt.Kind, uint32(v))
	}
}

func (p pbufEncoder) AppendBoolField(dst []byte, key string, v bool, fFmt logfmt.LogFieldFormat) []byte {
	switch {
	case fFmt.HasFmt && p.enableFmt:
		return p.appendFmt(dst, key, v, fFmt)
	case v:
		return p.appendVarint(dst, key, fFmt.Kind, 1)
	default:
		return p.appendVarint(dst, key, fFmt.Kind, 0)
	}
}

func (p pbufEncoder) AppendFloatField(dst []byte, key string, v float64, fFmt logfmt.LogFieldFormat) []byte {
	switch {
	case fFmt.HasFmt && p.enableFmt:
		if fFmt.Kind == reflect.Float32 {
			return p.appendFmt(dst, key, float32(v), fFmt)
		}
		return p.appendFmt(dst, key, v, fFmt)

	case fFmt.Kind == reflect.Float32:
		return p.appendUint32(dst, key, fFmt.Kind, math.Float32bits(float32(v)))
	default:
		return p.appendUint64(dst, key, fFmt.Kind, math.Float64bits(v))
	}
}

func (p pbufEncoder) AppendComplexField(dst []byte, key string, v complex128, fFmt logfmt.LogFieldFormat) []byte {
	if fFmt.HasFmt && p.enableFmt {
		return p.appendFmt(dst, key, v, fFmt)
	}
	return p.appendStrf(dst, key, fFmt.Kind, "%v", v)
}

func (p pbufEncoder) AppendTimeField(dst []byte, key string, v time.Time, fFmt logfmt.LogFieldFormat) []byte {
	if fFmt.HasFmt && p.enableFmt {
		return p.appendFmt(dst, key, v, fFmt)
	}
	return p.appendUint64(dst, key, TimeFieldType, uint64(v.UnixNano()))
}

func (p pbufEncoder) AppendStrField(dst []byte, key string, v string, fFmt logfmt.LogFieldFormat) []byte {
	if fFmt.HasFmt {
		return p.appendFmt(dst, key, v, fFmt)
	}
	return p.appendStr(dst, key, fFmt.Kind, v)
}

func (p pbufEncoder) AppendIntfField(dst []byte, key string, v interface{}, fFmt logfmt.LogFieldFormat) []byte {
	if fFmt.HasFmt {
		return p.appendFmt(dst, key, v, fFmt)
	}

	switch vv := v.(type) {
	case []byte:
		return p.appendBytes(dst, key, vv)
	case string:
		return p.appendStr(dst, key, fFmt.Kind, vv)
	}

	return p.appendStr(dst, key, fFmt.Kind, p.sformat(v))
}

func (p pbufEncoder) AppendRawJSONField(dst []byte, key string, v interface{}, fFmt logfmt.LogFieldFormat) []byte {
	if fFmt.HasFmt {
		return p.appendFmt(dst, key, v, fFmt)
	}

	switch vv := v.(type) {
	case string:
		return p.appendStr(dst, key, fFmt.Kind, vv)
	case []byte:
		return p.appendBytes(dst, key, vv)
	}

	marshaled, err := json.Marshal(v)
	if err != nil {
		return p.appendStr(dst, key, ErrorFieldType, "marshaling error: "+err.Error())
	}
	return p.appendBytes(dst, key, marshaled)
}
