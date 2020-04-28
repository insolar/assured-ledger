// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate go run $GOPATH/src/github.com/rigidus/in-solar/org/analyse.go -f=$GOPACKAGE -c

package sm_object // nolint:golint

import (
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/artifacts"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_artifact"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_sender"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

// nolint
type ObjectInfo struct {
	ObjectReference insolar.Reference
	IsReadyToWork   bool

	artifactClient *s_artifact.ArtifactClientServiceAdapter
	sender         *s_sender.SenderServiceAdapter
	pulseSlot      *conveyor.PulseSlot
	externalError  error

	ObjectLatestDescriptor artifacts.ObjectDescriptor

	ImmutableExecute smachine.SyncLink
	MutableExecute   smachine.SyncLink
	ReadyToWork      smachine.SyncLink

	PreviousExecutorState payload.PreviousExecutorState
}

type SharedObjectState struct {
	SemaphorePreviousResultSaved smachine.SyncLink

	SemaphorePreviousExecutorFinished smachine.SyncLink
	PreviousExecutorFinished          smsync.BoolConditionalLink

	ObjectInfo
}

func NewStateMachineObject(objectReference insolar.Reference, exists bool) *SMObject {
	return &SMObject{
		SharedObjectState: SharedObjectState{
			ObjectInfo: ObjectInfo{ObjectReference: objectReference},
		},
		oldObject: exists,
	}
}

type SMObject struct {
	smachine.StateMachineDeclTemplate

	SharedObjectState

	readyToWorkCtl      smsync.BoolConditionalLink
	previousResultSaved smsync.BoolConditionalLink

	oldObject bool
}

func (s *SharedObjectState) SetObjectDescriptor(l smachine.Logger, newObjectDescriptor artifacts.ObjectDescriptor) {
	l.Trace("setting new object descriptor")
	s.ObjectLatestDescriptor = newObjectDescriptor
}

/* -------- Declaration ------------- */

func (sm *SMObject) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	injector.MustInject(&sm.artifactClient)
	injector.MustInject(&sm.sender)
	injector.MustInject(&sm.pulseSlot)
}

func (sm *SMObject) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

/* -------- Instance ------------- */

func (sm *SMObject) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *SMObject) sendPayloadToVirtual(ctx smachine.ExecutionContext, pl payload.Payload) {
	goCtx := ctx.GetContext()

	resultsMessage, err := payload.NewMessage(pl)
	if err == nil {
		objectReference := sm.ObjectReference

		sm.sender.PrepareNotify(ctx, func(svc s_sender.SenderService) {
			_, done := svc.SendRole(goCtx, resultsMessage, insolar.DynamicRoleVirtualExecutor, objectReference)
			done()
		}).DelayedSend()
	} else {
		logger := inslogger.FromContext(goCtx)
		logger.Error("Failed to serialize message: ", err.Error())
	}
}

func (sm *SMObject) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(sm.migrateSendStateBeforeExecution)

	sm.readyToWorkCtl = smsync.NewConditionalBool(false, "readyToWork")
	sm.ReadyToWork = sm.readyToWorkCtl.SyncLink()

	sm.PreviousExecutorFinished = smsync.NewConditionalBool(false, "PreviousExecutorFinished")
	sm.SemaphorePreviousExecutorFinished = sm.readyToWorkCtl.SyncLink()

	sm.previousResultSaved = smsync.NewConditionalBool(false, "previousResultSaved")
	sm.SemaphorePreviousResultSaved = sm.previousResultSaved.SyncLink()

	sm.ImmutableExecute = smsync.NewSemaphore(30, "immutable calls").SyncLink()
	sm.MutableExecute = smsync.NewSemaphore(1, "mutable calls").SyncLink() // TODO here we need an ORDERED queue

	sdl := ctx.Share(&sm.SharedObjectState, 0)
	if !ctx.Publish(sm.ObjectReference, sdl) {
		return ctx.Stop()
	}
	return ctx.Jump(sm.stepCheckPreviousExecutor)
}

func (sm *SMObject) stepCheckPreviousExecutor(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch sm.PreviousExecutorState {
	case payload.PreviousExecutorUnknown:
		return ctx.Jump(sm.stepGetPendingsInformation)
	case payload.PreviousExecutorProbablyExecutes, payload.PreviousExecutorExecutes:
		// we should wait here till PendingFinished/ExecutorResults came, retry and then change state to PreviousExecutorFinished
		if ctx.AcquireForThisStep(sm.SemaphorePreviousExecutorFinished).IsNotPassed() {
			return ctx.Sleep().ThenRepeat()
		}

		// we shouldn't be here
		// if we came to that place - means MutableRequestsAreReady, but PreviousExecutor still executes)
		panic("unreachable")

	case payload.PreviousExecutorFinished:
		return ctx.Jump(sm.stepGetLatestValidatedState)
	default:
		panic("unreachable")
	}
}

func (sm *SMObject) stepGetPendingsInformation(ctx smachine.ExecutionContext) smachine.StateUpdate {
	goCtx := ctx.GetContext()

	objectReference := sm.ObjectReference

	return sm.artifactClient.PrepareAsync(ctx, func(svc s_artifact.ArtifactClientService) smachine.AsyncResultFunc {
		logger := inslogger.FromContext(goCtx)

		hasAbandonedRequests, err := svc.HasPendings(goCtx, objectReference)
		if err != nil {
			logger.Error("couldn't check pending state: ", err.Error())
		}

		var newState payload.PreviousExecutorState
		if hasAbandonedRequests {
			logger.Debug("ledger has requests older than one pulse")
			newState = payload.PreviousExecutorProbablyExecutes
		} else {
			logger.Debug("no requests on ledger older than one pulse")
			newState = payload.PreviousExecutorFinished
		}

		return func(ctx smachine.AsyncResultContext) {
			if sm.PreviousExecutorState == payload.PreviousExecutorUnknown {
				sm.PreviousExecutorState = newState
			} else {
				logger.Info("state already changed, ignoring check")
			}
		}
	}).DelayedStart().Sleep().ThenJump(sm.stepCheckPreviousExecutor)
}

