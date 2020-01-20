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

package bilog

import (
	"sort"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/msgencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

var _ logfmt.LogObjectWriter = &objectEncoder{}

type objectEncoder struct {
	fieldEncoder msgencoder.Encoder
	content      []byte
	reportedAt   time.Time
}

func (p *objectEncoder) AddIntField(key string, v int64, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendIntField(p.content, key, v, fmt)
}

func (p *objectEncoder) AddUintField(key string, v uint64, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendUintField(p.content, key, v, fmt)
}

func (p *objectEncoder) AddBoolField(key string, v bool, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendBoolField(p.content, key, v, fmt)
}

func (p *objectEncoder) AddFloatField(key string, v float64, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendFloatField(p.content, key, v, fmt)
}

func (p *objectEncoder) AddComplexField(key string, v complex128, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendComplexField(p.content, key, v, fmt)
}

func (p *objectEncoder) AddStrField(key string, v string, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendStrField(p.content, key, v, fmt)
}

func (p *objectEncoder) AddIntfField(key string, v interface{}, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendIntfField(p.content, key, v, fmt)
}

func (p *objectEncoder) AddRawJSONField(key string, v interface{}, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendRawJSONField(p.content, key, v, fmt)
}

func (p *objectEncoder) AddTimeField(key string, v time.Time, fmt logfmt.LogFieldFormat) {
	p.content = p.fieldEncoder.AppendTimeField(p.content, key, v, fmt)
}

func (p *objectEncoder) addIntfFields(fields map[string]interface{}) {
	names := make([]string, 0, len(fields))
	for k := range fields {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		p.AddIntfField(k, fields[k], logfmt.LogFieldFormat{})
	}
}
