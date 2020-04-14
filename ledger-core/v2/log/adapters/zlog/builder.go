// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package zlog

import (
	"errors"
	"io"

	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

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

func NewFactory() logcommon.Factory {
	return zerologFactory{}
}

/* ============================ */

var _ logcommon.Factory = zerologFactory{}

type zerologFactory struct {
	writeDelayPreferTrim bool
}

func (zf zerologFactory) GetGlobalLogAdapter() logcommon.GlobalLogAdapter {
	return zerologGlobalAdapter
}

func (zf zerologFactory) PrepareBareOutput(output logcommon.BareOutput, metrics *logcommon.MetricsHelper, config logcommon.BuildConfig) (io.Writer, error) {
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

func (zf zerologFactory) createNewLogger(output zerolog.LevelWriter, params logcommon.NewLoggerParams, template *zerologAdapter,
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
		case template != nil && params.Reqs&logcommon.RequiresParentDynFields != 0 && len(template.dynFields) > 0:
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

	// NB! only add hooks below this line, DON'T set the context as it can be replaced below

	if params.Config.Instruments.MetricsMode&logcommon.LogMetricsTimestamp != 0 {
		lc = lc.Timestamp()
	}

	if callerMode == logcommon.CallerField {
		lc = lc.CallerWithSkipFrameCount(skipFrames)
	}

	if template != nil && params.Reqs&logcommon.RequiresParentCtxFields != 0 {
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

func (zf zerologFactory) copyLogger(template *zerologAdapter, params logcommon.CopyLoggerParams) logcommon.EmbeddedLogger {

	if params.Reqs&logcommon.RequiresParentDynFields == 0 {
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

		if params.Reqs&logcommon.RequiresParentCtxFields == 0 {
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

func (zf zerologFactory) createOutputWrapper(config logcommon.Config, reqs logcommon.FactoryRequirementFlags) zerolog.LevelWriter {
	if reqs&logcommon.RequiresLowLatency != 0 {
		return zerologAdapterLLOutput{config.LoggerOutput}
	}
	return zerologAdapterOutput{config.LoggerOutput}
}

func (zf zerologFactory) CreateNewLogger(params logcommon.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
	output := zf.createOutputWrapper(params.Config, params.Reqs)
	return zf.createNewLogger(output, params, nil)
}

func (zf zerologFactory) CanReuseMsgBuffer() bool {
	// zerolog does recycling of []byte buffers
	return false
}

/* ============================ */

var _ logcommon.Template = &zerologTemplate{}

type zerologTemplate struct {
	zerologFactory
	template *zerologAdapter
}

func (zf zerologTemplate) GetTemplateLevel() logcommon.Level {
	return FromZerologLevel(zf.template.logger.GetLevel())
}

func (zf zerologTemplate) GetLoggerOutput() logcommon.LoggerOutput {
	return zf.template.GetLoggerOutput()
}

func (zf zerologTemplate) GetTemplateConfig() logcommon.Config {
	return *zf.template.config
}

func (zf zerologTemplate) CopyTemplateLogger(params logcommon.CopyLoggerParams) logcommon.EmbeddedLogger {
	return zf.copyLogger(zf.template, params)
}

func (zf zerologTemplate) CreateNewLogger(params logcommon.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
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
