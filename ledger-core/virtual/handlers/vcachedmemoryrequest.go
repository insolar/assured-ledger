// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
//go:generate sm-uml-gen -f $GOFILE
package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

type SMVCachedMemoryRequest struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCachedMemoryRequest
}

/* -------- Declaration ------------- */
var dSMVCachedMemoryRequestInstance smachine.StateMachineDeclaration = &dSMVCachedMemoryRequest{}

type dSMVCachedMemoryRequest struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVCachedMemoryRequest) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ injector.DependencyInjector) {
}
func (*dSMVCachedMemoryRequest) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVCachedMemoryRequest)
	return s.Init
}

/* -------- Instance ------------- */
func (s *SMVCachedMemoryRequest) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVCachedMemoryRequestInstance
}
func (s *SMVCachedMemoryRequest) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
