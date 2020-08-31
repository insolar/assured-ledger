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
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SMRegisterOnLMN struct {
	// translate from VCallRequest.Payload. If set we need register incoming before any other registration
	incoming rms.BasicRecord

	isConstructor bool
	object        reference.Global
	outgoing      reference.Global
	interference  isolation.InterferenceFlag
	newState      *execution.Update

	registeredOutbound bool

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
	if s.incoming != nil {
		return ctx.Jump(s.stepRegisterIncoming)
	}
	return ctx.Jump(s.stepRegisterRecords)
}

func (s *SMRegisterOnLMN) stepRegisterIncoming(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.sendIncoming(ctx, s.incoming)
	return ctx.Jump(s.stepRegisterRecords)
}

func (s *SMRegisterOnLMN) stepRegisterRecords(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.outgoing.IsEmpty() {
		return ctx.Jump(s.stepRegisterDone)
	}
	return ctx.Jump(s.stepRegisterOutgoing)
}

func (s *SMRegisterOnLMN) stepRegisterOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	records := make([]rms.BasicRecord, 0)
	if s.registeredOutbound {
		records = append(records, &rms.ROutboundResponse{})
	}

	switch s.interference {
	case isolation.CallTolerable:
		records = append(records, &rms.RLineInboundRequest{})
	case isolation.CallIntolerable:
		records = append(records, &rms.RInboundRequest{})
	}
	records = append(records, &rms.ROutboundRequest{})
	s.sendRecords(ctx, records)

	return ctx.Stop()
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
	s.sendRecords(ctx, records)

	return ctx.Stop()
}

func (s *SMRegisterOnLMN) sendIncoming(ctx smachine.ExecutionContext, record rms.BasicRecord) {
	req := &rms.LRegisterRequest{
		Flags: rms.RegistrationFlags_FastSafe,
	}
	if err := req.SetAsLazy(record); err != nil {
		panic(err)
	}

	rlv := req.TryGetLazy()
	if rlv.IsZero() {
		panic(throw.IllegalValue())
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, req, affinity.DynamicRoleLightExecutor, s.object, s.pulseSlot.CurrentPulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
				panic(err)
			}
		}
	}).WithoutAutoWakeUp().Start()
}

func (s *SMRegisterOnLMN) sendRecords(ctx smachine.ExecutionContext, records []rms.BasicRecord) {
	lazyRecords := make([]rms.AnyRecordLazy, len(records))
	for i, record := range records {
		if err := lazyRecords[i].SetAsLazy(record); err != nil {
			ctx.Log().Error("create lazyRecord fail", err)
			panic(err)
		}
		rlv := lazyRecords[i].TryGetLazy()
		if rlv.IsZero() {
			panic(throw.IllegalValue())
		}
	}
	req := &rms.LRegisterRequest{
		Flags:   rms.RegistrationFlags_FastSafe,
		Records: lazyRecords,
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, req, affinity.DynamicRoleLightExecutor, s.object, s.pulseSlot.CurrentPulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
				panic(err)
			}
		}
	}).WithoutAutoWakeUp().Start()
}
