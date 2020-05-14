// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type State int32

const (
	Unknown State = iota
	Missing
	Inactive
	HasState
)

type Info struct {
	Reference   reference.Global
	descriptor  descriptor.Object
	Deactivated bool

	ImmutableExecute smachine.SyncLink
	MutableExecute   smachine.SyncLink
	ReadyToWork      smachine.SyncLink

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

func (i *Info) IncrementPotentialPendingCounter(isOrdered bool) {
	if isOrdered {
		i.PotentialMutablePendingCount++
	} else {
		i.PotentialImmutablePendingCount++
	}
}

func (i *Info) DecrementPotentialPendingCounter(isOrdered bool) {
	if isOrdered {
		i.PotentialMutablePendingCount--
	} else {
		i.PotentialImmutablePendingCount--
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

// StackRelation shows relevance of 2 stacks
type InitReason int8

const (
	InitReasonCTConstructor InitReason = iota
	InitReasonCTMethod
	InitReasonVStateReport
)

func NewStateMachineObject(objectReference reference.Global, reason InitReason) *SMObject {
	return &SMObject{
		SharedState: SharedState{
			Info: Info{Reference: objectReference},
		},
		initReason: reason,
	}
}

type SMObject struct {
	smachine.StateMachineDeclTemplate

	SharedState

	readyToWorkCtl smsync.BoolConditionalLink
	initReason     InitReason

	waitGetStateUntil time.Time
	stateWasRequested bool

	// dependencies
	messageSender *messageSenderAdapter.MessageSender
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

	waitDuration := time.Second * time.Duration(sm.pulseSlot.PulseData().NextPulseDelta) / 10
	sm.waitGetStateUntil = sm.pulseSlot.PulseStartedAt().Add(waitDuration)

	switch sm.initReason {
	case InitReasonCTConstructor:
		return ctx.Jump(sm.stepReadyToWork)
	case InitReasonVStateReport:
		sm.stateWasRequested = true
		fallthrough
	case InitReasonCTMethod:
		return ctx.Jump(sm.stepWaitState)
	default:
		panic(throw.IllegalValue())
	}
}

// we get CallMethod but we have no object data
// we need to ask previous executor
func (sm *SMObject) stepGetObjectState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	flags := payload.StateRequestContentFlags(0)
	flags.Set(payload.RequestLatestDirtyState, payload.RequestLatestValidatedState,
		payload.RequestMutableQueue, payload.RequestImmutableQueue)
	msg := payload.VStateRequest{
		Callee:           sm.Reference,
		RequestedContent: flags,
	}

	goCtx := ctx.GetContext()
	prevPulse := sm.pulseSlot.PulseData().PrevPulseNumber()
	ref := sm.Reference

	sm.messageSender.PrepareNotify(ctx, func(svc messagesender.Service) {
		_ = svc.SendRole(goCtx, &msg, insolar.DynamicRoleVirtualExecutor, ref, prevPulse)
	}).Send()

	sm.stateWasRequested = true
	return ctx.Jump(sm.stepWaitState)
}

func (sm *SMObject) stepWaitState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.IsReady() {
		return ctx.Jump(sm.stepReadyToWork)
	}

	if !sm.stateWasRequested && time.Now().After(sm.waitGetStateUntil) {
		return ctx.Jump(sm.stepGetObjectState)
	}

	return ctx.WaitAnyUntil(sm.waitGetStateUntil).ThenRepeat()
}

func (sm *SMObject) stepReadyToWork(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.ApplyAdjustment(sm.readyToWorkCtl.NewValue(true))
	return ctx.Jump(sm.stepWaitIndefinitely)
}

func (sm *SMObject) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
