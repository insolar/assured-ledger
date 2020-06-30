// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package object

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object/finalizedstate"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tables"
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

	UnorderedExecute smachine.SyncLink
	OrderedExecute   smachine.SyncLink
	ReadyToWork      smachine.SyncLink
	SummaryDone      smachine.SyncLink

	AwaitPendingOrdered smachine.BargeIn

	// KnownRequests holds requests that were seen on current pulse
	KnownRequests tables.WorkingTable
	// PendingTable holds requests that are known to be processed by other executors
	PendingTable tables.PendingTable

	PreviousExecutorUnorderedPendingCount uint8
	PreviousExecutorOrderedPendingCount   uint8

	// Potential means pendings on this executor
	PotentialUnorderedPendingCount uint8
	PotentialOrderedPendingCount   uint8

	UnorderedPendingEarliestPulse pulse.Number
	OrderedPendingEarliestPulse   pulse.Number

	OrderedPendingListFilled   smachine.SyncLink
	UnorderedPendingListFilled smachine.SyncLink

	OrderedPendingListFilledCallback   smachine.BargeIn
	UnorderedPendingListFilledCallback smachine.BargeIn

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

func (i *Info) SetPendingListFilled(ctx smachine.ExecutionContext, tolerance contract.InterferenceFlag) {
	switch tolerance {
	case contract.CallIntolerable:
		ctx.CallBargeIn(i.UnorderedPendingListFilledCallback)
	case contract.CallTolerable:
		ctx.CallBargeIn(i.OrderedPendingListFilledCallback)
	default:
		panic(throw.IllegalValue())
	}
}

func (i *Info) IncrementPotentialPendingCounter(isolation contract.MethodIsolation) {
	switch isolation.Interference {
	case contract.CallIntolerable:
		i.PotentialUnorderedPendingCount++
	case contract.CallTolerable:
		i.PotentialOrderedPendingCount++
	default:
		panic(throw.Unsupported())
	}
}

