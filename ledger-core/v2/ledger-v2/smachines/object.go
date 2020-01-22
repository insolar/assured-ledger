package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type filament struct {
	record record.Record
	prev   *filament
}

type sharedObject struct {
	mutex smachine.SyncLink
	state *filament
}

func (s *sharedObject) appendState(state record.Record) {
	s.state = &filament{
		record: state,
		prev:   s.state,
	}
}

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
	id insolar.ID

	index sharedObject
}

func NewObject(id insolar.ID) *Object {
	return &Object{id: id}
}

func (s *Object) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declObject
}

func (s *Object) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.Publish(s.id, &s.index)
	return ctx.Stop()
}
