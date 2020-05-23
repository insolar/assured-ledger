// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/convlog"
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
	return true
}

func (c ConveyorLogger) CreateAsyncLogger(ctx context.Context, _ *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return c.tracerCtx, c
}


func (c ConveyorLogger) LogInternal(data smachine.StepLoggerData, updateType string) {
	// TODO LogInternal
}

func (c ConveyorLogger) LogAdapter(data smachine.StepLoggerData, adapterID smachine.AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller) {
	// TODO LogAdapter
}

type LogStepMessage struct {
	*log.Msg

	Message   string
	Component string `txt:"sm"`
	TraceID   string `opt:""`

	MachineName interface{} `fmt:"%T"`
	MachineID   string
	SlotID      smachine.SlotID
	SlotStepNo  uint32
	CycleNo     uint32
	From        string
	To          string `opt:""`

	Error     string `opt:""`
	Backtrace string `opt:""`

	ExecutionTime  int64 `opt:""`
	InactivityTime int64 `opt:""`
}

func (c ConveyorLogger) LogEvent(data smachine.StepLoggerData, msg interface{}, fields []logfmt.LogFieldMarshaller) {
	dm := c.logger.FieldsOf(data)
	if dm == nil {
		panic(throw.IllegalState())
	}

	if len(fields) == 0 {
		c.logger.Errorm(msg, dm)
		return
	}
	f := make([]logfmt.LogFieldMarshaller, 0, len(fields) + 1)
	f[0] = dm
	c.logger.Errorm(msg, append(f, fields...)...)
}

func (c ConveyorLogger) LogUpdate(stepLoggerData smachine.StepLoggerData, stepLoggerUpdateData smachine.StepLoggerUpdateData) {
	special := ""

	switch stepLoggerData.EventType {
	case smachine.StepLoggerUpdate:
	case smachine.StepLoggerMigrate:
		special = "migrate "
	default:
		panic(throw.Impossible())
	}

	smachine.PrepareStepName(&stepLoggerData.CurrentStep)
	smachine.PrepareStepName(&stepLoggerUpdateData.NextStep)

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
		var st throw.StackTrace
		err, st = throw.ErrorWithTopOrMinimizedStack(stepLoggerData.Error, convlog.StackMinimizePackage, true)
		if st != nil {
			backtrace = st.StackTraceAsText()
		}
	}

	if err != "" {
		fmt.Println(err)
	}

	msg := LogStepMessage{
		Message: special + stepLoggerUpdateData.UpdateType + suffix,

		MachineName: stepLoggerData.Declaration,
		MachineID:   stepLoggerData.StepNo.MachineID(),
		CycleNo:     stepLoggerData.CycleNo,
		SlotID:      stepLoggerData.StepNo.SlotID(),
		SlotStepNo:  stepLoggerData.StepNo.StepNo(),

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

	if stepLoggerData.Error == nil {
		c.logger.Debug(msg)
	} else {
		c.logger.Error(msg)
	}
}
