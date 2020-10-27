// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package vnlmn

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type ReceivedType int

const (
	ReceivedNothing ReceivedType = iota
	ReceivedLRegisterResponse
	ReceivedLStateVerifyResponse
)

type SMWaitSafeResponse struct {
	ObjectSharedState   object.SharedStateAccessor
	ExpectedKey         ResultAwaitKey
	SafeResponseCounter smachine.SharedDataLink
	LMNContext          *RegistrationCtx

	receivedType    ReceivedType
	receivedMessage interface{}
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

func (s *SMWaitSafeResponse) bargeInLRegisterResponseHandler(param interface{}) smachine.BargeInCallbackFunc {
	res, ok := param.(*rms.LRegisterResponse)
	if !ok || res == nil {
		panic(throw.IllegalValue())
	}

	return func(ctx smachine.BargeInContext) smachine.StateUpdate {
		if !res.AnticipatedRef.Equal(&s.ExpectedKey.AnticipatedRef) || res.Flags != s.ExpectedKey.RequiredFlag {
			panic(throw.IllegalValue())
		}

		if s.receivedType <= ReceivedLRegisterResponse {
			s.receivedType = ReceivedLRegisterResponse
		}

		s.receivedMessage = param
		return ctx.WakeUp()
	}
}

// func (s *SMWaitSafeResponse) bargeInLStateVerifyResponseHandler(param interface{}) smachine.BargeInCallbackFunc {
// 	res, ok := param.(*rms.LStateVerifyResponse)
// 	if !ok || res == nil {
// 		panic(throw.IllegalValue())
// 	}
//
// 	return func(ctx smachine.BargeInContext) smachine.StateUpdate {
// 		if res.AnticipatedRef != s.ExpectedKey.AnticipatedRef || res.Flags != s.ExpectedKey.RequiredFlag {
// 			panic(throw.IllegalValue())
// 		}
//
// 		if s.receivedType <= ReceivedLStateVerifyResponse {
// 			s.receivedType = ReceivedLStateVerifyResponse
// 		}
//
// 		s.receivedMessage = param
// 		return ctx.WakeUp()
// 	}
// }

func (s *SMWaitSafeResponse) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	var (
		expectedKey = s.ExpectedKey
		bargeIn     = ctx.NewBargeInWithParam(s.bargeInLRegisterResponseHandler)
	)

	// ctx.SetDefaultMigration(s.migrateDefault)

	if !ctx.PublishGlobalAliasAndBargeIn(expectedKey, bargeIn) {
		panic(throw.E("failed to publish bargeIn"))
	}
	return ctx.Jump(s.stepWaitResult)
}

// func (s *SMWaitSafeResponse) migrateDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
// 	ctx.Unpublish(s.ExpectedKey)
//
// 	return ctx.Jump(s.stepSendCheckStatus)
// }

func (s *SMWaitSafeResponse) stepWaitResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.receivedType == ReceivedNothing {
		return ctx.Sleep().ThenRepeat()
	} else if s.receivedType == ReceivedLStateVerifyResponse {
		panic(throw.IllegalState())
	}

	stateUpdate := s.LMNContext.DecrementSafeResponse(ctx)
	if !stateUpdate.IsEmpty() {
		return stateUpdate
	}

	return ctx.Stop()
}

// func (s *SMWaitSafeResponse) stepSendCheckStatus(ctx smachine.ExecutionContext) smachine.StateUpdate {
// 	// TODO: send message here
//
// 	return ctx.Jump(s.stepWaitStatusResult)
// }
//
// func (s *SMWaitSafeResponse) stepWaitStatusResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
// 	if s.receivedType < ReceivedLStateVerifyResponse {
// 		return ctx.Sleep().ThenRepeat()
// 	}
//
// 	// TODO: check everything here
//
// 	stateUpdate := shared.CounterDecrement(ctx, s.SafeResponseCounter)
// 	if !stateUpdate.IsEmpty() {
// 		return stateUpdate
// 	}
//
// 	return ctx.Stop()
// }
