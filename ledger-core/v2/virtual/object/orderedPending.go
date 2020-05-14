package object

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type AwaitOrderedPendingSM struct {
	smachine.StateMachineDeclTemplate
	sync smachine.SyncLink
	stop smachine.BargeIn
}

func (sm *AwaitOrderedPendingSM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
	// No-op.
}

func (sm *AwaitOrderedPendingSM) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

func (sm *AwaitOrderedPendingSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *AwaitOrderedPendingSM) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if !ctx.Acquire(sm.sync) {
		panic("failed to acquire semaphore")
	}

	// Sync will be released on machine stop.
	sm.stop = ctx.NewBargeIn().WithStop()

	return ctx.Jump(sm.stepWaitIndefinitely)
}

// Await until SM will be destroyed
func (sm *AwaitOrderedPendingSM) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
