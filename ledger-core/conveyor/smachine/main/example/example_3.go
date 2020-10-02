// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package example

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type StateMachine3 struct {
	smachine.StateMachineDeclTemplate
	count int
}

func (StateMachine3) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	return sm.(*StateMachine3).Init
}

/* -------- Instance ------------- */

func (s *StateMachine3) GetSubroutineInitState(smachine.SubroutineStartContext) smachine.InitFunc {
	return s.Init
}

func (s *StateMachine3) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *StateMachine3) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.State0)
}

func (s *StateMachine3) State0(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.count++
	return ctx.Jump(s.State1)
}

func (s *StateMachine3) State1(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.count&1 == 1 {
		ctx.SetTerminationResult(s.count)
		return ctx.Stop()
	}
	// return ctx.Stop()
	panic("stop by panic")
}
