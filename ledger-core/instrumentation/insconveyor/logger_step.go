// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insconveyor

import (
	"context"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StepLogger = ConveyorLogger{}

type ConveyorLogger struct {
	tracerCtx context.Context
	tracerID  smachine.TracerID

	logger log.Logger
}

func (c ConveyorLogger) GetTracerID() smachine.TracerID {
	return c.tracerID
}

func (c ConveyorLogger) GetLoggerContext() context.Context {
	return c.tracerCtx
}

func (c ConveyorLogger) CanLogEvent(eventType smachine.StepLoggerEvent, stepLevel smachine.StepLogLevel) bool {
	level, _ := convlog.MapLogEvent(eventType, stepLevel)
	return c.logger.Is(level)
}

func (c ConveyorLogger) CreateAsyncLogger(_ context.Context, _ *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return c.tracerCtx, c
}

func (c ConveyorLogger) LogInternal(data smachine.StepLoggerData, updateType string) {
	level, _ := convlog.MapBasicLogLevel(data)

	convlog.PrepareStepName(&data.CurrentStep)
	logMsg := c.logPrepare(data)
	msgText := data.FormatForLog("internal " + updateType)

	if err := data.Error; err != nil {
		c.logger.Eventm(level, throw.W(err, msgText, logMsg))
	} else {
		logMsg.Message = msgText
		c.logger.Eventm(level, logMsg)
	}
}

func (ConveyorLogger) CanLogTestEvent() bool {
	return false
}

func (ConveyorLogger) LogTestEvent(smachine.StepLoggerData, interface{}) {}

func (c ConveyorLogger) LogEvent(data smachine.StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller) {
	level, _ := convlog.MapCustomLogLevel(data)

	convlog.PrepareStepName(&data.CurrentStep)
	logMsg := c.logPrepare(data)
	msgText := data.FormatForLog("custom")

	if err, ok := customEvent.(error); ok && err == nil {
		c.logger.Eventm(level, throw.W(err, msgText, logMsg), fields...)
	} else if err = data.Error; err != nil {
		c.logger.Eventm(level, throw.W(err, msgText, logMsg), fields...)
	} else {
		dm := global.Logger().FieldsOf(logMsg)
		logMsg.Message = msgText
		c.logger.Eventm(level, customEvent, logfmt.JoinFields(fields, dm)...)
	}
}

func (c ConveyorLogger) LogAdapter(data smachine.StepLoggerData, adapterID smachine.AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller) {
	level, _ := convlog.MapBasicLogLevel(data)

	convlog.PrepareStepName(&data.CurrentStep)
	logMsg := c.logPrepare(data)
	logMsg.Message = data.FormatForLog("")

	logMsg.AdapterID = adapterID
	logMsg.CallID = callID

	if err := data.Error; err != nil {
		c.logger.Eventm(level, throw.WithDetails(err, logMsg), fields...)
	} else {
		c.logger.Eventm(level, logMsg, fields...)
	}
}

func (c ConveyorLogger) LogUpdate(data smachine.StepLoggerData, updateData smachine.StepLoggerUpdateData) {
	if _, ok := data.Declaration.(*conveyor.PulseSlotMachine); ok {
		return
	}

	level, _ := convlog.MapBasicLogLevel(data)

	convlog.PrepareStepName(&data.CurrentStep)
	convlog.PrepareStepName(&updateData.NextStep)
	logMsg := c.logPrepare(data)
	logMsg.Message = data.FormatForLog(updateData.UpdateType)

	logMsg.NextStep = updateData.NextStep.GetStepName()

	if updateData.ActivityNano > 0 {
		logMsg.ExecutionTime = updateData.ActivityNano.Nanoseconds()
	}
	if updateData.InactivityNano > 0 {
		logMsg.InactivityTime = updateData.InactivityNano.Nanoseconds()
	}

	if err := data.Error; err != nil {
		c.logger.Eventm(level, throw.WithDetails(err, logMsg))
	} else {
		c.logger.Eventm(level, logMsg)
	}
}

func (c ConveyorLogger) logPrepare(data smachine.StepLoggerData) LogStepInfo {
	return LogStepInfo{
		MachineID:   data.StepNo.MachineID(),
		CycleNo:     data.CycleNo,
		Declaration: data.Declaration,
		SlotID:      data.StepNo.SlotID(),
		SlotStepNo:  data.StepNo.StepNo(),

		CurrentStep: data.CurrentStep.GetStepName(),
	}
}

type LogStepInfo struct {
	Message   string

	MachineID   string
	CycleNo     uint32
	Declaration smachine.StateMachineHelper `fmt:"%T"`
	SlotID      smachine.SlotID
	SlotStepNo  uint32

	CurrentStep string
	NextStep    string `opt:""`

	ExecutionTime  int64 `opt:""`
	InactivityTime int64 `opt:""`

	AdapterID smachine.AdapterID `opt:""`
	CallID    uint64 `opt:""`
}

// DisableLogStepInfoMarshaller is for benchmarking
var DisableLogStepInfoMarshaller bool

func (v LogStepInfo) GetLogObjectMarshaller() logfmt.LogObjectMarshaller {
	if DisableLogStepInfoMarshaller {
		return nil
	}
	return v
}

func (v LogStepInfo) MarshalLogObject(w logfmt.LogObjectWriter, _ logfmt.LogObjectMetricCollector) (msg string, defMsg bool) {
	w.AddStrField("Component", "sm", logfmt.LogFieldFormat{Kind: reflect.String})
	w.AddStrField("MachineID", v.MachineID, logfmt.LogFieldFormat{Kind: reflect.String})

	w.AddUintField("CycleNo", uint64(v.CycleNo), logfmt.LogFieldFormat{Kind: reflect.Uint32})

	if v.Declaration != nil {
		w.AddStrField("Declaration", reflect.TypeOf(v.Declaration).String(), logfmt.LogFieldFormat{Kind: reflect.String})
	}

	w.AddUintField("SlotID", uint64(v.SlotID), logfmt.LogFieldFormat{Kind: reflect.Uint32})
	w.AddUintField("SlotStepNo", uint64(v.SlotStepNo), logfmt.LogFieldFormat{Kind: reflect.Uint32})

	w.AddStrField("CurrentStep", v.CurrentStep, logfmt.LogFieldFormat{Kind: reflect.String})
	if v.NextStep != "" {
		w.AddStrField("NextStep", v.NextStep, logfmt.LogFieldFormat{Kind: reflect.String})
	}

	if v.ExecutionTime != 0 {
		w.AddIntField("ExecutionTime", v.ExecutionTime, logfmt.LogFieldFormat{Kind: reflect.Int64})
	}
	if v.InactivityTime != 0 {
		w.AddIntField("InactivityTime", v.InactivityTime, logfmt.LogFieldFormat{Kind: reflect.Int64})
	}

	if v.AdapterID != "" {
		w.AddStrField("AdapterID", string(v.AdapterID), logfmt.LogFieldFormat{Kind: reflect.String})
		w.AddUintField("CallID", v.CallID, logfmt.LogFieldFormat{Kind: reflect.Uint64})
	}

	return v.Message, false
}

