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

package zlogadapter

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"

	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logmetrics"
)

func init() {
	initLevelMappings()
}

type zerologMapping struct {
	zl      zerolog.Level
	fn      func(*zerolog.Logger) *zerolog.Event
	metrics context.Context
}

func (v zerologMapping) IsEmpty() bool {
	return v.fn == nil
}

var zerologLevelMapping = []zerologMapping{
	logcommon.DebugLevel: {zl: zerolog.DebugLevel, fn: (*zerolog.Logger).Debug},
	logcommon.InfoLevel:  {zl: zerolog.InfoLevel, fn: (*zerolog.Logger).Info},
	logcommon.WarnLevel:  {zl: zerolog.WarnLevel, fn: (*zerolog.Logger).Warn},
	logcommon.ErrorLevel: {zl: zerolog.ErrorLevel, fn: (*zerolog.Logger).Error},
	logcommon.FatalLevel: {zl: zerolog.FatalLevel, fn: (*zerolog.Logger).Fatal},
	logcommon.PanicLevel: {zl: zerolog.PanicLevel, fn: (*zerolog.Logger).Panic},
	logcommon.NoLevel:    {zl: zerolog.NoLevel, fn: (*zerolog.Logger).Log},
	logcommon.Disabled:   {zl: zerolog.Disabled, fn: func(*zerolog.Logger) *zerolog.Event { return nil }},
}

var zerologReverseMapping []logcommon.LogLevel

func initLevelMappings() {
	var zLevelMax zerolog.Level
	for i := range zerologLevelMapping {
		if zerologLevelMapping[i].IsEmpty() {
			continue
		}
		if zLevelMax < zerologLevelMapping[i].zl {
			zLevelMax = zerologLevelMapping[i].zl
		}
		zerologLevelMapping[i].metrics = logmetrics.GetLogLevelContext(logcommon.LogLevel(i))
	}

	zerologReverseMapping = make([]logcommon.LogLevel, zLevelMax+1)
	for i := range zerologReverseMapping {
		zerologReverseMapping[i] = logcommon.Disabled
	}

	for i := range zerologLevelMapping {
		if zerologLevelMapping[i].IsEmpty() {
			zerologLevelMapping[i] = zerologLevelMapping[logcommon.Disabled]
		} else {
			zl := zerologLevelMapping[i].zl
			if zerologReverseMapping[zl] != logcommon.Disabled {
				panic("duplicate level mapping")
			}
			zerologReverseMapping[zl] = logcommon.LogLevel(i)
		}
	}
}

func getLevelMapping(insLevel logcommon.LogLevel) zerologMapping {
	if int(insLevel) > len(zerologLevelMapping) {
		return zerologLevelMapping[logcommon.Disabled]
	}
	return zerologLevelMapping[insLevel]
}

func ToZerologLevel(insLevel logcommon.LogLevel) zerolog.Level {
	return getLevelMapping(insLevel).zl
}

func FromZerologLevel(zLevel zerolog.Level) logcommon.LogLevel {
	if int(zLevel) > len(zerologReverseMapping) {
		return zerologReverseMapping[zerolog.Disabled]
	}
	return zerologReverseMapping[zLevel]
}

func selectFormatter(format logcommon.LogFormat, output io.Writer) (io.Writer, error) {
	switch format {
	case logcommon.TextFormat:
		return newDefaultTextOutput(output), nil
	case logcommon.JSONFormat:
		return output, nil
	default:
		return nil, errors.New("unknown formatter " + format.String())
	}
}

const ZerologSkipFrameCount = 4

func NewBuilder(cfg logadapter.Config, level logcommon.LogLevel) logcommon.LoggerBuilder {
	return logadapter.NewBuilder(zerologFactory{}, cfg, level)
}

/* ============================ */

type zerologMarshaller struct {
	event *zerolog.Event
	*logadapter.MsgFormatConfig
}

func (m zerologMarshaller) AddIntField(key string, v int64, fFmt logcommon.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Int64(key, v)
	}
}

func (m zerologMarshaller) AddUintField(key string, v uint64, fFmt logcommon.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Uint64(key, v)
	}
}

func (m zerologMarshaller) AddBoolField(key string, v bool, fFmt logcommon.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Bool(key, v)
	}
}

func (m zerologMarshaller) AddFloatField(key string, v float64, fFmt logcommon.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Float64(key, v)
	}
}

func (m zerologMarshaller) AddComplexField(key string, v complex128, fFmt logcommon.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Str(key, m.Sformat(v))
	}
}

func (m zerologMarshaller) AddStrField(key string, v string, fFmt logcommon.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Str(key, v)
	}
}

