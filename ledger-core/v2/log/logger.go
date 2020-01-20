//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package log

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

const DefaultOutputParallelLimit = 5

func WrapEmbeddedLogger(embedded logcommon.EmbeddedLogger) Logger {
	if embedded == nil {
		panic("illegal value")
	}
	return Logger{embedded}
}

type Logger struct {
	embedded logcommon.EmbeddedLogger
}

// Returns true when the given level will reach the output
func (z Logger) Is(level Level) bool {
	return z.embedded.Is(level)
}

// Returns a builder to create a new logger that inherits current settings.
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

// Returns an internal representation of the logger.
// Do not use directly as caller handling may be incorrect.
func (z Logger) Embeddable() logcommon.EmbeddedLogger {
	return z.embedded
}

// Logs an event of the given (level).
// When a single argument is provided - the argument is evaluated as a logging struct.
// Multiple arguments or single non-record argument are formatted into a string by MsgFormatConfig.Sformat()
func (z Logger) Event(level Level, args ...interface{}) {
	if fn := z.embedded.NewEvent(level); fn != nil {
		fn(args)
	}
}

// Logs an event of the given (level).
// Argument(s) are formatted into a string by MsgFormatConfig.Sformatf()
func (z Logger) Eventf(level Level, fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(level); fn != nil {
		fn(fmt, args)
	}
}

// Logs an event of the given (level).
// Value of (msgStruct) is processed as a logging struct.
// And the event will also include provided (fields).
func (z Logger) Events(level Level, msgStruct interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(level); fn != nil {
		fn(msgStruct, fields)
	}
}

func (z Logger) Debug(args ...interface{}) {
	if fn := z.embedded.NewEvent(DebugLevel); fn != nil {
		fn(args)
	}
}

func (z Logger) Debugf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(DebugLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z Logger) Debugm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(DebugLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z Logger) Info(args ...interface{}) {
	if fn := z.embedded.NewEvent(InfoLevel); fn != nil {
		fn(args)
	}
}

func (z Logger) Infof(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(InfoLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z Logger) Infom(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(InfoLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z Logger) Warn(args ...interface{}) {
	if fn := z.embedded.NewEvent(WarnLevel); fn != nil {
		fn(args)
	}
}

func (z Logger) Warnf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(WarnLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z Logger) Warnm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(WarnLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z Logger) Error(args ...interface{}) {
	if fn := z.embedded.NewEvent(ErrorLevel); fn != nil {
		fn(args)
	}
}

func (z Logger) Errorf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(ErrorLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z Logger) Errorm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(ErrorLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z Logger) Fatal(args ...interface{}) {
	if fn := z.embedded.NewEvent(FatalLevel); fn != nil {
		fn(args)
	}
}

func (z Logger) Fatalf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(FatalLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z Logger) Fatalm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(FatalLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z Logger) Panic(args ...interface{}) {
	if fn := z.embedded.NewEvent(PanicLevel); fn != nil {
		fn(args)
	}
}

func (z Logger) Panicf(fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(PanicLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z Logger) Panicm(msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(PanicLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z Logger) IsZero() bool {
	return z.embedded == nil
}
