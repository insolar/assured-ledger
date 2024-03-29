package rmsreg

import (
	"reflect"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
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

	digester  cryptkit.DataDigester
	digesters map[uint64]cryptkit.DataDigester
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

func (p *TypeRegistry) SetDefaultPayloadDigester(d cryptkit.DataDigester) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.digester = d
}

func (p *TypeRegistry) SetPayloadDigester(id uint64, d cryptkit.DataDigester) {
	if id == 0 {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.digesters == nil {
		p.digesters = map[uint64]cryptkit.DataDigester{}
	}

	p.digesters[id] = d
}

func (p *TypeRegistry) UnsetPayloadDigester(id uint64) {
	if id == 0 {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.digesters == nil {
		return
	}

	delete(p.digesters, id)
}

func (p *TypeRegistry) GetPayloadDigester(id uint64) cryptkit.DataDigester {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if d, ok := p.digesters[id]; ok {
		return d
	}
	return p.digester
}

type PayloadDigesterFunc = func(uint64) cryptkit.DataDigester
type UnmarshalTypeFunc = func(uint64) reflect.Type

func UnmarshalType(b []byte, typeFn UnmarshalTypeFunc) (uint64, reflect.Type, error) {
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
	if skipFn == nil {
		return obj.(unmarshaler).Unmarshal(b)
	}
	if un, ok := obj.(unmarshalerWithUnknownCallback); ok {
		_, err := un.UnmarshalWithUnknownCallback(b, skipFn)
		return err
	}
	return throw.Unsupported()
}

func UnmarshalAsWithPos(b []byte, obj interface{}, skipFn UnknownCallbackFunc) (int, error) {
	if un, ok := obj.(unmarshalerWithUnknownCallback); ok {
		return un.UnmarshalWithUnknownCallback(b, skipFn)
	}
	if skipFn != nil {
		return 0, throw.Unsupported()
	}
	if err := obj.(unmarshaler).Unmarshal(b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func UnmarshalAsType(b []byte, vType reflect.Type, skipFn UnknownCallbackFunc) (interface{}, error) {
	if vType.Kind() == reflect.Ptr {
		vType = vType.Elem()
	}

	obj := reflect.New(vType).Interface()
	if err := UnmarshalAs(b, obj, skipFn); err != nil {
		return nil, err
	}
	return obj, nil
}

func UnmarshalCustom(b []byte, typeFn UnmarshalTypeFunc, skipFn UnknownCallbackFunc) (uint64, interface{}, error) {
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
