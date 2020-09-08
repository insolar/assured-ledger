// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package execute

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RecordList []rms.BasicRecord

func (c RecordList) Append(message rms.BasicRecord) RecordList {
	return append(c, message)
}

func (c RecordList) AsAnyRecordLazyList() ([]rms.AnyRecordLazy, error) {
	rv := make([]rms.AnyRecordLazy, len(c))
	for pos, record := range c {
		if err := rv[pos].SetAsLazy(record); err != nil {
			return nil, err
		}
	}
	return rv, nil
}

// TODO: make all input arguments to be distinguishable from output arguments
// TODO: make all input and output arguments to be capitalized
// TODO: make all internal data to be separate
type SMRegisterOnLMN struct {
	// translate from VCallRequest.Payload. If set we need register incoming before any other registration
	incoming rms.BasicRecord

	isConstructor bool
	object        reference.Global
	outgoing      reference.Global
	interference  isolation.InterferenceFlag
	newState      *execution.Update

	registeredOutbound bool
	messages           []payload.Marshaler

	// result flags show registration status
	incomingRegistered bool
	messagesRegistered bool

	// DI
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
}

var dSMRegisterOnLMNInstance smachine.StateMachineDeclaration = &dSMRegisterOnLMN{}

type dSMRegisterOnLMN struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMRegisterOnLMN) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMRegisterOnLMN)

	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.pulseSlot)
}

func (*dSMRegisterOnLMN) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return nil
}

func (s *SMRegisterOnLMN) GetSubroutineInitState(ctx smachine.SubroutineStartContext) smachine.InitFunc {
	ctx.SetSubroutineCleanupMode(smachine.SubroutineCleanupAliasesAndShares)

	return s.Init
}
func (s *SMRegisterOnLMN) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMRegisterOnLMNInstance
}

func (s *SMRegisterOnLMN) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	// possible variants here:
	// all
	// CTConstructor -> Register (RLifelineStart and RInboundRequest) in one message (??)
	// Register (RInboundRequest + ROutboundRequest) in two separate messages
	// Register ROutboundRequest
	// Register ROutboundResponse
	// Register *InboundResult* (all possible variants, different step, one message)
	// Register (RInboundRequest + *InboundResult*) in two separate messages
	switch {
	case s.incoming != nil: // if we need to register incoming
		return ctx.Jump(s.stepRegisterIncoming)
	// case -- if we need to register outgoing
	case s.registeredOutbound: // if we need to register outgoing
		return ctx.Jump(s.stepRegisterOutgoingResult)
	// case -- if we need to register incoming result
	// case -- if we need to register outgoing result
	default: // should be panic(throw.IllegalState) after all fixes
		return ctx.Jump(s.stepRegisterRecords)
	}
}

func (s *SMRegisterOnLMN) stepRegisterIncoming(ctx smachine.ExecutionContext) smachine.StateUpdate {
	records := RecordList{}

	if s.isConstructor {
		records = records.Append(&rms.RLifelineStart{})
	}

	records = records.Append(s.incoming)

	lazyRecords, err := records.AsAnyRecordLazyList()
	if err != nil {
		panic(err)
	}

	req := &rms.LRegisterRequest{
		Flags:   rms.RegistrationFlags_FastSafe,
		Records: lazyRecords,
	}

	s.messages = append(s.messages, req)

	return ctx.Jump(s.stepRegisterRecords)
}

func (s *SMRegisterOnLMN) stepRegisterRecords(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.outgoing.IsEmpty() { // we should register result
		return ctx.Jump(s.stepRegisterDone)
	}

	// we should register outgoing request
	return ctx.Jump(s.stepRegisterOutgoing)
}

func (s *SMRegisterOnLMN) stepRegisterOutgoingResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	records := RecordList{&rms.ROutboundResponse{}}

	lazyRecords, err := records.AsAnyRecordLazyList()
	if err != nil {
		panic(err)
	}

	req := &rms.LRegisterRequest{
		Flags:   rms.RegistrationFlags_FastSafe,
		Records: lazyRecords,
	}

	s.messages = append(s.messages, req)

	return ctx.Jump(s.stepSendMessages)
}

func (s *SMRegisterOnLMN) stepRegisterOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	records := RecordList{}

	if s.registeredOutbound {
	} else {
		panic(throw.IllegalState())
	}

	switch s.interference {
	case isolation.CallTolerable:
		records = append(records, &rms.RLineInboundRequest{})
	case isolation.CallIntolerable:
		records = append(records, &rms.RInboundRequest{})
	}
	records = append(records, &rms.ROutboundRequest{})

	lazyRecords := make([]rms.AnyRecordLazy, len(records))
	for i, record := range records {
		if err := lazyRecords[i].SetAsLazy(record); err != nil {
			ctx.Log().Error("create lazyRecord fail", err)
			panic(err)
		}
	}

	req := &rms.LRegisterRequest{
		Flags:   rms.RegistrationFlags_FastSafe,
		Records: lazyRecords,
	}

	s.messages = append(s.messages, req)

	return ctx.Jump(s.stepSendMessages)
}

func (s *SMRegisterOnLMN) stepRegisterDone(ctx smachine.ExecutionContext) smachine.StateUpdate {
	records := make([]rms.BasicRecord, 0)
	if s.registeredOutbound {
		records = append(records, &rms.ROutboundResponse{})
	} else if s.isConstructor {
		records = append(records, &rms.RLineMemoryInit{})
	}
	records = append(records, &rms.RInboundResponse{})
	if s.newState.Type == execution.Error {
		records = append(records, &rms.RLineDeactivate{})
	} else {
		records = append(records, &rms.RLineMemory{})
	}
	if s.isConstructor {
		records = append(records, &rms.RLineActivate{})
	}

	lazyRecords := make([]rms.AnyRecordLazy, len(records))
	for i, record := range records {
		if err := lazyRecords[i].SetAsLazy(record); err != nil {
			ctx.Log().Error("create lazyRecord fail", err)
			panic(err)
		}
	}

	req := &rms.LRegisterRequest{
		Flags:   rms.RegistrationFlags_FastSafe,
		Records: lazyRecords,
	}

	s.messages = append(s.messages, req)

	return ctx.Jump(s.stepSendMessages)
}

func (s *SMRegisterOnLMN) stepSendMessages(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		object       = s.object
		messages     = s.messages
		currentPulse = s.pulseSlot.CurrentPulseNumber()
	)

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		for _, msg := range messages {
			err := svc.SendRole(goCtx, msg, affinity.DynamicRoleLightExecutor, object, currentPulse)
			if err != nil {
				return func(ctx smachine.AsyncResultContext) {
					ctx.Log().Error("failed to send message", err)
				}
			}
		}

		return nil
	}).WithoutAutoWakeUp().Start()

	return ctx.Jump(s.stepWaitResponses)
}

func (s *SMRegisterOnLMN) stepWaitResponses(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// TODO: we should wait here for given number of FAST (and maybe safe) responses, so we should set
	//       the right types and kinds of barge-ins in stepRegister***
	return ctx.Stop()
}