func (i *Info) FinishRequest(
	isolation contract.MethodIsolation,
	requestRef reference.Global,
	result *payload.VCallResult,
) {
	switch isolation.Interference {
	case contract.CallIntolerable:
		i.PotentialUnorderedPendingCount--
	case contract.CallTolerable:
		i.PotentialOrderedPendingCount--
	default:
		panic(throw.Unsupported())
	}
	i.KnownRequests.Finish(isolation.Interference, requestRef, result)
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

func (i Info) GetEarliestPulse(tolerance contract.InterferenceFlag) pulse.Number {
	minPulse := i.PendingTable.GetList(tolerance).EarliestPulse()
	knownPulse := i.KnownRequests.GetList(tolerance).EarliestPulse()
	if knownPulse != pulse.Unknown && (minPulse == pulse.Unknown || knownPulse < minPulse) {
		minPulse = knownPulse
	}
	return minPulse
}

func (i *Info) BuildStateReport() payload.VStateReport {
	previousExecutorUnorderedPendingCount := i.PendingTable.GetList(contract.CallIntolerable).CountActive()
	previousExecutorOrderedPendingCount := i.PendingTable.GetList(contract.CallTolerable).CountActive()
	res := payload.VStateReport{
		Object:                        i.Reference,
		UnorderedPendingCount:         int32(previousExecutorUnorderedPendingCount) + int32(i.PotentialUnorderedPendingCount),
		UnorderedPendingEarliestPulse: i.GetEarliestPulse(contract.CallIntolerable),
		OrderedPendingCount:           int32(previousExecutorOrderedPendingCount) + int32(i.PotentialOrderedPendingCount),
		OrderedPendingEarliestPulse:   i.GetEarliestPulse(contract.CallTolerable),
		ProvidedContent:               &payload.VStateReport_ProvidedContentBody{},
	}

	switch i.GetState() {
	case Unknown:
		panic(throw.IllegalState())
	case Missing:
		res.Status = payload.Missing
	case Inactive:
		res.Status = payload.Inactive
	case Empty:
		if i.PotentialOrderedPendingCount == uint8(0) {
			// constructor has not started
			res.Status = payload.Missing
		} else {
			res.Status = payload.Empty
		}
	case HasState:
		// ok case
		res.Status = payload.Ready
	default:
		panic(throw.IllegalValue())
	}

	if objDescriptor := i.Descriptor(); objDescriptor != nil {
		res.LatestDirtyState = objDescriptor.HeadRef()
	}

	return res
}

func (i *Info) BuildLatestDirtyState() *payload.ObjectState {
	if objDescriptor := i.Descriptor(); objDescriptor != nil {
		class, _ := objDescriptor.Class()
		return &payload.ObjectState{
			Reference:   objDescriptor.StateID(),
			Parent:      objDescriptor.Parent(),
			Class:       class,
			State:       objDescriptor.Memory(),
			Deactivated: i.Deactivated,
		}
	}
	return nil
}

type SharedState struct {
	Info
}

func NewStateMachineObject(objectReference reference.Global) *SMObject {
	return &SMObject{
		SharedState: SharedState{
			Info: Info{
				Reference:     objectReference,
				KnownRequests: tables.NewWorkingTable(),
				PendingTable:  tables.NewRequestTable(),
			},
		},
	}
}

type SMObject struct {
	smachine.StateMachineDeclTemplate

	SharedState

	readyToWorkCtl smsync.BoolConditionalLink
	summaryDoneCtl smsync.BoolConditionalLink

	waitGetStateUntil time.Time
	smFinalizer       *finalizedstate.SMStateFinalizer

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
	if sm.pulseSlot.State() != conveyor.Present {
		return ctx.Stop()
	}

	sm.readyToWorkCtl = smsync.NewConditionalBool(false, "readyToWork")
	sm.ReadyToWork = sm.readyToWorkCtl.SyncLink()

	sm.summaryDoneCtl = smsync.NewConditionalBool(false, "summaryDone")
	sm.SummaryDone = sm.summaryDoneCtl.SyncLink()

	sm.UnorderedExecute = smsync.NewSemaphore(30, "immutable calls").SyncLink()
	sm.OrderedExecute = smsync.NewSemaphore(1, "mutable calls").SyncLink() // TODO here we need an ORDERED queue

	sm.OrderedPendingListFilled = smsync.NewSemaphore(1, "ordered pending list filled").SyncLink()
	sm.UnorderedPendingListFilled = smsync.NewSemaphore(1, "unordered pending list filled").SyncLink()

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
	waitDuration := pulseDuration / 100 * waitStatePulsePercent
	pulseStartedAt := sm.pulseSlot.PulseStartedAt()

	sm.waitGetStateUntil = pulseStartedAt.Add(waitDuration)
}

func (sm *SMObject) stepSendStateRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	flags := payload.StateRequestContentFlags(0)
	flags.Set(payload.RequestLatestDirtyState, payload.RequestLatestValidatedState,
		payload.RequestOrderedQueue, payload.RequestUnorderedQueue)
	prevPulse := sm.pulseSlot.PulseData().PrevPulseNumber()
	msg := payload.VStateRequest{
		AsOf:             prevPulse,
		Object:           sm.Reference,
		RequestedContent: flags,
	}

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

func (sm *SMObject) stepGetState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.IsReady() {
		return ctx.Jump(sm.stepGotState)
	}

	if !time.Now().After(sm.waitGetStateUntil) {
		return ctx.WaitAnyUntil(sm.waitGetStateUntil).ThenRepeat()
	}

	return ctx.Jump(sm.stepSendStateRequest)
}

func (sm *SMObject) stepWaitState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.IsReady() {
		return ctx.Jump(sm.stepGotState)
	}

	return ctx.Sleep().ThenRepeat()
}

func (sm *SMObject) stepGotState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.PreviousExecutorOrderedPendingCount > 0 {
		sm.createWaitPendingOrderedSM(ctx)
		sm.createWaitOrderedPendingTableSM(ctx)
	}

	if sm.PreviousExecutorUnorderedPendingCount > 0 {
		sm.createWaitUnorderedPendingTableSM(ctx)
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

func (sm *SMObject) createWaitOrderedPendingTableSM(ctx smachine.ExecutionContext) {
	syncSM := SMAwaitTableFill{
		sync: sm.OrderedPendingListFilled,
	}

	// syncSM acquire OrderedPendingListFilledCallback semaphore in init step.
	ctx.InitChildWithPostInit(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &syncSM
	}, func() {
		sm.OrderedPendingListFilledCallback = syncSM.stop
	})
	if sm.OrderedPendingListFilledCallback.IsZero() {
		panic(throw.IllegalState())
	}
}

func (sm *SMObject) createWaitUnorderedPendingTableSM(ctx smachine.ExecutionContext) {
	syncSM := SMAwaitTableFill{
		sync: sm.UnorderedPendingListFilled,
	}

	// syncSM acquire UnorderedPendingListFilledCallback semaphore in init step.
	ctx.InitChildWithPostInit(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &syncSM
	}, func() {
		sm.UnorderedPendingListFilledCallback = syncSM.stop
	})
	if sm.UnorderedPendingListFilledCallback.IsZero() {
		panic(throw.IllegalState())
	}
}

