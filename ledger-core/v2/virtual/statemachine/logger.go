// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

type ConveyorLogger struct {
	smachine.StepLoggerStub

	logger log.Logger
}

func (c ConveyorLogger) CanLogEvent(eventType smachine.StepLoggerEvent, stepLevel smachine.StepLogLevel) bool {
	return true
}

func (c ConveyorLogger) CreateAsyncLogger(ctx context.Context, _ *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return ctx, c
}

type LogStepMessage struct {
	*log.Msg

	Message   string
	Component string `txt:"sm"`
	TraceID   string `opt:""`

	MachineName interface{} `fmt:"%T"`
	MachineID   string
	SlotStep    string
	From        string
	To          string `opt:""`

	Error     string `opt:""`
	Backtrace string `opt:""`

	ExecutionTime  int64 `opt:""`
	InactivityTime int64 `opt:""`
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

func prepareStepName(sd *smachine.StepDeclaration) {
	if !sd.IsNameless() {
		return
	}
	sd.Name = getStepName(sd.Transition)
}

func (c ConveyorLogger) LogEvent(data smachine.StepLoggerData, msg interface{}, fields []logfmt.LogFieldMarshaller) {
	c.logger.Errorm(msg, fields...)
}

func (c ConveyorLogger) LogUpdate(stepLoggerData smachine.StepLoggerData, stepLoggerUpdateData smachine.StepLoggerUpdateData) {
	special := ""

	switch stepLoggerData.EventType {
	case smachine.StepLoggerUpdate:
	case smachine.StepLoggerMigrate:
		special = "migrate "
	default:
		panic("illegal value")
	}

	prepareStepName(&stepLoggerData.CurrentStep)
	prepareStepName(&stepLoggerUpdateData.NextStep)

	suffix := ""
	if stepLoggerData.Flags&smachine.StepLoggerDetached != 0 {
		suffix = " (detached)"
	}

	if _, ok := stepLoggerData.Declaration.(*conveyor.PulseSlotMachine); ok {
		return
	}

	var (
		backtrace string
		err       string
	)
	if stepLoggerData.Error != nil {
		if slotPanicError, ok := stepLoggerData.Error.(smachine.SlotPanicError); ok {
			backtrace = string(slotPanicError.Stack)
		}
		err = stepLoggerData.Error.Error()
	}

	msg := LogStepMessage{
		Message: special + stepLoggerUpdateData.UpdateType + suffix,

		MachineName: stepLoggerData.Declaration,
		MachineID:   fmt.Sprintf("%s[%3d]", stepLoggerData.StepNo.MachineId(), stepLoggerData.CycleNo),
		SlotStep:    fmt.Sprintf("%03d @ %03d", stepLoggerData.StepNo.StepNo(), stepLoggerData.StepNo.StepNo()),

		From: stepLoggerData.CurrentStep.GetStepName(),
		To:   stepLoggerUpdateData.NextStep.GetStepName(),

		Error:     err,
		Backtrace: backtrace,
	}

	if stepLoggerUpdateData.ActivityNano > 0 {
		msg.ExecutionTime = stepLoggerUpdateData.ActivityNano.Nanoseconds()
	}
	if stepLoggerUpdateData.InactivityNano > 0 {
		msg.InactivityTime = stepLoggerUpdateData.InactivityNano.Nanoseconds()
	}

	c.logger.Error(msg)
}

type ConveyorLoggerFactory struct {
}

func (c ConveyorLoggerFactory) CreateStepLogger(ctx context.Context, _ smachine.StateMachine, traceID smachine.TracerId) smachine.StepLogger {
	_, logger := inslogger.WithTraceField(context.Background(), traceID)
	return &ConveyorLogger{
		StepLoggerStub: smachine.StepLoggerStub{TracerId: traceID},
		logger:         logger,
	}
}

type LogInternal struct {
	*log.Msg `txt:"internal"`

	Message   string `fmt:"internal - %s"`
	Component string `txt:"sm"`

	MachineID string
	SlotStep  string
	Error     error  `opt:""`
	Backtrace string `opt:""`
}

func (ConveyorLoggerFactory) LogMachineInternal(slotMachineData smachine.SlotMachineData, msg string) {
	backtrace := ""
	if slotMachineData.Error != nil {
		if slotPanicError, ok := slotMachineData.Error.(smachine.SlotPanicError); ok {
			backtrace = string(slotPanicError.Stack)
		}
	}
	global.Logger().Error(LogInternal{
		Message: msg,

		MachineID: fmt.Sprintf("%s[%3d]", slotMachineData.StepNo.MachineId(), slotMachineData.CycleNo),
		SlotStep:  fmt.Sprintf("%03d @ %03d", slotMachineData.StepNo.StepNo(), slotMachineData.StepNo.StepNo()),
		Error:     slotMachineData.Error,
		Backtrace: backtrace,
	})
}

type LogCritical struct {
	*log.Msg `txt:"internal"`

	Message   string `fmt:"internal critical - %s"`
	Component string `txt:"sm"`

	MachineID string
	SlotStep  string
	Error     error  `opt:""`
	Backtrace string `opt:""`
}

func (ConveyorLoggerFactory) LogMachineCritical(slotMachineData smachine.SlotMachineData, msg string) {
	backtrace := ""
	if slotMachineData.Error != nil {
		if slotPanicError, ok := slotMachineData.Error.(smachine.SlotPanicError); ok {
			backtrace = string(slotPanicError.Stack)
		}
	}
	global.Logger().Error(LogCritical{
		Message: msg,

		MachineID: fmt.Sprintf("%s[%3d]", slotMachineData.StepNo.MachineId(), slotMachineData.CycleNo),
		SlotStep:  fmt.Sprintf("%03d @ %03d", slotMachineData.StepNo.StepNo(), slotMachineData.StepNo.StepNo()),
		Error:     slotMachineData.Error,
		Backtrace: backtrace,
	})
}
