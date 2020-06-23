// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ GoGoSerializableWithText = &AnyLazy{}

type AnyLazy struct {
	value goGoMarshaler
}

func (p *AnyLazy) Get() LazyValue {
	if vv, ok := p.value.(LazyValue); ok {
		return vv
	}
	return LazyValue{}
}

func (p *AnyLazy) Set(v GoGoSerializable) {
	p.value = v
}

func (p *AnyLazy) ProtoSize() int {
	if p.value != nil {
		return p.value.ProtoSize()
	}
	return 0
}

func (p *AnyLazy) Unmarshal(b []byte) error {
	return p.UnmarshalCustom(b, true, GetRegistry().Get, nil)
}

func (p *AnyLazy) UnmarshalCustom(b []byte, copyBytes bool, typeFn func(uint64) reflect.Type, skipFn UnknownCallbackFunc) error {
	_, t, err := UnmarshalType(b, typeFn)
	if err != nil {
		p.value = nil
		return err
	}
	if copyBytes {
		b = append([]byte(nil), b...)
	}
	p.value = LazyValue{ b, skipFn, t }
	return nil
}

func (p *AnyLazy) MarshalTo(b []byte) (int, error) {
	if p.value != nil {
		return p.value.MarshalTo(b)
	}
	return 0, nil
}

func (p *AnyLazy) MarshalToSizedBuffer(b []byte) (int, error) {
	if p.value != nil {
		return p.value.MarshalToSizedBuffer(b)
	}
	return 0, nil
}

func (p *AnyLazy) MarshalText() ([]byte, error) {
	if tm, ok := p.value.(encoding.TextMarshaler); ok {
		return tm.MarshalText()
	}
	return []byte(fmt.Sprintf("AnyLazy{%T}", p.value)), nil
}

/************************/

type anyLazy = AnyLazy
type AnyLazyNoCopy struct {
	anyLazy
}

func (p *AnyLazyNoCopy) Unmarshal(b []byte) error {
	return p.UnmarshalCustom(b, false, GetRegistry().Get, nil)
}

/************************/

var _ goGoMarshaler = LazyValue{}

type LazyValue struct {
	value []byte
	skipFn UnknownCallbackFunc
	vType  reflect.Type
}

func (p LazyValue) IsZero() bool {
	return p.value == nil && p.vType == nil
}

func (p LazyValue) IsEmpty() bool {
	return len(p.value) == 0
}

func (p LazyValue) Type() reflect.Type {
	return p.vType
}

func (p LazyValue) Unmarshal() (GoGoSerializable, error) {
	switch {
	case p.value == nil:
		return nil, nil
	case p.vType == nil:
		panic(throw.IllegalState())
	}
	return p.UnmarshalAsType(p.vType, p.skipFn)
}

var typeGoGoSerializable = reflect.TypeOf((*GoGoSerializable)(nil)).Elem()

func (p LazyValue) UnmarshalAsType(vType reflect.Type, skipFn UnknownCallbackFunc) (GoGoSerializable, error) {
	switch {
	case vType == nil || !vType.Implements(typeGoGoSerializable):
		panic(throw.IllegalValue())
	case p.value == nil:
		return nil, nil
	}
	
	obj, err := UnmarshalAsType(p.value, vType, skipFn)
	if err != nil {
		return nil, err		
	}
	return obj.(GoGoSerializable), nil
}

func (p LazyValue) UnmarshalAs(v GoGoSerializable, skipFn UnknownCallbackFunc) (bool, error) {
	if p.value == nil {
		return false, nil
	}
	return true, UnmarshalAs(p.value, v, skipFn)
}

func (p LazyValue) ProtoSize() int {
	return len(p.value)
}

func (p LazyValue) MarshalTo(b []byte) (int, error) {
	return copy(b, p.value), nil
}

func (p LazyValue) MarshalToSizedBuffer(b []byte) (int, error) {
	if len(b) != len(p.value) {
		return 0, throw.IllegalState()
	}
	return copy(b, p.value), nil
}

func (p LazyValue) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("LazyValue{[%d]byte, %v}", len(p.value), p.vType)), nil
}

