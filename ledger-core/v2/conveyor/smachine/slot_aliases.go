// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

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

func (a *slotAliases) RetainAll(parent *slotAliases, removeFn func(k interface{})) {
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
	if i == 0 {
		a.keys = nil
		return
	}

	keys := a.keys[i:]
	a.keys = a.keys[:i]
	for i, k := range keys {
		keys[i] = nil
		removeFn(k)
	}
}

// ONLY to be used by a holder of a slot
func (s *Slot) registerBoundAlias(k, v interface{}) bool {
	if k == nil {
		panic("illegal value")
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
func (s *Slot) unregisterBoundAlias(k interface{}) bool {
	if k == nil {
		panic("illegal value")
	}

	switch keyExists, wasUnpublished, _ := s.machine.unpublishUnbound(k); {
	case !keyExists:
		return false
	case wasUnpublished:
		return true
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
	all := true
	switch mode & SubroutineCleanupAliasesAndShares {
	case SubroutineCleanupNone:
		return
	case SubroutineCleanupAliases:
		all = false
	}

	mm := &s.machine.localRegistry // SAFE for concurrent use
	var key interface{} = slotIDKey(s.GetSlotID())

	if isa, ok := mm.Load(key); ok {
		sa := isa.(*slotAliases)
		sar := s.machine.config.SlotAliasRegistry
		sa.RetainAll(parent, func(k interface{}) {
			if !all {
				if _, ok := k.(*uniqueAliasKey); ok {
					// retain shares
					return
				}
			}

			mm.Delete(k)

			if sar != nil {
				if ga, ok := k.(globalAliasKey); ok {
					sar.UnpublishAlias(ga.key)
				}
			}
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

		// reversed search as aliases are more likely to be LIFO
		for i := len(sa.keys) - 1; i > 0; i-- {
			kk := sa.keys[i]
			if k != kk {
				continue
			}

			m.localRegistry.Delete(k)
			if sar := m.config.SlotAliasRegistry; sar != nil {
				if ga, ok := k.(globalAliasKey); ok {
					sar.UnpublishAlias(ga.key)
				}
			}

			if i == 0 && len(sa.keys) == 1 {
				m.localRegistry.Delete(key)
				// slot.slotFlags &^= slotHadAliases
			} else {
				switch last := len(sa.keys) - 1; {
				case i < last:
					copy(sa.keys[i:], sa.keys[i+1:])
					fallthrough
				default:
					sa.keys[last] = nil
					sa.keys = sa.keys[:last]
				}
			}
			return true
		}
	}
	return false
}
