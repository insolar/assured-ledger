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

func (c DebugStepLogger) CanLogEvent(eventType smachine.StepLoggerEvent, stepLevel smachine.StepLogLevel) bool {
	return true
}

func (c DebugStepLogger) CreateAsyncLogger(ctx context.Context, _ *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return c.GetLoggerContext(), c
}

func (c DebugStepLogger) LogEvent(data smachine.StepLoggerData, msg interface{}, fields []logfmt.LogFieldMarshaller) {
	c.StepLogger.LogEvent(data, msg, fields)
}

func (c DebugStepLogger) LogUpdate(stepLoggerData smachine.StepLoggerData, stepLoggerUpdateData smachine.StepLoggerUpdateData) {
	c.StepLogger.LogUpdate(stepLoggerData, stepLoggerUpdateData)

	c.events <- UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     stepLoggerData,
		Update:   stepLoggerUpdateData,
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
