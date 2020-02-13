// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package log

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

const DefaultOutputParallelLimit = 5

// WrapEmbeddedLogger returns initialized Logger for given embedded adapter
func WrapEmbeddedLogger(embedded logcommon.EmbeddedLogger) Logger {
	if embedded == nil {
		panic("illegal value")
	}
	return Logger{embedded}
}

// A Logger represents an active logging object
type Logger struct {
	embedded logcommon.EmbeddedLogger
}

// Is returns true when the given level will reach the output
func (z Logger) Is(level Level) bool {
	return z.embedded.Is(level)
}

// Copy returns a builder to create a new logger that inherits current settings.
func (z Logger) Copy() LoggerBuilder {
	template := z.embedded.Copy()
	return NewBuilderWithTemplate(template, template.GetTemplateLevel())
}

// WithFields adds fields for to-be-built logger. Fields are deduplicated within a single builder only.
func (z Logger) WithFields(fields map[string]interface{}) Logger {
	if len(fields) == 0 {
		return z
	}
	if assist, ok := z.embedded.(logcommon.EmbeddedLoggerOptional); ok {
		return WrapEmbeddedLogger(assist.WithFields(fields))
	}
	if logger, err := z.Copy().WithFields(fields).Build(); err != nil {
		panic(err)
	} else {
		return logger
	}
}

// WithField add a fields for to-be-built logger. Fields are deduplicated within a single builder only.
func (z Logger) WithField(name string, value interface{}) Logger {
	if assist, ok := z.embedded.(logcommon.EmbeddedLoggerOptional); ok {
		return WrapEmbeddedLogger(assist.WithField(name, value))
	}
	if logger, err := z.Copy().WithField(name, value).Build(); err != nil {
		panic(err)
	} else {
		return logger
	}
}

// Embeddable returns an internal representation of the logger.
// Do not use directly as caller handling may be incorrect.
func (z Logger) Embeddable() logcommon.EmbeddedLogger {
	return z.embedded
}

// Event logs an event of the given (level).
// When a single argument is provided - the argument is evaluated as a logging struct.
// Multiple arguments or single non-record argument are formatted into a string by MsgFormatConfig.Sformat()
func (z Logger) Event(level Level, args ...interface{}) {
	if fn := z.embedded.NewEvent(level); fn != nil {
		fn(args)
	}
}

// Eventf logs an event of the given (level).
// Argument(s) are formatted into a string by MsgFormatConfig.Sformatf()
func (z Logger) Eventf(level Level, fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(level); fn != nil {
		fn(fmt, args)
	}
}

// Events logs an event of the given (level).
// Value of (msgStruct) is processed as a logging struct.
// And the event will also include provided (fields).
func (z Logger) Events(level Level, msgStruct interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(level); fn != nil {
		fn(msgStruct, fields)
	}
}

// Debug logs an event with level Debug.
// When a single argument is provided - the argument is evaluated as a logging struct.
// Multiple arguments or single non-record argument are formatted into a string by MsgFormatConfig.Sformat()
func (z Logger) Debug(args ...interface{}) {
	if fn := z.embedded.NewEvent(DebugLevel); fn != nil {
		fn(args)
	}
}

// Debugf logs an event with level Debug.
// Argument(s) are formatted into a string by MsgFormatConfig.Sformatf()
func (z Logger) Debugf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(DebugLevel); fn != nil {
		fn(fmt, args)
	}
}

// Debugm logs an event with level Debug.
// msg is expected to be logging struct, extra fields could be provided
func (z Logger) Debugm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(DebugLevel); fn != nil {
		fn(msg, fields)
	}
}

// Info logs an event with level Info.
// When a single argument is provided - the argument is evaluated as a logging struct.
// Multiple arguments or single non-record argument are formatted into a string by MsgFormatConfig.Sformat()
func (z Logger) Info(args ...interface{}) {
	if fn := z.embedded.NewEvent(InfoLevel); fn != nil {
		fn(args)
	}
}

