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

package msgencoder

import (
	"io"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

type FactoryDispatcherFunc func(string) EncoderFactory

type EncoderFactory interface {
	CreateEncoder(config logfmt.MsgFormatConfig) Encoder
	CreateMetricWriter(downstream io.Writer, fieldName string, reportFn logcommon.DurationReportFunc) (io.Writer, error)
}

type Encoder interface {
	PrepareBuffer(dst []byte, key string, level log.Level) []byte
	FinalizeBuffer(dst []byte, metricTime time.Time) []byte

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
