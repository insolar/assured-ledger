package smachine

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
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

func (v StepLoggerEvent) IsEvent() bool {
	return v >= StepLoggerTrace
}

type StepLoggerFlags uint32

const (
	stepLoggerUpdateErrorBit0 StepLoggerFlags = 1 << iota
	stepLoggerUpdateErrorBit1
	stepLoggerUpdateAdapterBit0
	stepLoggerUpdateAdapterBit1
	stepLoggerUpdateAdapterBit2
	StepLoggerDetached
	StepLoggerInline
	StepLoggerShortLoop
	StepLoggerElevated
)

const (
	StepLoggerUpdateErrorDefault        StepLoggerFlags = 0
	StepLoggerUpdateErrorMuted                          = stepLoggerUpdateErrorBit0
	StepLoggerUpdateErrorRecovered                      = stepLoggerUpdateErrorBit1
	StepLoggerUpdateErrorRecoveryDenied                 = stepLoggerUpdateErrorBit0 | stepLoggerUpdateErrorBit1
)
const StepLoggerErrorMask = stepLoggerUpdateErrorBit0 | stepLoggerUpdateErrorBit1

func (v StepLoggerFlags) ErrorFlags() StepLoggerFlags {
	return v & StepLoggerErrorMask
}

const (
	StepLoggerAdapterSyncCall           StepLoggerFlags = 0
	StepLoggerAdapterNotifyCall                         = stepLoggerUpdateAdapterBit0
	StepLoggerAdapterAsyncCall                          = stepLoggerUpdateAdapterBit2
	StepLoggerAdapterAsyncResult                        = stepLoggerUpdateAdapterBit2 | stepLoggerUpdateAdapterBit1
	StepLoggerAdapterAsyncCancel                        = stepLoggerUpdateAdapterBit2 | stepLoggerUpdateAdapterBit1 | stepLoggerUpdateAdapterBit0
	StepLoggerAdapterAsyncExpiredResult                 = stepLoggerUpdateAdapterBit1
	StepLoggerAdapterAsyncExpiredCancel                 = stepLoggerUpdateAdapterBit1 | stepLoggerUpdateAdapterBit0
)

const StepLoggerAdapterMask = stepLoggerUpdateAdapterBit0 | stepLoggerUpdateAdapterBit1 | stepLoggerUpdateAdapterBit2

func (v StepLoggerFlags) AdapterFlags() StepLoggerFlags {
	return v & StepLoggerAdapterMask
}

func (v StepLoggerFlags) WithAdapterFlags(flags StepLoggerFlags) StepLoggerFlags {
	return (v & ^StepLoggerAdapterMask) | (flags & StepLoggerAdapterMask)
}

// SlotMachineData describes an event that is not connected to a specific SM, or when the related SM is already dead.
type SlotMachineData struct {
	// CycleNo is a cycle number of a SlotMachine when this event has happened
	CycleNo uint32
	// StepNo is an link to SM and step, the issue was related to. This SM is dead. Can be zero, when was not related to an SM.
	StepNo StepLink
	Error  error
}

// StepLoggerData describes an event that is connected to a specific SM.
type StepLoggerData struct {
	// CycleNo is a cycle number of a SlotMachine when this event has happened.
	CycleNo uint32
	// StepNo is an link to SM and step, the issue was related to.
	StepNo StepLink
	// CurrentStep is a step and its declaration data the SM is currently at.
	CurrentStep StepDeclaration
	// Declaration is a declaration of SM
	Declaration StateMachineHelper
	Error       error
	// EventType is a type of this event
	EventType StepLoggerEvent
	// Flags provide additional details about the event. Flags depend on (EventType)
	Flags StepLoggerFlags
}

// StepLoggerUpdateData describes an StateUpdate event applied to a specific SM. Invoked after each step.
type StepLoggerUpdateData struct {
	// UpdateType is a symbolic name of the update type, e.g. jump, sleep etc
	UpdateType string
	// NextStep is a step and its declaration data to be applied to SM.
	NextStep StepDeclaration

	// AppliedMigrate is set on migration update
	AppliedMigrate MigrateFunc

	// InactivityNano is a duration since the previous update, a time for which SM did not run any step.
	// Zero or negative value means that duration is not applicable / not valid.
	InactivityNano time.Duration
	// ActivityNano is a duration spent during inside the last (current) step.
	// Zero or negative value means that duration is not applicable / not valid.
	ActivityNano time.Duration
}

