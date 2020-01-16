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
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/bilogadapter/bilogencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

var (
	CallerMarshalFunc = func(file string, line int) string {
		return file + ":" + strconv.Itoa(line)
	}
)

var _ logcommon.EmbeddedLogger = &binLogAdapter{}
var _ logcommon.EmbeddedLoggerAssistant = &binLogAdapter{}

type binLogAdapter struct {
	config  *logadapter.Config
	encoder bilogencoder.Encoder
	writer  logcommon.LogLevelWriter

	parentStatic []byte
	staticFields []byte
	dynFields    dynFieldList // sorted

	expectedEventLen int

	levelFilter logcommon.LogLevel
}

const loggerSkipFrameCount = 3

func (v binLogAdapter) prepareEncoder(level logcommon.LogLevel, preallocate int) objectEncoder {
	if preallocate < minEventBuffer {
		preallocate += maxEventBufferIncrement
	}

	encoder := objectEncoder{v.encoder, make([]byte, 0, preallocate)}
	v.encoder.PrepareBuffer(&encoder.content)

	if level.IsValid() {
		encoder.AddStrField(logadapter.LevelFieldName, level.String(), logcommon.LogFieldFormat{})
	} else {
		encoder.AddStrField(logadapter.LevelFieldName, "", logcommon.LogFieldFormat{})
	}

	encoder.AddTimeField(logadapter.TimestampFieldName, time.Now(), logcommon.LogFieldFormat{})

	switch withFuncName := false; v.config.Instruments.CallerMode {
	case logcommon.CallerFieldWithFuncName:
		withFuncName = true
		fallthrough
	case logcommon.CallerField:
		skipFrameCount := int(v.config.Instruments.SkipFrameCountBaseline) + int(v.config.Instruments.SkipFrameCount)

		fileName, funcName, line := logadapter.GetCallInfo(skipFrameCount + loggerSkipFrameCount)
		fileName = CallerMarshalFunc(fileName, line)

		encoder.AddStrField(logadapter.CallerFieldName, fileName, logcommon.LogFieldFormat{})
		if withFuncName {
			encoder.AddStrField(logadapter.FuncFieldName, funcName, logcommon.LogFieldFormat{})
		}
	}

	encoder.content = append(encoder.content, v.parentStatic...)
	encoder.content = append(encoder.content, v.staticFields...)

	for _, field := range v.dynFields {
		val := field.Getter()
		encoder.AddIntfField(field.Name, val, logcommon.LogFieldFormat{})
	}

	return encoder
}

func (v binLogAdapter) sendEvent(level logcommon.LogLevel, encoder objectEncoder, msg string) {
	encoder.AddStrField(logadapter.MessageFieldName, msg, logcommon.LogFieldFormat{})
	v.encoder.FinalizeBuffer(&encoder.content)

	switch _, err := v.writer.LogLevelWrite(level, encoder.content); {
	case err == nil:
	case v.config.ErrorFn != nil:
		v.config.ErrorFn(err)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "bilog: could not write event: %v\n", err)
	}
}

func (v binLogAdapter) postEvent(level logcommon.LogLevel, msgStr string) {
	switch level {
	case logcommon.FatalLevel:
		os.Exit(1)
	case logcommon.PanicLevel:
		panic(msgStr)
	}
}

func (v binLogAdapter) doneEvent(recovered interface{}) {
	var err error
	if e, ok := recovered.(error); ok {
		err = e
	} else {
		err = errors.New(fmt.Sprintf("internal error (%v) stack:\n%s", recovered, debug.Stack()))
	}

	switch {
	case v.config.ErrorFn != nil:
		v.config.ErrorFn(err)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "bilog: could not handle event: %v\n", err)
	}
}

func (v binLogAdapter) NewEventStruct(level logcommon.LogLevel) func(interface{}, []logcommon.LogFieldMarshaller) {
	if !v.Is(level) {
		return nil
	}
	return func(arg interface{}, fields []logcommon.LogFieldMarshaller) {
		completed := false
		defer func() {
			if !completed {
				v.doneEvent(recover())
			}
		}()

		event := v.prepareEncoder(level, v.expectedEventLen)

		for _, f := range fields {
			f.MarshalLogFields(&event)
		}

		obj, msgStr := v.config.MsgFormat.FmtLogStruct(arg)
		if obj != nil {
			collector := v.config.Metrics.GetMetricsCollector()
			msgStr = obj.MarshalLogObject(&event, collector)
		}
		v.sendEvent(level, event, msgStr)
		completed = true
		v.postEvent(level, msgStr)
	}
}

