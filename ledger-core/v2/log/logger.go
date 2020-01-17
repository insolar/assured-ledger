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

func (z Logger) Is(level Level) bool {
	return z.embedded.Is(level)
}

func (z Logger) Copy() LoggerBuilder {
	template := z.embedded.Copy()
	return NewBuilderWithTemplate(template, template.GetTemplateLevel())
}

func (z Logger) WithFields(fields map[string]interface{}) Logger {
	if len(fields) == 0 {
		return z
	}
	if assist, ok := z.embedded.(logcommon.EmbeddedLoggerAssistant); ok {
		return WrapEmbeddedLogger(assist.WithFields(fields))
	}
	if logger, err := z.Copy().WithFields(fields).Build(); err != nil {
		panic(err)
	} else {
		return logger
	}
}

func (z Logger) WithField(name string, value interface{}) Logger {
	if assist, ok := z.embedded.(logcommon.EmbeddedLoggerAssistant); ok {
		return WrapEmbeddedLogger(assist.WithField(name, value))
	}
	if logger, err := z.Copy().WithField(name, value).Build(); err != nil {
		panic(err)
	} else {
		return logger
	}
}

func (z Logger) Embeddable() logcommon.EmbeddedLogger {
	return z.embedded
}

func (z Logger) Event(level Level, args ...interface{}) {
	if fn := z.embedded.NewEvent(level); fn != nil {
		fn(args)
	}
}

func (z Logger) Eventf(level Level, fmt string, args ...interface{}) {
	if fn := z.embedded.NewEventFmt(level); fn != nil {
		fn(fmt, args)
	}
}

func (z Logger) Events(level Level, msg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := z.embedded.NewEventStruct(level); fn != nil {
		fn(msg, fields)
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