// we should check here only if not creation request here
func (sm *SMObject) stepGetLatestValidatedState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(sm.migrateStop)

	goCtx := ctx.GetContext()
	objectReference := sm.ObjectReference

	if !sm.oldObject {
		sm.oldObject = true
		sm.IsReadyToWork = true
		return ctx.Jump(sm.stateGotLatestValidatedStatePrototypeAndCode)
	}

	return sm.artifactClient.PrepareAsync(ctx, func(svc s_artifact.ArtifactClientService) smachine.AsyncResultFunc {
		var err error

		failCallback := func(ctx smachine.AsyncResultContext) {
			inslogger.FromContext(goCtx).Error("Failed to obtain objects: ", err)
			sm.externalError = err
		}

		inslogger.FromContext(goCtx).Debugf("NewObject fetched %s", objectReference.String())
		objectDescriptor, err := svc.GetObject(goCtx, objectReference, nil)
		if err != nil {
			err = errors.Wrap(err, "Failed to obtain object descriptor")
			return failCallback
		}

		return func(ctx smachine.AsyncResultContext) {
			sm.SetObjectDescriptor(ctx.Log(), objectDescriptor)
			sm.IsReadyToWork = true
		}
	}).DelayedStart().Sleep().ThenJump(sm.stateGotLatestValidatedStatePrototypeAndCode)
}

func (sm *SMObject) stateGotLatestValidatedStatePrototypeAndCode(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.externalError != nil {
		ctx.Error(sm.externalError)
	} else if !sm.IsReadyToWork {
		return ctx.Sleep().ThenJump(sm.stateGotLatestValidatedStatePrototypeAndCode)
	}

	ctx.ApplyAdjustment(sm.readyToWorkCtl.NewValue(true))

	return ctx.JumpExt(smachine.SlotStep{
		Transition: sm.waitForMigration,
		Migration:  sm.migrateSendStateAfterExecution,
	})
}

func (sm *SMObject) waitForMigration(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

// //////////////////////////////////////

func (sm *SMObject) migrateSendStateBeforeExecution(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(nil)

	return ctx.Jump(sm.stateSendStateBeforeExecution)
}

func (sm *SMObject) stateSendStateBeforeExecution(ctx smachine.ExecutionContext) smachine.StateUpdate {
	_, immutableLeft := sm.ImmutableExecute.GetCounts()
	_, mutableLeft := sm.MutableExecute.GetCounts()

	ledgerHasMoreRequests := immutableLeft+mutableLeft > 0

	var newState payload.PreviousExecutorState

	switch sm.PreviousExecutorState {
	case payload.PreviousExecutorUnknown:
		newState = payload.PreviousExecutorFinished
	case payload.PreviousExecutorProbablyExecutes:
		newState = payload.PreviousExecutorFinished
	case payload.PreviousExecutorExecutes:
		newState = payload.PreviousExecutorUnknown
	case payload.PreviousExecutorFinished:
		newState = payload.PreviousExecutorFinished
	default:
		panic("unreachable")
	}

	sm.sendPayloadToVirtual(ctx, &payload.ExecutorResults{
		ObjectReference:       sm.ObjectReference,
		LedgerHasMoreRequests: ledgerHasMoreRequests,
		State:                 newState,
	})

	return ctx.Stop()
}

// //////////////////////////////////////

func (sm *SMObject) migrateSendStateAfterExecution(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(nil)

	return ctx.Jump(sm.stateSendStateAfterExecution)
}

func (sm *SMObject) stateSendStateAfterExecution(ctx smachine.ExecutionContext) smachine.StateUpdate {
	immutableInProgress, immutableLeft := sm.ImmutableExecute.GetCounts()
	mutableInProgress, mutableLeft := sm.MutableExecute.GetCounts()

	ledgerHasMoreRequests := immutableLeft+mutableLeft > 0
	pendingCount := uint32(immutableInProgress + mutableInProgress)

	var newState payload.PreviousExecutorState

	switch sm.PreviousExecutorState {
	case payload.PreviousExecutorFinished:
		if pendingCount > 0 {
			newState = payload.PreviousExecutorExecutes
		} else {
			newState = payload.PreviousExecutorFinished
		}
	default:
		panic("unreachable")
	}

	if pendingCount > 0 || ledgerHasMoreRequests {
		sm.sendPayloadToVirtual(ctx, &payload.ExecutorResults{
			ObjectReference:       sm.ObjectReference,
			LedgerHasMoreRequests: ledgerHasMoreRequests,
			State:                 newState,
		})
	}

	return ctx.Jump(sm.stateWaitFinishExecutionAfterMigration)
}

func (sm *SMObject) stateWaitFinishExecutionAfterMigration(ctx smachine.ExecutionContext) smachine.StateUpdate {
	mc, _ := sm.MutableExecute.GetCounts()
	ic, _ := sm.ImmutableExecute.GetCounts()
	if mc > 0 || ic > 0 {
		return ctx.Poll().ThenRepeat()
	}

	sm.sendPayloadToVirtual(ctx, &payload.PendingFinished{
		ObjectRef: sm.ObjectReference,
	})

	return ctx.Stop()
}

func (sm *SMObject) migrateStop(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}
