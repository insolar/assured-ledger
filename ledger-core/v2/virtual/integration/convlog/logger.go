// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package convlog

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type MachineLogger struct {
	EchoToGlobal bool
}

func (l MachineLogger) LogMachineInternal(data smachine.SlotMachineData, msg string) {
	fmt.Printf("[MACHINE][LOG] %s[%3d]: %03d @ %03d: internal %s%s\n", data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(), msg, formatErrorStack(data.Error))
	if l.EchoToGlobal {
		global.Infom(throw.E(msg, data))
	}
}

func (MachineLogger) LogMachineCritical(data smachine.SlotMachineData, msg string) {
	fmt.Printf("[MACHINE][ERR] %s[%3d]: %03d @ %03d: internal %s%s\n", data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(), msg, formatErrorStack(data.Error))
	global.Errorm(throw.W(data.Error, msg, data))
}

func (l MachineLogger) CreateStepLogger(ctx context.Context, sm smachine.StateMachine, tracer smachine.TracerID) smachine.StepLogger {
	ctxWithTrace, _ := inslogger.WithTraceField(ctx, tracer)
	return conveyorStepLogger{ctxWithTrace, sm, tracer, l.EchoToGlobal}
}

type conveyorStepLogger struct {
	ctxWithTrace context.Context
	sm           smachine.StateMachine
	tracer       smachine.TracerID
	echoToGlobal bool
}

func (conveyorStepLogger) CanLogEvent(_ smachine.StepLoggerEvent, _ smachine.StepLogLevel) bool {
	return true
}

func (v conveyorStepLogger) GetTracerID() smachine.TracerID {
	return v.tracer
}

func (v conveyorStepLogger) GetLoggerContext() context.Context {
	return v.ctxWithTrace
}

func (v conveyorStepLogger) CreateAsyncLogger(_ context.Context, _ *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return v.ctxWithTrace, v
}

func GetStepName(step interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(step).Pointer()).Name()
	if lastIndex := strings.LastIndex(fullName, "/"); lastIndex >= 0 {
		fullName = fullName[lastIndex+1:]
	}
	if firstIndex := strings.Index(fullName, "."); firstIndex >= 0 {
		fullName = fullName[firstIndex+1:]
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
	sd.Name = GetStepName(sd.Transition)
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

		if v.echoToGlobal {
			global.Infom(throw.E(special+upd.UpdateType+detached, data,
				struct{ CurrentStep, NextStep string }{
					data.CurrentStep.GetStepName(), upd.NextStep.GetStepName(),
				}))
		}
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

	fmt.Printf("[ERR] %s[%3d]: %03d @ %03d: %s%s%s%s current=%v next=%v payload=%T tracer=%v%s\n", data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(),
		special, errSpecial, upd.UpdateType, detached, data.CurrentStep.GetStepName(), upd.NextStep.GetStepName(), v.sm, v.tracer,
		formatErrorStack(data.Error))

	global.Errorm(throw.W(data.Error, special+errSpecial+upd.UpdateType+detached, data,
		struct{ CurrentStep, NextStep string }{
			data.CurrentStep.GetStepName(), upd.NextStep.GetStepName(),
		}))
}

func (v conveyorStepLogger) LogInternal(data smachine.StepLoggerData, updateType string) {
	v.prepareStepName(&data.CurrentStep)

	if data.Error == nil {
		fmt.Printf("[LOG] %s[%3d]: %03d @ %03d: internal %s current=%v payload=%T tracer=%v\n", data.StepNo.MachineID(), data.CycleNo,
			data.StepNo.SlotID(), data.StepNo.StepNo(),
			updateType, data.CurrentStep.GetStepName(), v.sm, v.tracer)

		return
	}

	fmt.Printf("[ERR] %s[%3d]: %03d @ %03d: internal %s current=%v payload=%T tracer=%v%s\n", data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(),
		updateType, data.CurrentStep.GetStepName(), v.sm, v.tracer, formatErrorStack(data.Error))

	global.Errorm(throw.W(data.Error, updateType, data,
		struct{ CurrentStep string }{data.CurrentStep.GetStepName()}))
}

func (v conveyorStepLogger) LogEvent(data smachine.StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller) {
	special := ""

	v.prepareStepName(&data.CurrentStep)

	var level log.Level
	switch data.EventType {
	case smachine.StepLoggerTrace:
		special = "TRC"
		level = log.DebugLevel
	case smachine.StepLoggerActiveTrace:
		special = "TRA"
		level = log.InfoLevel
	case smachine.StepLoggerWarn:
		special = "WRN"
		level = log.WarnLevel
	case smachine.StepLoggerError:
		special = "ERR"
		level = log.ErrorLevel
	case smachine.StepLoggerFatal:
		special = "FTL"
		level = log.FatalLevel
	default:
		special = fmt.Sprintf("U%d", data.EventType)
		level = log.WarnLevel
	}
	fmt.Printf("[%s] %s[%3d]: %03d @ %03d: current=%v payload=%T tracer=%v custom=%v\n", special, data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(),
		data.CurrentStep.GetStepName(), v.sm, v.tracer, customEvent)

	if !v.echoToGlobal && level < log.ErrorLevel {
		return
	}

	err := data.Error
	if e, ok := customEvent.(error); ok && err == nil {
		err = e
	}

	if err != nil {
		global.Eventm(level,
			throw.WithDetails(err, data, struct {
				CurrentStep string
				CustomEvent interface{}
			}{data.CurrentStep.GetStepName(), customEvent}),
			fields...)
	} else {
		global.Eventm(level, customEvent, fields...)
	}

	if data.EventType == smachine.StepLoggerFatal {
		panic("os.Exit(1)")
	}
}

func (v conveyorStepLogger) LogAdapter(data smachine.StepLoggerData, adapterID smachine.AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller) {
	// case smachine.StepLoggerAdapterCall:
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

	if v.echoToGlobal {
		global.Infom(struct{ msg, CurrentStep string }{s, data.CurrentStep.GetStepName()})
	}
}

const StackMinimizePackage = "github.com/insolar/assured-ledger/ledger-core/v2/conveyor"

func formatErrorStack(err error) string {
	if err == nil {
		return ""
	}
	st := throw.DeepestStackTraceOf(err)
	if st == nil {
		return " err=" + err.Error()
	}
	st = throw.MinimizeStackTrace(st, StackMinimizePackage, true)
	return throw.JoinStackText(" err="+err.Error(), st)
}