func (m zerologMarshaller) AddIntfField(key string, v interface{}, fFmt logcommon.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Interface(key, v)
	}
}

func (m zerologMarshaller) AddTimeField(key string, v time.Time, fFmt logcommon.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, v.Format(fFmt.Fmt))
	} else {
		// NB! here we ignore MsgFormatConfig.TimeFmt as it should be set to zerolog.TimeFieldFormat
		m.event.Time(key, v)
	}
}

func (m zerologMarshaller) AddRawJSONField(key string, v interface{}, fFmt logcommon.LogFieldFormat) {
	m.event.RawJSON(key, []byte(m.Sformatf(fFmt.Fmt, v)))
}

/* ============================ */

var _ logcommon.EmbeddedLogger = &zerologAdapter{}
var _ logcommon.EmbeddedLoggerAssistant = &zerologAdapter{}

type zerologAdapter struct {
	logger    zerolog.Logger
	dynFields logcommon.DynFieldMap
	config    *logadapter.Config
}

func (z *zerologAdapter) WithFields(fields map[string]interface{}) logcommon.Logger {
	zCtx := z.logger.With()
	for key, value := range fields {
		zCtx = zCtx.Interface(key, value)
	}

	zCopy := *z
	zCopy.logger = zCtx.Logger()
	return logcommon.WrapEmbeddedLogger(&zCopy)
}

func (z *zerologAdapter) WithField(key string, value interface{}) logcommon.Logger {
	zCopy := *z
	zCopy.logger = z.logger.With().Interface(key, value).Logger()
	return logcommon.WrapEmbeddedLogger(&zCopy)
}

func (z *zerologAdapter) newEvent(level logcommon.LogLevel) *zerolog.Event {
	m := getLevelMapping(level)
	z.config.Metrics.OnNewEvent(m.metrics, level)
	event := m.fn(&z.logger)
	if event == nil {
		return nil
	}
	z.config.Metrics.OnFilteredEvent(m.metrics, level)
	return event
}

func (z *zerologAdapter) NewEventStruct(level logcommon.LogLevel) func(interface{}, []logcommon.LogFieldMarshaller) {
	switch event := z.newEvent(level); {
	case event == nil:
		//collector := z.config.Metrics.GetMetricsCollector()
		//if collector != nil {
		//	if obj := z.config.MsgFormat.PrepareMutedLogObject(arg); obj != nil {
		//		obj.MarshalMutedLogObject(collector)
		//	}
		//}
		return nil
	default:
		return func(arg interface{}, fields []logcommon.LogFieldMarshaller) {
			m := zerologMarshaller{event, &z.config.MsgFormat}
			for _, f := range fields {
				f.MarshalLogFields(m)
			}
			obj, msgStr := z.config.MsgFormat.FmtLogStruct(arg)
			if obj != nil {
				collector := z.config.Metrics.GetMetricsCollector()
				msgStr = obj.MarshalLogObject(zerologMarshaller{event, &z.config.MsgFormat}, collector)
			}
			event.Msg(msgStr)
		}
	}
}

func (z *zerologAdapter) NewEvent(level logcommon.LogLevel) func(args []interface{}) {
	switch event := z.newEvent(level); {
	case event == nil:
		return nil
	default:
		return func(args []interface{}) {
			if len(args) != 1 {
				msgStr := z.config.MsgFormat.FmtLogObject(args...)
				event.Msg(msgStr)
				return
			}

			obj, msgStr := z.config.MsgFormat.FmtLogStructOrObject(args[0])
			if obj != nil {
				collector := z.config.Metrics.GetMetricsCollector()
				msgStr = obj.MarshalLogObject(zerologMarshaller{event, &z.config.MsgFormat}, collector)
			}
			event.Msg(msgStr)
		}
	}
}

func (z *zerologAdapter) NewEventFmt(level logcommon.LogLevel) func(fmt string, args []interface{}) {
	event := z.newEvent(level)
	if event == nil {
		return nil
	}
	return func(fmt string, args []interface{}) {
		event.Msg(z.config.MsgFormat.Sformatf(fmt, args...))
	}
}

func (z *zerologAdapter) EmbeddedFlush(msg string) {
	if len(msg) > 0 {
		z.newEvent(logcommon.WarnLevel).Msg(msg)
	}
	_ = z.config.LoggerOutput.Flush()
}

func (z *zerologAdapter) Is(level logcommon.LogLevel) bool {
	return z.newEvent(level) != nil
}

