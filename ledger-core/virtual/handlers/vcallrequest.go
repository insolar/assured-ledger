//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
)

type SMVCallRequest struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VCallRequest

	pulseSlot *conveyor.PulseSlot
}

/* -------- Declaration ------------- */

var dSMVCallRequestInstance smachine.StateMachineDeclaration = &dSMVCallRequest{}

type dSMVCallRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVCallRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVCallRequest)

	injector.MustInject(&s.pulseSlot)
}

func (*dSMVCallRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVCallRequest)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVCallRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVCallRequestInstance
}

func (s *SMVCallRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	if s.pulseSlot.State() != conveyor.Present {
		ctx.Log().Trace("stop processing VCallRequest since we are not in present pulse")
		return ctx.Stop()
	}
	ctx.SetDefaultMigration(s.migrationDefault)
	return ctx.Jump(s.stepExecute)
}

func (s *SMVCallRequest) migrationDefault(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.Log().Trace("stop processing VCallRequest since pulse was changed")
	return ctx.Stop()
}

func (s *SMVCallRequest) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &execute.SMExecute{Meta: s.Meta, Payload: s.Payload}
	})
}
