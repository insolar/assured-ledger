// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
//go:generate sm-uml-gen -f $GOFILE
package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

type SMVObjectValidationReport struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VObjectValidationReport
}

/* -------- Declaration ------------- */

var dSMVObjectValidationReportInstance smachine.StateMachineDeclaration = &dSMVObjectValidationReport{}

type dSMVObjectValidationReport struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVObjectValidationReport) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ injector.DependencyInjector) {
}

func (*dSMVObjectValidationReport) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVObjectValidationReport)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVObjectValidationReport) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVObjectValidationReportInstance
}

func (s *SMVObjectValidationReport) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
