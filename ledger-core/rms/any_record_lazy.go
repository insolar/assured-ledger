// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
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
	if !payloads.IsEmpty() {
		return throw.FailHere("too many payloads")
	}
	return nil
}

func (p *AnyRecordLazy) Unmarshal(b []byte) error {
	return p.unmarshalCustom(b, false, GetRegistry().Get)
}

func (p *AnyRecordLazy) UnmarshalCustom(b []byte, copyBytes bool, typeFn func(uint64) reflect.Type) error {
	return p.unmarshalCustom(b, copyBytes, typeFn)
}

func stopAfterRecordBodyField(b []byte) (int, error) {
	u, n := protokit.DecodeVarintFromBytes(b)
	if n != recordBodyTagSize {
		return 0, nil // it is something else
	}
	wt, err := protokit.SafeWireTag(u)
	if err != nil {
		return 0, err
	}
	if wt.FieldID() > RecordBodyField {
		// NB! Fields MUST be sorted
		return -1, nil // don't read other fields
	}
	return 0, nil
}

func (p *AnyRecordLazy) unmarshalCustom(b []byte, copyBytes bool, typeFn func(uint64) reflect.Type) error {
	v, err := p.anyLazy.unmarshalCustom(b, copyBytes, typeFn)
	if err != nil {
		p.value = nil
		return err
	}

	p.value = LazyRecordValue{ v, nil }
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

var _ goGoMarshaler = &LazyRecordValue{}
var _ BasicRecord = &LazyRecordValue{}

type lazyValue = LazyValue
type LazyRecordValue struct {
	lazyValue
	body *RecordBodyForLazy
}

func (p *LazyRecordValue) Visit(RecordVisitor) error {
	return nil
}

func (p *LazyRecordValue) GetRecordBody() *RecordBody {
	if p.body == nil {
		body := &RecordBodyForLazy{}
		if err := body.UnmarshalWithUnknownCallback(p.value, stopAfterRecordBodyField); err != nil {
			p.body = &RecordBodyForLazy{} // empty
		} else {
			p.body = body
		}
	}
	return &p.body.RecordBody
}

func (p *LazyRecordValue) GetRecordPayloads() RecordPayloads {
	body := p.GetRecordBody()
	if body != nil {
		return body.GetRecordPayloads()
	}
	return RecordPayloads{}
}

func (p *LazyRecordValue) SetRecordPayloads(payloads RecordPayloads, digester cryptkit.DataDigester) error {
	if p.body == nil {
		p.body = &RecordBodyForLazy{}
	}
	return p.body.RecordBody.SetRecordPayloads(payloads, digester)
}

func (p *LazyRecordValue) Unmarshal() (BasicRecord, error) {
	switch {
	case p.value == nil:
		return nil, nil
	case p.vType == nil:
		panic(throw.IllegalState())
	}
	return p.UnmarshalAsType(p.vType, nil)
}

var typeBasicRecord = reflect.TypeOf((*BasicRecord)(nil)).Elem()

func (p *LazyRecordValue) UnmarshalAsType(vType reflect.Type, skipFn UnknownCallbackFunc) (BasicRecord, error) {
	switch {
	case vType == nil:
		panic(throw.IllegalValue())
	case !vType.Implements(typeBasicRecord) && !reflect.PtrTo(vType).Implements(typeBasicRecord):
		panic(throw.IllegalValue())
	case p.value == nil:
		return nil, nil
	}

	obj, err := UnmarshalAsType(p.value, vType, skipFn)
	if err != nil {
		return nil, err
	}

	br := obj.(BasicRecord)
	if p.body == nil || p.body.RecordBody.isEmptyForCopy() {
		return br, nil
	}

	if err = br.SetRecordPayloads(p.body.RecordBody.GetRecordPayloads(), p.body.RecordBody.digester); err != nil {
		return nil, err
	}

	return br, nil
}

func (p *LazyRecordValue) UnmarshalAs(v BasicRecord, skipFn UnknownCallbackFunc) (bool, error) {
	if p.value == nil {
		return false, nil
	}
	return true, UnmarshalAs(p.value, v, skipFn)
}
