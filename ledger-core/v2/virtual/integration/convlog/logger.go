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

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)


var _ smachine.SlotMachineLogger = MachineLogger{}
type MachineLogger struct {
	EchoToGlobal bool
}

func (l MachineLogger) LogMachineInternal(data smachine.SlotMachineData, msg string) {
	if data.Error != nil {
		l.LogMachineCritical(data, msg)
		return
	}

	printMachineData(data, "LOG", msg)

	if l.EchoToGlobal {
		global.Infom(throw.E(msg, data))
	}
}

func (MachineLogger) LogMachineCritical(data smachine.SlotMachineData, msg string) {
	printMachineData(data, "ERR", msg)

	if data.Error == nil {
		data.Error = throw.New("error is missing")
	}

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

func getStepName(step interface{}) string {
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
	sd.Name = getStepName(sd.Transition)
}

func (v conveyorStepLogger) LogUpdate(data smachine.StepLoggerData, upd smachine.StepLoggerUpdateData) {

	v.prepareStepName(&data.CurrentStep)
	v.prepareStepName(&upd.NextStep)

	durations := ""
	if upd.InactivityNano > 0 || upd.ActivityNano > 0 {
		durations = fmt.Sprintf(" timing=%s/%s", upd.InactivityNano, upd.ActivityNano)
	}

	msg := printStepData(data, "", upd.UpdateType,
		fmt.Sprintf("next=%v%s payload=%T tracer=%v", upd.NextStep.GetStepName(), durations, v.sm, v.tracer))

	switch {
	case data.Error != nil:
		global.Errorm(throw.W(data.Error, msg, data,
			struct{ CurrentStep, NextStep string }{
				data.CurrentStep.GetStepName(), upd.NextStep.GetStepName(),
			}))

	case v.echoToGlobal:
		global.Infom(throw.E(msg, data,
			struct{ CurrentStep, NextStep string }{
				data.CurrentStep.GetStepName(), upd.NextStep.GetStepName(),
			}))
	}
}

func (v conveyorStepLogger) LogInternal(data smachine.StepLoggerData, updateType string) {

	v.prepareStepName(&data.CurrentStep)

	msg := printStepData(data, "", "internal " + updateType,
		fmt.Sprintf("payload=%T tracer=%v", v.sm, v.tracer))

	switch {
	case data.Error != nil:
		global.Errorm(throw.W(data.Error, msg, data,
			struct{ CurrentStep string }{
				data.CurrentStep.GetStepName(),
			}))

	case v.echoToGlobal:
		global.Infom(throw.E(msg, data,
			struct{ CurrentStep string }{
				data.CurrentStep.GetStepName(),
			}))
	}
}

func (v conveyorStepLogger) LogEvent(data smachine.StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller) {
	levelName := ""

	v.prepareStepName(&data.CurrentStep)

	var level log.Level
	switch data.EventType {
	case smachine.StepLoggerTrace:
		levelName = "TRC"
		level = log.DebugLevel
	case smachine.StepLoggerActiveTrace:
		levelName = "TRA"
		level = log.InfoLevel
	case smachine.StepLoggerWarn:
		levelName = "WRN"
		level = log.WarnLevel
	case smachine.StepLoggerError:
		levelName = "ERR"
		level = log.ErrorLevel
	case smachine.StepLoggerFatal:
		levelName = "FTL"
		level = log.FatalLevel
	default:
		levelName = fmt.Sprintf("U%d", data.EventType)
		level = log.WarnLevel
	}

	printStepData(data, levelName, fmt.Sprintf("custom %v", customEvent),
		fmt.Sprintf("payload=%T tracer=%v", v.sm, v.tracer))

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
	msg := "adapter-unknown-event"
	switch data.Flags & smachine.StepLoggerAdapterMask {
	case smachine.StepLoggerAdapterSyncCall:
		msg = "sync-call"
	case smachine.StepLoggerAdapterAsyncCall:
		msg = "async-call"
	case smachine.StepLoggerAdapterNotifyCall:
		msg = "notify-call"
	case smachine.StepLoggerAdapterAsyncResult:
		msg = "async-result"
	case smachine.StepLoggerAdapterAsyncCancel:
		msg = "async-cancel"
	}

	msg = printStepData(data, "", msg,
		fmt.Sprintf("adapter=%v/0x%x payload=%T tracer=%v",
			adapterID, callID,
			v.sm, v.tracer))

	switch {
	case data.Error != nil:
		global.Errorm(throw.W(data.Error, msg, data,
			struct{ CurrentStep string }{
				data.CurrentStep.GetStepName(),
			}), fields...)

	case v.echoToGlobal:
		global.Infom(throw.E(msg, data,
			struct{ CurrentStep string }{
				data.CurrentStep.GetStepName(),
			}), fields...)
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

func printTimestamp() string {
	return time.Now().Format(inslogger.TimestampFormat)
}

func printMachineData(data smachine.SlotMachineData, level, msg string) {


	fmt.Printf("%s %s %s[%3d] [MACHINE] %03d @ %03d: internal %s%s\n", printTimestamp(),
		level, data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(), msg, formatErrorStack(data.Error))
}

func printStepData(data smachine.StepLoggerData, level, msg, extra string) string {
	special := ""
	detached := ""

	if data.Flags&smachine.StepLoggerDetached != 0 {
		detached = " (detached)"
	}

	switch data.EventType {
	case smachine.StepLoggerUpdate:
	case smachine.StepLoggerMigrate:
		special = "migrate "
	case smachine.StepLoggerAdapterCall:
		detached = ""
	}


	errSpecial := ""
	switch {
	case data.Error != nil:
		if level == "" {
			level = "ERR"
		}
		switch data.Flags & smachine.StepLoggerErrorMask {
		case smachine.StepLoggerUpdateErrorMuted:
			errSpecial = "muted "
		case smachine.StepLoggerUpdateErrorRecovered:
			errSpecial = "recovered "
		case smachine.StepLoggerUpdateErrorRecoveryDenied:
			errSpecial = "recover-denied "
		}
	case level == "":
		level = "LOG"
	}

	msg = errSpecial+special+msg+detached

	fmt.Printf("%s %s %s[%3d] %03d @ %03d: %s current=%v %s%s\n", printTimestamp(),
		level, data.StepNo.MachineID(), data.CycleNo,
		data.StepNo.SlotID(), data.StepNo.StepNo(),
		msg,
		data.CurrentStep.GetStepName(), extra, formatErrorStack(data.Error))

	return msg
}