func (sm *SMObject) createWaitPendingOrderedSM(ctx smachine.ExecutionContext) {
	syncSM := SMAwaitDelegate{
		sync: sm.OrderedExecute,
	}

	// syncSM acquire OrderedExecute semaphore in init step.
	ctx.InitChildWithPostInit(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &syncSM
	}, func() {
		sm.AwaitPendingOrdered = syncSM.stop
	})

	if sm.AwaitPendingOrdered.IsZero() {
		panic(throw.IllegalState())
	}
}

func (sm *SMObject) migrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	if sm.GetState() == Unknown {
		ctx.Log().Trace("SMObject migration happened when object is not ready yet")
		return ctx.Stop()
	}

	ctx.UnpublishAll()

	sm.smFinalizer = &finalizedstate.SMStateFinalizer{
		Reference: sm.Reference,
	}

	sm.checkPendingCounters(ctx.Log())
	sm.smFinalizer.Report = sm.BuildStateReport()
	if sm.Descriptor() != nil {
		sm.smFinalizer.Report.ProvidedContent.LatestDirtyState = sm.BuildLatestDirtyState()
	}

	sdlStateReport := ctx.Share(&sm.smFinalizer.Report, 0)
	if !ctx.Publish(finalizedstate.BuildReportKey(sm.Reference), sdlStateReport) {
		ctx.Log().Warn(struct {
			*log.Msg  `txt:"failed to publish state report"`
			Reference string
		}{
			Reference: sm.Reference.String(),
		})
		return ctx.Error(throw.New("failed to publish state report"))
	}

	sdlCallSummarySync := ctx.Share(&sm.SummaryDone, 0)

	if !ctx.Publish(callsummary.BuildSummarySyncKey(sm.Reference), sdlCallSummarySync) {
		ctx.Log().Warn(struct {
			*log.Msg  `txt:"failed to publish call summary sync key"`
			Reference string
		}{
			Reference: sm.Reference.String(),
		})
		return ctx.Error(throw.New("failed to publish call summary sync key"))
	}

	return ctx.Jump(sm.stepPublishCallSummary)
}

func (sm *SMObject) stepPublishCallSummary(ctx smachine.ExecutionContext) smachine.StateUpdate {
	pulseNumber := sm.pulseSlot.PulseData().PulseNumber
	summarySharedStateAccessor, ok := callsummary.GetSummarySMSharedAccessor(ctx, pulseNumber)

	if !ok {
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return callsummary.NewStateMachineCallSummary(pulseNumber)
		})

		summarySharedStateAccessor, _ = callsummary.GetSummarySMSharedAccessor(ctx, pulseNumber)
	}

	action := func(shared *callsummary.SharedCallSummary) {
		shared.Requests.AddObjectRequests(sm.Reference, sm.KnownRequests)

		if !ctx.Unpublish(callsummary.BuildSummarySyncKey(sm.Reference)) {
			ctx.Log().Warn(struct {
				*log.Msg  `txt:"failed to unpublish call summary sync key"`
				Reference string
			}{
				Reference: sm.Reference.String(),
			})
			ctx.Error(throw.New("failed to unpublish call summary sync key"))
		}
	}

	switch summarySharedStateAccessor.Prepare(action).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(summarySharedStateAccessor.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		panic(throw.NotImplemented())
	case smachine.Passed:
		// go further
	default:
		panic(throw.Impossible())
	}

	ctx.ApplyAdjustment(sm.summaryDoneCtl.NewValue(true))

	return ctx.Jump(sm.stepFinalize)
}

func (sm *SMObject) stepFinalize(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.ReplaceWith(sm.smFinalizer)
}

type pendingCountersWarnMsg struct {
	*log.Msg     `txt:"Pending counter does not match active records count in table"`
	CounterType  string
	PendingCount uint8
	CountActive  int
}

func (sm *SMObject) checkPendingCounters(logger smachine.Logger) {
	unorderedPendingList := sm.PendingTable.GetList(contract.CallIntolerable)
	if int(sm.PreviousExecutorUnorderedPendingCount) != unorderedPendingList.Count() {
		logger.Warn(pendingCountersWarnMsg{
			CounterType:  "Unordered",
			PendingCount: sm.PreviousExecutorUnorderedPendingCount,
			CountActive:  unorderedPendingList.Count(),
		})
	}

	orderedPendingList := sm.PendingTable.GetList(contract.CallTolerable)
	if int(sm.PreviousExecutorOrderedPendingCount) != orderedPendingList.Count() {
		logger.Warn(pendingCountersWarnMsg{
			CounterType:  "Ordered",
			PendingCount: sm.PreviousExecutorOrderedPendingCount,
			CountActive:  orderedPendingList.Count(),
		})
	}
}
