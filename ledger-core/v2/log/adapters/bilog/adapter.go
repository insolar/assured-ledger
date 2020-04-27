// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bilog

import (
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/msgencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"
)

var (
	CallerMarshalFunc = func(file string, line int) string {
		return file + ":" + strconv.Itoa(line)
	}
)

var _ logcommon.EmbeddedLogger = &binLogAdapter{}
var _ logcommon.EmbeddedLoggerOptional = &binLogAdapter{}

type binLogAdapter struct {
	config  *logcommon.Config
	encoder msgencoder.Encoder
	writer  logcommon.LoggerOutput

	parentStatic []byte
	staticFields []byte
	dynFields    dynFieldList // sorted

	expectedEventLen int

	levelFilter log.Level
	lowLatency  bool
	recycleBuf  bool
}

const loggerSkipFrameCount = 3

func (v binLogAdapter) prepareEncoder(level log.Level, preallocate int) objectEncoder {
	if preallocate < minEventBuffer {
		preallocate = minEventBuffer
	} else if preallocate > minEventBuffer {
		preallocate += maxEventBufferIncrement
	}

	encoder := objectEncoder{v.encoder, nil, nil, time.Now(), level == logcommon.DebugLevel, level}
	switch {
	case v.recycleBuf:
		encoder.poolBuf = allocateBuffer(preallocate)
		if encoder.poolBuf != nil {
			encoder.content = (*encoder.poolBuf)[:0]
			break
		}
		fallthrough
	default:
		encoder.content = make([]byte, 0, preallocate)
	}

	encoder.content = v.encoder.PrepareBuffer(encoder.content, logoutput.LevelFieldName, level)

	if v.config.Instruments.MetricsMode&logcommon.LogMetricsTimestamp != 0 {
		encoder.AddTimeField(logoutput.TimestampFieldName, encoder.reportedAt, logfmt.LogFieldFormat{})
	}

	switch withFuncName := false; v.config.Instruments.CallerMode {
	case logcommon.CallerFieldWithFuncName:
		withFuncName = true
		fallthrough
	case logcommon.CallerField:
		skipFrameCount := int(v.config.Instruments.SkipFrameCountBaseline) + int(v.config.Instruments.SkipFrameCount)

		fileName, funcName, line := logoutput.GetCallerInfo(skipFrameCount + loggerSkipFrameCount)
		fileName = CallerMarshalFunc(fileName, line)

		encoder.AddStrField(logoutput.CallerFieldName, fileName, logfmt.LogFieldFormat{})
		if withFuncName {
			encoder.AddStrField(logoutput.FuncFieldName, funcName, logfmt.LogFieldFormat{})
		}
	}

	encoder.content = append(encoder.content, v.parentStatic...)
	encoder.content = append(encoder.content, v.staticFields...)

	for _, field := range v.dynFields {
		val := field.Getter()
		if val == nil {
			continue
		}
		encoder.AddIntfField(field.Name, val, logfmt.LogFieldFormat{})
	}

	return encoder
}

func (v binLogAdapter) sendEvent(encoder objectEncoder, msgStr string, completed *bool) {
	encoder.AddStrField(logoutput.MessageFieldName, msgStr, logfmt.LogFieldFormat{})

	var content []byte
	if v.config.Instruments.MetricsMode.HasWriteMetric() {
		content = v.encoder.FinalizeBuffer(encoder.content, encoder.reportedAt)
	} else {
		content = v.encoder.FinalizeBuffer(encoder.content, time.Time{})
	}

	// NB! writer's methods can call runtime.Goexit() etc
	// So we have to mark our part as completed before it
	*completed = true

	level := encoder.level

	var err error
	if v.lowLatency {
		_, err = v.writer.LowLatencyWrite(level, content)
	} else {
		_, err = v.writer.LogLevelWrite(level, content)
	}

	if level == log.FatalLevel {
		// MUST NOT do it before writer's call to properly handle runtime.Goexit()
		// make sure not to loose Fatal by an internal panic
		defer os.Exit(1)
	}

	if encoder.poolBuf != nil {
		reuseBuffer(encoder.poolBuf)
	}

	switch {
	case err == nil:
		//
	case v.config.ErrorFn != nil:
		v.config.ErrorFn(err)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "bilog: could not write event: %v\n", err)
	}

	if level == log.PanicLevel {
		panic(msgStr)
	}
}

func (v binLogAdapter) doneEvent(level log.Level, recovered interface{}) {
	switch {
	case recovered == nil:
		// this is runtime.Goexit() - probably is caused by testing.T.FatalNow() - don't use os.Exit()
		return
	case level == log.FatalLevel:
		// make sure not to loose Fatal by an internal panic
		defer os.Exit(1)
	}

	var err error
	if e, ok := recovered.(error); ok {
		err = e
	} else {
		err = fmt.Errorf("internal error (%v)", recovered)
	}

	switch {
	case v.config.ErrorFn != nil:
		v.config.ErrorFn(err)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "bilog: could not handle event, %v\n\tstack:\n%s", err, debug.Stack())
	}
}