func (v binLogAdapter) NewEvent(level logcommon.LogLevel) func(args []interface{}) {
	if !v.Is(level) {
		return nil
	}
	return func(args []interface{}) {
		completed := false
		defer func() {
			if !completed {
				v.doneEvent(recover())
			}
		}()

		if len(args) != 1 {
			msgStr := v.config.MsgFormat.FmtLogObject(args...)
			event := v.prepareEncoder(level, v.expectedEventLen)
			v.sendEvent(level, event, msgStr)
			completed = true
			v.postEvent(level, msgStr)
			return
		}

		obj, msgStr := v.config.MsgFormat.FmtLogStructOrObject(args[0])

		event := v.prepareEncoder(level, v.expectedEventLen)
		if obj != nil {
			collector := v.config.Metrics.GetMetricsCollector()
			msgStr = obj.MarshalLogObject(&event, collector)
		}
		v.sendEvent(level, event, msgStr)
		completed = true
		v.postEvent(level, msgStr)
	}
}

func (v binLogAdapter) NewEventFmt(level logcommon.LogLevel) func(fmt string, args []interface{}) {
	if !v.Is(level) {
		return nil
	}
	return func(fmt string, args []interface{}) {
		completed := false
		defer func() {
			if !completed {
				v.doneEvent(recover())
			}
		}()

		msgStr := v.config.MsgFormat.Sformatf(fmt, args...)
		event := v.prepareEncoder(level, len(msgStr))
		v.sendEvent(level, event, msgStr)
		completed = true
		v.postEvent(level, msgStr)
	}
}

const flushEventLevel = logcommon.WarnLevel

func (v binLogAdapter) EmbeddedFlush(msgStr string) {
	completed := false
	defer func() {
		if !completed {
			v.doneEvent(recover())
		}
	}()

	const level = flushEventLevel
	if len(msgStr) > 0 {
		event := v.prepareEncoder(level, len(msgStr))
		v.sendEvent(level, event, msgStr)
		completed = true
	}
	_ = v.config.LoggerOutput.Flush()
	//	v.postEvent(level, msgStr)
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

func (v binLogAdapter) _prepareAppendFields(nExpected int) objectEncoder {
	var buf []byte
	switch required, available := estimateBufferSizeByFields(nExpected), cap(v.staticFields)-len(v.staticFields); {
	case required <= available:
		buf = make([]byte, 0, cap(v.staticFields))
	default:
		buf = make([]byte, 0, len(v.staticFields)+required)
	}
	buf = append(buf, v.staticFields...)
	return objectEncoder{v.encoder, buf}
}

func (v binLogAdapter) WithFields(fields map[string]interface{}) logcommon.Logger {
	switch len(fields) {
	case 0:
		return logcommon.WrapEmbeddedLogger(v)
	case 1:
		// avoids sorting inside addIntfFields()
		for k, val := range fields {
			return v.WithField(k, val)
		}
	}
	objEncoder := v._prepareAppendFields(len(fields))
	objEncoder.addIntfFields(fields)
	v.staticFields = objEncoder.content
	return logcommon.WrapEmbeddedLogger(v)
}

func (v binLogAdapter) WithField(name string, value interface{}) logcommon.Logger {
	objEncoder := v._prepareAppendFields(1)
	objEncoder.AddIntfField(name, value, logcommon.LogFieldFormat{})
	v.staticFields = objEncoder.content
	return logcommon.WrapEmbeddedLogger(v)
}

// Can ONLY be used by the builder
func (v *binLogAdapter) _addFieldsByBuilder(fields map[string]interface{}) {
	if len(fields) == 0 {
		return
	}
	var newFields *[]byte
	if n := len(v.staticFields); n > 0 {
		buf := make([]byte, 0, n+len(v.parentStatic)+estimateBufferSizeByFields(len(fields)))
		buf = append(buf, v.parentStatic...)
		buf = append(buf, v.staticFields...)
		v.parentStatic = buf
		v.staticFields = nil
		newFields = &v.parentStatic
	} else {
		v.staticFields = make([]byte, 0, maxEventBufferIncrement)
		newFields = &v.staticFields
	}

	objEncoder := objectEncoder{v.encoder, *newFields}
	objEncoder.addIntfFields(fields)
	*newFields = objEncoder.content
}

// Can ONLY be used by the builder
func (v *binLogAdapter) _addDynFieldsByBuilder(newFields logcommon.DynFieldMap) {
	switch {
	case len(newFields) == 0:
		return
	case len(v.dynFields) > 0:
		fields := make(logcommon.DynFieldMap, len(newFields)+len(v.dynFields))
		for _, ve := range v.dynFields {
			fields[ve.Name] = ve.Getter
		}
		for k, v := range newFields {
			fields[k] = v
		}
		newFields = fields
	}

	v.dynFields = make(dynFieldList, 0, len(newFields))
	for k, val := range newFields {
		v.dynFields = append(v.dynFields, dynFieldEntry{k, val})
	}
	v.dynFields.Sort()
}

/* =========================== */

type dynFieldEntry struct {
	Name   string
	Getter logcommon.DynFieldFunc
}

type dynFieldList []dynFieldEntry

func (v dynFieldList) Sort() {
	sort.Sort(v)
}

func (v dynFieldList) Len() int {
	return len(v)
}

func (v dynFieldList) Less(i, j int) bool {
	return v[i].Name < v[j].Name
}

func (v dynFieldList) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}
