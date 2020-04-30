// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
	"time"

	conveyor2 "github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
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

	factoryFn := func(pn pulse.Number, _ pulse.Range, v conveyor2.InputEvent) (pulse.Number, smachine.CreateFunc) {
		return 0, func(ctx smachine.ConstructionContext) smachine.StateMachine {
			sm := &AppEventSM{eventValue: v, pn: pn}
			return sm
		}
	}
	machineConfig.SlotMachineLogger = conveyorSlotMachineLogger{}

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
		noError(conveyor.CommitPulseChange(pd.AsRange(), time.Now()))
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

func (sm *AppEventSM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
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
	if su, wait := ctx.WaitAnyUntil(sm.expiry).ThenRepeatOrElse(); wait {
		ctx.Log().Trace(fmt.Sprint("wait: ", sm.eventValue, sm.pn))
		return su
	}
	ctx.Log().Trace(fmt.Sprint("stop: ", sm.eventValue, sm.pn, "late=", time.Since(sm.expiry)))
	return ctx.Stop()
}

/* ===================================================== */

type conveyorSlotMachineLogger struct {
}

func (conveyorSlotMachineLogger) LogMachineInternal(data smachine.SlotMachineData, msg string) {
	fmt.Printf("[MACHINE][LOG] %s[%3d]: %03d @ %03d: internal %s err=%v\n", data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(), msg, data.Error)
}

func (conveyorSlotMachineLogger) LogMachineCritical(data smachine.SlotMachineData, msg string) {
	fmt.Printf("[MACHINE][ERR] %s[%3d]: %03d @ %03d: internal %s err=%v\n", data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(), msg, data.Error)
}

func (conveyorSlotMachineLogger) CreateStepLogger(ctx context.Context, sm smachine.StateMachine, tracer smachine.TracerID) smachine.StepLogger {
	return conveyorStepLogger{ctx, sm, tracer}
}

type conveyorStepLogger struct {
	ctx    context.Context
	sm     smachine.StateMachine
	tracer smachine.TracerID
}

func (conveyorStepLogger) CanLogEvent(eventType smachine.StepLoggerEvent, stepLevel smachine.StepLogLevel) bool {
	return true
}

func (v conveyorStepLogger) GetTracerID() smachine.TracerID {
	return v.tracer
}

func (v conveyorStepLogger) CreateAsyncLogger(ctx context.Context, _ *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return ctx, v
}

func getStepName(step interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(step).Pointer()).Name()
	if lastIndex := strings.LastIndex(fullName, "."); lastIndex >= 0 {
		fullName = fullName[lastIndex+1:]
	}
	if lastIndex := strings.LastIndex(fullName, "-"); lastIndex >= 0 {
		fullName = fullName[:lastIndex]
	}

	return fullName
}

func (v conveyorStepLogger) prepareStepName(sd *smachine.StepDeclaration) {
	if !sd.IsNameless() {
		return
	}
	sd.Name = getStepName(sd.Transition)
}

func (v conveyorStepLogger) LogUpdate(data smachine.StepLoggerData, upd smachine.StepLoggerUpdateData) {
	special := ""

	switch data.EventType {
	case smachine.StepLoggerUpdate:
	case smachine.StepLoggerMigrate:
		special = "migrate "
	default:
		panic("illegal value")
	}

	v.prepareStepName(&data.CurrentStep)
	v.prepareStepName(&upd.NextStep)

	detached := ""
	if data.Flags&smachine.StepLoggerDetached != 0 {
		detached = "(detached)"
	}

	durations := ""
	if upd.InactivityNano > 0 || upd.ActivityNano > 0 {
		durations = fmt.Sprintf(" timing=%s/%s", upd.InactivityNano, upd.ActivityNano)
	}

	if data.Error == nil {
		fmt.Printf("[LOG] %s[%3d]: %03d @ %03d: %s%s%s%s current=%v next=%v payload=%T tracer=%v\n", data.StepNo.MachineID(), data.CycleNo,
			data.StepNo.SlotID(), data.StepNo.StepNo(),
			special, upd.UpdateType, detached, durations,
			data.CurrentStep.GetStepName(), upd.NextStep.GetStepName(), v.sm, v.tracer)
		return
	}

	errSpecial := ""
	switch data.Flags & smachine.StepLoggerErrorMask {
	case smachine.StepLoggerUpdateErrorMuted:
		errSpecial = "muted "
	case smachine.StepLoggerUpdateErrorRecovered:
		errSpecial = "recovered "
	case smachine.StepLoggerUpdateErrorRecoveryDenied:
		errSpecial = "recover-denied "
	}

	fmt.Printf("[ERR] %s[%3d]: %03d @ %03d: %s%s%s%s current=%v next=%v payload=%T tracer=%v err=%v\n", data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(),
		special, errSpecial, upd.UpdateType, detached, data.CurrentStep.GetStepName(), upd.NextStep.GetStepName(), v.sm, v.tracer, data.Error)
}

