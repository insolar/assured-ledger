//go:generate sm-uml-gen -f $GOFILE

package object

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callsummary"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object/preservedstatereport"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
	"github.com/insolar/assured-ledger/ledger-core/virtual/validation"
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

const UnorderedMaxParallelism = 30

type Info struct {
	Reference           reference.Global
	descriptorDirty     descriptor.Object
	descriptorValidated descriptor.Object

	UnorderedExecute           smachine.SyncLink
	OrderedExecute             smachine.SyncLink
	ReadyToWork                smachine.SyncLink
	SummaryDone                smachine.SyncLink
	PendingConstructorFinished smachine.SyncLink

	SignalOrderedPendingFinished smachine.BargeInWithParam

	// KnownRequests holds requests that were seen on current pulse
	KnownRequests callregistry.WorkingTable
	// PendingTable holds requests that are known to be processed by other executors
	PendingTable callregistry.PendingTable

	PreviousExecutorUnorderedPendingCount uint8
	PreviousExecutorOrderedPendingCount   uint8

	UnorderedPendingEarliestPulse pulse.Number
	OrderedPendingEarliestPulse   pulse.Number

	objectState State

	Transcript validation.Transcript
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

func (i *Info) FinishRequest(
	isolation contract.MethodIsolation,
	requestRef reference.Global,
	result *rms.VCallResult,
) {
	i.KnownRequests.Finish(isolation.Interference, requestRef, result)
}

func (i *Info) SetDescriptorDirty(objectDescriptor descriptor.Object) {
	i.descriptorDirty = objectDescriptor
}

func (i *Info) SetDescriptorValidated(objectDescriptor descriptor.Object) {
	i.descriptorValidated = objectDescriptor
}

func (i *Info) DescriptorDirty() descriptor.Object {
	return i.descriptorDirty
}

func (i *Info) DescriptorValidated() descriptor.Object {
	return i.descriptorValidated
}

func (i Info) GetEarliestPulse(tolerance isolation.InterferenceFlag) pulse.Number {
	var (
		minPendingRequestPulse = i.PendingTable.GetList(tolerance).EarliestPulse()
		minKnownRequestPulse   = i.KnownRequests.GetList(tolerance).EarliestPulse()
	)

	switch {
	case !minKnownRequestPulse.IsUnknownOrTimePulse(), !minPendingRequestPulse.IsUnknownOrTimePulse():
		panic(throw.IllegalState())
	case minKnownRequestPulse.IsUnknown(), minPendingRequestPulse.IsBefore(minKnownRequestPulse):
		return minPendingRequestPulse
	default:
		return minKnownRequestPulse
	}
}

func (i *Info) BuildStateReport() rms.VStateReport {
	unorderedPendingCount := i.PendingTable.GetList(isolation.CallIntolerable).CountActive()
	unorderedPendingCount += i.KnownRequests.GetList(isolation.CallIntolerable).CountActive()

	orderedPendingCount := i.PendingTable.GetList(isolation.CallTolerable).CountActive()
	orderedPendingCount += i.KnownRequests.GetList(isolation.CallTolerable).CountActive()

	res := rms.VStateReport{
		Object:                        rms.NewReference(i.Reference),
		UnorderedPendingCount:         int32(unorderedPendingCount),
		UnorderedPendingEarliestPulse: i.GetEarliestPulse(isolation.CallIntolerable),
		OrderedPendingCount:           int32(orderedPendingCount),
		OrderedPendingEarliestPulse:   i.GetEarliestPulse(isolation.CallTolerable),
		ProvidedContent:               &rms.VStateReport_ProvidedContentBody{},
	}

	switch i.GetState() {
	case Unknown:
		panic(throw.IllegalState())
	case Missing:
		res.Status = rms.StateStatusMissing
	case Inactive:
		res.Status = rms.StateStatusInactive
	case Empty:
		if i.KnownRequests.GetList(isolation.CallTolerable).CountActive() == 0 {
			// constructor has not started
			res.Status = rms.StateStatusMissing
		} else {
			res.Status = rms.StateStatusEmpty
		}
	case HasState:
		// ok case
		res.Status = rms.StateStatusReady
	default:
		panic(throw.IllegalValue())
	}

	switch objDescriptor := i.DescriptorDirty(); {
	case objDescriptor == nil:
	case objDescriptor.Deactivated():
	case len(objDescriptor.Memory()) == 0:
	default:
		res.LatestDirtyState.Set(objDescriptor.HeadRef())
	}

	return res
}

func (i *Info) BuildLatestDirtyState() *rms.ObjectState {
	if objDescriptor := i.DescriptorDirty(); objDescriptor != nil {
		return &rms.ObjectState{
			Reference:   rms.NewReference(objDescriptor.State()),
			Class:       rms.NewReference(objDescriptor.Class()),
			Memory:      rms.NewBytes(objDescriptor.Memory()),
			Deactivated: objDescriptor.Deactivated(),
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
				KnownRequests: callregistry.NewWorkingTable(),
				PendingTable:  callregistry.NewRequestTable(),
				Transcript:    validation.NewTranscript(),
			},
		},
	}
}

