package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

/* ------- Slot-dependant aliases and mappings ------------- */

type slotIDKey SlotID

type slotAliases struct {
	keys []interface{}
	//	keys map[interface{}
}

func (a *slotAliases) Equal(parent *slotAliases) bool {
	switch {
	case parent == nil:
		return false
	case a == parent:
		return true
	case len(a.keys) != len(parent.keys):
		return false
	}

	for i := range a.keys {
		if a.keys[i] != parent.keys[i] {
			return false
		}
	}
	return true
}

func (a slotAliases) Copy() *slotAliases {
	return &slotAliases{append([]interface{}{}, a.keys...)}
}

func (a *slotAliases) RetainAll(parent *slotAliases, removeFn func(k interface{}) bool) {
	i, n := 0, len(a.keys)
	if n == 0 {
		return
	}
	if parent != nil {
		for _, k := range parent.keys {
			if a.keys[i] == k {
				i++
				if i >= n {
					break
				}
			}
		}
	}

	keys := a.keys[i:]
	a.keys = a.keys[:i]
	for i, k := range keys {
		keys[i] = nil
		if !removeFn(k) {
			a.keys = append(a.keys, k)
		}
	}

	if len(a.keys) == 0 {
		a.keys = nil
	}
}

// ONLY to be used by a holder of a slot
func (s *Slot) registerUnboundAlias(k, v interface{}) bool {
	if k == nil {
		panic(throw.IllegalValue())
	}

	m := &s.machine.localRegistry
	_, loaded := m.LoadOrStore(k, v)
	return !loaded
}

// ONLY to be used by a holder of a slot
func (s *Slot) registerBoundAlias(k, v interface{}) bool {
	if k == nil {
		panic(throw.IllegalValue())
	}

	m := &s.machine.localRegistry
	if _, loaded := m.LoadOrStore(k, v); loaded {
		return false
	}

	var key interface{} = slotIDKey(s.GetSlotID())

	switch isa, ok := m.Load(key); {
	case !ok:
		isa, _ = m.LoadOrStore(key, &slotAliases{ /* owner:s */ })
		fallthrough
	default:
		sa := isa.(*slotAliases)
		sa.keys = append(sa.keys, k)
		s.slotFlags |= slotHadAliases
	}

	if sar := s.machine.config.SlotAliasRegistry; sar != nil {
		if ga, ok := k.(globalAliasKey); ok {
			return sar.PublishAlias(ga.key, v.(SlotAliasValue))
		}
	}

	return true
}

// ONLY to be used by a holder of a slot
func (s *Slot) replaceBoundAlias(k, v interface{}) bool {
	if k == nil {
		panic(throw.IllegalValue())
	}

	m := &s.machine.localRegistry
	if _, loaded := m.Load(k); !loaded {
		return false
	}

	var key interface{} = slotIDKey(s.GetSlotID())

	isa, ok := m.Load(key)
	if !ok {
		return false
	}
	sa := isa.(*slotAliases)

	return sa.FindAndRemove(k, func(_ interface{}, last bool) (remove bool) {
		m.Store(k, v) // replace value

		if sar := s.machine.config.SlotAliasRegistry; sar != nil {
			if ga, ok := k.(globalAliasKey); ok {
				sar.ReplaceAlias(ga.key, v.(SlotAliasValue))
			}
		}

		if sdl, ok := v.(SharedDataLink); ok && sdl.IsUnbound() {
			if last {
				m.Delete(key)
				return false
			}
			return true
		}
		return false
	})
}

// ONLY to be used by a holder of a slot
func (s *Slot) unregisterAlias(k interface{}) bool {
	if k == nil {
		panic(throw.IllegalValue())
	}

	switch keyExists, wasUnpublished, _ := s.machine.unpublishUnbound(k); {
	case !keyExists:
		return false
	case wasUnpublished:
		return true
	}
	return s.machine._unregisterSlotBoundAlias(s.GetSlotID(), k)
}

// ONLY to be used by a holder of a slot
func (s *Slot) unregisterBoundAlias(k interface{}) bool {
	if k == nil {
		panic(throw.IllegalValue())
	}

	return s.machine._unregisterSlotBoundAlias(s.GetSlotID(), k)
}

func (s *Slot) storeSubroutineAliases(parent *slotAliases, mode SubroutineCleanupMode) *slotAliases {
	mm := &s.machine.localRegistry // SAFE for concurrent use
	var key interface{} = slotIDKey(s.GetSlotID())

	if isa, ok := mm.Load(key); ok {
		switch sa := isa.(*slotAliases); {
		case mode&SubroutineCleanupAliasesAndShares == SubroutineCleanupNone:
			return sa
		case sa.Equal(parent):
			return parent
		default:
			return sa.Copy()
		}
	}
	return nil
}

func (s *Slot) restoreSubroutineAliases(parent *slotAliases, mode SubroutineCleanupMode) {
	mode &= SubroutineCleanupAliasesAndShares
	if mode == SubroutineCleanupNone {
		return
	}

	mm := &s.machine.localRegistry // SAFE for concurrent use
	var key interface{} = slotIDKey(s.GetSlotID())

	if isa, ok := mm.Load(key); ok {
		sa := isa.(*slotAliases)
		sar := s.machine.config.SlotAliasRegistry
		sa.RetainAll(parent, func(k interface{}) bool {
			switch kk := k.(type) {
			case *uniqueSharedKey: // shared data
				if mode != SubroutineCleanupAliasesAndShares {
					return false
				}
			case globalAliasKey:
				// always
				if sar != nil {
					sar.UnpublishAlias(kk.key)
				}
			default:
				if mode == SubroutineCleanupGlobalsOnly {
					return false
				}
			}

			mm.Delete(k)
			return true
		})
	}
}

// ONLY to be used by a holder of a slot
func (m *SlotMachine) unregisterBoundAliases(id SlotID) {
	mm := &m.localRegistry // SAFE for concurrent use
	var key interface{} = slotIDKey(id)

	if isa, ok := mm.Load(key); ok {
		sa := isa.(*slotAliases)
		mm.Delete(key)

		sar := m.config.SlotAliasRegistry
		for _, k := range sa.keys {
			mm.Delete(k)

			if sar != nil {
				if ga, ok := k.(globalAliasKey); ok {
					sar.UnpublishAlias(ga.key)
				}
			}
		}
	}
}

// ONLY to be used by a holder of a slot
func (m *SlotMachine) _unregisterSlotBoundAlias(slotID SlotID, k interface{}) bool {
	var key interface{} = slotIDKey(slotID)

	if isa, loaded := m.localRegistry.Load(key); loaded {
		sa := isa.(*slotAliases)

		return sa.FindAndRemove(k, func(_ interface{}, last bool) (remove bool) {
			m.localRegistry.Delete(k)
			if sar := m.config.SlotAliasRegistry; sar != nil {
				if ga, ok := k.(globalAliasKey); ok {
					sar.UnpublishAlias(ga.key)
				}
			}

			if last {
				m.localRegistry.Delete(key)
				return false
			}
			return true
		})
	}
	return false
}

func (a *slotAliases) FindAndRemove(k interface{}, fn func(k interface{}, last bool) (remove bool)) bool {
	// reversed search as aliases are more likely to be LIFO
	last := len(a.keys) - 1
	for i := last; i >= 0; i-- {
		kk := a.keys[i]
		if k != kk {
			continue
		}

		if fn(k, last == 0) {
			if i < last {
				copy(a.keys[i:], a.keys[i+1:])
			}
			a.keys[last] = nil
			a.keys = a.keys[:last]
		}

		return true
	}
	return false
}
