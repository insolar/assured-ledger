// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ GoGoSerializableWithText = &AnyRecordLazy{}
var _ BasicRecord = &AnyRecordLazy{}

type AnyRecordLazy struct {
	anyLazy
}

func (p *AnyRecordLazy) TryGetLazy() LazyRecordValue {
	if vv, ok := p.value.(LazyRecordValue); ok {
		return vv
	}
	return LazyRecordValue{}
}

func (p *AnyRecordLazy) Set(v BasicRecord) {
	p.value = v.(goGoMarshaler)
}

func (p *AnyRecordLazy) TryGet() (isLazy bool, r BasicRecord) {
	switch p.value.(type) {
	case nil:
		return false, nil
	case LazyRecordValue:
		return true, nil
	}
	return false, p.value.(BasicRecord)
}

func (p *AnyRecordLazy) Visit(visitor RecordVisitor) error {
	if r, ok := p.value.(BasicRecord); ok {
		return r.Visit(visitor)
	}
	return nil
}

func (p *AnyRecordLazy) GetRecordPayloads() RecordPayloads {
	if r, ok := p.value.(BasicRecord); ok {
		return r.GetRecordPayloads()
	}
	return RecordPayloads{}
}

func (p *AnyRecordLazy) SetRecordPayloads(payloads RecordPayloads, digester cryptkit.DataDigester) error {
	if r, ok := p.value.(BasicRecord); ok {
		return r.SetRecordPayloads(payloads, digester)
	}
	panic(throw.IllegalState())
}

func (p *AnyRecordLazy) Unmarshal(b []byte) error {
	return p.unmarshalCustom(b, false, GetRegistry().Get)
}

func (p *AnyRecordLazy) UnmarshalCustom(b []byte, copyBytes bool, typeFn func(uint64) reflect.Type) error {
	return p.unmarshalCustom(b, copyBytes, typeFn)
}


func (p *AnyRecordLazy) unmarshalCustom(b []byte, copyBytes bool, typeFn func(uint64) reflect.Type) error {
	v, err := p.anyLazy.unmarshalCustom(b, copyBytes, typeFn)
	if err != nil {
		p.value = nil
		return err
	}
	p.value = LazyRecordValue{ v }
	return nil
}

func (p *AnyRecordLazy) Equal(that interface{}) bool {
	switch {
	case that == nil:
		return p == nil
	case p == nil:
		return false
	}

	var thatValue goGoMarshaler
	switch tt := that.(type) {
	case *AnyRecordLazy:
		thatValue = tt.value
	case AnyRecordLazy:
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


/************************/

type anyRecordLazy = AnyRecordLazy
type AnyRecordLazyCopy struct {
	anyRecordLazy
}

func (p *AnyRecordLazyCopy) Unmarshal(b []byte) error {
	return p.UnmarshalCustom(b, true, GetRegistry().Get)
}

/************************/

var _ goGoMarshaler = LazyRecordValue{}

type lazyValue = LazyValue
type LazyRecordValue struct {
	lazyValue
}

func (p LazyRecordValue) Unmarshal() (BasicRecord, error) {
	switch {
	case p.value == nil:
		return nil, nil
	case p.vType == nil:
		panic(throw.IllegalState())
	}
	return p.UnmarshalAsType(p.vType, nil)
}

var typeBasicRecord = reflect.TypeOf((*BasicRecord)(nil)).Elem()

func (p LazyRecordValue) UnmarshalAsType(vType reflect.Type, skipFn UnknownCallbackFunc) (BasicRecord, error) {
	switch {
	case vType == nil || !vType.Implements(typeBasicRecord):
		panic(throw.IllegalValue())
	case p.value == nil:
		return nil, nil
	}

	obj, err := UnmarshalAsType(p.value, vType, skipFn)
	if err != nil {
		return nil, err
	}
	return obj.(BasicRecord), nil
}

func (p LazyRecordValue) UnmarshalAs(v BasicRecord, skipFn UnknownCallbackFunc) (bool, error) {
	if p.value == nil {
		return false, nil
	}
	return true, UnmarshalAs(p.value, v, skipFn)
}
