package datafinder

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMFindRecord{}

type SMFindRecord struct {
	smachine.StateMachineDeclTemplate

	Unresolved lineage.UnresolvedDependency
	FindCallback    smachine.BargeInWithParam
}

func (p *SMFindRecord) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMFindRecord) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMFindRecord) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	panic(throw.NotImplemented()) // TODO
}