func (v binLogAdapter) NewEventStruct(level log.Level) func(interface{}, []logfmt.LogFieldMarshaller) {
	if !v.Is(level) {
		return nil
	}
	return func(arg interface{}, fields []logfmt.LogFieldMarshaller) {
		completed := false
		defer func() {
			if !completed {
				v.doneEvent(level, recover())
			}
		}()

		event := v.prepareEncoder(level, v.expectedEventLen)

		for _, f := range fields {
			f.MarshalLogFields(&event)
		}

		obj, msgStr := v.config.MsgFormat.FmtLogStruct(arg)
		if obj != nil {
			collector := v.config.Metrics.GetMetricsCollector()
			msgStr, _ = obj.MarshalLogObject(&event, collector)
		}
		v.sendEvent(event, msgStr, &completed)
	}
}

func (v binLogAdapter) NewEvent(level log.Level) func(args []interface{}) {
	if !v.Is(level) {
		return nil
	}
	return func(args []interface{}) {
		completed := false
		defer func() {
			if !completed {
				v.doneEvent(level, recover())
			}
		}()

		if len(args) != 1 {
			msgStr := v.config.MsgFormat.FmtLogObject(args...)
			event := v.prepareEncoder(level, v.expectedEventLen)
			v.sendEvent(event, msgStr, &completed)
			return
		}

		obj, msgStr := v.config.MsgFormat.FmtLogStructOrObject(args[0])

		event := v.prepareEncoder(level, v.expectedEventLen)
		if obj != nil {
			collector := v.config.Metrics.GetMetricsCollector()
			msgStr, _ = obj.MarshalLogObject(&event, collector)
		}
		v.sendEvent(event, msgStr, &completed)
	}
}

func (v binLogAdapter) NewEventFmt(level log.Level) func(fmt string, args []interface{}) {
	if !v.Is(level) {
		return nil
	}
	return func(fmt string, args []interface{}) {
		completed := false
		defer func() {
			if !completed {
				v.doneEvent(level, recover())
			}
		}()

		msgStr := v.config.MsgFormat.Sformatf(fmt, args...)
		event := v.prepareEncoder(level, len(msgStr))
		v.sendEvent(event, msgStr, &completed)
	}
}

func (v binLogAdapter) EmbeddedFlush(msgStr string) {
	if len(msgStr) == 0 {
		_ = v.config.LoggerOutput.Flush()
		return
	}

	const level = log.WarnLevel
	completed := false
	defer func() {
		if !completed {
			v.doneEvent(level, recover())
		}
		_ = v.config.LoggerOutput.Flush()
	}()

	event := v.prepareEncoder(level, len(msgStr))
	v.sendEvent(event, msgStr, &completed)
}

func (v binLogAdapter) Is(level log.Level) bool {
	return level >= v.levelFilter && level >= getGlobalFilter()
}

// NB! Reference receiver allows the builder to return the existing instance when nothing was changed, otherwise it will make a copy
func (v *binLogAdapter) Copy() logcommon.EmbeddedLoggerBuilder {
	return binLogTemplate{template: v, binLogFactory: binLogFactory{recycleBuf: v.recycleBuf}}
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
	return objectEncoder{v.encoder, nil, buf, time.Time{}, false, 0}
}

// NB! Reference receiver allows return of the existing instance when nothing was changed, otherwise it will make a copy
func (v *binLogAdapter) WithFields(fields map[string]interface{}) logcommon.EmbeddedLogger {
	switch len(fields) {
	case 0:
		return v
	case 1:
		// avoids sorting inside addIntfFields()
		for k, val := range fields {
			return v.WithField(k, val)
		}
	}
	return v.withFields(fields)
}

func (v binLogAdapter) withFields(fields map[string]interface{}) logcommon.EmbeddedLogger {
	objEncoder := v._prepareAppendFields(len(fields))
	objEncoder.addIntfFields(fields)
	v.staticFields = objEncoder.content
	return &v
}

func (v binLogAdapter) WithField(name string, value interface{}) logcommon.EmbeddedLogger {
	objEncoder := v._prepareAppendFields(1)
	objEncoder.AddIntfField(name, value, logfmt.LogFieldFormat{})
	v.staticFields = objEncoder.content
	return &v
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

	objEncoder := objectEncoder{v.encoder, nil, *newFields, time.Time{}, false, 0}
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
			if v == nil {
				delete(fields, k)
			} else {
				fields[k] = v
			}
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
