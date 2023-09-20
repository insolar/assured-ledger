//go:generate sm-uml-gen -f $GOFILE

package example

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func NewVMObjectSM(objKey longbits.ByteString) *vmObjectSM { // nolint:golint
	return &vmObjectSM{SharedObjectState: SharedObjectState{ObjectInfo: ObjectInfo{ObjKey: objKey}}}
}

type vmObjectSM struct {
	smachine.StateMachineDeclTemplate

	SharedObjectState
	readyToWorkCtl smsync.BoolConditionalLink
}

type ObjectInfo struct {
	ObjKey        longbits.ByteString
	IsReadyToWork bool

	ArtifactClient *ArtifactClientServiceAdapter
	ContractRunner *ContractRunnerServiceAdapter

	ObjectLatestValidState ArtifactBinary
	ObjectLatestValidCode  ArtifactBinary

	ImmutableExecute smachine.SyncLink
	MutableExecute   smachine.SyncLink
}

type SharedObjectState struct {
	SemaReadyToWork smachine.SyncLink
	ObjectInfo
}

//////////////////////////

func (sm *vmObjectSM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&sm.ArtifactClient)
	injector.MustInject(&sm.ContractRunner)
}

func (sm *vmObjectSM) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return sm.Init
}

func (sm *vmObjectSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *vmObjectSM) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(sm.migrateStop)

	sm.readyToWorkCtl = smsync.NewConditionalBool(false, "readyToWork")
	sm.SemaReadyToWork = sm.readyToWorkCtl.SyncLink()
	sm.ImmutableExecute = smsync.NewFixedSemaphore(5, "immutable calls")
	sm.MutableExecute = smsync.NewFixedSemaphore(1, "mutable calls") // TODO here we need an ORDERED queue

	sdl := ctx.Share(&sm.SharedObjectState, 0)
	if !ctx.Publish(sm.ObjKey, sdl) {
		return ctx.Stop()
	}
	return ctx.Jump(sm.stateGetLatestValidatedState)
}

func (sm *vmObjectSM) stateGetLatestValidatedState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	sm.ArtifactClient.PrepareAsync(ctx, func(svc ArtifactClientService) smachine.AsyncResultFunc {
		stateObj, codeObj := svc.GetLatestValidatedStateAndCode()

		return func(ctx smachine.AsyncResultContext) {
			sm.ObjectLatestValidState = stateObj
			sm.ObjectLatestValidCode = codeObj
			sm.IsReadyToWork = true
			ctx.WakeUp()
		}
	})

	return ctx.Sleep().ThenJump(sm.stateGotLatestValidatedState)
}

func (sm *vmObjectSM) stateGotLatestValidatedState(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if sm.ObjectLatestValidState == nil {
		return ctx.Sleep().ThenRepeat()
	}
	ctx.ApplyAdjustment(sm.readyToWorkCtl.NewValue(true))

	return ctx.JumpExt(smachine.SlotStep{Transition: sm.waitForMigration, Migration: sm.migrateSendState})
}

func (sm *vmObjectSM) waitForMigration(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}

func (sm *vmObjectSM) migrateStop(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (sm *vmObjectSM) migrateSendState(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(nil)
	return ctx.Jump(sm.stateCompleteExecution)
}

func (sm *vmObjectSM) stateCompleteExecution(ctx smachine.ExecutionContext) smachine.StateUpdate {
	/*  TODO
	Steps:
	1. Send last state to next executor
	2. Send transcript to validators
	*/
	return ctx.Stop()
}
