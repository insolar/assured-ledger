package handlers

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
)

type SMVCachedMemoryRequest struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VCachedMemoryRequest

	messageSender messageSenderAdapter.MessageSender
	memoryCache   memoryCacheAdapter.MemoryCache

	object   descriptor.Object
	response *rms.VCachedMemoryResponse
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
	var (
		objectStateRef = s.Payload.State.GetValue()
	)

	return s.memoryCache.PrepareAsync(ctx, func(ctx context.Context, svc memorycache.Service) smachine.AsyncResultFunc {
		objectDescriptor, err := svc.Get(ctx, objectStateRef)

		return func(ctx smachine.AsyncResultContext) {
			s.object = objectDescriptor
			if err != nil {
				ctx.Log().Error("failed to get memory", err)
			}
		}
	}).DelayedStart().ThenJump(s.stepWaitResult)
}

func (s *SMVCachedMemoryRequest) stepWaitResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.object == nil {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(s.stepBuildResult)
}

func (s *SMVCachedMemoryRequest) stepBuildResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.object.HeadRef().IsEmpty() {
		s.response = &rms.VCachedMemoryResponse{
			CallStatus: rms.CachedMemoryStateUnknown,
			State: rms.ObjectState{
				Reference: s.Payload.State,
			},
		}
	} else {
		s.response = &rms.VCachedMemoryResponse{
			CallStatus: rms.CachedMemoryStateFound,
			State: rms.ObjectState{
				Reference:   s.Payload.State,
				Class:       rms.NewReference(s.object.Class()),
				Memory:      rms.NewBytes(s.object.Memory()),
				Deactivated: s.object.Deactivated(),
			},
		}
	}

	return ctx.Jump(s.stepSendResult)
}

func (s *SMVCachedMemoryRequest) stepSendResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		response = s.response
		target   = s.Meta.Sender.GetValue()
	)

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendTarget(goCtx, response, target)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Stop()
}
