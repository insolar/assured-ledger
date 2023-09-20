package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SMLRegisterRequest struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.LRegisterRequest
}

/* -------- Declaration ------------- */

var dSMLRegisterRequestInstance smachine.StateMachineDeclaration = &dSMLRegisterRequest{}

type dSMLRegisterRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMLRegisterRequest) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
}

func (*dSMLRegisterRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMLRegisterRequest)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMLRegisterRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMLRegisterRequestInstance
}

func (s *SMLRegisterRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	// THIS IS STUB, LATER IT'LL BE REMOVED
	return ctx.Error(throw.IllegalState())
}
