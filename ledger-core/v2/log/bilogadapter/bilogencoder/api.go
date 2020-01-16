package bilogencoder

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

type EncoderFactory interface {
	CreateEncoder(config logadapter.MsgFormatConfig) Encoder
}

type Encoder interface {
	PrepareBuffer(dst *[]byte)
	FinalizeBuffer(dst *[]byte)

	AppendIntField(b *[]byte, key string, v int64, fmt logcommon.LogFieldFormat)
	AppendUintField(b *[]byte, key string, v uint64, fmt logcommon.LogFieldFormat)
	AppendBoolField(b *[]byte, key string, v bool, fmt logcommon.LogFieldFormat)
	AppendFloatField(b *[]byte, key string, v float64, fmt logcommon.LogFieldFormat)
	AppendComplexField(b *[]byte, key string, v complex128, fmt logcommon.LogFieldFormat)
	AppendStrField(b *[]byte, key string, v string, fmt logcommon.LogFieldFormat)
	AppendIntfField(b *[]byte, key string, v interface{}, fmt logcommon.LogFieldFormat)
	AppendRawJSONField(b *[]byte, key string, v interface{}, fmt logcommon.LogFieldFormat)
	AppendTimeField(b *[]byte, key string, v time.Time, fmt logcommon.LogFieldFormat)
}
