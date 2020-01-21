package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type SetResult struct {
}

/* -------- Declaration ------------- */

var declSetResult smachine.StateMachineDeclaration = &declarationSetResult{}

type declarationSetResult struct {
	smachine.StateMachineDeclTemplate
}

func (*declarationSetResult) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (*declarationSetResult) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*SetResult).Init
}

/* -------- Instance ------------- */

func (s *SetResult) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declSetResult
}

func (s *SetResult) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