type SMObject struct {
	smachine.StateMachineDeclTemplate

	SharedState

	readyToWorkCtl        smsync.BoolConditionalLink
	orderedSemaphoreCtl   smsync.SemaChildLink
	unorderedSemaphoreCtl smsync.SemaChildLink
	summaryDoneCtl        smsync.BoolConditionalLink

	waitGetStateUntil time.Time
	smFinalizer       *preservedstatereport.SMPreservedStateReport

	// dependencies
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
	globalLimiter tool.RunnerLimiter
}

/* -------- Declaration ------------- */

func (sm *SMObject) InjectDependencies(stateMachine smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := stateMachine.(*SMObject)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.globalLimiter)
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
		ctx.Log().Trace("stop creating SMObject since we are not in present pulse")
		return ctx.Stop()
	}

	sm.readyToWorkCtl = smsync.NewConditionalBool(false, "readyToWork")
	sm.ReadyToWork = sm.readyToWorkCtl.SyncLink()

	sm.summaryDoneCtl = smsync.NewConditionalBool(false, "summaryDone")
	sm.SummaryDone = sm.summaryDoneCtl.SyncLink()

	sm.unorderedSemaphoreCtl = sm.globalLimiter.NewChildSemaphore(0, "unordered calls")
	sm.UnorderedExecute = sm.unorderedSemaphoreCtl.SyncLink()

	sm.orderedSemaphoreCtl = sm.globalLimiter.NewChildSemaphore(0, "ordered calls")
	sm.OrderedExecute = sm.orderedSemaphoreCtl.SyncLink()

	sdl := ctx.Share(&sm.SharedState, 0)
	if !ctx.Publish(sm.Reference, sdl) {
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
	flags := rms.StateRequestContentFlags(0)
	flags.Set(rms.RequestLatestDirtyState, rms.RequestLatestValidatedState,
		rms.RequestOrderedQueue, rms.RequestUnorderedQueue)

	prevPulse := sm.pulseSlot.PrevOperationPulseNumber()
	if prevPulse.IsUnknown() {
		// unable to identify exact prev pulse
		panic(throw.NotImplemented())
	}

	msg := rms.VStateRequest{
		AsOf:             prevPulse,
		Object:           rms.NewReference(sm.Reference),
		RequestedContent: flags,
	}

	ref := sm.Reference

	sm.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, affinity.DynamicRoleVirtualExecutor, ref, prevPulse)
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
	state := sm.GetState()

	if sm.PreviousExecutorOrderedPendingCount == 0 {
		sm.releaseOrderedExecutionPath()
		sm.releaseUnorderedExecutionPath()
	} else {
		var pendingConstructorFinishedCtl smsync.BoolConditionalLink
		if state == Empty {
			pendingConstructorFinishedCtl = smsync.NewConditionalBool(false, "pendingConstructorFinished")
			sm.PendingConstructorFinished = pendingConstructorFinishedCtl.SyncLink()
		} else {
			sm.releaseUnorderedExecutionPath()
		}

		sm.SignalOrderedPendingFinished = ctx.NewBargeInWithParam(func(interface{}) smachine.BargeInCallbackFunc {
			return func(ctx smachine.BargeInContext) smachine.StateUpdate {
				if state == Empty {
					smachine.ApplyAdjustmentAsync(pendingConstructorFinishedCtl.NewValue(true))
					sm.releaseUnorderedExecutionPath()
				}
				sm.releaseOrderedExecutionPath()

				return ctx.Stay()
			}
		})
	}

	ctx.ApplyAdjustment(sm.readyToWorkCtl.NewValue(true))

	return ctx.Jump(sm.stepWaitIndefinitely)
}

