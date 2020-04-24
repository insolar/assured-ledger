// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package zlog

import (
	"os"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"

	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

type zerologMarshaller struct {
	event *zerolog.Event
	*logfmt.MsgFormatConfig
}

func (m zerologMarshaller) AddIntField(key string, v int64, fFmt logfmt.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Int64(key, v)
	}
}

func (m zerologMarshaller) AddUintField(key string, v uint64, fFmt logfmt.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Uint64(key, v)
	}
}

func (m zerologMarshaller) AddBoolField(key string, v bool, fFmt logfmt.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Bool(key, v)
	}
}

func (m zerologMarshaller) AddFloatField(key string, v float64, fFmt logfmt.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Float64(key, v)
	}
}

func (m zerologMarshaller) AddComplexField(key string, v complex128, fFmt logfmt.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Str(key, m.Sformat(v))
	}
}

func (m zerologMarshaller) AddStrField(key string, v string, fFmt logfmt.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Str(key, v)
	}
}

func (m zerologMarshaller) AddIntfField(key string, v interface{}, fFmt logfmt.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, m.Sformatf(fFmt.Fmt, v))
	} else {
		m.event.Interface(key, v)
	}
}

func (m zerologMarshaller) AddTimeField(key string, v time.Time, fFmt logfmt.LogFieldFormat) {
	if fFmt.HasFmt {
		m.event.Str(key, v.Format(fFmt.Fmt))
	} else {
		// NB! here we ignore MsgFormatConfig.TimeFmt as it should be set to zerolog.TimeFieldFormat
		m.event.Time(key, v)
	}
}

func (m zerologMarshaller) AddRawJSONField(key string, v interface{}, fFmt logfmt.LogFieldFormat) {
	m.event.RawJSON(key, []byte(m.Sformatf(fFmt.Fmt, v)))
}

func (m zerologMarshaller) AddErrorField(msg string, stack throw.StackTrace, severity throw.Severity, hasPanic bool) {
	if msg != "" {
		m.event.Str(zerolog.ErrorFieldName, msg)
	}
	if stack != nil {
		m.event.Str(zerolog.ErrorStackFieldName, stack.StackTraceAsText())
	}
}

/* ============================ */

var _ logcommon.EmbeddedLogger = &zerologAdapter{}
var _ logcommon.EmbeddedLoggerOptional = &zerologAdapter{}

type zerologAdapter struct {
	logger    zerolog.Logger
	dynFields logcommon.DynFieldMap
	config    *logcommon.Config
}

func (z *zerologAdapter) WithFields(fields map[string]interface{}) logcommon.EmbeddedLogger {
	zCtx := z.logger.With()
	for key, value := range fields {
		zCtx = zCtx.Interface(key, value)
	}

	zCopy := *z
	zCopy.logger = zCtx.Logger()
	return &zCopy
}

func (z *zerologAdapter) WithField(key string, value interface{}) logcommon.EmbeddedLogger {
	zCopy := *z
	zCopy.logger = z.logger.With().Interface(key, value).Logger()
	return &zCopy
}

func (z *zerologAdapter) newEvent(level logcommon.Level) *zerolog.Event {
	m := getLevelMapping(level)
	z.config.Metrics.OnNewEvent(m.metrics, level)
	event := m.fn(&z.logger)
	if event == nil {
		return nil
	}
	z.config.Metrics.OnFilteredEvent(m.metrics, level)
	return event
}

func (z *zerologAdapter) NewEventStruct(level logcommon.Level) func(interface{}, []logfmt.LogFieldMarshaller) {
	event := z.newEvent(level)
	if event == nil {
		//collector := z.config.Metrics.GetMetricsCollector()
		//if collector != nil {
		//	if obj := z.config.MsgFormat.PrepareMutedLogObject(arg); obj != nil {
		//		obj.MarshalMutedLogObject(collector)
		//	}
		//}
		return nil
	}

	return func(arg interface{}, fields []logfmt.LogFieldMarshaller) {
		if level == logcommon.FatalLevel {
			defer func() {
				if recover() != nil {
					os.Exit(logoutput.LogFatalExitCode)
				}
			}()
		}

		msgStr := z.fmtEventStruct(event, arg, fields)
		event.Msg(msgStr)

		if level == logcommon.FatalLevel {
			os.Exit(logoutput.LogFatalExitCode)
		}
	}
}

