package smachines

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
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

	objLink     smachine.SharedDataLink
	objLifeline record.Lifeline
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
			return NewObject()
		})
		return ctx.WaitShared(lifelineLink).ThenRepeat()
	}

	accessDecision := lifelineLink.PrepareAccess(func(val interface{}) (wakeup bool) {
		s.objLifeline = val.(record.Lifeline)
		return false
	}).TryUse(ctx).GetDecision()

	switch accessDecision {
	case smachine.NotPassed:
		return ctx.WaitShared(lifelineLink).ThenRepeat()
	case smachine.Impossible:
		return ctx.Stop()
	case smachine.Passed:
	default:
		panic("unknown state from TryUse")
	}

	return ctx.Jump(s.saveResult)
}

func (s *SetResult) saveResult(ctx smachine.ExecutionContext) smachine.StateUpdate {

}
