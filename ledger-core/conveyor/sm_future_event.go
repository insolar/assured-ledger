//go:generate sm-uml-gen -f $GOFILE

package conveyor

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// This SM delays actual creation of an event handler when the event has arrived to early.
// Also, when its pulseNumber won't match when the slot will became present, then SM will stop.
// Before such stop, this SM will attempt to capture and fire a termination handler for the event.

func newFutureEventSM(pn pulse.Number, ps *PulseSlot, createFn smachine.CreateFunc) smachine.StateMachine {
	return &futureEventSM{wrapEventSM{pn: pn, ps: ps, createFn: createFn}}
}

type futureEventSM struct {
	wrapEventSM
}

func (sm *futureEventSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *futureEventSM) GetInitStateFor(machine smachine.StateMachine) smachine.InitFunc {
	if sm != machine {
		panic("illegal value")
	}
	return sm.stepInit
}

func (sm *futureEventSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(sm.stepMigration)
	return ctx.Jump(sm.stepWaitMigration)
}

func (sm *futureEventSM) stepWaitMigration(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch isFuture, isAccepted := sm.ps.isAcceptedFutureOrPresent(sm.pn); {
	case !isAccepted:
		return sm.stepTerminate(ctx)
	case isFuture: // make sure that this slot isn't late
		return ctx.Sleep().ThenRepeat()
	default:
		return ctx.Replace(sm.createFn)
	}
}

func (sm *futureEventSM) stepMigration(ctx smachine.MigrationContext) smachine.StateUpdate {
	switch isFuture, isAccepted := sm.ps.isAcceptedFutureOrPresent(sm.pn); {
	case !isAccepted:
		return ctx.Jump(sm.stepTerminate)
	case isFuture: // make sure that this slot isn't late
		return ctx.Error(fmt.Errorf("impossible state for future pulse number: pn=%v", sm.pn))
	default:
		return ctx.WakeUp()
	}
}

func (sm *futureEventSM) stepTerminate(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Error(fmt.Errorf("incorrect future pulse number: pn=%v", sm.pn))
}

func (sm *futureEventSM) IsConsecutive(_, _ smachine.StateFunc) bool {
	// WARNING! DO NOT DO THIS ANYWHERE ELSE
	// Without CLEAR understanding of consequences this can lead to infinite loops
	return true // allow faster transition between steps
}
