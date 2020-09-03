// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SMVDelegatedRequestFinished struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VDelegatedRequestFinished

	objectSharedState object.SharedStateAccessor
	objectReadyToWork smachine.SyncLink

	// dependencies
	objectCatalog object.Catalog
	pulseSlot     *conveyor.PulseSlot
	memoryCache   memoryCacheAdapter.MemoryCache
}

type stateIsNotReady struct {
	*log.Msg `txt:"State is not ready"`
	Object   reference.Holder
}

type unexpectedVDelegateRequestFinished struct {
	*log.Msg `txt:"Unexpected VDelegateRequestFinished"`
	Object   reference.Holder
	Request  reference.Holder
	Ordered  bool
}

type noLatestStateTolerableVDelegateRequestFinished struct {
	*log.Msg `txt:"Tolerable VDelegateRequestFinished on Empty object has no LatestState"`
	Object   reference.Holder
	Request  reference.Holder
}

var dSMVDelegatedRequestFinishedInstance smachine.StateMachineDeclaration = &dSMVDelegatedRequestFinished{}

type dSMVDelegatedRequestFinished struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVDelegatedRequestFinished) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVDelegatedRequestFinished)

	injector.MustInject(&s.objectCatalog)
	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.memoryCache)
}

func (*dSMVDelegatedRequestFinished) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVDelegatedRequestFinished)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVDelegatedRequestFinished) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVDelegatedRequestFinishedInstance
}

func (s *SMVDelegatedRequestFinished) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if s.pulseSlot.State() != conveyor.Present {
		ctx.Log().Warn("stop processing VDelegatedRequestFinished since we are not in present pulse")
		return ctx.Stop()
	}
	ctx.SetDefaultMigration(s.migrationDefault)

	return ctx.Jump(s.stepGetObject)
}

func (s *SMVDelegatedRequestFinished) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.Log().Trace("stop processing SMVDelegatedRequestFinished since pulse was changed")
	return ctx.Stop()
}

func (s *SMVDelegatedRequestFinished) stepGetObject(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.objectSharedState = s.objectCatalog.GetOrCreate(ctx, s.Payload.Callee)

	var (
		semaphoreReadyToWork smachine.SyncLink
	)

	action := func(state *object.SharedState) {
		semaphoreReadyToWork = state.ReadyToWork
	}

	switch s.objectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	s.objectReadyToWork = semaphoreReadyToWork

	return ctx.Jump(s.awaitObjectReady)
}

func (s *SMVDelegatedRequestFinished) awaitObjectReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if ctx.AcquireForThisStep(s.objectReadyToWork).IsNotPassed() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepProcess)
}

func (s *SMVDelegatedRequestFinished) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	setStateFunc := func(data interface{}) (wakeup bool) {
		state := data.(*object.SharedState)
		if !state.IsReady() {
			ctx.Log().Trace(stateIsNotReady{Object: s.Payload.Callee})
			return false
		}

		s.updateSharedState(ctx, state)

		return false
	}

	switch s.objectSharedState.PrepareAccess(setStateFunc).TryUse(ctx).GetDecision() {
	case smachine.Passed:
	case smachine.NotPassed:
		return ctx.WaitShared(s.objectSharedState.SharedDataLink).ThenRepeat()
	default:
		panic(throw.Impossible())
	}

	s.updateMemoryCache(ctx)

	return ctx.Stop()
}

func (s *SMVDelegatedRequestFinished) updateSharedState(
	ctx smachine.ExecutionContext,
	state *object.SharedState,
) {
	objectRef := s.Payload.Callee
	requestRef := s.Payload.CallOutgoing

	// Update object state.
	if s.hasLatestState() {
		state.SetDescriptorDirty(s.latestState())
		s.updateObjectState(state)
	} else if s.Payload.CallFlags.GetInterference() == isolation.CallTolerable &&
		s.Payload.CallType == payload.CallTypeConstructor &&
		state.GetState() == object.Empty {

		ctx.Log().Warn(noLatestStateTolerableVDelegateRequestFinished{
			Object:  objectRef,
			Request: requestRef,
		})
		state.SetState(object.Missing)
	}

	pendingList := state.PendingTable.GetList(s.Payload.CallFlags.GetInterference())
	if !pendingList.Finish(requestRef) {
		panic(throw.E("delegated request was not registered", struct {
			Object  string
			Request string
		}{
			Object:  objectRef.String(),
			Request: requestRef.String(),
		}))
	}

	switch s.Payload.CallFlags.GetInterference() {
	case isolation.CallIntolerable:
		if state.PreviousExecutorUnorderedPendingCount == 0 {
			ctx.Log().Warn(unexpectedVDelegateRequestFinished{
				Object:  objectRef,
				Request: requestRef,
				Ordered: false,
			})
		}
	case isolation.CallTolerable:
		if state.PreviousExecutorOrderedPendingCount == 0 {
			ctx.Log().Warn(unexpectedVDelegateRequestFinished{
				Object:  objectRef,
				Request: requestRef,
				Ordered: true,
			})
		}
		if pendingList.CountActive() == 0 {
			// If we do not have pending ordered, release sync object.
			if !ctx.CallBargeInWithParam(state.SignalOrderedPendingFinished, nil) {
				ctx.Log().Warn("SignalOrderedPendingFinished BargeIn receive false")
			}
		}
	}
}

func (s *SMVDelegatedRequestFinished) updateObjectState(state *object.SharedState) {
	switch state.GetState() {
	case object.Empty:
		state.SetState(object.HasState)
	case object.HasState:
		// ok
	default:
		panic(throw.Impossible())
	}
}

func (s *SMVDelegatedRequestFinished) updateMemoryCache(ctx smachine.ExecutionContext) {
	if !s.hasLatestState() {
		return
	}
	objectDescriptor := s.latestState()

	s.memoryCache.PrepareAsync(ctx, func(ctx context.Context, svc memorycache.Service) smachine.AsyncResultFunc {
		stateRef := reference.NewSelf(objectDescriptor.StateID())
		err := svc.Set(ctx, objectDescriptor.HeadRef(), stateRef, objectDescriptor)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to set dirty memory", err)
			}
		}
	}).WithoutAutoWakeUp().Start()
}

func (s *SMVDelegatedRequestFinished) hasLatestState() bool {
	return s.Payload.LatestState != nil
}

func (s *SMVDelegatedRequestFinished) latestState() descriptor.Object {
	state := s.Payload.LatestState
	if state == nil {
		panic(throw.IllegalState())
	}

	return descriptor.NewObject(
		s.Payload.Callee,
		state.Reference,
		state.Class,
		state.State,
		state.Deactivated,
	)
}
