// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ rmsreg.GoGoSerializableWithText = &AnyRecord{}
var _ BasicRecord = &AnyRecord{}

type AnyRecord struct {
	value serializableBasicRecord
}

type serializableBasicRecord interface {
	rmsreg.GoGoSerializable
	BasicRecord
}

func (p *AnyRecord) Get() BasicRecord {
	return p.value
}

func (p *AnyRecord) Set(v BasicRecord) {
	p.value = v.(serializableBasicRecord)
}

func (p *AnyRecord) Visit(visitor RecordVisitor) error {
	if p.value != nil {
		return p.value.Visit(visitor)
	}
	return nil
}

func (p *AnyRecord) GetRecordPayloads() RecordPayloads {
	if p.value != nil {
		return p.value.GetRecordPayloads()
	}
	return RecordPayloads{}
}

func (p *AnyRecord) SetRecordPayloads(payloads RecordPayloads, digester cryptkit.DataDigester) error {
	if p.value != nil {
		return p.value.SetRecordPayloads(payloads, digester)
	}
	panic(throw.IllegalState())
}

func (p *AnyRecord) ProtoSize() int {
	if p.value != nil {
		return p.value.ProtoSize()
	}
	return 0
}

func (p *AnyRecord) Unmarshal(b []byte) error {
	return p.UnmarshalCustom(b, rmsreg.GetRegistry().Get, nil)
}

func (p *AnyRecord) UnmarshalCustom(b []byte, typeFn func(uint64) reflect.Type, skipFn rmsreg.UnknownCallbackFunc) error {
	_, v, err := rmsreg.UnmarshalCustom(b, typeFn, skipFn)
	if err != nil {
		p.value = nil
		return err
	}
	if vv, ok := v.(serializableBasicRecord); ok {
		p.value = vv
		return nil
	}
	return throw.IllegalValue()
}

func (p *AnyRecord) MarshalTo(b []byte) (int, error) {
	if p.value != nil {
		return p.value.MarshalTo(b)
	}
	return 0, nil
}

func (p *AnyRecord) MarshalToSizedBuffer(b []byte) (int, error) {
	if p.value != nil {
		return p.value.MarshalToSizedBuffer(b)
	}
	return 0, nil
}

func (p *AnyRecord) MarshalText() ([]byte, error) {
	if tm, ok := p.value.(encoding.TextMarshaler); ok {
		return tm.MarshalText()
	}
	return []byte(fmt.Sprintf("AnyRecord{%T}", p.value)), nil
}

func (p *AnyRecord) Equal(that interface{}) bool {
	switch {
	case that == nil:
		return p == nil
	case p == nil:
		return false
	}

	var thatValue serializableBasicRecord
	switch tt := that.(type) {
	case *AnyRecord:
		thatValue = tt.value
	case AnyRecord:
		thatValue = tt.value
	default:
		return false
	}

	switch {
	case thatValue == nil:
		return p.value == nil
	case p.value == nil:
		return false
	}

	if eq, ok := thatValue.(interface{ Equal(that interface{}) bool}); ok {
		return eq.Equal(p.value)
	}
	return false
}