func (sm *SMObject) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

func (sm *SMObject) releaseOrderedExecutionPath() {
	adjustment := sm.orderedSemaphoreCtl.NewValue(1)
	smachine.ApplyAdjustmentAsync(adjustment)
}

func (sm *SMObject) releaseUnorderedExecutionPath() {
	adjustment := sm.unorderedSemaphoreCtl.NewValue(UnorderedMaxParallelism)
	smachine.ApplyAdjustmentAsync(adjustment)
}

func (sm *SMObject) migrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	if sm.GetState() == Unknown {
		ctx.Log().Trace("SMObject migration happened when object is not ready yet")
		return ctx.Stop()
	}

	ctx.UnpublishAll()

	sm.smFinalizer = &preservedstatereport.SMPreservedStateReport{
		Reference: sm.Reference,
	}

	sm.checkPendingCounters(ctx.Log())
	if sm.descriptorDirty != nil && sm.descriptorDirty.Deactivated() {
		sm.SetState(Inactive)
	}
	sm.smFinalizer.Report = sm.BuildStateReport()
	if sm.DescriptorDirty() != nil && sm.smFinalizer.Report.GetStatus() != rms.StateStatusInactive {
		state := sm.BuildLatestDirtyState()
		sm.smFinalizer.Report.ProvidedContent.LatestDirtyState = state
		sm.smFinalizer.Report.ProvidedContent.LatestValidatedState = state
	}

	if err := sm.sharedAndPublishStateReport(ctx, &sm.smFinalizer.Report); err != nil {
		return ctx.Error(err)
	}

	if err := sm.sharedAndPublishSMCallSummarySyncLink(ctx, &sm.SummaryDone); err != nil {
		return ctx.Error(err)
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

		summarySharedStateAccessor, ok = callsummary.GetSummarySMSharedAccessor(ctx, pulseNumber)

		if !ok {
			// we should get accessor always after InitChild in this step
			panic(throw.IllegalState())
		}
	}

	action := func(shared *callsummary.SharedCallSummary) {
		callResults := callregistry.ObjectCallResults{
			CallResults: sm.KnownRequests.GetResults(),
		}

		if !shared.Requests.AddObjectCallResults(sm.Reference, callResults) {
			// result for this object already exist
			panic(throw.Impossible())
		}

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
	unorderedPendingList := sm.PendingTable.GetList(isolation.CallIntolerable)
	if int(sm.PreviousExecutorUnorderedPendingCount) != unorderedPendingList.Count() {
		logger.Warn(pendingCountersWarnMsg{
			CounterType:  "Unordered",
			PendingCount: sm.PreviousExecutorUnorderedPendingCount,
			CountActive:  unorderedPendingList.Count(),
		})
	}

	orderedPendingList := sm.PendingTable.GetList(isolation.CallTolerable)
	if int(sm.PreviousExecutorOrderedPendingCount) != orderedPendingList.Count() {
		logger.Warn(pendingCountersWarnMsg{
			CounterType:  "Ordered",
			PendingCount: sm.PreviousExecutorOrderedPendingCount,
			CountActive:  orderedPendingList.Count(),
		})
	}
}

func (sm *SMObject) sharedAndPublishStateReport(
	ctx smachine.MigrationContext,
	report *rms.VStateReport,
) error {
	sdlStateReport := ctx.Share(report, 0)

	if !ctx.Publish(preservedstatereport.BuildReportKey(sm.Reference), sdlStateReport) {
		return throw.New("failed to publish state report", struct {
			Reference reference.Holder
		}{sm.Reference})
	}

	return nil
}

func (sm *SMObject) sharedAndPublishSMCallSummarySyncLink(
	ctx smachine.MigrationContext,
	summaryDone *smachine.SyncLink,
) error {
	sdlCallSummarySync := ctx.Share(summaryDone, 0)

	if !ctx.Publish(callsummary.BuildSummarySyncKey(sm.Reference), sdlCallSummarySync) {
		return throw.New("failed to publish call summary sync key", struct {
			Reference reference.Holder
		}{sm.Reference})
	}

	return nil
}
