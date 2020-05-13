package object

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type WaitPendingSM struct {
	smachine.StateMachineDeclTemplate
	sync smachine.SyncLink
	stop smachine.BargeIn
}

func (sm *WaitPendingSM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
	// No-op.
}

func (sm *WaitPendingSM) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

func (sm *WaitPendingSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *WaitPendingSM) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if !ctx.Acquire(sm.sync) {
		panic("failed to acquire semaphore")
	}

	// Sync will be released on machine stop.
	sm.stop = ctx.NewBargeIn().WithStop()

	return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		return ctx.Sleep().ThenRepeat()
	})
}
