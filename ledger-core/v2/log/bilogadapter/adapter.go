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

package bilogadapter

import (
	"fmt"
	"os"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

var _ logcommon.EmbeddedLogger = &binLogAdapter{}
var _ logcommon.EmbeddedLoggerAssistant = &binLogAdapter{}

type binLogAdapter struct {
	config  *logadapter.Config
	encoder Encoder
	writer  logcommon.LogLevelWriter

	parentStatic  []byte
	staticFields  []byte
	dynamicHooks  map[string]logcommon.LogObjectMarshallerFunc // TODO sorted list
	dynamicFields logcommon.DynFieldList                       // sorted

	levelFilter logcommon.LogLevel
}

func (v binLogAdapter) prepareEncoder(level logcommon.LogLevel) objectEncoder {
	buf := v.encoder.InitBuffer(level, nil, [][]byte{v.parentStatic, v.staticFields})
	encoder := objectEncoder{v.encoder, buf}

	for _, field := range v.dynamicFields {
		// TODO handle panic
		val := field.Getter()
		encoder.AddIntfField(field.Name, val, logcommon.LogFieldFormat{})
	}

	for _, hook := range v.dynamicHooks {
		// TODO handle panic
		hook(&encoder)
	}

	return encoder
}

func (v binLogAdapter) sendEvent(level logcommon.LogLevel, encoder objectEncoder, msg string) {
	encoder.AddStrField("Msg", msg, logcommon.LogFieldFormat{})

	switch _, err := v.writer.LogLevelWrite(level, encoder.content); {
	case err == nil:
	case v.config.ErrorFn != nil:
		v.config.ErrorFn(err)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "inslog: could not write event: %v\n", err)
	}
}

func (v binLogAdapter) NewEventStruct(level logcommon.LogLevel) func(interface{}, []logcommon.LogFieldMarshaller) {
	if !v.Is(level) {
		return nil
	}
	return func(arg interface{}, fields []logcommon.LogFieldMarshaller) {
		event := v.prepareEncoder(level)

		for _, f := range fields {
			f.MarshalLogFields(&event)
		}

		obj, msgStr := v.config.MsgFormat.FmtLogStruct(arg)
		if obj != nil {
			collector := v.config.Metrics.GetMetricsCollector()
			msgStr = obj.MarshalLogObject(&event, collector)
		}
		v.sendEvent(level, event, msgStr)
	}
}

func (v binLogAdapter) NewEvent(level logcommon.LogLevel) func(args []interface{}) {
	if !v.Is(level) {
		return nil
	}
	return func(args []interface{}) {

		if len(args) != 1 {
			msgStr := v.config.MsgFormat.FmtLogObject(args...)
			event := v.prepareEncoder(level)
			v.sendEvent(level, event, msgStr)
			return
		}

		obj, msgStr := v.config.MsgFormat.FmtLogStructOrObject(args[0])

		event := v.prepareEncoder(level)
		if obj != nil {
			collector := v.config.Metrics.GetMetricsCollector()
			msgStr = obj.MarshalLogObject(&event, collector)
		}
		v.sendEvent(level, event, msgStr)
	}
}

func (v binLogAdapter) NewEventFmt(level logcommon.LogLevel) func(fmt string, args []interface{}) {
	if !v.Is(level) {
		return nil
	}
	return func(fmt string, args []interface{}) {
		event := v.prepareEncoder(level)
		msgStr := v.config.MsgFormat.Sformatf(fmt, args...)
		v.sendEvent(level, event, msgStr)
	}
}

const flushEventLevel = logcommon.WarnLevel

func (v binLogAdapter) EmbeddedFlush(msg string) {
	if len(msg) > 0 {
		event := v.prepareEncoder(flushEventLevel)
		v.sendEvent(flushEventLevel, event, msg)
	}
	_ = v.config.LoggerOutput.Flush()
}

func (v binLogAdapter) Is(level logcommon.LogLevel) bool {
	return level >= v.levelFilter && level >= getGlobalFilter()
}

func (v binLogAdapter) Copy() logcommon.LoggerBuilder {
	return logadapter.NewBuilderWithTemplate(binLogTemplate{template: &v}, v.levelFilter)
}

func (v binLogAdapter) GetLoggerOutput() logcommon.LoggerOutput {
	return v.config.LoggerOutput
}

func (v binLogAdapter) WithFields(fields map[string]interface{}) logcommon.Logger {
	panic("implement me")
}

func (v binLogAdapter) WithField(name string, value interface{}) logcommon.Logger {
	panic("implement me")
}
