// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"reflect"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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
	mutex    sync.RWMutex
	commons  map[uint64]reflect.Type
	specials map[string]map[uint64]reflect.Type
}

func (p *TypeRegistry) getMap(special string) map[uint64]reflect.Type {
	if special == "" {
		return p.commons
	}
	return p.specials[special]
}

func (p *TypeRegistry) makeMap(special string) map[uint64]reflect.Type {
	switch {
	case special != "":
		if p.specials == nil {
			m := map[uint64]reflect.Type{}
			p.specials = map[string]map[uint64]reflect.Type{special: m}
			return m
		}
		m := p.specials[special]
		if m == nil {
			m = map[uint64]reflect.Type{}
			p.specials[special] = m
		}
		return m

	case p.commons == nil:
		p.commons = map[uint64]reflect.Type{}
		return p.commons
	default:
		return p.commons
	}
}

func (p *TypeRegistry) Put(id uint64, t reflect.Type) {
	p.PutSpecial(id, "", t)
}

func (p *TypeRegistry) PutSpecial(id uint64, special string, t reflect.Type) {
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
			ID      uint64
			Special string
			Type    reflect.Type
		}{id, special, t}))
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()
	types := p.makeMap(special)
	if t2 := types[id]; t2 != nil {
		panic(throw.E("duplicate id", struct {
			ID       uint64
			Special  string
			Type     reflect.Type
			Existing reflect.Type
		}{id, special, t, t2}))
	}
	types[id] = t
}

func (p *TypeRegistry) Get(id uint64) reflect.Type {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.commons[id]
}

func (p *TypeRegistry) GetSpecial(id uint64, special string) reflect.Type {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.getMap(special)[id]
}

func UnmarshalType(b []byte, typeFn func(uint64) reflect.Type) (uint64, reflect.Type, error) {
	switch ct, id, err := protokit.PeekContentTypeAndPolymorphIDFromBytes(b); {
	case err != nil:
		return 0, nil, err
	case ct != protokit.ContentPolymorph:
		return 0, nil, throw.E("unexpected content", struct{ protokit.ContentType }{ct})
	case id == 0:
		return 0, nil, throw.E("untyped object")
	default:
		t := typeFn(id)
		if t == nil {
			return 0, nil, throw.E("unknown object", struct{ ID uint64 }{id})
		}
		return id, t, nil
	}
}

func UnmarshalAs(b []byte, obj interface{}, skipFn UnknownCallbackFunc) error {
	var err error
	if skipFn == nil {
		err = obj.(unmarshaler).Unmarshal(b)
	} else if un, ok := obj.(unmarshalerWithUnknownCallback); ok {
		err = un.UnmarshalWithUnknownCallback(b, skipFn)
	} else {
		return throw.E("wrong type")
	}

	return err
}

func UnmarshalAsType(b []byte, vType reflect.Type, skipFn UnknownCallbackFunc) (interface{}, error) {
	obj := reflect.New(vType).Interface()
	if err := UnmarshalAs(b, obj, skipFn); err != nil {
		return nil, err
	}
	return obj, nil
}

func UnmarshalCustom(b []byte, typeFn func(uint64) reflect.Type, skipFn UnknownCallbackFunc) (uint64, interface{}, error) {
	switch id, t, err := UnmarshalType(b, typeFn); {
	case err == nil:
		obj, err := UnmarshalAsType(b, t, skipFn)
		return id, obj, err
	case id != 0:
		return id, nil, throw.WithDetails(err, struct{ ID uint64 }{id})
	default:
		return id, nil, err
	}
}

func Unmarshal(b []byte) (uint64, interface{}, error) {
	return UnmarshalCustom(b, GetRegistry().Get, nil)
}

func UnmarshalSpecial(b []byte, special string) (uint64, interface{}, error) {
	return UnmarshalCustom(b, func(id uint64) reflect.Type {
		return GetRegistry().GetSpecial(id, special)
	}, nil)
}
