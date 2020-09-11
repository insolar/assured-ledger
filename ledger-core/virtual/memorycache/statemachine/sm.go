// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
)

//go:generate sm-uml-gen -f $GOFILE

type CachedMemoryReportAwaitKey struct {
	Object reference.Global
	State  reference.Local
}

type SMGetCachedMemory struct {
	// input arguments
	Meta    *payload.Meta
	Payload *rms.VObjectTranscriptReport

	messageSender messageSenderAdapter.MessageSender
	memoryCache   memoryCacheAdapter.MemoryCache

	Object reference.Global
	State  reference.Local
	Result descriptor.Object

	stateGlobal reference.Global
	response    *payload.VCachedMemoryResponse
}

/* -------- Declaration ------------- */

var dSMGetCachedMemoryInstance smachine.StateMachineDeclaration = &dSMGetCachedMemory{}

type dSMGetCachedMemory struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMGetCachedMemory) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMGetCachedMemory)

	injector.MustInject(&s.memoryCache)
	injector.MustInject(&s.messageSender)
}

func (*dSMGetCachedMemory) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMGetCachedMemory)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMGetCachedMemory) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMGetCachedMemoryInstance
}

func (s *SMGetCachedMemory) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	switch {
	case s.Object.IsEmpty():
		panic(throw.IllegalValue())
	case s.State.IsEmpty():
		panic(throw.IllegalValue())
	}

	s.stateGlobal = reference.NewRecordOf(s.Object, s.State)

	return ctx.Jump(s.stepProcess)
}

func (s *SMGetCachedMemory) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	done := false
	s.memoryCache.PrepareAsync(ctx, func(ctx context.Context, svc memorycache.Service) smachine.AsyncResultFunc {
		obj, err := svc.Get(ctx, s.stateGlobal)
		return func(ctx smachine.AsyncResultContext) {
			defer func() { done = true }()
			s.Result = obj
			if err != nil {
				ctx.Log().Error("failed to get memory", err)
			}
		}
	}).Start()

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if !done {
			return ctx.Sleep().ThenRepeat()
		}
		if s.Result == nil {
			return ctx.Jump(s.stepRequestMemory)
		}
		return ctx.Stop()
	})
}

func (s *SMGetCachedMemory) stepRequestMemory(ctx smachine.ExecutionContext) smachine.StateUpdate {
	bargeInCallback := ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*payload.VCachedMemoryResponse)
		if !ok || res == nil {
			panic(throw.IllegalValue())
		}

		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			s.response = res

			return ctx.WakeUp()
		}
	})

	key := CachedMemoryReportAwaitKey{Object: s.Object, State: s.State}
	if !ctx.PublishGlobalAliasAndBargeIn(key, bargeInCallback) {
		return ctx.Error(throw.E("failed to publish bargeIn callback"))
	}

	msg := payload.VCachedMemoryRequest{
		Object:  s.Object,
		StateID: s.State,
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, affinity.DynamicRoleVirtualExecutor, s.Object, s.State.GetPulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
				return
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if s.response == nil {
			return ctx.Sleep().ThenRepeat()
		}
		return ctx.Jump(s.stepProcessResponse)
	})
}

func (s *SMGetCachedMemory) stepProcessResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch s.response.CallStatus {
	case payload.CachedMemoryStateFound:
		// ok
	case payload.CachedMemoryStateMissing:
		return ctx.Error(throw.E("Not existing state"))
	case payload.CachedMemoryStateUnknown:
		panic(throw.NotImplemented())
	default:
		panic(throw.Impossible())
	}

	s.Result = descriptor.NewObject(
		s.Object,
		s.State,
		reference.Global{},
		s.response.Memory,
		s.response.Inactive,
	)

	s.memoryCache.PrepareAsync(ctx, func(ctx context.Context, svc memorycache.Service) smachine.AsyncResultFunc {
		err := svc.Set(ctx, s.stateGlobal, s.Result)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to set memory", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Stop()
}
