// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

// This SM is responsible for pulling in old pulse.Data for events with very old PulseNumber before firing the event.
// Otherwise, the event will not be able to get its pulse.Data from a cache.
func newAntiqueEventSM(pn pulse.Number, ps *PulseSlot, createFn smachine.CreateFunc) smachine.StateMachine {
	return &antiqueEventSM{wrapEventSM: wrapEventSM{pn: pn, ps: ps, createFn: createFn}}
}

type antiqueEventSM struct {
	wrapEventSM
}

func (sm *antiqueEventSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *antiqueEventSM) GetInitStateFor(machine smachine.StateMachine) smachine.InitFunc {
	if sm != machine {
		panic("illegal value")
	}
	return sm.stepInit
}

func (sm *antiqueEventSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(sm.stepRequestOldPulseData)
}

func (sm *antiqueEventSM) stepRequestOldPulseData(ctx smachine.ExecutionContext) smachine.StateUpdate {

	return sm.ps.pulseManager.PreparePulseDataRequest(ctx, sm.pn, func(_ bool, _ pulse.Data) {
		// we don't need to store PD as it will also be in the cache for a while
	}).DelayedStart().Sleep().ThenJump(sm.stepGotAnswer)
}

func (sm *antiqueEventSM) stepGotAnswer(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if cps, ok := sm.ps.pulseManager.getCachedPulseSlot(sm.pn); ok {
		var createDefaults smachine.CreateDefaultValues
		createDefaults.PutOverride(injector.GetDefaultInjectionId(cps), cps)
		return ctx.ReplaceExt(sm.createFn, createDefaults)
	}

	ctx.SetDefaultTerminationResult(fmt.Errorf("unable to find pulse data: pn=%v", sm.pn))
	return sm.stepTerminateEvent(ctx)
}
