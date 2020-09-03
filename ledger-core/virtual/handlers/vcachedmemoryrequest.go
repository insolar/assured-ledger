// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
//go:generate sm-uml-gen -f $GOFILE
package handlers

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
)

type SMVCachedMemoryRequest struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCachedMemoryRequest

	messageSender messageSenderAdapter.MessageSender
	memoryCache   memoryCacheAdapter.MemoryCache

	object   descriptor.Object
	response *payload.VCachedMemoryResponse
}

/* -------- Declaration ------------- */

var dSMVCachedMemoryRequestInstance smachine.StateMachineDeclaration = &dSMVCachedMemoryRequest{}

type dSMVCachedMemoryRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVCachedMemoryRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVCachedMemoryRequest)

	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.memoryCache)
}

func (*dSMVCachedMemoryRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVCachedMemoryRequest)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVCachedMemoryRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVCachedMemoryRequestInstance
}

func (s *SMVCachedMemoryRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepGetMemory)
}

func (s *SMVCachedMemoryRequest) stepGetMemory(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.memoryCache.PrepareAsync(ctx, func(ctx context.Context, svc memorycache.Service) smachine.AsyncResultFunc {
		obj, err := svc.Get(ctx, s.Payload.Object, s.Payload.StateID)
		return func(ctx smachine.AsyncResultContext) {
			s.object = obj
			if err != nil {
				ctx.Log().Error("failed to get memory", err)
			}
		}
	}).Start()
	return ctx.Jump(s.stepWaitResult)
}

func (s *SMVCachedMemoryRequest) stepWaitResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.object == nil {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(s.stepBuildResult)
}

func (s *SMVCachedMemoryRequest) stepBuildResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.object.HeadRef().IsEmpty() {
		s.response = &payload.VCachedMemoryResponse{
			Object:     s.Payload.Object,
			StateID:    s.Payload.StateID,
			CallStatus: payload.CachedMemoryStateUnknown,
		}
		return ctx.Jump(s.stepSendResult)
	}

	s.response = &payload.VCachedMemoryResponse{
		Object:     s.Payload.Object,
		StateID:    s.Payload.StateID,
		CallStatus: payload.CachedMemoryStateFound,
		// Node:        s.object.HeadRef(),
		// PrevStateID: s.object.StateID(),
		Inactive: s.object.Deactivated(),
		Memory:   s.object.Memory(),
	}
	return ctx.Jump(s.stepSendResult)
}

func (s *SMVCachedMemoryRequest) stepSendResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendTarget(goCtx, s.response, s.Meta.Sender)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()
	return ctx.Stop()
}
