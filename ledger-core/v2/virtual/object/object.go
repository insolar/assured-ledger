// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type State int32

const (
	Unknown State = iota
	Missing
	Inactive
	Empty
	HasState
)

const waitStatePulsePercent = 10

type Info struct {
	Reference   reference.Global
	descriptor  descriptor.Object
	Deactivated bool

	ImmutableExecute smachine.SyncLink
	MutableExecute   smachine.SyncLink
	ReadyToWork      smachine.SyncLink

	AwaitPendingOrdered smachine.BargeIn

	KnownRequests map[reference.Global]struct{}
	PendingTable  PendingTable

	ActiveImmutablePendingCount    uint8
	ActiveMutablePendingCount      uint8
	PotentialImmutablePendingCount uint8
	PotentialMutablePendingCount   uint8

	objectState State
}

func (i *Info) IsReady() bool {
	return i.objectState != Unknown
}

func (i *Info) SetState(state State) {
	i.objectState = state
}

func (i *Info) GetState() State {
	return i.objectState
}

func (i *Info) IncrementPotentialPendingCounter(isolation contract.MethodIsolation) {
	switch isolation.Interference {
	case contract.CallIntolerable:
		i.PotentialImmutablePendingCount++
	case contract.CallTolerable:
		i.PotentialMutablePendingCount++
	default:
		panic(throw.Unsupported())
	}
}

func (i *Info) DecrementPotentialPendingCounter(isolation contract.MethodIsolation) {
	switch isolation.Interference {
	case contract.CallIntolerable:
		i.PotentialImmutablePendingCount--
	case contract.CallTolerable:
		i.PotentialMutablePendingCount--
	default:
		panic(throw.Unsupported())
	}
}

func (i *Info) SetDescriptor(objectDescriptor descriptor.Object) {
	i.descriptor = objectDescriptor
}

func (i *Info) Deactivate() {
	i.Deactivated = true
}

func (i *Info) Descriptor() descriptor.Object {
	return i.descriptor
}

type SharedState struct {
	Info
}

func NewStateMachineObject(objectReference reference.Global) *SMObject {
	return &SMObject{
		SharedState: SharedState{
			Info: Info{
				Reference:     objectReference,
				KnownRequests: make(map[reference.Global]struct{}),
				PendingTable:  NewPendingTable(),
			},
		},
	}
}

type SMObject struct {
	smachine.StateMachineDeclTemplate

	SharedState

	readyToWorkCtl    smsync.BoolConditionalLink
	migrateTransition smachine.StateFunc

	waitGetStateUntil time.Time

	// dependencies
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
}

/* -------- Declaration ------------- */

func (sm *SMObject) InjectDependencies(stateMachine smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	s := stateMachine.(*SMObject)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.pulseSlot)
}

func (sm *SMObject) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

/* -------- Instance ------------- */

func (sm *SMObject) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *SMObject) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	sm.readyToWorkCtl = smsync.NewConditionalBool(false, "readyToWork")
	sm.ReadyToWork = sm.readyToWorkCtl.SyncLink()

	sm.ImmutableExecute = smsync.NewSemaphore(30, "immutable calls").SyncLink()
	sm.MutableExecute = smsync.NewSemaphore(1, "mutable calls").SyncLink() // TODO here we need an ORDERED queue

	sdl := ctx.Share(&sm.SharedState, 0)
	if !ctx.Publish(sm.Reference.String(), sdl) {
		return ctx.Stop()
	}

	sm.initWaitGetStateUntil()

	ctx.SetDefaultMigration(sm.migrate)

	return ctx.Jump(sm.stepGetState)
}

func (sm *SMObject) initWaitGetStateUntil() {
	pulseDuration := time.Second * time.Duration(sm.pulseSlot.PulseData().NextPulseDelta)
	waitDuration := pulseDuration / waitStatePulsePercent
	pulseStartedAt := sm.pulseSlot.PulseStartedAt()

	sm.waitGetStateUntil = pulseStartedAt.Add(waitDuration)
}

func (sm *SMObject) stepGetState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.IsReady() {
		return ctx.Jump(sm.stepGotState)
	}

	if !time.Now().After(sm.waitGetStateUntil) {
		return ctx.WaitAnyUntil(sm.waitGetStateUntil).ThenRepeat()
	}

	return ctx.Jump(sm.stepSendStateRequest)
}