// Infof logs an event with level Info.
// Argument(s) are formatted into a string by MsgFormatConfig.Sformatf()
func (z Logger) Infof(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(InfoLevel); fn != nil {
		fn(fmt, args)
	}
}

// Infom logs an event with level Info.
// msg is expected to be logging struct, extra fields could be provided
func (z Logger) Infom(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(InfoLevel); fn != nil {
		fn(msg, fields)
	}
}

// Warn logs an event with level Warn.
// When a single argument is provided - the argument is evaluated as a logging struct.
// Multiple arguments or single non-record argument are formatted into a string by MsgFormatConfig.Sformat()
func (z Logger) Warn(args ...interface{}) {
	if fn := z.embedded.NewEvent(WarnLevel); fn != nil {
		fn(args)
	}
}

// Warnf logs an event with level Warn.
// Argument(s) are formatted into a string by MsgFormatConfig.Sformatf()
func (z Logger) Warnf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(WarnLevel); fn != nil {
		fn(fmt, args)
	}
}

// Warnm logs an event with level Warn.
// msg is expected to be logging struct, extra fields could be provided
func (z Logger) Warnm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(WarnLevel); fn != nil {
		fn(msg, fields)
	}
}

// Error logs an event with level Error.
// When a single argument is provided - the argument is evaluated as a logging struct.
// Multiple arguments or single non-record argument are formatted into a string by MsgFormatConfig.Sformat()
func (z Logger) Error(args ...interface{}) {
	if fn := z.embedded.NewEvent(ErrorLevel); fn != nil {
		fn(args)
	}
}

// Errorf logs an event with level Error.
// Argument(s) are formatted into a string by MsgFormatConfig.Sformatf()
func (z Logger) Errorf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(ErrorLevel); fn != nil {
		fn(fmt, args)
	}
}

// Errorm logs an event with level Error.
// msg is expected to be logging struct, extra fields could be provided
func (z Logger) Errorm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(ErrorLevel); fn != nil {
		fn(msg, fields)
	}
}

// Fatal logs an event with level Fatal and calls os.Exit(1) after log message has been written
// When a single argument is provided - the argument is evaluated as a logging struct.
// Multiple arguments or single non-record argument are formatted into a string by MsgFormatConfig.Sformat()
func (z Logger) Fatal(args ...interface{}) {
	if fn := z.embedded.NewEvent(FatalLevel); fn != nil {
		fn(args)
	}
}

// Fatalf logs an event with level Fatal and calls os.Exit(1) after log message has been written.
// Argument(s) are formatted into a string by MsgFormatConfig.Sformatf()
func (z Logger) Fatalf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(FatalLevel); fn != nil {
		fn(fmt, args)
	}
}

// Fatalm logs an event with level Fatal and calls os.Exit(1) after log message has been written.
// msg is expected to be logging struct, extra fields could be provided
func (z Logger) Fatalm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(FatalLevel); fn != nil {
		fn(msg, fields)
	}
}

// Panic logs an event with level Panic and calls panic after log message has been written
// When a single argument is provided - the argument is evaluated as a logging struct.
// Multiple arguments or single non-record argument are formatted into a string by MsgFormatConfig.Sformat()
func (z Logger) Panic(args ...interface{}) {
	if fn := z.embedded.NewEvent(PanicLevel); fn != nil {
		fn(args)
	}
}

// Panicf logs an event with level Panic and calls panic after log message has been written.
// Argument(s) are formatted into a string by MsgFormatConfig.Sformatf()
func (z Logger) Panicf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(PanicLevel); fn != nil {
		fn(fmt, args)
	}
}

// Panicm logs an event with level Panic and calls panic after log message has been written.
// msg is expected to be logging struct, extra fields could be provided
func (z Logger) Panicm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(PanicLevel); fn != nil {
		fn(msg, fields)
	}
}

// IsZero returns true if logger has been initialized with non zero embedded
func (z Logger) IsZero() bool {
	return z.embedded == nil
}
