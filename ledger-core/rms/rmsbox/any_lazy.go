// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"encoding"
	"fmt"
	"io"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ rmsreg.GoGoSerializableWithText = &AnyLazy{}

type AnyLazy struct {
	value rmsreg.GoGoMarshaler
}

func (p *AnyLazy) IsZero() bool {
	return p.value == nil
}

func (p *AnyLazy) TryGetLazy() LazyValue {
	if vv, ok := p.value.(LazyValue); ok {
		return vv
	}
	return LazyValue{}
}

//nolint:interfacer
func (p *AnyLazy) Set(v rmsreg.GoGoSerializable) {
	p.value = v
}

func (p *AnyLazy) asLazy(v rmsreg.MarshalerTo) (LazyValue, error) {
	n := v.ProtoSize()
	b := make([]byte, n)
	if n > 0 {
		switch n2, err := v.MarshalTo(b); {
		case err != nil:
			return LazyValue{}, err
		case n != n2:
			return LazyValue{}, io.ErrShortWrite
		}
	}
	return LazyValue{b, reflect.TypeOf(v) }, nil
}

func (p *AnyLazy) SetAsLazy(v rmsreg.MarshalerTo) error {
	lv, err := p.asLazy(v)
	if err != nil {
		return err
	}

	p.value = lv
	return nil
}

func (p *AnyLazy) TryGet() (isLazy bool, r rmsreg.GoGoSerializable) {
	switch p.value.(type) {
	case nil:
		return false, nil
	case LazyValue:
		return true, nil
	}
	return false, p.value.(rmsreg.GoGoSerializable)
}

func (p *AnyLazy) ProtoSize() int {
	if p.value != nil {
		return p.value.ProtoSize()
	}
	return 0
}

func (p *AnyLazy) Unmarshal(b []byte) error {
	return p.UnmarshalCustom(b, false, rmsreg.GetRegistry().Get)
}

func (p *AnyLazy) UnmarshalCustom(b []byte, copyBytes bool, typeFn func(uint64) reflect.Type) error {
	v, err := p.unmarshalCustom(b, copyBytes, typeFn)
	if err != nil {
		p.value = nil
		return err
	}
	p.value = v
	return nil
}

func (p *AnyLazy) unmarshalCustom(b []byte, copyBytes bool, typeFn func(uint64) reflect.Type) (LazyValue, error) {
	_, t, err := rmsreg.UnmarshalType(b, typeFn)
	if err != nil {
		return LazyValue{}, err
	}
	if copyBytes {
		b = append([]byte(nil), b...)
	}
	return LazyValue{b, t }, nil
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
	return []byte(fmt.Sprintf("%T{%T}", p, p.value)), nil
}

func (p *AnyLazy) Equal(that interface{}) bool {
	switch {
	case that == nil:
		return p == nil
	case p == nil:
		return false
	}

	var thatValue rmsreg.GoGoMarshaler
	switch tt := that.(type) {
	case *AnyLazy:
		thatValue = tt.value
	case AnyLazy:
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

type anyLazy = AnyLazy
type AnyLazyCopy struct {
	anyLazy
}

func (p *AnyLazyCopy) Unmarshal(b []byte) error {
	return p.UnmarshalCustom(b, true, rmsreg.GetRegistry().Get)
}

/************************/

var _ rmsreg.GoGoMarshaler = LazyValue{}
var _ io.WriterTo = LazyValue{}

type LazyValueReader interface {
	Type() reflect.Type
	UnmarshalAsAny(v interface{}, skipFn rmsreg.UnknownCallbackFunc) (bool, error)
}

type LazyValue struct {
	value []byte
	vType  reflect.Type
}

func (p LazyValue) WriteTo(w io.Writer) (int64, error) {
	if p.value == nil {
		panic(throw.IllegalState())
	}
	n, err := w.Write(p.value)
	return int64(n), err
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

func (p LazyValue) Unmarshal() (rmsreg.GoGoSerializable, error) {
	switch {
	case p.value == nil:
		return nil, nil
	case p.vType == nil:
		panic(throw.IllegalState())
	}
	return p.UnmarshalAsType(p.vType, nil)
}

var typeGoGoSerializable = reflect.TypeOf((*rmsreg.GoGoSerializable)(nil)).Elem()

func (p LazyValue) UnmarshalAsType(vType reflect.Type, skipFn rmsreg.UnknownCallbackFunc) (rmsreg.GoGoSerializable, error) {
	switch {
	case vType == nil || !vType.Implements(typeGoGoSerializable):
		panic(throw.IllegalValue())
	case p.value == nil:
		return nil, nil
	}
	
	obj, err := rmsreg.UnmarshalAsType(p.value, vType, skipFn)
	if err != nil {
		return nil, err		
	}
	return obj.(rmsreg.GoGoSerializable), nil
}

func (p LazyValue) UnmarshalAs(v rmsreg.GoGoSerializable, skipFn rmsreg.UnknownCallbackFunc) (bool, error) {
	if p.value == nil {
		return false, nil
	}
	return true, rmsreg.UnmarshalAs(p.value, v, skipFn)
}

func (p LazyValue) UnmarshalAsAny(v interface{}, skipFn rmsreg.UnknownCallbackFunc) (bool, error) {
	if p.value == nil {
		return false, nil
	}
	return true, rmsreg.UnmarshalAs(p.value, v, skipFn)
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
	return []byte(fmt.Sprintf("%T{[%d]byte, %v}", p, len(p.value), p.vType)), nil
}