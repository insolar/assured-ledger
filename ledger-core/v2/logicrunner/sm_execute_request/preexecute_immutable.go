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

type SMPreExecuteImmutable struct {
	smachine.StateMachineDeclTemplate

	*ExecuteIncomingCommon
}

/* -------- Declaration ------------- */

func (s *SMPreExecuteImmutable) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return s.Init
}

func (s *SMPreExecuteImmutable) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
}

/* -------- Instance ------------- */

func (s *SMPreExecuteImmutable) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *SMPreExecuteImmutable) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepTakeLock)
}

func (s *SMPreExecuteImmutable) stepTakeLock(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.RequestInfo.Result != nil {
		return ctx.Jump(s.stepSpawnExecution)
	}

	if !ctx.AcquireAndRelease(s.objectInfo.ImmutableExecute) {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepSpawnExecution)
}

func (s *SMPreExecuteImmutable) stepSpawnExecution(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &SMExecute{
			ExecuteIncomingCommon: s.ExecuteIncomingCommon,
		}
	})
}
