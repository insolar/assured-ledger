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
	state   *filament
	updates []record.Record
}

func (s *sharedObject) appendState(state record.Record) {
	s.state = &filament{
		record: state,
		prev:   s.state,
	}
	s.updates = append(s.updates, state)
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

	ownedObject     sharedObject
	sharedDropBatch smachine.SharedDataLink
}

func NewObject(id insolar.ID) *Object {
	return &Object{id: id}
}

func (s *Object) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declObject
}

func (s *Object) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	link := ctx.Share(s.ownedObject, smachine.ShareDataWakesUpAfterUse)
	if !ctx.Publish(s.id, link) {
		return ctx.Stop()
	}
	return ctx.Jump(s.propagateChangeToDrop)
}

func (s *Object) propagateChangeToDrop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if len(s.ownedObject.updates) == 0 {
		ctx.Sleep()
	}

	// TODO: calculate jet.
	jetID := insolar.JetID(s.id)

	s.sharedDropBatch = ctx.GetPublishedLink(jetID)
	if s.sharedDropBatch.IsZero() {
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return NewDropBatch(jetID)
		})
		s.sharedDropBatch = ctx.GetPublishedLink(jetID)
		if s.sharedDropBatch.IsZero() {
			ctx.Stop()
		}
	}

	decision := s.sharedDropBatch.PrepareAccess(func(val interface{}) (wakeup bool) {
		batch := val.(*sharedDropBatch)
		batch.appendRecords(s.ownedObject.updates)
		s.ownedObject.updates = nil
		return true
	}).TryUse(ctx).GetDecision()

	switch decision {
	case smachine.NotPassed:
		return ctx.WaitShared(s.sharedDropBatch).ThenRepeat()
	case smachine.Impossible:
		return ctx.Stop()
	default:
		panic("unknown state from TryUse")
	}

	return ctx.Stop()
}
