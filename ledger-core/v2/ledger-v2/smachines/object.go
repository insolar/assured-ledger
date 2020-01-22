package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

/* -------- Declaration ------------- */

var declObject smachine.StateMachineDeclaration = &declarationObject{}

type declarationObject struct {
	smachine.StateMachineDeclTemplate
}

func (*declarationObject) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (*declarationObject) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*Object).Init
}

/* -------- Instance ------------- */

type Object struct {
}

func NewObject() *Object {
	return &Object{}
}

func (s *Object) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declObject
}

func (s *Object) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
