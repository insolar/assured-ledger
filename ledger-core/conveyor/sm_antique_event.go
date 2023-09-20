//go:generate sm-uml-gen -f $GOFILE

package conveyor

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

// This SM is responsible for pulling in old pulse.Data for events with very old Number before firing the event.
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

	return sm.ps.pulseManager.preparePulseDataRequest(ctx, sm.pn, func(BeatData) {
		// we don't need to store PD as it will also be in the cache for a while
	}).DelayedStart().Sleep().ThenJump(sm.stepGotAnswer)
}

func (sm *antiqueEventSM) stepGotAnswer(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if cps := sm.ps.pulseManager.getCachedPulseSlot(sm.pn); cps != nil {
		createDefaults := smachine.CreateDefaultValues{
			InheritAllDependencies: true,
		}
		createDefaults.PutOverride(injector.GetDefaultInjectionID(cps), cps)
		return ctx.ReplaceExt(sm.createFn, createDefaults)
	}

	return ctx.Error(fmt.Errorf("unable to find pulse data: pn=%v", sm.pn))
}
