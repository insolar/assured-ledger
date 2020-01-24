package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

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

type SetResult struct {
	meta    payload.Meta
	message payload.V2SetRequestResult

	sharedObject smachine.SharedDataLink
	syncFinished smachine.SyncLink
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
	return ctx.Jump(s.saveResult)
}

func (s *SetResult) saveResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.sharedObject = sharedObjectLink(ctx, s.message.ObjectID)
	decision := s.sharedObject.PrepareAccess(func(val interface{}) (wakeup bool) {
		object := val.(*sharedObject)
		object.appendState(s.message.SideEffect)
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
		panic("unknown state from TryUse")
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
		panic("unknown state from TryUse")
	}
}

func (s *SetResult) waitForSync(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch ctx.Acquire(s.syncFinished).GetDecision() {
	case smachine.Passed:
		// TODO: return OK.
		return ctx.Stop()
	case smachine.NotPassed:
		return ctx.Sleep().ThenRepeat()
	case smachine.Impossible:
		return ctx.Stop()
	default:
		panic("unknown state from TryUse")
	}
}
