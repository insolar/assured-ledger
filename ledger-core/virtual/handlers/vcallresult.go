//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SMVCallResult struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VCallResult
}

/* -------- Declaration ------------- */

var dSMVCallResultInstance smachine.StateMachineDeclaration = &dSMVCallResult{}

type dSMVCallResult struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVCallResult) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ injector.DependencyInjector) {
}

func (*dSMVCallResult) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVCallResult)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVCallResult) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVCallResultInstance
}

func (s *SMVCallResult) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepProcess)
}

func (s *SMVCallResult) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.Payload.CallType != rms.CallTypeMethod && s.Payload.CallType != rms.CallTypeConstructor {
		panic(throw.IllegalValue())
	}

	link, bargeInCallback := ctx.GetPublishedGlobalAliasAndBargeIn(s.Payload.CallOutgoing)
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
