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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
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
	return LazyValue{b, reflect.TypeOf(v)}, nil
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
	return p.unmarshalCustom(b, false, rmsreg.GetRegistry())
}

func (p *AnyLazy) UnmarshalCustom(b []byte, registry *rmsreg.TypeRegistry) error {
	if registry == nil {
		panic(throw.IllegalValue())
	}

	return p.unmarshalCustom(b, false, registry)
}

func (p *AnyLazy) unmarshalCustom(b []byte, copyBytes bool, registry *rmsreg.TypeRegistry) error {
	p.value = p.unmarshalValue(b, copyBytes, registry)
	return nil
}

var emptyBytes = make([]byte, 0) // to distinguish zero and empty states of LazyValue

func (p *AnyLazy) unmarshalValue(b []byte, copyBytes bool, registry *rmsreg.TypeRegistry) LazyValue {
	switch {
	case len(b) == 0:
		b = emptyBytes // to distinguish zero and empty states of LazyValue
	case copyBytes:
		b = append([]byte(nil), b...)
	}
	return LazyValue{b, registry}
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

	if eq, ok := thatValue.(interface{ Equal(that interface{}) bool }); ok {
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
	return p.unmarshalCustom(b, true, rmsreg.GetRegistry())
}

func (p *AnyLazyCopy) UnmarshalWithRegistry(b []byte, registry *rmsreg.TypeRegistry) error {
	if registry == nil {
		panic(throw.IllegalValue())
	}

	return p.unmarshalCustom(b, true, registry)
}

/************************/

var _ rmsreg.GoGoMarshaler = LazyValue{}
var _ io.WriterTo = LazyValue{}

type LazyValueReader interface {
	UnmarshalAsAny(v interface{}, skipFn rmsreg.UnknownCallbackFunc) (bool, error)
}

type lazyValueType interface {
	// either *rmsreg.TypeRegistry or reflect.Type
}

type LazyValue struct {
	value []byte
	vType lazyValueType
}

func (p LazyValue) WriteTo(w io.Writer) (int64, error) {
	if p.value == nil {
		panic(throw.IllegalState())
	}
	n, err := w.Write(p.value)
	return int64(n), err
}

func (p LazyValue) IsZero() bool {
	return p.value == nil
}

func (p LazyValue) IsEmpty() bool {
	return len(p.value) == 0
}

func (p LazyValue) Unmarshal() (rmsreg.GoGoSerializable, error) {
	switch vType := p.vType.(type) {
	case nil:
		return p.UnmarshalCustom(rmsreg.GetRegistry(), nil)
	case *rmsreg.TypeRegistry:
		return p.UnmarshalCustom(vType, nil)
	case reflect.Type:
		return p.UnmarshalAsType(vType, nil)
	default:
		panic(throw.Impossible())
	}
}

var typeGoGoSerializable = reflect.TypeOf((*rmsreg.GoGoSerializable)(nil)).Elem()

func (p LazyValue) UnmarshalAsType(vType reflect.Type, skipFn rmsreg.UnknownCallbackFunc) (rmsreg.GoGoSerializable, error) {
	switch {
	case vType == nil:
		panic(throw.IllegalValue())
	case !vType.Implements(typeGoGoSerializable) && !reflect.PtrTo(vType).Implements(typeGoGoSerializable):
		panic(throw.IllegalValue())
	case p.value == nil:
		return nil, nil
	}

	v := reflect.New(vType).Interface()
	if err := p.unmarshalAsAny(v, skipFn); err != nil {
		return nil, err
	}
	return v.(rmsreg.GoGoSerializable), nil
}

var typeBasicMessage = reflect.TypeOf((*BasicMessage)(nil)).Elem()

func (p LazyValue) UnmarshalCustom(registry *rmsreg.TypeRegistry, skipFn rmsreg.UnknownCallbackFunc) (rmsreg.GoGoSerializable, error) {
	switch {
	case registry == nil:
		panic(throw.IllegalValue())
	case p.value == nil:
		return nil, nil
	}

	id, t, err := rmsreg.UnmarshalType(p.value, registry.Get)

	var v interface{}
	switch {
	case err != nil:
	case !t.Implements(typeGoGoSerializable):
		err = throw.E("incompatible with GoGoSerializable", struct { Type reflect.Type }{ t })

	case t.Implements(typeBasicMessage):
		payloads := RecordPayloads{}
		skipFn = payloads.WrapSkipFunc(skipFn)

		v, err = rmsreg.UnmarshalAsType(p.value, t, skipFn)

		if err == nil && !payloads.IsEmpty() {
			// postprocessing for message payloads
			digester := registry.GetPayloadDigester(id)
			_, err = UnmarshalMessageApplyPayloads(v, digester, payloads)
		}

	default:
		v, err = rmsreg.UnmarshalAsType(p.value, t, skipFn)
	}

	if err != nil {
		return nil, throw.WithDetails(err, struct{ ID uint64 }{id})
	}

	return v.(rmsreg.GoGoSerializable), nil
}

func (p LazyValue) UnmarshalAsAny(v interface{}, skipFn rmsreg.UnknownCallbackFunc) (bool, error) {
	if p.value == nil {
		return false, nil
	}
	return true, p.unmarshalAsAny(v, skipFn)
}

func (p LazyValue) unmarshalAsAny(v interface{}, skipFn rmsreg.UnknownCallbackFunc) error {
	if _, ok := v.(BasicMessage); !ok {
		return rmsreg.UnmarshalAs(p.value, v, skipFn)
	}

	payloads := RecordPayloads{}
	skipFn = payloads.WrapSkipFunc(skipFn)

	err := rmsreg.UnmarshalAs(p.value, v, skipFn)

	if err == nil && !payloads.IsEmpty() {
		// postprocessing for message payloads
		_, err = UnmarshalMessageApplyPayloads(v, nil, payloads)
	}
	if err == nil {
		return nil
	}

	// try to get polymorph id
	if id, _, err2 := protokit.DecodePolymorphFromBytes(p.value, false); err2 == nil {
		err = throw.WithDetails(err, struct{ ID uint64 }{id })
	}

	return throw.WithDetails(err, struct{ Type reflect.Type }{ reflect.TypeOf(v) })
}

func (p LazyValue) ProtoSize() int {
	return len(p.value)
}

func (p LazyValue) MarshalTo(b []byte) (int, error) {
	return copy(b, p.value), nil
}

func (p LazyValue) MarshalToSizedBuffer(b []byte) (int, error) {
	if len(b) < len(p.value) {
		return 0, io.ErrShortBuffer
	}
	return copy(b[len(b) - len(p.value):], p.value), nil
}

func (p LazyValue) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("LazyValue{[%d]byte}", len(p.value))), nil
}
