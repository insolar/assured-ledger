// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate go run $GOPATH/src/github.com/insolar/assured-ledger/ledger-core/v2/scripts/gen_plantuml.go -f $GOFILE

package sm_request // nolint:golint

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type StateMachinePendingFinished struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.PendingFinished
}

/* -------- Declaration ------------- */

var declPendingFinished smachine.StateMachineDeclaration = &declarationPendingFinished{}

type declarationPendingFinished struct {
	smachine.StateMachineDeclTemplate
}

func (declarationPendingFinished) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachinePendingFinished)
	return s.Init
}

func (declarationPendingFinished) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	_ = sm.(*StateMachinePendingFinished)
}

/* -------- Instance ------------- */

func (s *StateMachinePendingFinished) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declPendingFinished
}

func (s *StateMachinePendingFinished) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Stop()
}