func (z *zerologAdapter) Copy() logcommon.LoggerBuilder {
	return logadapter.NewBuilderWithTemplate(zerologTemplate{template: z}, FromZerologLevel(z.logger.GetLevel()))
}

func (z *zerologAdapter) GetLoggerOutput() logcommon.LoggerOutput {
	return z.config.LoggerOutput
}

/* =========================== */

var _ logadapter.Factory = zerologFactory{}
var _ logcommon.GlobalLogAdapterFactory = zerologFactory{}

type zerologFactory struct {
	writeDelayPreferTrim bool
}

func (zf zerologFactory) GetGlobalLogAdapter() logcommon.GlobalLogAdapter {
	return zerologGlobalAdapter
}

func (zf zerologFactory) PrepareBareOutput(output logadapter.BareOutput, metrics *logmetrics.MetricsHelper, config logadapter.BuildConfig) (io.Writer, error) {
	outputWriter, err := selectFormatter(config.Output.Format, output.Writer)

	if err != nil {
		return nil, err
	}

	if ok, name, reportFn := getWriteDelayConfig(metrics, config); ok {
		outputWriter = newWriteDelayPostHook(outputWriter, name, zf.writeDelayPreferTrim, reportFn)
	}

	return outputWriter, nil
}

func checkNewLoggerOutput(output zerolog.LevelWriter) zerolog.LevelWriter {
	if output == nil {
		panic("illegal value")
	}
	//
	return output
}

func (zf zerologFactory) createNewLogger(output zerolog.LevelWriter, params logadapter.NewLoggerParams, template *zerologAdapter,
) (logcommon.EmbeddedLogger, error) {

	instruments := params.Config.Instruments
	skipFrames := int(instruments.SkipFrameCountBaseline) + int(instruments.SkipFrameCount)
	callerMode := instruments.CallerMode

	cfg := params.Config

	la := zerologAdapter{
		// NB! We have to create a new logger and pass the context separately
		// Otherwise, zerolog will also copy hooks - which we need to get rid of some.
		logger: zerolog.New(checkNewLoggerOutput(output)).Level(ToZerologLevel(params.Level)),
		config: &cfg,
	}

	if ok, name, _ := getWriteDelayConfig(params.Config.Metrics, params.Config.BuildConfig); ok {
		la.logger = la.logger.Hook(newWriteDelayPreHook(name, zf.writeDelayPreferTrim))
	}

	{ // replacement and inheritance for dynFields
		switch newFields := params.DynFields; {
		case template != nil && params.Reqs&logadapter.RequiresParentDynFields != 0 && len(template.dynFields) > 0:
			prevFields := template.dynFields

			if len(newFields) > 0 {
				for k, v := range prevFields {
					if _, ok := newFields[k]; !ok {
						newFields[k] = v
					}
				}
			} else {
				newFields = prevFields
			}
			fallthrough
		case len(newFields) > 0:
			la.dynFields = newFields
			la.logger = la.logger.Hook(newDynFieldsHook(newFields))
		}
	}

	if callerMode == logcommon.CallerFieldWithFuncName {
		la.logger = la.logger.Hook(newCallerHook(skipFrames))
	}
	lc := la.logger.With()

	// only add hooks, DON'T set the context as it can be replaced below
	lc = lc.Timestamp()
	if callerMode == logcommon.CallerField {
		lc = lc.CallerWithSkipFrameCount(skipFrames)
	}

	if template != nil && params.Reqs&logadapter.RequiresParentCtxFields != 0 {
		la.logger = lc.Logger()     // save hooks
		lc = template.logger.With() // get a copy of the inherited context
	}
	for k, v := range params.Fields {
		lc = lc.Interface(k, v)
	}

	la.logger.UpdateContext(func(zerolog.Context) zerolog.Context {
		return lc
	})

	return &la, nil
}

