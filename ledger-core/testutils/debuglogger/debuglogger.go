// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package debuglogger

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StepLogger = &DebugStepLogger{}

type UpdateEvent struct {
	notEmpty    bool
	SM          smachine.StateMachine
	Data        smachine.StepLoggerData
	Update      smachine.StepLoggerUpdateData
	CustomEvent interface{}
	AdapterID   smachine.AdapterID
	CallID      uint64
}

func (e UpdateEvent) IsEmpty() bool {
	return !e.notEmpty
}

type DebugStepLogger struct {
	smachine.StepLogger

	sm          smachine.StateMachine
	events      *updateChan
	continueSig synckit.SignalChannel
}

func (c DebugStepLogger) CanLogEvent(smachine.StepLoggerEvent, smachine.StepLogLevel) bool {
	return true
}

func (c DebugStepLogger) CreateAsyncLogger(context.Context, *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return c.GetLoggerContext(), c
}

func (c DebugStepLogger) LogInternal(data smachine.StepLoggerData, updateType string) {
	c.StepLogger.LogInternal(data, updateType)

	c.events.send(UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     data,
		Update:   smachine.StepLoggerUpdateData{UpdateType: updateType},
	})

	<-c.continueSig
}

func (c DebugStepLogger) LogUpdate(data smachine.StepLoggerData, update smachine.StepLoggerUpdateData) {
	c.StepLogger.LogUpdate(data, update)

	c.events.send(UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     data,
		Update:   update,
	})

	<-c.continueSig
}

func (c DebugStepLogger) CanLogTestEvent() bool {
	return true
}

func (c DebugStepLogger) LogTestEvent(data smachine.StepLoggerData, customEvent interface{}) {
	if c.StepLogger.CanLogTestEvent() {
		c.StepLogger.LogTestEvent(data, customEvent)
	}

	c.events.send(UpdateEvent{
		notEmpty:    true,
		SM:          c.sm,
		Data:        data,
		CustomEvent: customEvent,
	})

	<-c.continueSig
}

func (c DebugStepLogger) LogEvent(data smachine.StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller) {
	c.StepLogger.LogEvent(data, customEvent, fields)

	c.events.send(UpdateEvent{
		notEmpty:    true,
		SM:          c.sm,
		Data:        data,
		CustomEvent: customEvent,
	})

	<-c.continueSig
}

func (c DebugStepLogger) LogAdapter(data smachine.StepLoggerData, adapterID smachine.AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller) {
	c.StepLogger.LogAdapter(data, adapterID, callID, fields)

	c.events.send(UpdateEvent{
		notEmpty:  true,
		SM:        c.sm,
		Data:      data,
		AdapterID: adapterID,
		CallID:    callID,
	})

	<-c.continueSig
}

type LoggerSlotPredicateFn func(smachine.StateMachine, smachine.TracerID) bool

type DebugMachineLogger struct {
	underlying   smachine.SlotMachineLogger
	events       *updateChan
	continueStep synckit.ClosableSignalChannel
}

func (v DebugMachineLogger) CreateStepLogger(ctx context.Context, sm smachine.StateMachine, traceID smachine.TracerID) smachine.StepLogger {
	underlying := v.underlying.CreateStepLogger(ctx, sm, traceID)

	continueStep := synckit.SignalChannel(v.continueStep)
	if continueStep == nil {
		continueStep = synckit.ClosedChannel()
	}

	return DebugStepLogger{
		StepLogger:  underlying,
		sm:          sm,
		events:      v.events,
		continueSig: continueStep,
	}
}

func (v DebugMachineLogger) LogMachineInternal(slotMachineData smachine.SlotMachineData, msg string) {
	v.underlying.LogMachineInternal(slotMachineData, msg)
}

func (v DebugMachineLogger) LogMachineCritical(slotMachineData smachine.SlotMachineData, msg string) {
	v.underlying.LogMachineCritical(slotMachineData, msg)
}

func (v DebugMachineLogger) LogStopping(m *smachine.SlotMachine) {
	v.underlying.LogStopping(m)
	v.continueAll()
}

func (v DebugMachineLogger) continueAll() {
	if v.continueStep != nil {
		select {
		case _, ok := <- v.continueStep:
			if !ok {
				return
			}
		default:
		}
		close(v.continueStep)
	}
}

func (v DebugMachineLogger) GetEvent() (ev UpdateEvent) {
	 ev, _ = v.events.receive()
	 return
}

func (v DebugMachineLogger) EventChan() <-chan UpdateEvent {
	return v.events.events
}

func (v DebugMachineLogger) Continue() {
	if v.continueStep != nil {
		v.continueStep <- struct{}{}
	}
}

func (v DebugMachineLogger) FlushEvents(flushDone synckit.SignalChannel, closeEvents bool) {
	for {
		select {
		case _, ok := <- v.events.events:
			if !ok {
				return
			}
		case <-flushDone:
			if closeEvents {
				v.events.close()
			}
			return
		}
	}
}

func NewDebugMachineLogger(underlying smachine.SlotMachineLogger) DebugMachineLogger {
	return DebugMachineLogger{
		underlying:    underlying,
		events:        &updateChan{ events: make(chan UpdateEvent, 1) },
		continueStep:  make(chan struct{}),
	}
}

func NewDebugMachineLoggerNoBlock(underlying smachine.SlotMachineLogger, eventBufLimit int) DebugMachineLogger {
	if eventBufLimit <= 0 {
		panic(throw.IllegalValue())
	}
	return DebugMachineLogger{
		underlying:    underlying,
		events:        &updateChan{ events: make(chan UpdateEvent, eventBufLimit) },
	}
}
