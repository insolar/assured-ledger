package msgencoder

import (
	"io"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
)

type FactoryDispatcherFunc func(string) EncoderFactory

type EncoderFactory interface {
	CreateEncoder(config logfmt.MsgFormatConfig) Encoder
	CreateMetricWriter(downstream io.Writer, fieldName string, reportFn logcommon.DurationReportFunc) (io.Writer, error)
}

type Encoder interface {
	PrepareBuffer(dst []byte, key string, level log.Level) []byte
	FinalizeBuffer(dst []byte, metricTime time.Time) []byte

	AppendParts(b []byte, bb [][]byte) []byte

	AppendIntField(b []byte, key string, v int64, fmt logfmt.LogFieldFormat) []byte
	AppendUintField(b []byte, key string, v uint64, fmt logfmt.LogFieldFormat) []byte
	AppendBoolField(b []byte, key string, v bool, fmt logfmt.LogFieldFormat) []byte
	AppendFloatField(b []byte, key string, v float64, fmt logfmt.LogFieldFormat) []byte
	AppendComplexField(b []byte, key string, v complex128, fmt logfmt.LogFieldFormat) []byte
	AppendStrField(b []byte, key string, v string, fmt logfmt.LogFieldFormat) []byte
	AppendIntfField(b []byte, key string, v interface{}, fmt logfmt.LogFieldFormat) []byte
	AppendRawJSONField(b []byte, key string, v interface{}, fmt logfmt.LogFieldFormat) []byte
	AppendTimeField(b []byte, key string, v time.Time, fmt logfmt.LogFieldFormat) []byte
}
