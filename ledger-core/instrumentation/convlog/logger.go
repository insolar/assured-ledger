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
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	mLog         = "LOG"
	mError       = "ERR"
	mWarning     = "WRN"
	mTrace       = "TRC"
	mActiveTrace = "TRA"
	mFatal       = "FTL"
)

var _ smachine.SlotMachineLogger = MachineLogger{}

type MachineLogger struct {
	EchoToGlobal bool
}

func (l MachineLogger) LogMachineInternal(data smachine.SlotMachineData, msg string) {
	if data.Error != nil {
		l.logMachineCritical(data, msg)
		return
	}

	printMachineData(data, mLog, msg)

	if l.EchoToGlobal {
		global.Infom(throw.E(msg, data))
	}
}

func (l MachineLogger) LogMachineCritical(data smachine.SlotMachineData, msg string) {
	l.logMachineCritical(data, msg)
}

func (MachineLogger) logMachineCritical(data smachine.SlotMachineData, msg string) {
	printMachineData(data, mError, msg)

	if data.Error == nil {
		data.Error = throw.New("error is missing")
	}

	global.Errorm(throw.W(data.Error, msg, data))
}

func (MachineLogger) LogStopping(*smachine.SlotMachine) {}

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

func (v conveyorStepLogger) LogUpdate(data smachine.StepLoggerData, upd smachine.StepLoggerUpdateData) {
	level, levelName := MapBasicLogLevel(data)

	PrepareStepName(&data.CurrentStep)
	PrepareStepName(&upd.NextStep)

	durations := ""
	switch {
	case upd.InactivityNano > 0 && upd.ActivityNano > 0:
		durations = fmt.Sprintf(" timing=%s/%s", upd.InactivityNano, upd.ActivityNano)
	case upd.InactivityNano > 0:
		durations = fmt.Sprintf(" timing=%s/_", upd.InactivityNano)
	case upd.ActivityNano > 0:
		durations = fmt.Sprintf(" timing=_/%s", upd.ActivityNano)
	}

	msg := data.FormatForLog(upd.UpdateType)
	printStepData(data, levelName, msg,
		fmt.Sprintf("next=%v%s payload=%T tracer=%v", upd.NextStep.GetStepName(), durations, v.sm, v.tracer))

	if !v.echoToGlobal && level < log.ErrorLevel {
		return
	}

	if err := data.Error; err != nil {
		data.Error = nil
		global.Eventm(level, throw.W(err, msg, data, upd))
	} else {
		global.Event(level, msg, data, upd)
	}
}

func (v conveyorStepLogger) LogInternal(data smachine.StepLoggerData, updateType string) {
	level, levelName := MapBasicLogLevel(data)

	PrepareStepName(&data.CurrentStep)

	msgText := data.FormatForLog("internal " + updateType)
	printStepData(data, levelName, msgText,
		fmt.Sprintf("payload=%T tracer=%v", v.sm, v.tracer))

	if !v.echoToGlobal && level < log.ErrorLevel {
		return
	}

	if err := data.Error; err != nil {
		data.Error = nil
		global.Eventm(level, throw.W(err, msgText, data))
	} else {
		global.Event(level, msgText, data)
	}
}

func (v conveyorStepLogger) CanLogTestEvent() bool {
	return true
}

func (v conveyorStepLogger) LogTestEvent(data smachine.StepLoggerData, customEvent interface{}) {
	_, levelName := MapCustomLogLevel(data)

	PrepareStepName(&data.CurrentStep)

	printStepData(data, levelName, fmt.Sprintf("custom_test  %+v", customEvent),
		fmt.Sprintf("payload=%T tracer=%v", v.sm, v.tracer))
}

func (v conveyorStepLogger) LogEvent(data smachine.StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller) {
	level, levelName := MapCustomLogLevel(data)

	PrepareStepName(&data.CurrentStep)

	printStepData(data, levelName, fmt.Sprintf("custom %+v", customEvent),
		fmt.Sprintf("payload=%T tracer=%v", v.sm, v.tracer))

	if !v.echoToGlobal && level < log.ErrorLevel {
		return
	}

	if err, ok := customEvent.(error); ok && err == nil {
		msgText := data.FormatForLog("custom")
		global.Eventm(level, throw.W(err, msgText, data), fields...)
	} else if err = data.Error; err != nil {
		data.Error = nil
		msgText := data.FormatForLog("custom")
		global.Eventm(level, throw.W(err, msgText, data), fields...)
	} else {
		dm := global.Logger().FieldsOf(data)
		global.Eventm(level, customEvent, logfmt.JoinFields(fields, dm)...)
	}

	if data.EventType == smachine.StepLoggerFatal {
		panic("os.Exit(1)")
	}
}

