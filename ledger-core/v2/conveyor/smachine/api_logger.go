// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

type StepLoggerEvent uint8

const (
	StepLoggerUpdate StepLoggerEvent = iota
	StepLoggerMigrate
	StepLoggerInternal
	StepLoggerAdapterCall

	StepLoggerTrace
	StepLoggerActiveTrace
	StepLoggerWarn
	StepLoggerError
	StepLoggerFatal
)

type StepLoggerFlags uint32

const (
	stepLoggerUpdateErrorBit0 StepLoggerFlags = 1 << iota
	stepLoggerUpdateErrorBit1
	stepLoggerUpdateAdapterBit0
	stepLoggerUpdateAdapterBit1
	stepLoggerUpdateAdapterBit2
	StepLoggerDetached
)

const (
	StepLoggerUpdateErrorDefault        StepLoggerFlags = 0
	StepLoggerUpdateErrorMuted                          = stepLoggerUpdateErrorBit0
	StepLoggerUpdateErrorRecovered                      = stepLoggerUpdateErrorBit1
	StepLoggerUpdateErrorRecoveryDenied                 = stepLoggerUpdateErrorBit0 | stepLoggerUpdateErrorBit1
)
const StepLoggerErrorMask = stepLoggerUpdateErrorBit0 | stepLoggerUpdateErrorBit1

const (
	StepLoggerAdapterSyncCall    StepLoggerFlags = 0
	StepLoggerAdapterNotifyCall                  = stepLoggerUpdateAdapterBit0
	StepLoggerAdapterAsyncCall                   = stepLoggerUpdateAdapterBit2
	StepLoggerAdapterAsyncResult                 = stepLoggerUpdateAdapterBit2 | stepLoggerUpdateAdapterBit1
	StepLoggerAdapterAsyncCancel                 = stepLoggerUpdateAdapterBit2 | stepLoggerUpdateAdapterBit1 | stepLoggerUpdateAdapterBit0
)
const StepLoggerAdapterMask = stepLoggerUpdateAdapterBit0 | stepLoggerUpdateAdapterBit1 | stepLoggerUpdateAdapterBit2

type SlotMachineData struct {
	CycleNo uint32
	StepNo  StepLink
	Error   error
}

type StepLoggerData struct {
	CycleNo     uint32
	StepNo      StepLink
	CurrentStep StepDeclaration
	Declaration StateMachineHelper
	EventType   StepLoggerEvent
	Error       error
	Flags       StepLoggerFlags
}

type StepLoggerUpdateData struct {
	UpdateType string
	PrevStepNo uint32
	NextStep   StepDeclaration

	InactivityNano time.Duration // zero or negative - means that value is not applicable / not valid
	ActivityNano   time.Duration // zero or negative - means that value is not applicable / not valid
}

type SlotMachineLogger interface {
	CreateStepLogger(context.Context, StateMachine, TracerID) StepLogger
	LogMachineInternal(data SlotMachineData, msg string)
	LogMachineCritical(data SlotMachineData, msg string)
}

type StepLoggerFactoryFunc func(context.Context, StateMachine, TracerID) StepLogger

type StepLogLevel uint8

const (
	StepLogLevelDefault StepLogLevel = iota
	StepLogLevelElevated
	StepLogLevelTracing
)

type StepLogger interface {
	CanLogEvent(eventType StepLoggerEvent, stepLevel StepLogLevel) bool
	//LogMetric()
	LogUpdate(StepLoggerData, StepLoggerUpdateData)
	LogInternal(data StepLoggerData, updateType string)
	LogEvent(data StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller)

	// (callId) is guaranteed to be unique per Slot for async calls.
	// For notify and sync calls there is no guarantees on (callId).
	// Type of call can be identified by (data.Flags).
	LogAdapter(data StepLoggerData, adapterID AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller)

	GetTracerID() TracerID

	CreateAsyncLogger(context.Context, *StepLoggerData) (context.Context, StepLogger)
}

type StepLoggerFunc func(*StepLoggerData, *StepLoggerUpdateData)

type TracerID = string