// SlotMachineLogger is a helper to facilitate reporting of SlotMachine operations into a log or console.
type SlotMachineLogger interface {
	// CreateStepLogger is invoked for every new Slot. Can return nil to disable any logging.
	CreateStepLogger(context.Context, StateMachine, TracerID) StepLogger
	// LogMachineInternal is invoked on non critical events, that aren't related to any active SM.
	LogMachineInternal(data SlotMachineData, msg string)
	// LogMachineCritical is invoked on critical events, that SlotMachine may not be able to handle properly.
	LogMachineCritical(data SlotMachineData, msg string)
	// LogStopping is invoked before SlotMachine will starts stopping operations. Can be cal
	LogStopping(*SlotMachine)
}

type StepLoggerFactoryFunc func(context.Context, StateMachine, TracerID) StepLogger

type StepLogLevel uint8

const (
	StepLogLevelDefault StepLogLevel = iota
	// StepLogLevelElevated indicates that this step was marked for detailed logging
	StepLogLevelElevated
	// StepLogLevelElevated indicates that this SM was marked for detailed logging
	StepLogLevelTracing
	StepLogLevelError
)

// StepLogger is a per-Slot helper to facilitate reporting of Slot operations into a log or console.
type StepLogger interface {
	// CanLogEvent returns true when the given type of the event should be logged. Followed by LogXXX() call
	CanLogEvent(eventType StepLoggerEvent, stepLevel StepLogLevel) bool
	// TODO LogMetric()
	// LogUpdate is invoked after each step of the Slot
	LogUpdate(StepLoggerData, StepLoggerUpdateData)
	// LogInternal is invoked on internal events related to the Slot
	LogInternal(data StepLoggerData, updateType string)
	// LogEvent is invoked on user calls to ctx.Log() methods
	LogEvent(data StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller)

	CanLogTestEvent() bool
	LogTestEvent(data StepLoggerData, customEvent interface{})

	// LogAdapter is invoked on user calls to ctx.LogAsync() methods
	// Value of (callId) is guaranteed to be unique per Slot per async call. For notify and sync calls there is no guarantees on (callId).
	// Type of call can be identified by (data.Flags).
	LogAdapter(data StepLoggerData, adapterID AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller)

	// GetTracerID returns TracerID assigned to this Slot during construction
	GetTracerID() TracerID
	// GetLoggerContext returns context for adapter calls etc, that must contain relevant TracerID
	// Implementation must reuse context, as this method can be called intensively.
	GetLoggerContext() context.Context

	// CreateAsyncLogger returns a logger for async operations attached to the given (StepLoggerData).
	// It received context of Slot, but can return any context.
	// Better reuse context, e.g. from GetLoggerContext(), as this method can be called intensively.
	CreateAsyncLogger(context.Context, *StepLoggerData) (context.Context, StepLogger)
}

type StepLoggerFunc func(*StepLoggerData, *StepLoggerUpdateData)

type TracerID = string

type Logger struct { // we use an explicit struct here to enable compiler optimizations when logging is not needed
	ctx    context.Context
	logger interface {
		getStepLogger() (StepLogger, StepLogLevel, uint32)
		getStepLoggerData() StepLoggerData
	}
}