func (zf zerologFactory) copyLogger(template *zerologAdapter, params logadapter.CopyLoggerParams) logcommon.EmbeddedLogger {

	if params.Reqs&logadapter.RequiresParentDynFields == 0 {
		// have to reset hooks, but zerolog can't reset hooks
		// so we have to create the logger from scratch
		return nil
	}

	hasUpdates := false
	la := *template

	if newFields := params.AppendDynFields; len(newFields) > 0 {
		if prevFields := la.dynFields; len(prevFields) > 0 {
			// NB! avoid modification of newFields when nil can be returned
			for k := range newFields {
				if _, ok := prevFields[k]; ok {
					// key collision
					// have to reset hooks, but zerolog can't reset hooks
					// so we have to create the logger from scratch
					return nil
				}
			}
			for k, v := range prevFields {
				newFields[k] = v
			}
		}
		la.dynFields = newFields
		la.logger = la.logger.Hook(newDynFieldsHook(newFields))
		hasUpdates = true
	}

	newLevel := ToZerologLevel(params.Level)
	if template.logger.GetLevel() != newLevel {
		la.logger = la.logger.Level(newLevel)
		hasUpdates = true
	}

	{
		hasCtxUpdates := false
		var lc zerolog.Context

		if params.Reqs&logadapter.RequiresParentCtxFields == 0 {
			// have to reset logger context
			lc = zerolog.New(nil).With()
			hasCtxUpdates = true
		}

		if len(params.AppendFields) > 0 {
			if !hasCtxUpdates {
				lc = la.logger.With()
			}
			for k, v := range params.AppendFields {
				lc = lc.Interface(k, v)
			}
			hasCtxUpdates = true
		}

		if hasCtxUpdates {
			la.logger.UpdateContext(func(zerolog.Context) zerolog.Context {
				return lc
			})
			hasUpdates = true
		}
	}

	if !hasUpdates {
		return template
	}
	return &la
}

func (zf zerologFactory) createOutputWrapper(config logadapter.Config, reqs logadapter.FactoryRequirementFlags) zerolog.LevelWriter {
	if reqs&logadapter.RequiresLowLatency != 0 {
		return zerologAdapterLLOutput{config.LoggerOutput}
	}
	return zerologAdapterOutput{config.LoggerOutput}
}

func (zf zerologFactory) CreateNewLogger(params logadapter.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
	output := zf.createOutputWrapper(params.Config, params.Reqs)
	return zf.createNewLogger(output, params, nil)
}

func (zf zerologFactory) CanReuseMsgBuffer() bool {
	// zerolog does recycling of []byte buffers
	return false
}

/* =========================== */

var zerologGlobalAdapter logcommon.GlobalLogAdapter = zerologGlobal{}

type zerologGlobal struct{}

func (zerologGlobal) SetGlobalLoggerFilter(level logcommon.LogLevel) {
	zerolog.SetGlobalLevel(ToZerologLevel(level))
}

func (zerologGlobal) GetGlobalLoggerFilter() logcommon.LogLevel {
	return FromZerologLevel(zerolog.GlobalLevel())
}

/* =========================== */

var _ logadapter.Template = &zerologTemplate{}

type zerologTemplate struct {
	zerologFactory
	template *zerologAdapter
}

func (zf zerologTemplate) GetLoggerOutput() logcommon.LoggerOutput {
	return zf.template.GetLoggerOutput()
}

func (zf zerologTemplate) GetTemplateConfig() logadapter.Config {
	return *zf.template.config
}

func (zf zerologTemplate) CopyTemplateLogger(params logadapter.CopyLoggerParams) logcommon.EmbeddedLogger {
	return zf.copyLogger(zf.template, params)
}

func (zf zerologTemplate) CreateNewLogger(params logadapter.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
	output := zf.createOutputWrapper(params.Config, params.Reqs)
	return zf.createNewLogger(output, params, zf.template)
}

/* ========================================= */

var _ zerolog.LevelWriter = &zerologAdapterOutput{}

type zerologAdapterOutput struct {
	logcommon.LoggerOutput
}

func (z zerologAdapterOutput) WriteLevel(level zerolog.Level, b []byte) (int, error) {
	return z.LoggerOutput.LogLevelWrite(FromZerologLevel(level), b)
}

func (z zerologAdapterOutput) Write(b []byte) (int, error) {
	panic("unexpected") // zerolog writes only to WriteLevel
}

/* ========================================= */

var _ zerolog.LevelWriter = &zerologAdapterLLOutput{}

type zerologAdapterLLOutput struct {
	logcommon.LoggerOutput
}

func (z zerologAdapterLLOutput) WriteLevel(level zerolog.Level, b []byte) (int, error) {
	return z.LoggerOutput.LowLatencyWrite(FromZerologLevel(level), b)
}

func (z zerologAdapterLLOutput) Write(b []byte) (int, error) {
	panic("unexpected") // zerolog writes only to WriteLevel
}

/* ========================================= */

func newDynFieldsHook(dynFields logcommon.DynFieldMap) zerolog.Hook {
	return dynamicFieldsHook{dynFields}
}

type dynamicFieldsHook struct {
	dynFields logcommon.DynFieldMap
}

func (v dynamicFieldsHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	for k, fn := range v.dynFields {
		if fn == nil {
			continue
		}
		vv := fn()
		if vv == nil {
			continue
		}
		e.Interface(k, vv)
	}
}
