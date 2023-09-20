//go:generate sm-uml-gen -f $GOFILE

package main

import (
	"context"
	"fmt"
	"math"
	"time"

	conveyor2 "github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func noError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	machineConfig := smachine.SlotMachineConfig{
		SlotPageSize:      1000,
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		ScanCountLimit:    100000,
		SlotAliasRegistry: &conveyor2.GlobalAliases{},
	}

	factoryFn := func(_ context.Context, v conveyor2.InputEvent, ic conveyor2.InputContext) (conveyor2.InputSetup, error) {
		return conveyor2.InputSetup{
			CreateFn: func(ctx smachine.ConstructionContext) smachine.StateMachine {
				sm := &AppEventSM{eventValue: v, pn: ic.PulseNumber}
				return sm
			},
		}, nil
	}
	machineConfig.SlotMachineLogger = convlog.MachineLogger{}

	conveyor := conveyor2.NewPulseConveyor(context.Background(), conveyor2.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        1 * time.Second,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, factoryFn, nil)

	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})

	conveyor.StartWorker(nil, nil)

	eventCount := 0
	noError(conveyor.AddInput(context.Background(), pd.PulseNumber, fmt.Sprintf("event-%d-pre", eventCount)))

	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 100; i++ {
		fmt.Println(">>>================================== ", pd, " ====================================")
		if i != 0 {
			noError(conveyor.PreparePulseChange(nil))
		}
		time.Sleep(100 * time.Millisecond)
		noError(conveyor.CommitPulseChange(pd.AsRange(), time.Now(), nil))
		fmt.Println("<<<================================== ", pd, " ====================================")
		pd = pd.CreateNextPulsarPulse(10, func() longbits.Bits256 {
			return longbits.Bits256{}
		})
		time.Sleep(10 * time.Millisecond)

		eventToCall := ""
		if eventCount < math.MaxInt32 {
			eventCount++
			noError(conveyor.AddInput(context.Background(), pd.NextPulseNumber(), fmt.Sprintf("event-%d-future", eventCount)))
			eventCount++
			noError(conveyor.AddInput(context.Background(), pd.PrevPulseNumber(), fmt.Sprintf("event-%d-past", eventCount)))

			for j := 0; j < 1; j++ {
				eventCount++
				eventToCall = fmt.Sprintf("event-%d-present", eventCount)
				noError(conveyor.AddInput(context.Background(), pd.PulseNumber, fmt.Sprintf("event-%d-present", eventCount)))
			}
		}

		time.Sleep(time.Second)

		if eventToCall != "" {
			link, _ := conveyor.GetPublishedGlobalAliasAndBargeIn(eventToCall)
			//require.False(t, link.IsEmpty())
			smachine.ScheduleCallTo(link, func(callContext smachine.MachineCallContext) {
				fmt.Println("Global call: ", callContext.GetMachineID(), link)
			}, false)
		}

	}
	time.Sleep(100 * time.Millisecond)
	fmt.Println("======================")
	conveyor.StopNoWait()
	time.Sleep(time.Hour)
}

/* ===================================================== */

type AppEventSM struct {
	smachine.StateMachineDeclTemplate

	pulseSlot *conveyor2.PulseSlot

	pn         pulse.Number
	eventValue interface{}
	expiry     time.Time
}

func (sm *AppEventSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return sm
}

func (sm *AppEventSM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&sm.pulseSlot)
}

func (sm *AppEventSM) GetInitStateFor(machine smachine.StateMachine) smachine.InitFunc {
	if sm != machine {
		panic("illegal value")
	}
	fmt.Println("new: ", sm.eventValue, sm.pn)
	return sm.stepInit
}

func (sm *AppEventSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.Log().Trace(fmt.Sprint("init: ", sm.eventValue, sm.pn))
	ctx.SetDefaultMigration(sm.migrateToClosing)
	ctx.PublishGlobalAlias(sm.eventValue)

	return ctx.Jump(sm.stepRun)
}

func (sm *AppEventSM) stepRun(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.Log().Trace(fmt.Sprint("run: ", sm.eventValue, sm.pn, sm.pulseSlot.PulseData()))
	ctx.LogAsync().Trace(fmt.Sprint("(via async log) run: ", sm.eventValue, sm.pn, sm.pulseSlot.PulseData()))
	return ctx.Poll().ThenRepeat()
}

func (sm *AppEventSM) migrateToClosing(ctx smachine.MigrationContext) smachine.StateUpdate {
	sm.expiry = time.Now().Add(2600 * time.Millisecond)
	ctx.Log().Trace(fmt.Sprint("migrate: ", sm.eventValue, sm.pn))
	ctx.SetDefaultMigration(nil)
	return ctx.Jump(sm.stepClosingRun)
}

func (sm *AppEventSM) stepClosingRun(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if wait := ctx.WaitAnyUntil(sm.expiry); wait.GetDecision().IsNotPassed() {
		ctx.Log().Trace(fmt.Sprint("wait: ", sm.eventValue, sm.pn))
		return wait.ThenRepeat()
	}
	ctx.Log().Trace(fmt.Sprint("stop: ", sm.eventValue, sm.pn, "late=", time.Since(sm.expiry)))
	return ctx.Stop()
}