func (p Logger) getStepLogger() (StepLogger, StepLogLevel, uint32) {
	if p.logger != nil {
		return p.logger.getStepLogger()
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

func (p Logger) _checkTestLog() StepLogger {
	if stepLogger, _, _ := p.getStepLogger(); stepLogger != nil && stepLogger.CanLogTestEvent() {
		return stepLogger
	}
	return nil
}

func (p Logger) _checkLog(eventType StepLoggerEvent) (StepLogger, uint32, StepLoggerEvent) {
	if stepLogger, stepLevel, stepUpdate := p.getStepLogger(); stepLogger != nil {
		if stepLogger.CanLogEvent(eventType, stepLevel) {
			if stepLevel >= StepLogLevelElevated && eventType == StepLoggerTrace {
				eventType = StepLoggerActiveTrace
			}
			return stepLogger, stepUpdate, eventType
		}
	}
	return nil, 0, 0
}

func (p Logger) getStepLoggerData(eventType StepLoggerEvent, stepUpdate uint32, err error) StepLoggerData {
	stepData := p.logger.getStepLoggerData()
	stepData.EventType = eventType
	stepData.Error = err

	if stepUpdate != 0 {
		stepData.StepNo.step = stepUpdate
	}
	return stepData
}

func (p Logger) _doTestLog(stepLogger StepLogger, msg interface{}) {
	stepLogger.LogTestEvent(p.getStepLoggerData(StepLoggerTrace, 0, nil), msg)
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

	if stepUpdate == 0 {
		stepData.StepNo.step = 0
		switch stepData.Flags.AdapterFlags() {
		case StepLoggerAdapterAsyncResult:
			stepData.Flags = stepData.Flags.WithAdapterFlags(StepLoggerAdapterAsyncExpiredResult)
		case StepLoggerAdapterAsyncCancel:
			stepData.Flags = stepData.Flags.WithAdapterFlags(StepLoggerAdapterAsyncExpiredCancel)
		}
	}

	stepLogger.LogAdapter(stepData, adapterID, callID, fields)
}

func (p Logger) adapterCall(flags StepLoggerFlags, adapterID AdapterID, callID uint64, err error, fields ...logfmt.LogFieldMarshaller) { // nolint:unparam
	if stepLogger, stepUpdate, _ := p._checkLog(StepLoggerAdapterCall); stepLogger != nil {
		p._doAdapterLog(stepLogger, stepUpdate, flags, adapterID, callID, fields, err)
	}
}

// Test will only send the given event when SM runs under step debugger. This method shall never report into a log.
// NB! keep method simple to ensure inlining
func (p Logger) Test(msg interface{}) {
	if stepLogger := p._checkTestLog(); stepLogger != nil {
		p._doTestLog(stepLogger, msg)
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

func (v StepLoggerData) FormatForLog(msg string) string {
	special := ""
	extra := ""
	sep := ""

	switch { // these flags are exclusive
	case v.Flags&StepLoggerDetached != 0:
		extra = " (detached)"
	case v.Flags&StepLoggerShortLoop != 0:
		extra = " (short)"
	}
	if v.Flags&StepLoggerInline != 0 {
		extra += " (inline)"
	}

	switch v.EventType {
	case StepLoggerUpdate:
	case StepLoggerMigrate:
		special = "migrate "
	case StepLoggerAdapterCall:
		// it is always (detached)
		extra = v.Flags.FormatAdapterForLog()
		if extra == "" {
			extra = "adapter-unknown-event"
		}
		if msg != "" {
			sep = " "
		}
	}

	errSpecial := v.Flags.FormatErrorForLog()

	return errSpecial + special + msg + sep + extra
}

func (v StepLoggerData) IsElevated() bool {
	return v.Flags&StepLoggerElevated != 0
}

func (v StepLoggerFlags) FormatAdapterForLog() string {
	switch v & StepLoggerAdapterMask {
	case StepLoggerAdapterSyncCall:
		return "sync-call"
	case StepLoggerAdapterAsyncCall:
		return "async-call"
	case StepLoggerAdapterNotifyCall:
		return "notify-call"
	case StepLoggerAdapterAsyncResult:
		return "async-result"
	case StepLoggerAdapterAsyncCancel:
		return "async-cancel"
	case StepLoggerAdapterAsyncExpiredResult:
		return "async-expired-result"
	case StepLoggerAdapterAsyncExpiredCancel:
		return "async-expired-cancel"
	}
	return ""
}

func (v StepLoggerFlags) FormatErrorForLog() string {
	switch v & StepLoggerErrorMask {
	case StepLoggerUpdateErrorMuted:
		return "muted "
	case StepLoggerUpdateErrorRecovered:
		return "recovered "
	case StepLoggerUpdateErrorRecoveryDenied:
		return "recover-denied "
	}
	return ""
}