func (sm *SMObject) stepSendStateRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	flags := payload.StateRequestContentFlags(0)
	flags.Set(payload.RequestLatestDirtyState, payload.RequestLatestValidatedState,
		payload.RequestMutableQueue, payload.RequestImmutableQueue)
	msg := payload.VStateRequest{
		Callee:           sm.Reference,
		RequestedContent: flags,
	}

	prevPulse := sm.pulseSlot.PulseData().PrevPulseNumber()
	ref := sm.Reference

	sm.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, node.DynamicRoleVirtualExecutor, ref, prevPulse)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send state", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Jump(sm.stepWaitState)
}

func (sm *SMObject) stepWaitState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.IsReady() {
		return ctx.Jump(sm.stepGotState)
	}

	return ctx.Sleep().ThenRepeat()
}

func (sm *SMObject) stepGotState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.ActiveMutablePendingCount > 0 {
		sm.createWaitPendingOrderedSM(ctx)
	}
	return ctx.Jump(sm.stepReadyToWork)
}

func (sm *SMObject) stepReadyToWork(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.ApplyAdjustment(sm.readyToWorkCtl.NewValue(true))
	return ctx.Jump(sm.stepWaitIndefinitely)
}

func (sm *SMObject) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

func (sm *SMObject) createWaitPendingOrderedSM(ctx smachine.ExecutionContext) {
	syncSM := AwaitOrderedPendingSM{
		sync: sm.MutableExecute,
	}

	// syncSM acquire MutableExecute semaphore in init step.
	ctx.InitChildWithPostInit(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &syncSM
	}, func() {
		sm.AwaitPendingOrdered = syncSM.stop
	})
}

func (sm *SMObject) stepSendVStateReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		pulseNumber = sm.pulseSlot.CurrentPulseNumber()
	)

	objDescriptor := sm.Descriptor()
	prototype, _ := objDescriptor.Prototype()
	msg := payload.VStateReport{
		AsOf:                  sm.pulseSlot.PulseData().PulseNumber,
		Callee:                sm.SharedState.Info.Reference,
		ImmutablePendingCount: int32(sm.ActiveImmutablePendingCount) + int32(sm.PotentialImmutablePendingCount),
		MutablePendingCount:   int32(sm.ActiveMutablePendingCount) + int32(sm.PotentialMutablePendingCount),
		LatestDirtyState:      objDescriptor.HeadRef(),
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference:   objDescriptor.StateID(),
				Parent:      objDescriptor.Parent(),
				Prototype:   prototype,
				State:       objDescriptor.Memory(),
				Deactivated: sm.Deactivated,
			},
		},
	}

	sm.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, node.DynamicRoleVirtualExecutor, sm.Reference, pulseNumber)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send state", err)
			}
		}
	}).WithoutAutoWakeUp().Start()
	// TODO: sm must be stopped here in PLAT-347
	return ctx.Jump(sm.migrateTransition)
}

func (sm *SMObject) stepSendVStateUnavailable(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		pulseNumber = sm.pulseSlot.CurrentPulseNumber()
		failReason  payload.VStateUnavailable_ReasonType
	)
	switch sm.GetState() {
	case Missing:
		failReason = payload.Missing
	case Inactive:
		failReason = payload.Inactive
	default:
		panic(throw.IllegalState())
	}

	msg := payload.VStateUnavailable{
		Reason:   failReason,
		Lifeline: sm.SharedState.Info.Reference,
	}

	sm.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, node.DynamicRoleVirtualExecutor, sm.SharedState.Info.Reference, pulseNumber)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send state", err)
			}
		}
	}).WithoutAutoWakeUp().Start()
	return ctx.Jump(sm.migrateTransition)
}

func (sm *SMObject) migrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	sm.migrateTransition = ctx.AffectedStep().Transition
	switch sm.GetState() {
	case Unknown:
		ctx.Log().Warn("SMObject migration happened when object is not ready yet")
		// TODO: should sm die here in PLAT-347?
		return ctx.Stay()
	case Missing:
		fallthrough
	case Inactive:
		return ctx.Jump(sm.stepSendVStateUnavailable)
	case Empty:
		if sm.PotentialMutablePendingCount == uint8(0) && sm.PotentialImmutablePendingCount == uint8(0) {
			// SMObject construction was interrupted by migration. Counters was not incremented yet
			panic(throw.NotImplemented())
		}
		fallthrough
	case HasState:
		return ctx.Jump(sm.stepSendVStateReport)
	default:
		panic(throw.IllegalState())
	}
}
