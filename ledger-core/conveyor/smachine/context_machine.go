package smachine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ MachineCallContext = &machineCallContext{}

type machineCallContext struct {
	contextTemplate
	m *SlotMachine
	w FixedSlotWorker
}

func (p *machineCallContext) Migrate(beforeFn func()) {
	p.ensureValid()
	p.m.migrateWithBefore(p.w, beforeFn)
}

func (p *machineCallContext) Cleanup() {
	p.ensureValid()
	p.m.Cleanup(p.w)
}

func (p *machineCallContext) Stop() {
	p.ensureValid()
	p.m.Stop()
}

func (p *machineCallContext) AddNew(ctx context.Context, sm StateMachine, defValues CreateDefaultValues) (SlotLink, bool) {
	p.ensureValid()
	if sm == nil {
		panic(throw.IllegalValue())
	}

	switch {
	case ctx != nil:
		defValues.Context = ctx
	case defValues.Context == nil:
		panic("illegal value")
	}

	link, ok := p.m.prepareNewSlotWithDefaults(nil, nil, sm, defValues)
	if ok {
		p.m.startNewSlot(link.s, p.w)
	}
	return link, ok
}

func (p *machineCallContext) AddNewByFunc(ctx context.Context, cf CreateFunc, defValues CreateDefaultValues) (SlotLink, bool) {
	p.ensureValid()

	switch {
	case ctx != nil:
		defValues.Context = ctx
	case defValues.Context == nil:
		panic("illegal value")
	}

	link, ok := p.m.prepareNewSlotWithDefaults(nil, cf, nil, defValues)
	if ok {
		p.m.startNewSlot(link.s, p.w)
	}
	return link, ok
}

func (p *machineCallContext) SlotMachine() *SlotMachine {
	p.ensureValid()
	return p.m
}

func (p *machineCallContext) GetMachineID() string {
	p.ensureValid()
	return p.m.GetMachineID()
}

func (p *machineCallContext) CallDirectBargeIn(link StepLink, fn BargeInCallbackFunc) bool {
	p.ensureValid()
	switch {
	case fn == nil:
		panic(throw.IllegalValue())
	case !link.IsValid():
		return false
	case !link.isMachine(p.m):
		return false
	}

	return p.m.executeBargeInDirect(link, fn, p.w)
}

func (p *machineCallContext) CallBargeInWithParam(b BargeInWithParam, param interface{}) bool {
	p.ensureValid()
	return b.callInline(p.m, param, p.w.asDetachable())
}

func (p *machineCallContext) CallBargeIn(b BargeIn) bool {
	p.ensureValid()
	return b.callInline(p.m, p.w.asDetachable())
}

func (p *machineCallContext) ApplyAdjustment(adj SyncAdjustment) bool {
	p.ensureValid()

	if adj.controller == nil {
		panic("illegal value")
	}

	released, activate := adj.controller.AdjustLimit(adj.adjustment, adj.isAbsolute)
	if activate {
		p.m.activateDependants(released, SlotLink{}, p.w)
	}

	// actually, we MUST NOT stop a slot from outside
	return len(released) > 0
}

func (p *machineCallContext) Check(link SyncLink) BoolDecision {
	p.ensureValid()

	if link.controller == nil {
		panic("illegal value")
	}

	return link.controller.CheckState()
}

func (p *machineCallContext) executeCall(fn MachineCallFunc) (err error) {
	p.setMode(updCtxMachineCall)
	defer func() {
		p.discardAndCapture("machine call", recover(), &err, 0)
	}()

	fn(p)
	return nil
}
