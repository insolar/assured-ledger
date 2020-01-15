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

package bilogadapter

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

type EncoderFactory interface {
	CreateEncoder(config logadapter.MsgFormatConfig) Encoder
}

type Encoder interface {
	InitBuffer(level logcommon.LogLevel, reuse []byte, parts [][]byte) []byte
	AppendIntField(b *[]byte, key string, v int64, fmt logcommon.LogFieldFormat)
	AppendUintField(b *[]byte, key string, v uint64, fmt logcommon.LogFieldFormat)
	AppendBoolField(b *[]byte, key string, v bool, fmt logcommon.LogFieldFormat)
	AppendFloatField(b *[]byte, key string, v float64, fmt logcommon.LogFieldFormat)
	AppendComplexField(b *[]byte, key string, v complex128, fmt logcommon.LogFieldFormat)
	AppendStrField(b *[]byte, key string, v string, fmt logcommon.LogFieldFormat)
	AppendIntfField(b *[]byte, key string, v interface{}, fmt logcommon.LogFieldFormat)
	AppendRawJSONField(b *[]byte, key string, v interface{}, fmt logcommon.LogFieldFormat)
}

var _ logcommon.LogObjectWriter = &objectEncoder{}

type objectEncoder struct {
	fieldEncoder Encoder
	content      []byte
}

func (p *objectEncoder) AddIntField(key string, v int64, fmt logcommon.LogFieldFormat) {
	p.fieldEncoder.AppendIntField(&p.content, key, v, fmt)
}

func (p *objectEncoder) AddUintField(key string, v uint64, fmt logcommon.LogFieldFormat) {
	p.fieldEncoder.AppendUintField(&p.content, key, v, fmt)
}

func (p *objectEncoder) AddBoolField(key string, v bool, fmt logcommon.LogFieldFormat) {
	p.fieldEncoder.AppendBoolField(&p.content, key, v, fmt)
}

func (p *objectEncoder) AddFloatField(key string, v float64, fmt logcommon.LogFieldFormat) {
	p.fieldEncoder.AppendFloatField(&p.content, key, v, fmt)
}

func (p *objectEncoder) AddComplexField(key string, v complex128, fmt logcommon.LogFieldFormat) {
	p.fieldEncoder.AppendComplexField(&p.content, key, v, fmt)
}

func (p *objectEncoder) AddStrField(key string, v string, fmt logcommon.LogFieldFormat) {
	p.fieldEncoder.AppendStrField(&p.content, key, v, fmt)
}

func (p *objectEncoder) AddIntfField(key string, v interface{}, fmt logcommon.LogFieldFormat) {
	p.fieldEncoder.AppendIntfField(&p.content, key, v, fmt)
}

func (p *objectEncoder) AddRawJSONField(key string, v interface{}, fmt logcommon.LogFieldFormat) {
	p.fieldEncoder.AppendRawJSONField(&p.content, key, v, fmt)
}
