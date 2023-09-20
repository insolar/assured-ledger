package smachine

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// this structure provides isolation of shared data to avoid SM being retained via SharedDataLink
type uniqueSharedKey struct {
	valueType reflect.Type
}

func (p *slotContext) ensureForShareData() {
	switch {
	case p.mode >= updCtxInit:
	case p.mode == updCtxFail:
	case p.mode == updCtxFinalize:
	default:
		panic(throw.IllegalState())
	}
}

func (p *slotContext) Share(data interface{}, flags ShareDataFlags) SharedDataLink {
	p.ensureForShareData()
	ensureShareValue(data)

	switch {
	case flags&ShareDataUnbound != 0: // ShareDataDirect is irrelevant
		return SharedDataLink{SlotLink{}, data, flags}
	case flags&ShareDataDirect != 0:
		return SharedDataLink{p.s.NewLink(), data, flags}
	default:
		alias := &uniqueSharedKey{reflect.TypeOf(data)}
		if !p.s.registerBoundAlias(alias, data) {
			panic("impossible")
		}
		return SharedDataLink{p.s.NewLink(), alias, flags}
	}
}

func (p *slotContext) Unshare(link SharedDataLink) bool {
	p.ensureForShareData()

	switch {
	case link.IsZero():
		return false
	case link.link.s != p.s: // covers ShareDataUnbound also
		return false
	default:
		return p.s.unregisterBoundAlias(link.data)
	}
}

func (p *slotContext) Publish(key, data interface{}) bool {
	p.ensureForShareData()
	ensurePublishKey(key)

	if sdl, ok := data.(SharedDataLink); ok && sdl.IsUnbound() {
		return p.s.registerUnboundAlias(key, data)
	}

	ensurePublishValue(data)
	return p.s.registerBoundAlias(key, data)
}

func (p *slotContext) PublishReplacement(key, data interface{}) bool {
	p.ensureForShareData()
	ensurePublishKey(key)
	ensurePublishValue(data)
	return p.s.replaceBoundAlias(key, data)
}

func (p *slotContext) Unpublish(key interface{}) bool {
	p.ensureForShareData()
	return p.s.unregisterAlias(key)
}

func (p *slotContext) UnpublishAll() {
	p.ensureForShareData()
	p.s.machine.unregisterBoundAliases(p.s.GetSlotID())
}

func (p *slotContext) GetPublished(key interface{}) interface{} {
	p.ensureValid()
	if v, ok := p.s.machine.getPublished(key); ok {
		return v
	}
	return nil
}

func (p *slotContext) PublishGlobalAlias(key interface{}) bool {
	return p.PublishGlobalAliasAndBargeIn(key, nil)
}

func (p *slotContext) PublishGlobalAliasAndBargeIn(key interface{}, b BargeInHolder) bool {
	p.ensureForShareData()
	ensurePublishKey(key)
	return p.s.registerBoundAlias(globalAliasKey{key}, SlotAliasValue{Link: p.s.NewLink(), BargeIn: b})
}

func (p *slotContext) UnpublishGlobalAlias(key interface{}) bool {
	p.ensureForShareData()
	return p.s.unregisterBoundAlias(globalAliasKey{key})
}

func (p *slotContext) GetPublishedGlobalAlias(key interface{}) SlotLink {
	p.ensureValid()
	av := p.s.machine.getGlobalPublished(key)
	return av.Link
}

func (p *slotContext) GetPublishedGlobalAliasAndBargeIn(key interface{}) (SlotLink, BargeInHolder) {
	p.ensureValid()
	av := p.s.machine.getGlobalPublished(key)
	return av.Link, av.BargeIn
}

func (p *machineCallContext) GetPublished(key interface{}) interface{} {
	p.ensureValid()
	if v, ok := p.m.getPublished(key); ok {
		return v
	}
	return nil
}

