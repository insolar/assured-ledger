package object

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

type SMAwaitTableFill struct {
	smachine.StateMachineDeclTemplate

	sync smachine.SyncLink
	stop smachine.BargeIn
}

func (sm *SMAwaitTableFill) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (sm *SMAwaitTableFill) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

func (sm *SMAwaitTableFill) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *SMAwaitTableFill) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if !ctx.Acquire(sm.sync) {
		panic("failed to acquire semaphore")
	}

	// Sync will be released on machine stop.
	sm.stop = ctx.NewBargeIn().WithStop()

	return ctx.Jump(sm.stepWaitIndefinitely)
}

// Await until SM will be destroyed
func (sm *SMAwaitTableFill) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
