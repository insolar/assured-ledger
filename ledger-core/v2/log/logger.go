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
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logmsgfmt"
)

const DefaultOutputParallelLimit = 5

func WrapEmbeddedLogger(embedded logcommon.EmbeddedLogger) Logger {
	if embedded == nil {
		panic("illegal value")
	}
	return &LoggerStruct{EmbeddedHelper{embedded}}
}

type Logger = *LoggerStruct // TODO get rid of pointer?

type LoggerStruct struct {
	helper EmbeddedHelper
}

func (z LoggerStruct) Is(level LogLevel) bool {
	return z.helper.embedded.Is(level)
}

func (z LoggerStruct) Copy() LoggerBuilder {
	template := z.helper.embedded.Copy()
	return NewBuilderWithTemplate(template, template.GetTemplateLevel())
}

// Deprecated: do not use, or use Builder
func (z LoggerStruct) Level(lvl LogLevel) Logger {
	template := z.helper.embedded.Copy()
	if template.GetTemplateLevel() == lvl {
		return &z
	}

	if logger, err := NewBuilderWithTemplate(template, lvl).Build(); err != nil {
		panic(err)
	} else {
		return logger
	}
}

func (z LoggerStruct) WithFields(fields map[string]interface{}) Logger {
	if len(fields) == 0 {
		return &z
	}
	if assist, ok := z.helper.embedded.(logcommon.EmbeddedLoggerAssistant); ok {
		return WrapEmbeddedLogger(assist.WithFields(fields))
	}
	if logger, err := z.Copy().WithFields(fields).Build(); err != nil {
		panic(err)
	} else {
		return logger
	}
}

func (z LoggerStruct) WithField(name string, value interface{}) Logger {
	if assist, ok := z.helper.embedded.(logcommon.EmbeddedLoggerAssistant); ok {
		return WrapEmbeddedLogger(assist.WithField(name, value))
	}
	if logger, err := z.Copy().WithField(name, value).Build(); err != nil {
		panic(err)
	} else {
		return logger
	}
}

func (z LoggerStruct) Embeddable() logcommon.EmbeddedLogger {
	return z.helper.embedded
}

func (z LoggerStruct) Event(level LogLevel, args ...interface{}) {
	if fn := z.helper.NewEvent(level); fn != nil {
		fn(args)
	}
}

func (z LoggerStruct) Eventf(level LogLevel, fmt string, args ...interface{}) {
	if fn := z.helper.NewEventFmt(level); fn != nil {
		fn(fmt, args)
	}
}

func (z LoggerStruct) Events(level LogLevel, msg interface{}, fields ...logmsgfmt.LogFieldMarshaller) {
	if fn := z.helper.NewEventStruct(level); fn != nil {
		fn(msg, fields)
	}
}

func (z LoggerStruct) Debug(args ...interface{}) {
	if fn := z.helper.NewEvent(DebugLevel); fn != nil {
		fn(args)
	}
}

func (z LoggerStruct) Debugf(fmt string, args ...interface{}) {
	if fn := z.helper.NewEventFmt(DebugLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z LoggerStruct) Debugm(msg interface{}, fields ...logmsgfmt.LogFieldMarshaller) {
	if fn := z.helper.NewEventStruct(DebugLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z LoggerStruct) Info(args ...interface{}) {
	if fn := z.helper.NewEvent(InfoLevel); fn != nil {
		fn(args)
	}
}

func (z LoggerStruct) Infof(fmt string, args ...interface{}) {
	if fn := z.helper.NewEventFmt(InfoLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z LoggerStruct) Infom(msg interface{}, fields ...logmsgfmt.LogFieldMarshaller) {
	if fn := z.helper.NewEventStruct(InfoLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z LoggerStruct) Warn(args ...interface{}) {
	if fn := z.helper.NewEvent(WarnLevel); fn != nil {
		fn(args)
	}
}

func (z LoggerStruct) Warnf(fmt string, args ...interface{}) {
	if fn := z.helper.NewEventFmt(WarnLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z LoggerStruct) Warnm(msg interface{}, fields ...logmsgfmt.LogFieldMarshaller) {
	if fn := z.helper.NewEventStruct(WarnLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z LoggerStruct) Error(args ...interface{}) {
	if fn := z.helper.NewEvent(ErrorLevel); fn != nil {
		fn(args)
	}
}

func (z LoggerStruct) Errorf(fmt string, args ...interface{}) {
	if fn := z.helper.NewEventFmt(ErrorLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z LoggerStruct) Errorm(msg interface{}, fields ...logmsgfmt.LogFieldMarshaller) {
	if fn := z.helper.NewEventStruct(ErrorLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z LoggerStruct) Fatal(args ...interface{}) {
	if fn := z.helper.NewEvent(FatalLevel); fn != nil {
		fn(args)
	}
}

func (z LoggerStruct) Fatalf(fmt string, args ...interface{}) {
	if fn := z.helper.NewEventFmt(FatalLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z LoggerStruct) Fatalm(msg interface{}, fields ...logmsgfmt.LogFieldMarshaller) {
	if fn := z.helper.NewEventStruct(FatalLevel); fn != nil {
		fn(msg, fields)
	}
}

func (z LoggerStruct) Panic(args ...interface{}) {
	if fn := z.helper.NewEvent(PanicLevel); fn != nil {
		fn(args)
	}
}

func (z LoggerStruct) Panicf(fmt string, args ...interface{}) {
	if fn := z.helper.NewEventFmt(PanicLevel); fn != nil {
		fn(fmt, args)
	}
}

func (z LoggerStruct) Panicm(msg interface{}, fields ...logmsgfmt.LogFieldMarshaller) {
	if fn := z.helper.NewEventStruct(PanicLevel); fn != nil {
		fn(msg, fields)
	}
}

type EmbeddedHelper struct {
	embedded logcommon.EmbeddedLogger
}

func (z EmbeddedHelper) NewEventStruct(level LogLevel) func(interface{}, []logmsgfmt.LogFieldMarshaller) {
	return z.embedded.NewEventStruct(level)
	//if em := z.embedded.NewEventMarshaller(level); em == nil {
	//	return nil
	//} else {
	//	return func(arg interface{}) {
	//		msh, msg := z.embedded.GetMarshaller(true, arg)
	//		if msh != nil {
	//
	//		}
	//	}
	//}
}

func (z EmbeddedHelper) NewEvent(level LogLevel) func([]interface{}) {
	return z.embedded.NewEvent(level)
	//if fn := z.embedded.NewEventMarshaller(level); fn == nil {
	//	return nil
	//} else {
	//	return func(args []interface{}) {
	//		if len(args) != 1 {
	//
	//		}
	//		msh, msg := fn(false, args[0])
	//	}
	//}
}

func (z EmbeddedHelper) NewEventFmt(level LogLevel) func(string, []interface{}) {
	return z.embedded.NewEventFmt(level)
	//if fn := z.embedded.NewEventFmtArgs(level); fn == nil {
	//	return nil
	//} else {
	//	// NB! This closure is required to keep same stack depth with other methods
	//	return func(s string, args []interface{}) {
	//		fn(s, args)
	//	}
	//}
}
