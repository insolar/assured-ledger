// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmn

//go:generate sm-uml-gen -f $GOFILE

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute/shared"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SerializableBasicMessage interface {
	rmsreg.GoGoSerializable
	rms.BasicMessage
}

type msgRegister interface {
	GetAnticipatedRef() rms.Reference
	GetFlags() rms.RegistrationFlags
}

type SubSMRegisterRecordSend struct {
	Messages            []SerializableBasicMessage
	Object              reference.Global
	ObjectSharedState   object.SharedStateAccessor
	SafeResponseCounter smachine.SharedDataLink

	// internal
	sendError    error
	sendResult   interface{}
	requiredSafe bool

	// DI
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
}

var dSubSMRegisterRecordSendInstance smachine.StateMachineDeclaration = &dSubSMRegisterRecordSend{}

type dSubSMRegisterRecordSend struct {
	smachine.StateMachineDeclTemplate
}

func (*dSubSMRegisterRecordSend) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SubSMRegisterRecordSend)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.messageSender)
}

func (*dSubSMRegisterRecordSend) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return nil
}

func (s *SubSMRegisterRecordSend) GetSubroutineInitState(ctx smachine.SubroutineStartContext) smachine.InitFunc {
	ctx.SetSubroutineCleanupMode(smachine.SubroutineCleanupAliasesAndShares)

	return s.Init
}

func (s *SubSMRegisterRecordSend) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSubSMRegisterRecordSendInstance
}

func (s *SubSMRegisterRecordSend) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if len(s.Messages) == 0 {
		panic(throw.IllegalValue())
	} else if s.pulseSlot == nil {
		panic(throw.IllegalState())
	}

	for _, msg := range s.Messages {
		switch msg.(type) {
		case *rms.LRegisterRequest:
		default:
			panic(throw.IllegalValue())
		}
	}

	return ctx.Jump(s.stepSendRequest)
}

func (s *SubSMRegisterRecordSend) popRequest() rmsreg.GoGoSerializable {
	defer func() { s.Messages = s.Messages[1:] }()

	return s.Messages[0]
}

func (s *SubSMRegisterRecordSend) bargeInHandler(param interface{}) smachine.BargeInCallbackFunc {
	res, ok := param.(*rms.LRegisterResponse)
	if !ok || res == nil {
		panic(throw.IllegalValue())
	}

	return func(ctx smachine.BargeInContext) smachine.StateUpdate {
		s.sendResult = param

		return ctx.WakeUp()
	}
}

func (s *SubSMRegisterRecordSend) stepSendRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if len(s.Messages) == 0 {
		return ctx.Stop()
	}

	var (
		obj          = s.Object
		msg          = s.popRequest()
		currentPulse = s.pulseSlot.CurrentPulseNumber()
	)

	msgConverted, ok := msg.(msgRegister)
	if !ok {
		panic(throw.IllegalValue())
	}

	var (
		anticipatedRef  = msgConverted.GetAnticipatedRef()
		flags, waitFlag = msgConverted.GetFlags(), msgConverted.GetFlags()
		bargeInKey      ResultAwaitKey
	)

	switch flags {
	case rms.RegistrationFlags_FastSafe:
		s.requiredSafe = true

		// TODO: add destructor to finish that SM's if not all answers were
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &SMWaitSafeResponse{
				ObjectSharedState:   s.ObjectSharedState,
				ExpectedKey:         NewResultAwaitKey(anticipatedRef, rms.RegistrationFlags_Safe),
				SafeResponseCounter: s.SafeResponseCounter,
			}
		})

		waitFlag = rms.RegistrationFlags_Fast

		fallthrough
	case rms.RegistrationFlags_Fast, rms.RegistrationFlags_Safe:
		bargeIn := ctx.NewBargeInWithParam(s.bargeInHandler)
		bargeInKey = NewResultAwaitKey(anticipatedRef, waitFlag)

		fmt.Println(bargeInKey.String())

		if !ctx.PublishGlobalAliasAndBargeIn(bargeInKey, bargeIn) {
			return ctx.Error(throw.E("failed to publish bargeIn callback"))
		}
	default:
		panic(throw.IllegalValue())
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, msg, affinity.DynamicRoleLightExecutor, obj, currentPulse)
		return func(ctx smachine.AsyncResultContext) {
			s.sendError = throw.W(err, "failed to send LRegisterRequest message")
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Jump(s.stepUpdateSafeCount)
}

func (s *SubSMRegisterRecordSend) stepUpdateSafeCount(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.requiredSafe {
		stateUpdate := shared.CounterIncrement(ctx, s.SafeResponseCounter, 1)
		if !stateUpdate.IsEmpty() {
			return stateUpdate
		}
	}

	s.requiredSafe = false

	return ctx.Jump(s.stepWaitResponse)
}

func (s *SubSMRegisterRecordSend) stepWaitResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.sendError != nil {
		return ctx.Error(s.sendError)
	} else if s.sendResult == nil {
		return ctx.Sleep().ThenRepeat()
	}

	// TODO: process result (in case of deduplication, for example)
	s.sendResult = nil

	return ctx.Jump(s.stepSendRequest)
}
