// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"reflect"
	"sync"
)

func GetMarshallerFactory() MarshallerFactory {
	return marshallerFactory
}

var marshallerFactory MarshallerFactory = &cachingMarshallerFactory{}

type cachingMarshallerFactory struct {
	mutex       sync.RWMutex
	marshallers map[reflect.Type]*typeMarshaller
	reporters   map[reflect.Type]FieldReporterFunc
	forceAddr   bool // enforce use of address/pointer-based access to fields
}

func (p *cachingMarshallerFactory) RegisterFieldReporter(fieldType reflect.Type, fn FieldReporterFunc) {
	if fn == nil {
		panic("illegal value")
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.reporters == nil {
		p.reporters = make(map[reflect.Type]FieldReporterFunc)
	}
	p.reporters[fieldType] = fn
}

func (p *cachingMarshallerFactory) CreateErrorMarshaller(e error) LogObjectMarshaller {
	em := errorMarshaller{}
	if em.fillLevels(e, p) {
		return em
	}
	return nil
}

func (p *cachingMarshallerFactory) CreateLogObjectMarshaller(o reflect.Value) LogObjectMarshaller {
	if o.Kind() != reflect.Struct {
		panic("illegal value")
	}
	t := p.getTypeMarshaller(o.Type())
	if t == nil {
		return nil
	}
	return objectMarshaller{t, t.prepareValue(o)} // do prepare for a repeated use of marshaller
}

func (p *cachingMarshallerFactory) getFieldReporter(t reflect.Type) FieldReporterFunc {
	p.mutex.RLock()
	fr := p.reporters[t]
	p.mutex.RUnlock()
	return fr
}

func (p *cachingMarshallerFactory) getTypeMarshaller(t reflect.Type) *typeMarshaller {
	p.mutex.RLock()
	tm, ok := p.marshallers[t]
	p.mutex.RUnlock()
	if ok {
		return tm
	}

	tm = p.buildTypeMarshaller(t) // do before lock to reduce in-lock time

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.marshallers == nil {
		p.marshallers = make(map[reflect.Type]*typeMarshaller)
	} else if tm2, ok := p.marshallers[t]; ok {
		return tm2
	}
	p.marshallers[t] = tm
	return tm
}

func (p *cachingMarshallerFactory) buildTypeMarshaller(t reflect.Type) *typeMarshaller {
	n := t.NumField()
	if n <= 0 {
		return nil
	}

	tm := typeMarshaller{printNeedsAddr: p.forceAddr, reportNeedsAddr: p.forceAddr}

	if !tm.getFieldsOf(t, 0, p.getFieldReporter) {
		return nil
	}
	return &tm
}
