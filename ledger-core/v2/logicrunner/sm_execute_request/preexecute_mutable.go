// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate go run $GOPATH/src/github.com/insolar/assured-ledger/ledger-core/v2/scripts/gen_plantuml.go -f $GOFILE

package sm_execute_request // nolint:golint

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type SMPreExecuteMutable struct {
	smachine.StateMachineDeclTemplate

	*ExecuteIncomingCommon
}

/* -------- Declaration ------------- */

func (s *SMPreExecuteMutable) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return s.Init
}

func (s *SMPreExecuteMutable) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
}

/* -------- Instance ------------- */

func (s *SMPreExecuteMutable) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *SMPreExecuteMutable) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepWaitObjectIsReady)
}

func (s *SMPreExecuteMutable) stepWaitObjectIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.RequestInfo.Result != nil {
		return ctx.Jump(s.stepSpawnExecution)
	}

	if !ctx.AcquireForThisStep(s.objectInfo.ReadyToWork) {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepTakeLock)
}

func (s *SMPreExecuteMutable) stepTakeLock(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !ctx.AcquireAndRelease(s.objectInfo.MutableExecute) {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepSpawnExecution)
}

func (s *SMPreExecuteMutable) stepSpawnExecution(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &SMExecute{
			ExecuteIncomingCommon: s.ExecuteIncomingCommon,
		}
	})
}
