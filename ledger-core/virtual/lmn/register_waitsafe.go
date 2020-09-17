// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package lmn

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute/shared"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SMWaitSafeResponse struct {
	ObjectSharedState   object.SharedStateAccessor
	ExpectedKey         ResultAwaitKey
	SafeResponseCounter smachine.SharedDataLink

	resultReceived bool
}

var dSMWaitSafeResponseInstance smachine.StateMachineDeclaration = &dSMWaitSafeResponse{}

type dSMWaitSafeResponse struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMWaitSafeResponse) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ injector.DependencyInjector) {
}

func (*dSMWaitSafeResponse) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMWaitSafeResponse)
	return s.Init
}

func (s *SMWaitSafeResponse) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMWaitSafeResponseInstance
}

func (s *SMWaitSafeResponse) bargeInHandler(param interface{}) smachine.BargeInCallbackFunc {
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
}

func (s *SMWaitSafeResponse) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	var (
		expectedKey = s.ExpectedKey
		bargeIn     = ctx.NewBargeInWithParam(s.bargeInHandler)
	)

	if !ctx.PublishGlobalAliasAndBargeIn(expectedKey, bargeIn) {
		panic(throw.E("failed to publish bargeIn"))
	}
	return ctx.JumpExt(smachine.SlotStep{
		Transition: s.stepWaitResult,
		Migration:  func(ctx smachine.MigrationContext) smachine.StateUpdate { return ctx.Stop() },
	})
}

func (s *SMWaitSafeResponse) stepWaitResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !s.resultReceived {
		return ctx.Sleep().ThenRepeat()
	}

	stateUpdate := shared.CounterDecrement(ctx, s.SafeResponseCounter)
	if !stateUpdate.IsEmpty() {
		return stateUpdate
	}

	return ctx.Stop()
}
