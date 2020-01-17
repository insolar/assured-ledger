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

package bilogencoder

// TODO rename logencoder

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logmsgfmt"
)

type EncoderFactoryDispatcherFunc func(string) EncoderFactory

type EncoderFactory interface {
	CreateEncoder(config logmsgfmt.MsgFormatConfig) Encoder
}

type Encoder interface {
	PrepareBuffer(dst *[]byte, key string, level log.LogLevel)
	FinalizeBuffer(dst *[]byte, reportedAt time.Time)

	AppendIntField(b *[]byte, key string, v int64, fmt logmsgfmt.LogFieldFormat)
	AppendUintField(b *[]byte, key string, v uint64, fmt logmsgfmt.LogFieldFormat)
	AppendBoolField(b *[]byte, key string, v bool, fmt logmsgfmt.LogFieldFormat)
	AppendFloatField(b *[]byte, key string, v float64, fmt logmsgfmt.LogFieldFormat)
	AppendComplexField(b *[]byte, key string, v complex128, fmt logmsgfmt.LogFieldFormat)
	AppendStrField(b *[]byte, key string, v string, fmt logmsgfmt.LogFieldFormat)
	AppendIntfField(b *[]byte, key string, v interface{}, fmt logmsgfmt.LogFieldFormat)
	AppendRawJSONField(b *[]byte, key string, v interface{}, fmt logmsgfmt.LogFieldFormat)
	AppendTimeField(b *[]byte, key string, v time.Time, fmt logmsgfmt.LogFieldFormat)
}
