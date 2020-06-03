// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bilog

import (
	"sort"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/msgencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

var _ logfmt.LogObjectWriter = &objectEncoder{}

type objectEncoder struct {
	fieldEncoder msgencoder.Encoder

	poolBuf    poolBuffer
	content    []byte
	reportedAt time.Time
	allowTrace bool
	level      logcommon.Level
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

func (p *objectEncoder) AddErrorField(msg string, stack throw.StackTrace, severity throw.Severity, hasPanic bool) {
	if msg != "" {
		p.content = p.fieldEncoder.AppendStrField(p.content, logoutput.ErrorMsgFieldName, msg, logfmt.LogFieldFormat{})
	}

	override := logcommon.Level(0)
	switch {
	case severity.IsFatal():
		override = logcommon.FatalLevel
	case severity.IsError():
		override = logcommon.ErrorLevel
	case severity.IsWarn():
		override = logcommon.WarnLevel
	}
	if p.level < override {
		p.level = override
	}

	if !p.allowTrace || stack == nil {
		return
	}
	s := stack.StackTraceAsText()
	p.content = p.fieldEncoder.AppendStrField(p.content, logoutput.StackTraceFieldName, s, logfmt.LogFieldFormat{})
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
