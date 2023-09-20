package datafinder

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.SubroutineStateMachine = &SMFindRecap{}

type SMFindRecap struct {
	smachine.StateMachineDeclTemplate
	RootRef  reference.Global
	RecapRef reference.Global
	RecapRec *rms.RLineRecap
}

func (p *SMFindRecap) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMFindRecap) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMFindRecap) GetSubroutineInitState(ctx smachine.SubroutineStartContext) smachine.InitFunc {
	ctx.SetSubroutineCleanupMode(smachine.SubroutineCleanupAliasesAndShares)
	return p.stepInit
}

func (p *SMFindRecap) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	panic(throw.NotImplemented()) // TODO
}