func (p *machineCallContext) GetPublishedGlobalAliasAndBargeIn(key interface{}) (SlotLink, BargeInHolder) {
	p.ensureValid()
	av := p.m.getGlobalPublished(key)
	return av.Link, av.BargeIn
}

func (m *SlotMachine) getPublished(key interface{}) (interface{}, bool) {
	if !isValidPublishKey(key) {
		return nil, false
	}
	return m._getPublished(key)
}

func (m *SlotMachine) _getPublished(key interface{}) (interface{}, bool) {
	return m.localRegistry.Load(key)
}

func (m *SlotMachine) getGlobalPublished(key interface{}) SlotAliasValue {
	if v, ok := m._getPublished(globalAliasKey{key}); ok {
		return v.(SlotAliasValue)
	}
	if sar := m.config.SlotAliasRegistry; sar != nil {
		return sar.GetPublishedAlias(key)
	}
	return SlotAliasValue{}
}

// Provides external access to published data.
// But SharedDataLink can only be accessed when is unbound.
// The bool value indicates presence of a valid key, but value can be nil when access is not allowed.
func (m *SlotMachine) GetPublished(key interface{}) (interface{}, bool) {
	if v, ok := m.getPublished(key); ok {
		// unwrap unbound values
		// but slot-bound values can NOT be accessed outside of a slot machine
		switch sdl := v.(type) {
		case SharedDataLink:
			if sdl.IsUnbound() {
				return sdl.getData(), true
			}
			return nil, true
		case nil:
			return v, true
		default:
			if isValidPublishValue(v) {
				return v, true
			}
			return nil, false
		}
	}
	return nil, false
}

func (m *SlotMachine) TryPublish(key, data interface{}) (interface{}, bool) {
	ensurePublishKey(key)

	if sdl, ok := data.(SharedDataLink); ok && !sdl.IsUnbound() {
		panic(throw.IllegalValue())
	}

	ensurePublishValue(data)
	v, loaded := m.localRegistry.LoadOrStore(key, data)
	return v, !loaded
}

// WARNING! USE WITH CAUTION. Interfering with published names may be unexpected by SM.
// This method can unpublish keys published by SMs, but it is not able to always do it in a right ways.
// As a result - a hidden registry of names published by SM can become inconsistent and will
// cause an SM to remove key(s) if they were later published by another SM.
func (m *SlotMachine) TryUnsafeUnpublish(key interface{}) (keyExists, wasUnpublished bool) {
	ensurePublishKey(key)

	// Lets try to make it right first
	switch keyExists, wasUnpublished, v := m.unpublishUnbound(key); {
	case !keyExists:
		return false, false
	case wasUnpublished:
		return true, true
	default:
		if sdl, ok := v.(SharedDataLink); ok {
			valueOwner := sdl.link
			// This is the most likely case ... yet it doesn't cover all the cases
			if valueOwner.IsValid() && valueOwner.isMachine(m) && m._unregisterSlotBoundAlias(valueOwner.SlotID(), key) {
				return true, true
			}
		}
	}

	// as there are no more options to do it right - then do it wrong
	m.localRegistry.Delete(key)
	return true, true
}

func (m *SlotMachine) unpublishUnbound(k interface{}) (keyExists, wasUnpublished bool, value interface{}) {
	v, ok := m.localRegistry.Load(k)
	if !ok {
		return false, false, nil
	}

	if sdl, ok := v.(SharedDataLink); ok && sdl.IsUnbound() {
		m.localRegistry.Delete(k)
		return true, true, v
	}
	return true, false, v
}

func _asSharedDataLink(v interface{}) SharedDataLink {
	if sdl, ok := v.(SharedDataLink); ok {
		return sdl
	}
	return SharedDataLink{}
}

func (p *slotContext) GetPublishedLink(key interface{}) SharedDataLink {
	return _asSharedDataLink(p.GetPublished(key))
}

func (p *machineCallContext) GetPublishedLink(key interface{}) SharedDataLink {
	return _asSharedDataLink(p.GetPublished(key))
}
