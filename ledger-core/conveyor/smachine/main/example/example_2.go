//go:generate sm-uml-gen -f $GOFILE

package example

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type StateMachine2 struct {
	smachine.StateMachineDeclTemplate
	count int
	Yield bool
}

func (StateMachine2) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachine2)
	return s.Init
}

var IterationCount uint64
var Limiter = smsync.NewFixedSemaphore(1000, "global")

/* -------- Instance ------------- */

func (s *StateMachine2) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *StateMachine2) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.State0)
}

func (s *StateMachine2) State0(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !ctx.AcquireForThisStep(Limiter) {
		return ctx.Sleep().ThenRepeat()
	}
	IterationCount++
	s.count++
	if s.count < 1000 {
		if s.Yield {
			return ctx.Yield().ThenRepeat()
		}
		return ctx.Repeat(math.MaxInt32)
	}
	s.count = 0
	return ctx.Yield().ThenJump(s.State0) //forces release of mutex
}
