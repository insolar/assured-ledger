package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

/* -------- Declaration ------------- */

var declDropBatch smachine.StateMachineDeclaration = &declarationDropBatch{}

type declarationDropBatch struct {
	smachine.StateMachineDeclTemplate
}

func (*declarationDropBatch) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (*declarationDropBatch) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*Object).Init
}

/* -------- Instance ------------- */

type DropBatch struct {
	recordNumber int
	records      []record.Record
}

func NewDropBatch() *DropBatch {
	return &DropBatch{}
}

func (s *DropBatch) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declDropBatch
}

func (s *DropBatch) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