func (v conveyorStepLogger) LogAdapter(data smachine.StepLoggerData, adapterID smachine.AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller) {
	level, levelName := MapBasicLogLevel(data)

	msg := data.FormatForLog("")

	printStepData(data, levelName, msg,
		fmt.Sprintf("adapter=%v/0x%x payload=%T tracer=%v", adapterID, callID, v.sm, v.tracer))

	if !v.echoToGlobal && level < log.ErrorLevel {
		return
	}

	extra := struct {
		AdapterID smachine.AdapterID
		CallID    uint64
	}{adapterID, callID}

	if err := data.Error; err != nil {
		data.Error = nil
		global.Eventm(level, throw.W(err, msg, data, extra), fields...)
	} else {
		dm := global.Logger().FieldsOf(data)
		em := global.Logger().FieldsOf(extra)
		global.Eventm(level, msg, logfmt.JoinFields(fields, dm, em)...)
	}
}

const StackMinimizePackage = "github.com/insolar/assured-ledger/ledger-core/conveyor"

func formatErrorStack(err error) string {
	if err == nil {
		return ""
	}
	st := throw.DeepestStackTraceOf(err)
	st = throw.MinimizeStackTrace(st, StackMinimizePackage, true)
	return throw.JoinStackText(" "+logoutput.ErrorMsgFieldName+"="+err.Error(), st)
}

func printTimestamp() string {
	return time.Now().Format(inslogger.TimestampFormat)
}

func printMachineData(data smachine.SlotMachineData, level, msg string) {
	fmt.Printf("%s %s %s[%3d] [MACHINE] %03d @ %03d: internal %s%s\n", printTimestamp(),
		level, data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(), msg, formatErrorStack(data.Error))
}

func printStepData(data smachine.StepLoggerData, level, msg, extra string) {
	switch {
	case data.Error != nil:
		if level == "" {
			level = mError
		}
	case level == "":
		level = mLog
	}

	fmt.Printf("%s %s %s[%3d] %03d @ %03d: %s current=%v %s%s\n", printTimestamp(),
		level, data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(),
		msg,
		data.CurrentStep.GetStepName(), extra,
		formatErrorStack(data.Error))
}

func MapCustomLogLevel(data smachine.StepLoggerData) (log.Level, string) {
	return _mapLogLevel(data.EventType, data)
}

func MapBasicLogLevel(data smachine.StepLoggerData) (log.Level, string) {
	if data.EventType >= smachine.StepLoggerTrace {
		panic(throw.IllegalValue())
	}
	return _mapLogLevel(data.EventType, data)
}

func _mapLogLevel(eventType smachine.StepLoggerEvent, data smachine.StepLoggerData) (log.Level, string) {
	if data.Error != nil {
		var replace smachine.StepLoggerEvent

		switch data.Flags.ErrorFlags() {
		case smachine.StepLoggerUpdateErrorMuted:
			replace = smachine.StepLoggerWarn
		case smachine.StepLoggerUpdateErrorRecovered:
			replace = smachine.StepLoggerTrace
		default:
			return MapLogEvent(eventType, smachine.StepLogLevelError)
		}

		if eventType > replace && eventType.IsEvent() {
			eventType = replace
		}
	}

	if data.IsElevated() {
		return MapLogEvent(eventType, smachine.StepLogLevelElevated)
	}
	return MapLogEvent(eventType, smachine.StepLogLevelDefault)
}

func MapLogEvent(eventType smachine.StepLoggerEvent, stepLevel smachine.StepLogLevel) (log.Level, string) {
	switch eventType {
	case smachine.StepLoggerError:
		return log.ErrorLevel, mError
	case smachine.StepLoggerFatal:
		return log.FatalLevel, mFatal
	}

	switch stepLevel {
	case smachine.StepLogLevelDefault:
		switch eventType {
		case smachine.StepLoggerTrace:
			return log.DebugLevel, mTrace
		case smachine.StepLoggerActiveTrace:
			return log.InfoLevel, mActiveTrace
		case smachine.StepLoggerWarn:
			return log.WarnLevel, mWarning
		case smachine.StepLoggerUpdate, smachine.StepLoggerMigrate, smachine.StepLoggerInternal, smachine.StepLoggerAdapterCall:
			return log.DebugLevel, mLog
		}
	case smachine.StepLogLevelError:
		return log.ErrorLevel, mError
	default:
		if eventType < smachine.StepLoggerWarn {
			return log.InfoLevel, mLog
		}
		if eventType == smachine.StepLoggerWarn {
			return log.WarnLevel, mWarning
		}
	}

	return log.InfoLevel, fmt.Sprintf("U%d", eventType)
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

func PrepareStepName(sd *smachine.StepDeclaration) {
	if !sd.IsNameless() {
		return
	}
	sd.Name = GetStepName(sd.Transition)
}