func (z *zerologAdapter) NewEvent(level logcommon.Level) func(args []interface{}) {
	event := z.newEvent(level)
	if event == nil {
		return nil
	}

	return func(args []interface{}) {
		if level == logcommon.FatalLevel {
			defer func() {
				if recover() != nil {
					os.Exit(logoutput.LogFatalExitCode)
				}
			}()
		}

		msgStr := z.fmtEventArgs(event, args)
		event.Msg(msgStr)

		if level == logcommon.FatalLevel {
			os.Exit(logoutput.LogFatalExitCode)
		}
	}
}

func (z *zerologAdapter) NewEventFmt(level logcommon.Level) func(fmt string, args []interface{}) {
	event := z.newEvent(level)
	if event == nil {
		return nil
	}

	return func(fmt string, args []interface{}) {
		if level == logcommon.FatalLevel {
			defer func() {
				if recover() != nil {
					os.Exit(logoutput.LogFatalExitCode)
				}
			}()
		}

		msgStr := z.fmtEventFmt(event, fmt, args)
		event.Msg(msgStr)

		if level == logcommon.FatalLevel {
			os.Exit(logoutput.LogFatalExitCode)
		}
	}
}

func (z *zerologAdapter) fmtEventStruct(event *zerolog.Event, arg interface{}, fields []logfmt.LogFieldMarshaller) string {
	m := zerologMarshaller{event, &z.config.MsgFormat}
	for _, f := range fields {
		f.MarshalLogFields(m)
	}
	obj, msgStr := z.config.MsgFormat.FmtLogStruct(arg)
	if obj != nil {
		collector := z.config.Metrics.GetMetricsCollector()
		msgStr, _ = obj.MarshalLogObject(zerologMarshaller{event, &z.config.MsgFormat}, collector)
	}
	return msgStr
}

func (z *zerologAdapter) fmtEventArgs(event *zerolog.Event, args []interface{}) string {
	if len(args) != 1 {
		return z.config.MsgFormat.FmtLogObject(args...)
	}
	obj, msgStr := z.config.MsgFormat.FmtLogStructOrObject(args[0])
	if obj != nil {
		collector := z.config.Metrics.GetMetricsCollector()
		msgStr, _ = obj.MarshalLogObject(zerologMarshaller{event, &z.config.MsgFormat}, collector)
	}
	return msgStr
}

func (z *zerologAdapter) fmtEventFmt(_ *zerolog.Event, fmt string, args []interface{}) string {
	return z.config.MsgFormat.Sformatf(fmt, args...)
}

func (z *zerologAdapter) EmbeddedFlush(msg string) {
	if len(msg) > 0 {
		z.newEvent(logcommon.WarnLevel).Msg(msg)
	}
	_ = z.config.LoggerOutput.Flush()
}

func (z *zerologAdapter) Is(level logcommon.Level) bool {
	return z.newEvent(level) != nil
}

func (z *zerologAdapter) Copy() logcommon.EmbeddedLoggerBuilder {
	return zerologTemplate{template: z}
}

func (z *zerologAdapter) GetLoggerOutput() logcommon.LoggerOutput {
	return z.config.LoggerOutput
}

/* =========================== */

var zerologGlobalAdapter logcommon.GlobalLogAdapter = zerologGlobal{}

type zerologGlobal struct{}

func (zerologGlobal) SetGlobalLoggerFilter(level logcommon.Level) {
	zerolog.SetGlobalLevel(ToZerologLevel(level))
}

func (zerologGlobal) GetGlobalLoggerFilter() logcommon.Level {
	return FromZerologLevel(zerolog.GlobalLevel())
}
