package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type GetObject struct {
}

/* -------- Declaration ------------- */

var declGetObject smachine.StateMachineDeclaration = &declarationGetObject{}

type declarationGetObject struct {
	smachine.StateMachineDeclTemplate
}

func (*declarationGetObject) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (*declarationGetObject) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*GetObject).Init
}

/* -------- Instance ------------- */

func (s *GetObject) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declGetObject
}

func (s *GetObject) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepEnsureIndex)
}

func (s *GetObject) stepEnsureIndex(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}
