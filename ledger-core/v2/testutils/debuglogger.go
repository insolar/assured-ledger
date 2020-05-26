// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ smachine.StepLogger = &DebugStepLogger{}

type UpdateEvent struct {
	notEmpty bool
	SM       smachine.StateMachine
	Data     smachine.StepLoggerData
	Update   smachine.StepLoggerUpdateData
	CustomEvent interface{}
	AdapterID smachine.AdapterID
	CallID   uint64
}

func (e UpdateEvent) IsEmpty() bool {
	return !e.notEmpty
}

type DebugStepLogger struct {
	smachine.StepLogger

	sm          smachine.StateMachine
	events      chan<- UpdateEvent
	continueSig <-chan struct{}
}

func (c DebugStepLogger) CanLogEvent(smachine.StepLoggerEvent, smachine.StepLogLevel) bool {
	return true
}

func (c DebugStepLogger) CreateAsyncLogger(context.Context, *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return c.GetLoggerContext(), c
}

func (c DebugStepLogger) LogInternal(data smachine.StepLoggerData, updateType string) {
	c.StepLogger.LogInternal(data, updateType)

	c.events <- UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     data,
		Update:   smachine.StepLoggerUpdateData{UpdateType: updateType},
	}

	<-c.continueSig
}

func (c DebugStepLogger) LogUpdate(data smachine.StepLoggerData, update smachine.StepLoggerUpdateData) {
	c.StepLogger.LogUpdate(data, update)

	c.events <- UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     data,
		Update:   update,
	}

	<-c.continueSig
}

func (c DebugStepLogger) LogEvent(data smachine.StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller) {
	c.StepLogger.LogEvent(data, customEvent, fields)

	c.events <- UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     data,
		CustomEvent: customEvent,
	}

	<-c.continueSig
}

func (c DebugStepLogger) LogAdapter(data smachine.StepLoggerData, adapterID smachine.AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller) {
	c.StepLogger.LogAdapter(data, adapterID, callID, fields)

	c.events <- UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     data,
		AdapterID: adapterID,
		CallID: callID,
	}

	<-c.continueSig
}

type LoggerSlotPredicateFn func(smachine.StateMachine, smachine.TracerID) bool

type DebugMachineLogger struct {
	underlying        smachine.SlotMachineLogger
	events            chan UpdateEvent
	continueExecution chan struct{}
	slotPredicate     LoggerSlotPredicateFn
}

func (v DebugMachineLogger) CreateStepLogger(ctx context.Context, sm smachine.StateMachine, traceID smachine.TracerID) smachine.StepLogger {
	underlying := v.underlying.CreateStepLogger(ctx, sm, traceID)

	if v.slotPredicate != nil && !v.slotPredicate(sm, traceID) {
		return underlying
	}

	return DebugStepLogger{
		StepLogger:  underlying,
		sm:          sm,
		events:      v.events,
		continueSig: v.continueExecution,
	}
}

func (v DebugMachineLogger) LogMachineInternal(slotMachineData smachine.SlotMachineData, msg string) {
	v.underlying.LogMachineInternal(slotMachineData, msg)
}

func (v DebugMachineLogger) LogMachineCritical(slotMachineData smachine.SlotMachineData, msg string) {
	v.underlying.LogMachineCritical(slotMachineData, msg)
}

func (v DebugMachineLogger) GetEvent() UpdateEvent {
	event, ok := <-v.events
	if !ok {
		return UpdateEvent{}
	}
	return event
}

func (v DebugMachineLogger) Continue() {
	select {
	case v.continueExecution <- struct{}{}:
	default:
	}
}

func (v *DebugMachineLogger) SetPredicate(fn LoggerSlotPredicateFn) {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	v.slotPredicate = fn
}

func (v *DebugMachineLogger) Stop() {
	close(v.continueExecution)
	v.events <- UpdateEvent{}
}

func NewDebugMachineLogger(underlying smachine.SlotMachineLogger) *DebugMachineLogger {
	return &DebugMachineLogger{
		underlying:        underlying,
		events:            make(chan UpdateEvent, 1),
		continueExecution: make(chan struct{}),
		slotPredicate:     nil,
	}
}