type Logger struct { // we use an explicit struct here to enable compiler optimizations when logging is not needed
	ctx      context.Context
	loggerFn interface {
		getStepLogger() (StepLogger, StepLogLevel, uint32)
		getStepLoggerData() StepLoggerData
	}
}

func (p Logger) getStepLogger() (StepLogger, StepLogLevel, uint32) {
	if p.loggerFn != nil {
		return p.loggerFn.getStepLogger()
	}
	return nil, 0, 0
}

func (p Logger) GetContext() context.Context {
	_, _, _ = p.getStepLogger() // check context availability
	return p.ctx
}

func (p Logger) GetTracerID() TracerID {
	if stepLogger, _, _ := p.getStepLogger(); stepLogger != nil {
		return stepLogger.GetTracerID()
	}
	return ""
}

func (p Logger) _checkLog(eventType StepLoggerEvent) (StepLogger, uint32, StepLoggerEvent) {
	if stepLogger, stepLevel, stepUpdate := p.getStepLogger(); stepLogger != nil {
		if stepLogger.CanLogEvent(eventType, stepLevel) {
			if stepLevel == StepLogLevelTracing && eventType == StepLoggerTrace {
				eventType = StepLoggerActiveTrace
			}
			return stepLogger, stepUpdate, eventType
		}
	}
	return nil, 0, 0
}

func (p Logger) getStepLoggerData(eventType StepLoggerEvent, stepUpdate uint32, err error) StepLoggerData {
	stepData := p.loggerFn.getStepLoggerData()
	stepData.EventType = eventType
	stepData.Error = err
	if stepUpdate != 0 {
		stepData.StepNo.step = stepUpdate
	}
	return stepData
}

func (p Logger) _doLog(stepLogger StepLogger, stepUpdate uint32, eventType StepLoggerEvent,
	msg interface{}, fields []logfmt.LogFieldMarshaller, err error,
) {
	stepLogger.LogEvent(p.getStepLoggerData(eventType, stepUpdate, err), msg, fields)
}

func (p Logger) _doAdapterLog(stepLogger StepLogger, stepUpdate uint32, extraFlags StepLoggerFlags,
	adapterID AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller, err error,
) {
	stepData := p.getStepLoggerData(StepLoggerAdapterCall, stepUpdate, err)
	stepData.Flags |= extraFlags
	stepLogger.LogAdapter(stepData, adapterID, callID, fields)
}

func (p Logger) adapterCall(flags StepLoggerFlags, adapterID AdapterID, callID uint64, err error, fields ...logfmt.LogFieldMarshaller) { // nolint:unparam
	if stepLogger, stepUpdate, _ := p._checkLog(StepLoggerAdapterCall); stepLogger != nil {
		p._doAdapterLog(stepLogger, stepUpdate, flags, adapterID, callID, fields, err)
	}
}

// NB! keep method simple to ensure inlining
func (p Logger) Trace(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if stepLogger, stepUpdate, eventType := p._checkLog(StepLoggerTrace); stepLogger != nil {
		p._doLog(stepLogger, stepUpdate, eventType, msg, fields, nil)
	}
}

// NB! keep method simple to ensure inlining
func (p Logger) Warn(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if stepLogger, stepUpdate, eventType := p._checkLog(StepLoggerWarn); stepLogger != nil {
		p._doLog(stepLogger, stepUpdate, eventType, msg, fields, nil)
	}
}

// NB! keep method simple to ensure inlining
func (p Logger) Error(msg interface{}, err error, fields ...logfmt.LogFieldMarshaller) {
	if stepLogger, stepUpdate, eventType := p._checkLog(StepLoggerError); stepLogger != nil {
		p._doLog(stepLogger, stepUpdate, eventType, msg, fields, err)
	}
}

// NB! keep method simple to ensure inlining
func (p Logger) Fatal(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if stepLogger, stepUpdate, eventType := p._checkLog(StepLoggerFatal); stepLogger != nil {
		p._doLog(stepLogger, stepUpdate, eventType, msg, fields, nil)
	}
}
