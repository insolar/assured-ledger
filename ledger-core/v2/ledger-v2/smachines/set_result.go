package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
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

	objLink  smachine.SharedDataLink
	objIndex *sharedObject
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
	return ctx.Jump(s.getObject)
}

func (s *SetResult) getObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	lifelineLink := ctx.GetPublishedLink(s.message.ObjectID)
	if lifelineLink.IsZero() {
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return NewObject(s.message.ObjectID)
		})
		lifelineLink = ctx.GetPublishedLink(s.message.ObjectID)
	}

	accessDecision := lifelineLink.PrepareAccess(func(val interface{}) (wakeup bool) {
		s.objIndex = val.(*sharedObject)
		return false
	}).TryUse(ctx).GetDecision()

	switch accessDecision {
	case smachine.NotPassed:
		return ctx.WaitShared(lifelineLink).ThenRepeat()
	case smachine.Impossible:
		return ctx.Stop()
	case smachine.Passed:
		if ctx.Acquire(s.objIndex.mutex).IsNotPassed() {
			return ctx.Sleep().ThenRepeat()
		}
	default:
		panic("unknown state from TryUse")
	}

	return ctx.Jump(s.saveResult)
}

func (s *SetResult) saveResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	defer ctx.Release(s.objIndex.mutex)

	s.objIndex.appendState(s.message.SideEffect)
	return ctx.Stop()
}
