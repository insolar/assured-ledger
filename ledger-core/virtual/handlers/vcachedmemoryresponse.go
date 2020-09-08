// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
//go:generate sm-uml-gen -f $GOFILE
package handlers

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

type SMVCachedMemoryResponse struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VCachedMemoryResponse
}

/* -------- Declaration ------------- */
var dSMVCachedMemoryResponseInstance smachine.StateMachineDeclaration = &dSMVCachedMemoryResponse{}

type dSMVCachedMemoryResponse struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVCachedMemoryResponse) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ injector.DependencyInjector) {
}
func (*dSMVCachedMemoryResponse) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVCachedMemoryResponse)
	return s.Init
}

/* -------- Instance ------------- */
func (s *SMVCachedMemoryResponse) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVCachedMemoryResponseInstance
}
func (s *SMVCachedMemoryResponse) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
