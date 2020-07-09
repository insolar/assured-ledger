// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
	return &FutureEventSM{wrapEventSM{pn: pn, ps: ps, createFn: createFn}}
}

type FutureEventSM struct {
	wrapEventSM
}

func (sm *FutureEventSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *FutureEventSM) GetInitStateFor(machine smachine.StateMachine) smachine.InitFunc {
	if sm != machine {
		panic("illegal value")
	}
	return sm.stepInit
}

func (sm *FutureEventSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(sm.stepMigration)
	return ctx.Jump(sm.StepWaitMigration)
}

func (sm *FutureEventSM) StepWaitMigration(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch isFuture, isAccepted := sm.ps.isAcceptedFutureOrPresent(sm.pn); {
	case !isAccepted:
		return sm.stepTerminate(ctx)
	case isFuture: // make sure that this slot isn't late
		return ctx.Sleep().ThenRepeat()
	default:
		return ctx.Replace(sm.createFn)
	}
}

func (sm *FutureEventSM) stepMigration(ctx smachine.MigrationContext) smachine.StateUpdate {
	switch isFuture, isAccepted := sm.ps.isAcceptedFutureOrPresent(sm.pn); {
	case !isAccepted:
		return ctx.Jump(sm.stepTerminate)
	case isFuture: // make sure that this slot isn't late
		return ctx.Error(fmt.Errorf("impossible state for future pulse number: pn=%v", sm.pn))
	default:
		return ctx.WakeUp()
	}
}

func (sm *FutureEventSM) stepTerminate(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Error(fmt.Errorf("incorrect future pulse number: pn=%v", sm.pn))
}

func (sm *FutureEventSM) IsConsecutive(_, _ smachine.StateFunc) bool {
	// WARNING! DO NOT DO THIS ANYWHERE ELSE
	// Without CLEAR understanding of consequences this can lead to infinite loops
	return true // allow faster transition between steps
}
