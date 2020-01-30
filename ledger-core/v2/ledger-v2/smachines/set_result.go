package smachines

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

/* -------- Declaration ------------- */

var declSetResult smachine.StateMachineDeclaration = &declarationSetResult{}

type declarationSetResult struct {
	smachine.StateMachineDeclTemplate
}

func (*declarationSetResult) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := sm.(*SetResult)
	injector.MustInjectById("bus.Sender", &s.sender)
}

func (*declarationSetResult) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*SetResult).Init
}

/* -------- Instance ------------- */

type SetResult struct {
	sender bus.Sender

	meta    payload.Meta
	message payload.V2SetRequestResult

	sharedObject smachine.SharedDataLink
	syncFinished smachine.SyncLink
	sideEffect   *store.Record
}

func NewSetResult(meta payload.Meta) *SetResult {
	return &SetResult{meta: meta}
}

func (s *SetResult) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declSetResult
}

func (s *SetResult) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	pl, err := payload.Unmarshal(s.meta.Payload)
	if err != nil {
		panic(err)
	}
	s.message = *pl.(*payload.V2SetRequestResult)
	s.sideEffect = &store.Record{Record: record.Unwrap(&s.message.SideEffect.Virtual)}
	return ctx.Jump(s.saveResult)
}

func (s *SetResult) saveResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.sharedObject = sharedObjectLink(ctx, s.message.ObjectID)
	decision := s.sharedObject.PrepareAccess(func(val interface{}) (wakeup bool) {
		object := val.(*sharedObject)
		object.appendState(s.sideEffect)
		return false
	}).TryUse(ctx).GetDecision()

	switch decision {
	case smachine.Passed:
		return s.getSyncLink(ctx)
	case smachine.NotPassed:
		return ctx.WaitShared(s.sharedObject).ThenRepeat()
	case smachine.Impossible:
		return ctx.Stop()
	default:
		panic(fmt.Sprintf("unknown state from TryUse: %v", decision))
	}

	return ctx.Stop()
}

func (s *SetResult) getSyncLink(ctx smachine.ExecutionContext) smachine.StateUpdate {
	objectID := s.message.ObjectID
	// TODO: calculate jet.
	jetID := insolar.JetID(objectID)

	decision := sharedDropBatchLink(ctx, jetID).PrepareAccess(func(val interface{}) (wakeup bool) {
		batch := val.(*sharedDropBatch)
		s.syncFinished = batch.syncFinished
		return false
	}).TryUse(ctx).GetDecision()

	switch decision {
	case smachine.Passed:
		return ctx.Jump(s.waitForSync)
	case smachine.NotPassed:
		return ctx.WaitShared(s.sharedObject).ThenRepeat()
	case smachine.Impossible:
		return ctx.Stop()
	default:
		panic(fmt.Sprintf("unknown state from TryUse: %v", decision))
	}
}

func (s *SetResult) waitForSync(ctx smachine.ExecutionContext) smachine.StateUpdate {
	decision := ctx.Acquire(s.syncFinished).GetDecision()
	switch decision {
	case smachine.Passed:
		s.sender.Reply(ctx.GetContext(), s.meta, payload.MustNewMessage(&payload.ID{
			ID: s.sideEffect.ID,
		}))
		return ctx.Stop()
	case smachine.NotPassed:
		return ctx.Sleep().ThenRepeat()
	case smachine.Impossible:
		return ctx.Stop()
	default:
		panic(fmt.Sprintf("unknown state from TryUse: %v", decision))
	}
}
