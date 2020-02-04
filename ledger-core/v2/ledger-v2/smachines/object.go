package smachines

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type filament struct {
	record *store.Record
	prev   *filament
}

type sharedObject struct {
	state   *filament
	updates []*store.Record
}

func (s *sharedObject) appendState(state *store.Record) {
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
	link := ctx.Share(&s.ownedObject, smachine.ShareDataWakesUpAfterUse)
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

	s.sharedDropBatch = sharedDropBatchLink(ctx, jetID)
	decision := s.sharedDropBatch.PrepareAccess(func(val interface{}) (wakeup bool) {
		batch := val.(*sharedDropBatch)
		batch.appendRecords(s.ownedObject.updates)
		s.ownedObject.updates = nil
		return true
	}).TryUse(ctx).GetDecision()

	switch decision {
	case smachine.Passed:
		ctx.Stop()
	case smachine.NotPassed:
		return ctx.WaitShared(s.sharedDropBatch).ThenRepeat()
	case smachine.Impossible:
		return ctx.Stop()
	default:
		panic(fmt.Sprintf("unknown state from TryUse: %v", decision))
	}

	return ctx.Stop()
}

func sharedObjectLink(ctx smachine.ExecutionContext, objID insolar.ID) smachine.SharedDataLink {
	link := ctx.GetPublishedLink(objID)
	if link.IsZero() {
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return NewObject(objID)
		})
		link = ctx.GetPublishedLink(objID)
		if link.IsZero() {
			panic("failed to acquire shared object")
		}
	}

	return link
}
