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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ rmsreg.GoGoSerializableWithText = &Any{}

type Any struct {
	value rmsreg.GoGoSerializable
}

func (p *Any) Get() rmsreg.GoGoSerializable {
	return p.value
}

func (p *Any) Set(v rmsreg.GoGoSerializable) {
	p.value = v
}

func (p *Any) ProtoSize() int {
	if p.value != nil {
		return p.value.ProtoSize()
	}
	return 0
}

func (p *Any) Unmarshal(b []byte) error {
	return p.UnmarshalCustom(b, rmsreg.GetRegistry().Get, nil)
}

func (p *Any) UnmarshalCustom(b []byte, typeFn func(uint64) reflect.Type, skipFn rmsreg.UnknownCallbackFunc) error {
	if len(b) == 0 {
		p.value = nil
		return nil
	}

	_, v, err := rmsreg.UnmarshalCustom(b, typeFn, skipFn)
	if err != nil {
		p.value = nil
		return err
	}
	if vv, ok := v.(rmsreg.GoGoSerializable); ok {
		p.value = vv
		return nil
	}
	return throw.IllegalValue()
}

var dummyType = reflect.TypeOf(1)

func dummyResolveType(uint64) reflect.Type {
	return dummyType
}

func (p *Any) MarshalTo(b []byte) (int, error) {
	if p.value == nil {
		return 0, nil
	}

	n, err := p.value.MarshalTo(b)
	if err == nil {
		_, _, err = rmsreg.UnmarshalType(b, dummyResolveType)
	}
	return n, err
}

func (p *Any) MarshalToSizedBuffer(b []byte) (int, error) {
	if p.value == nil {
		return 0, nil
	}

	n, err := p.value.MarshalToSizedBuffer(b)
	if err == nil {
		_, _, err = rmsreg.UnmarshalType(b[len(b)-n:], dummyResolveType)
	}
	return n, err
}

func (p *Any) MarshalText() ([]byte, error) {
	if tm, ok := p.value.(encoding.TextMarshaler); ok {
		return tm.MarshalText()
	}
	return []byte(fmt.Sprintf("Any{%T}", p.value)), nil
}

func (p *Any) Equal(that interface{}) bool {
	switch {
	case that == nil:
		return p == nil
	case p == nil:
		return false
	}

	var thatValue rmsreg.GoGoMarshaler
	switch tt := that.(type) {
	case *Any:
		thatValue = tt.value
	case Any:
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

	if eq, ok := thatValue.(interface{ Equal(that interface{}) bool }); ok {
		return eq.Equal(p.value)
	}
	return false
}
