package refmap

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func NewRefLocatorMap() RefLocatorMap {
	return RefLocatorMap{keys: NewUpdateableKeyMap()}
}

type ValueLocator int64

type RefLocatorMap struct {
	keys   UpdateableKeyMap
	values map[ValueSelector]ValueLocator
}

func (m *RefLocatorMap) Intern(ref reference.PtrHolder) reference.Holder {
	return m.keys.InternHolder(ref)
}

func (m *RefLocatorMap) Get(ref reference.PtrHolder) (ValueLocator, bool) {
	if selector, ok := m.keys.Find(ref); !ok || selector.State == 0 {
		return 0, false
	} else {
		v, ok := m.values[selector.ValueSelector]
		return v, ok
	}
}

func (m *RefLocatorMap) Contains(ref reference.PtrHolder) bool {
	_, ok := m.Get(ref)
	return ok
}

func (m *RefLocatorMap) Len() int {
	return len(m.values)
}

func (m *RefLocatorMap) Put(ref reference.PtrHolder, v ValueLocator) (internedRef reference.Holder) {
	m.keys.TryPut(ref, func(internedKey reference.Holder, selector BucketValueSelector) BucketState {
		internedRef = internedKey

		n := len(m.values)
		if m.values == nil {
			m.values = make(map[ValueSelector]ValueLocator)
		}
		m.values[selector.ValueSelector] = v
		switch {
		case n != len(m.values):
			return selector.State + 1
		case n == 0:
			panic("illegal state")
		default:
			return selector.State
		}
	})
	return internedRef
}

func (m *RefLocatorMap) Delete(ref reference.PtrHolder) {
	m.keys.TryTouch(ref, func(selector BucketValueSelector) BucketState {
		n := len(m.values)
		delete(m.values, selector.ValueSelector)
		switch {
		case n == len(m.values):
			return selector.State
		case n == 0:
			panic("illegal state")
		case selector.State == 0:
			panic("illegal state")
		default:
			return selector.State - 1
		}
	})
}

func (m *RefLocatorMap) FillLocatorBuckets(config WriteBucketerConfig) WriteBucketer {
	wb := NewWriteBucketer(&m.keys, m.Len(), config)
	for k, v := range m.values {
		wb.AddValue(k, v)
	}
	return wb
}
