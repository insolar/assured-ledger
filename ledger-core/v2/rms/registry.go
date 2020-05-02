// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"reflect"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var registry = &TypeRegistry{}

func GetRegistry() *TypeRegistry {
	return registry
}

func SetRegistry(r *TypeRegistry) {
	if r == nil {
		panic(throw.IllegalValue())
	}
	registry = r
}

var unmarshalerType = reflect.TypeOf((*unmarshaler)(nil)).Elem()

type TypeRegistry struct {
	mutex sync.RWMutex
	types map[uint64]reflect.Type
}

func (p *TypeRegistry) Put(id uint64, t reflect.Type) {
	switch {
	case t == nil:
		panic(throw.IllegalValue())
	case id == 0:
		panic(throw.E("zero id", struct {
			ID   uint64
			Type reflect.Type
		}{id, t}))
	case t.Implements(unmarshalerType):
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct {
			break
		}
		fallthrough
	default:
		panic(throw.E("illegal type", struct {
			ID   uint64
			Type reflect.Type
		}{id, t}))
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.types == nil {
		p.types = map[uint64]reflect.Type{id: t}
		return
	}
	if t2 := p.types[id]; t2 != nil {
		panic(throw.E("duplicate id", struct {
			ID             uint64
			Type, Existing reflect.Type
		}{id, t, t2}))
	}
	p.types[id] = t
}

func (p *TypeRegistry) Get(id uint64) reflect.Type {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.types[id]
}

func Unmarshal(b []byte) (uint64, interface{}, error) {
	ct, id, err := protokit.PeekContentTypeAndPolymorphIDFromBytes(b)
	switch {
	case err != nil:
		return 0, nil, err
	case ct != protokit.ContentObject:
		return 0, nil, throw.E("unexpected content", struct{ protokit.ContentType }{ct})
	case id == 0:
		return 0, nil, throw.E("untyped object")
	default:
		t := GetRegistry().Get(id)
		if t == nil {
			return 0, nil, throw.E("unknown object", struct{ ID uint64 }{id})
		}
		obj := reflect.New(t).Interface()
		err := obj.(unmarshaler).Unmarshal(b)
		return id, obj, err
	}
}
