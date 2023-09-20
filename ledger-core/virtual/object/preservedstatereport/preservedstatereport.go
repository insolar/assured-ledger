package preservedstatereport

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SMPreservedStateReport struct {
	Reference reference.Global

	Report rms.VStateReport

	smachine.StateMachineDeclTemplate

	// dependencies
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
}

/* -------- Declaration ------------- */

func (*SMPreservedStateReport) InjectDependencies(stateMachine smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := stateMachine.(*SMPreservedStateReport)
	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.pulseSlot)
}

func (*SMPreservedStateReport) GetInitStateFor(stateMachine smachine.StateMachine) smachine.InitFunc {
	return stateMachine.(*SMPreservedStateReport).Init
}

/* -------- Instance ------------- */

func (sm *SMPreservedStateReport) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *SMPreservedStateReport) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (sm *SMPreservedStateReport) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if sm.Reference.IsZero() {
		panic(throw.IllegalState())
	}

	return ctx.Jump(sm.stepSendVStateReport)
}

func (sm *SMPreservedStateReport) stepSendVStateReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		currentPulseNumber = sm.pulseSlot.CurrentPulseNumber()
	)

	msg := sm.Report

	msg.AsOf = sm.pulseSlot.PulseData().PulseNumber

	sm.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, affinity.DynamicRoleVirtualExecutor, sm.Reference, currentPulseNumber)
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send state", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	// after report is sent we can stop
	ctx.SetDefaultMigration(sm.migrationDefault)

	return ctx.Jump(sm.stepWaitIndefinitely)
}

func (sm *SMPreservedStateReport) stepWaitIndefinitely(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Sleep().ThenRepeat()
}
