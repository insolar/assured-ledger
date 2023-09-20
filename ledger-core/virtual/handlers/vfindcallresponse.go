//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
)

type SMVFindCallResponse struct {
	// input arguments
	Meta      *rms.Meta
	Payload   *rms.VFindCallResponse
	pulseSlot *conveyor.PulseSlot
}

/* -------- Declaration ------------- */

var dSMVFindCallResponseInstance smachine.StateMachineDeclaration = &dSMVFindCallResponse{}

type dSMVFindCallResponse struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVFindCallResponse) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVFindCallResponse)
	injector.MustInject(&s.pulseSlot)
}

func (*dSMVFindCallResponse) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVFindCallResponse)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVFindCallResponse) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVFindCallResponseInstance
}

func (s *SMVFindCallResponse) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if s.pulseSlot.State() != conveyor.Present {
		ctx.Log().Warn("stop processing VFindCallResponse since we are not in present pulse")
		return ctx.Stop()
	}
	ctx.SetDefaultMigration(s.migrationDefault)
	return ctx.Jump(s.stepProcess)
}

func (s *SMVFindCallResponse) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.Log().Trace("stop processing VFindCallResponse since pulse was changed")
	return ctx.Stop()
}

func (s *SMVFindCallResponse) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	key := execute.DeduplicationBargeInKey{
		LookAt:   s.Payload.LookedAt,
		Callee:   s.Payload.Callee.GetValue(),
		Outgoing: s.Payload.Outgoing.GetValue(),
	}

	link, bargeInCallback := ctx.GetPublishedGlobalAliasAndBargeIn(key)
	if link.IsZero() {
		return ctx.Error(throw.E("no one is waiting"))
	}
	if bargeInCallback == nil {
		return ctx.Error(throw.Impossible())
	}

	done := bargeInCallback.CallWithParam(s.Payload)
	if !done {
		return ctx.Error(throw.E("no one is waiting anymore"))
	}

	return ctx.Stop()
}
