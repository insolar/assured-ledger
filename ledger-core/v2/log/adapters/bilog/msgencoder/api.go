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
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

type EncoderFactoryDispatcherFunc func(string) EncoderFactory

type EncoderFactory interface {
	CreateEncoder(config logfmt.MsgFormatConfig) Encoder
}

type Encoder interface {
	PrepareBuffer(dst *[]byte, key string, level log.Level)
	FinalizeBuffer(dst *[]byte, reportedAt time.Time)

	AppendIntField(b *[]byte, key string, v int64, fmt logfmt.LogFieldFormat)
	AppendUintField(b *[]byte, key string, v uint64, fmt logfmt.LogFieldFormat)
	AppendBoolField(b *[]byte, key string, v bool, fmt logfmt.LogFieldFormat)
	AppendFloatField(b *[]byte, key string, v float64, fmt logfmt.LogFieldFormat)
	AppendComplexField(b *[]byte, key string, v complex128, fmt logfmt.LogFieldFormat)
	AppendStrField(b *[]byte, key string, v string, fmt logfmt.LogFieldFormat)
	AppendIntfField(b *[]byte, key string, v interface{}, fmt logfmt.LogFieldFormat)
	AppendRawJSONField(b *[]byte, key string, v interface{}, fmt logfmt.LogFieldFormat)
	AppendTimeField(b *[]byte, key string, v time.Time, fmt logfmt.LogFieldFormat)
}
