// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package execute

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SMLMNSafeHandler struct {
	ObjectSharedState object.SharedStateAccessor
	ExpectedKey       LMNResultAwaitKey

	resultReceived bool
}

var dSMLMNSafeHandlerInstance smachine.StateMachineDeclaration = &dSMLMNSafeHandler{}

type dSMLMNSafeHandler struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMLMNSafeHandler) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ injector.DependencyInjector) {
}

func (*dSMLMNSafeHandler) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMLMNSafeHandler)
	return s.Init
}

func (s *SMLMNSafeHandler) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMLMNSafeHandlerInstance
}

func (s *SMLMNSafeHandler) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	bargeIn := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*rms.LRegisterResponse)
		if !ok || res == nil {
			panic(throw.IllegalValue())
		}

		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			if res.AnticipatedRef != s.ExpectedKey.AnticipatedRef || res.Flags != s.ExpectedKey.RequiredFlag {
				panic(throw.IllegalValue())
			}
			s.resultReceived = true
			return ctx.WakeUp()
		}
	})

	if !ctx.PublishGlobalAliasAndBargeIn(s.ExpectedKey, bargeIn) {
		panic(throw.E("failed to publish bargeIn"))
	}
	return ctx.Stop()
}

func (s *SMLMNSafeHandler) stepWaitResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !s.resultReceived {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(s.stepDecreaseCounter)
}

func (s *SMLMNSafeHandler) stepDecreaseCounter(ctx smachine.ExecutionContext) smachine.StateUpdate {
	action := func(state *object.SharedState) {
		state.RequiredSafeCounter--
	}

	switch s.ObjectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.ObjectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Passed:
		return ctx.Stop()
	default:
		panic(throw.Impossible())
	}
}
