//go:generate sm-uml-gen -f $GOFILE

package example

import (
	"fmt"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type StateMachine1 struct {
	serviceA *ServiceAdapterA
	catalogC CatalogC

	mutex   smachine.SyncLink
	testKey longbits.ByteString
	result  string
	count   int
	start   time.Time
}

/* -------- Declaration ------------- */

var declarationStateMachine1 smachine.StateMachineDeclaration = &stateMachine1Declaration{}

type stateMachine1Declaration struct {
	smachine.StateMachineDeclTemplate
}

func (stateMachine1Declaration) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*StateMachine1)
	injector.MustInject(&s.serviceA)
	injector.MustInject(&s.catalogC)
}

func (stateMachine1Declaration) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachine1)
	return s.Init
}

/* -------- Instance ------------- */

func (s *StateMachine1) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declarationStateMachine1
}

func (s *StateMachine1) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	s.testKey = longbits.WrapStr("testObjectID")
	s.start = time.Now()

	//fmt.Printf("init: %v %v\n", ctx.SlotLink(), time.Now())
	return ctx.Jump(s.State1)
}

func (s *StateMachine1) State1(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch s.catalogC.GetOrCreate(ctx, s.testKey).
		Prepare(func(state *CustomSharedState) {
			if state.GetKey() != s.testKey {
				panic("wtf?")
			}
			before := state.Text
			state.Counter++
			state.Text = fmt.Sprintf("last-%v", ctx.SlotLink())
			fmt.Printf("shared: accessed=%d %v -> %v\n", state.Counter, before, state.Text)
			s.mutex = state.Mutex
		}).
		TryUse(ctx).GetDecision() {
	case smachine.Passed:
		return ctx.Jump(s.State2)
	case smachine.NotPassed:
		return ctx.Yield().ThenRepeat()
	default:
		return ctx.Jump(s.State5)
	}
}

func (s *StateMachine1) State2(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.serviceA.PrepareAsync(ctx, func(svc ServiceA) smachine.AsyncResultFunc {
		result := svc.DoSomething("y")

		//if result != "" {
		//	panic("test panic")
		//}

		return func(ctx smachine.AsyncResultContext) {
			fmt.Printf("state1 async: %v %v\n", ctx.SlotLink(), result)
			s.result = result
			ctx.WakeUp()
		}
	}).Start() // result of async will only be applied _after_ leaving this state

	s.serviceA.PrepareSync(ctx, func(svc ServiceA) {
		s.result = svc.DoSomething("x")
	}).Call()

	fmt.Printf("state1: %v %v\n", ctx.SlotLink(), s.result)

	if ctx.SlotLink().SlotID()&1 == 0 {
		return ctx.Jump(s.State3)
	}
	return ctx.Jump(s.State2a)
}

func (s *StateMachine1) State2a(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.WaitAnyUntil(s.start.Add(200 * time.Millisecond)).ThenRepeatOrJump(s.State3)
}

func (s *StateMachine1) State3(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if ctx.Acquire(s.mutex).IsNotPassed() {
		//if ctx.AcquireForThisStep(s.mutex).IsNotPassed() {
		//active, inactive := s.mutex.GetCounts()
		//fmt.Println("Mutex queue: ", active, inactive)
		s.mutex.DebugPrint(10)
		return ctx.Sleep().ThenRepeat()
	}

	s.count++
	if s.count < 5 {
		//return ctx.Yield().ThenRepeat()
		//return ctx.Repeat(10)
		return ctx.Poll().ThenRepeat()
	}

	return ctx.Jump(s.State4)
}

func (s *StateMachine1) State4(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if ctx.GetPendingCallCount() > 0 {
		return ctx.WaitAny().ThenRepeat()
	}

	if ctx.SlotLink().SlotID()&1 == 1 {
		ctx.NewChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &StateMachine1{}
		})
		ctx.NewChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &StateMachine1{}
		})
		ctx.NewChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &StateMachine1{}
		})
	}

	fmt.Printf("wait: %d %v result:%v\n", ctx.SlotLink().SlotID(), time.Now(), s.result)
	s.count = 0

	//return ctx.Jump(s.State5)
	return ctx.WaitAnyUntil(time.Now().Add(time.Second)).ThenJump(s.State5)
}

func (s *StateMachine1) State5(ctx smachine.ExecutionContext) smachine.StateUpdate {
	id := int(ctx.SlotLink().SlotID())

	if id&2 == 0 {
		subroutineSM := &StateMachine3{count: id}
		return ctx.CallSubroutine(subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
			param := ctx.EventParam()
			direct := subroutineSM.count
			fmt.Printf("return: %d %v '%v'\n", ctx.SlotLink().SlotID(), direct, param)
			return ctx.Jump(s.State6)
		})
	}

	ctx.NewChild(func(ctx2 smachine.ConstructionContext) smachine.StateMachine {
		subroutineSM := &StateMachine3{count: id}
		ctx2.SetTerminationCallback(ctx, func(param interface{}, err error) smachine.AsyncResultFunc {
			direct := subroutineSM.count
			return func(ctx smachine.AsyncResultContext) {
				fmt.Printf("return: %d %v '%v'\n", ctx.SlotLink().SlotID(), direct, param)
				ctx.WakeUp()
			}
		})
		return subroutineSM
	})

	return ctx.Sleep().ThenJump(s.State6)
}

func (s *StateMachine1) State6(ctx smachine.ExecutionContext) smachine.StateUpdate {
	fmt.Printf("stop: %d %v\n", ctx.SlotLink().SlotID(), time.Now())
	return ctx.Stop()
}

//func (s *StateMachine1) State50(ctx smachine.ExecutionContext) smachine.StateUpdate {
//
//	////s.serviceA.
//	//result := ""
//	s.serviceA.PrepareSync(ctx, func(svc ServiceA) {
//		result = svc.DoSomething("x")
//	}).Call()
//
//	return s.serviceA.PrepareAsync(ctx, func(svc ServiceA) smachine.AsyncResultFunc {
//		asyncResult := svc.DoSomething("x")
//
//		return func(ctx smachine.AsyncResultContext) {
//			s.result = asyncResult
//			ctx.WakeUp()
//		}
//	}).DelayedStart().ThenJump(s.State5)
//}
//
//func (s *StateMachine1) State60(ctx smachine.ExecutionContext) smachine.StateUpdate {
//
//	//mx := s.mutexB.JoinMutex(ctx, "mutex Key", mutexCallback)
//	//if !mx.IsHolder() {
//	//	return ctx.Idle()
//	//}
//	//
//	//mb.Broadcast(info)
//	//// do something
//
//	return ctx.Jump(nil)
//}
