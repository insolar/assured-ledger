package smsync

import "github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"

// ConditionalBool allows Acquire() call to pass through when current value is true
func NewConditionalBool(isOpen bool, name string) BoolConditionalLink {
	ctl := &boolConditionalSync{}
	ctl.controller.Init(name, &ctl.mutex, &ctl.controller)

	deps, _ := ctl.AdjustLimit(boolToConditional(isOpen), false)
	if len(deps) != 0 {
		panic("illegal state")
	}
	return BoolConditionalLink{ctl}
}

type BoolConditionalLink struct {
	ctl *boolConditionalSync
}

func boolToConditional(isOpen bool) int {
	if isOpen {
		return 1
	}
	return 0
}

func (v BoolConditionalLink) IsZero() bool {
	return v.ctl == nil
}

// Creates an adjustment that sets the given value when applied with SynchronizationContext.ApplyAdjustment()
// Can be applied multiple times.
func (v BoolConditionalLink) NewValue(isOpen bool) smachine.SyncAdjustment {
	if v.ctl == nil {
		panic("illegal state")
	}
	return smachine.NewSyncAdjustment(v.ctl, boolToConditional(isOpen), true)
}

// Creates an adjustment that toggles the conditional when the adjustment is applied with SynchronizationContext.ApplyAdjustment()
// Can be applied multiple times.
func (v BoolConditionalLink) NewToggle() smachine.SyncAdjustment {
	if v.ctl == nil {
		panic("illegal state")
	}
	return smachine.NewSyncAdjustment(v.ctl, 1, false)
}

func (v BoolConditionalLink) SyncLink() smachine.SyncLink {
	return smachine.NewSyncLink(v.ctl)
}

type boolConditionalSync struct {
	conditionalSync
}

func (p *boolConditionalSync) AdjustLimit(limit int, absolute bool) ([]smachine.StepLink, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case absolute:
		if limit > 0 {
			if p.controller.state > 0 {
				return nil, false
			}
			return p.setLimit(1)
		}
	case limit == 0:
		return nil, false
	case p.controller.state == 0: // flip-flop
		return p.setLimit(1)
	}
	return p.setLimit(0)
}