func (v conveyorStepLogger) LogInternal(data smachine.StepLoggerData, updateType string) {
	v.prepareStepName(&data.CurrentStep)

	if data.Error == nil {
		fmt.Printf("[LOG] %s[%3d]: %03d @ %03d: internal %s current=%v payload=%T tracer=%v\n", data.StepNo.MachineID(), data.CycleNo,
			data.StepNo.SlotID(), data.StepNo.StepNo(),
			updateType, data.CurrentStep.GetStepName(), v.sm, v.tracer)
	} else {
		fmt.Printf("[ERR] %s[%3d]: %03d @ %03d: internal %s current=%v payload=%T tracer=%v err=%v\n", data.StepNo.MachineID(), data.CycleNo,
			data.StepNo.SlotID(), data.StepNo.StepNo(),
			updateType, data.CurrentStep.GetStepName(), v.sm, v.tracer, data.Error)
	}
}

func (v conveyorStepLogger) LogEvent(data smachine.StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller) {
	special := ""

	v.prepareStepName(&data.CurrentStep)

	switch data.EventType {
	case smachine.StepLoggerTrace:
		special = "TRC"
	case smachine.StepLoggerActiveTrace:
		special = "TRA"
	case smachine.StepLoggerWarn:
		special = "WRN"
	case smachine.StepLoggerError:
		special = "ERR"
	case smachine.StepLoggerFatal:
		special = "FTL"
	default:
		fmt.Printf("[U%d] %s[%3d]: %03d @ %03d: current=%v payload=%T tracer=%v custom=%v\n", data.EventType, data.StepNo.MachineID(), data.CycleNo,
			data.StepNo.SlotID(), data.StepNo.StepNo(),
			data.CurrentStep.GetStepName(), v.sm, v.tracer, customEvent)
		return
	}
	fmt.Printf("[%s] %s[%3d]: %03d @ %03d: current=%v payload=%T tracer=%v custom=%v\n", special, data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(),
		data.CurrentStep.GetStepName(), v.sm, v.tracer, customEvent)

	if data.EventType == smachine.StepLoggerFatal {
		panic("os.Exit(1)")
	}
}

func (v conveyorStepLogger) LogAdapter(data smachine.StepLoggerData, adapterID smachine.AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller) {
	//case smachine.StepLoggerAdapterCall:
	s := "?"
	switch data.Flags & smachine.StepLoggerAdapterMask {
	case smachine.StepLoggerAdapterSyncCall:
		s = "sync-call"
	case smachine.StepLoggerAdapterAsyncCall:
		s = "async-call"
	case smachine.StepLoggerAdapterAsyncResult:
		s = "async-result"
	case smachine.StepLoggerAdapterAsyncCancel:
		s = "async-cancel"
	}
	fmt.Printf("[ADP] %s %s[%3d]: %03d @ %03d: current=%v payload=%T tracer=%v adapter=%v/%v\n", s, data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(),
		data.CurrentStep.GetStepName(), v.sm, v.tracer, adapterID, callID)
}
