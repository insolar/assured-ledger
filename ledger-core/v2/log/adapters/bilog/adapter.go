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

package bilog

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/bilogencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logmsgfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"
)

var (
	CallerMarshalFunc = func(file string, line int) string {
		return file + ":" + strconv.Itoa(line)
	}
)

var _ logcommon.EmbeddedLogger = &binLogAdapter{}
var _ logcommon.EmbeddedLoggerAssistant = &binLogAdapter{}

type binLogAdapter struct {
	config  *logcommon.Config
	encoder bilogencoder.Encoder
	writer  logcommon.LoggerOutput

	parentStatic []byte
	staticFields []byte
	dynFields    dynFieldList // sorted

	expectedEventLen int

	levelFilter log.LogLevel
	lowLatency  bool
}

const loggerSkipFrameCount = 3

func (v binLogAdapter) prepareEncoder(level log.LogLevel, preallocate int) objectEncoder {
	if preallocate < minEventBuffer {
		preallocate += maxEventBufferIncrement
	}

	encoder := objectEncoder{v.encoder, make([]byte, 0, preallocate), time.Now()}

	v.encoder.PrepareBuffer(&encoder.content, logoutput.LevelFieldName, level)
	encoder.AddTimeField(logoutput.TimestampFieldName, encoder.reportedAt, logmsgfmt.LogFieldFormat{})

	switch withFuncName := false; v.config.Instruments.CallerMode {
	case logcommon.CallerFieldWithFuncName:
		withFuncName = true
		fallthrough
	case logcommon.CallerField:
		skipFrameCount := int(v.config.Instruments.SkipFrameCountBaseline) + int(v.config.Instruments.SkipFrameCount)

		fileName, funcName, line := logoutput.GetCallerInfo(skipFrameCount + loggerSkipFrameCount)
		fileName = CallerMarshalFunc(fileName, line)

		encoder.AddStrField(logoutput.CallerFieldName, fileName, logmsgfmt.LogFieldFormat{})
		if withFuncName {
			encoder.AddStrField(logoutput.FuncFieldName, funcName, logmsgfmt.LogFieldFormat{})
		}
	}

	encoder.content = append(encoder.content, v.parentStatic...)
	encoder.content = append(encoder.content, v.staticFields...)

	for _, field := range v.dynFields {
		val := field.Getter()
		encoder.AddIntfField(field.Name, val, logmsgfmt.LogFieldFormat{})
	}

	return encoder
}

func (v binLogAdapter) sendEvent(level log.LogLevel, encoder objectEncoder, msg string) {
	encoder.AddStrField(logoutput.MessageFieldName, msg, logmsgfmt.LogFieldFormat{})
	v.encoder.FinalizeBuffer(&encoder.content, encoder.reportedAt)

	var err error
	if v.lowLatency {
		_, err = v.writer.LowLatencyWrite(level, encoder.content)
	} else {
		_, err = v.writer.LogLevelWrite(level, encoder.content)
	}

	switch {
	case err == nil:
		return
	case v.config.ErrorFn != nil:
		v.config.ErrorFn(err)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "bilog: could not write event: %v\n", err)
	}
}

func (v binLogAdapter) postEvent(level log.LogLevel, msgStr string) {
	switch level {
	case log.FatalLevel:
		os.Exit(1)
	case log.PanicLevel:
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

func (v binLogAdapter) NewEventStruct(level log.LogLevel) func(interface{}, []logmsgfmt.LogFieldMarshaller) {
	if !v.Is(level) {
		return nil
	}
	return func(arg interface{}, fields []logmsgfmt.LogFieldMarshaller) {
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

func (v binLogAdapter) NewEvent(level log.LogLevel) func(args []interface{}) {
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

func (v binLogAdapter) NewEventFmt(level log.LogLevel) func(fmt string, args []interface{}) {
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

const flushEventLevel = log.WarnLevel

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

func (v binLogAdapter) Is(level log.LogLevel) bool {
	return level >= v.levelFilter && level >= getGlobalFilter()
}

func (v binLogAdapter) Copy() logcommon.EmbeddedLoggerBuilder {
	return binLogTemplate{template: &v}
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
	return objectEncoder{v.encoder, buf, time.Time{}}
}

func (v binLogAdapter) WithFields(fields map[string]interface{}) logcommon.EmbeddedLogger {
	switch len(fields) {
	case 0:
		return v
	case 1:
		// avoids sorting inside addIntfFields()
		for k, val := range fields {
			return v.WithField(k, val)
		}
	}
	objEncoder := v._prepareAppendFields(len(fields))
	objEncoder.addIntfFields(fields)
	v.staticFields = objEncoder.content
	return v
}

func (v binLogAdapter) WithField(name string, value interface{}) logcommon.EmbeddedLogger {
	objEncoder := v._prepareAppendFields(1)
	objEncoder.AddIntfField(name, value, logmsgfmt.LogFieldFormat{})
	v.staticFields = objEncoder.content
	return v
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

	objEncoder := objectEncoder{v.encoder, *newFields, time.Time{}}
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
