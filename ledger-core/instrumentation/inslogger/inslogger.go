// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inslogger

import (
	"context"
	"errors"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/prettylog"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
)

const TimestampFormat = prettylog.TimestampFormat

const insolarPrefix = "github.com/insolar/assured-ledger/ledger-core/"
const skipFrameBaselineAdjustment = 0

func init() {
	initBilog()
	initZlog()

	// NB! initialize adapters' globals before the next call
	global.TrySetDefaultInitializer(func() (log.LoggerBuilder, error) {
		return NewLogBuilder(defaultLogConfig())
	})
}

func defaultLogConfig() configuration.Log {
	logCfg := configuration.NewLog()

	// enforce buffer-less for a non-configured logger
	logCfg.BufferSize = 0
	logCfg.LLBufferSize = -1
	return logCfg
}

func DefaultTestLogConfig() configuration.Log {
	logCfg := defaultLogConfig()
	logCfg.Level = logcommon.DebugLevel.String()
	return logCfg
}

func fileLineMarshaller(file string, line int) string {
	var skip = 0
	if idx := strings.Index(file, insolarPrefix); idx != -1 {
		skip = idx + len(insolarPrefix)
	}
	return file[skip:] + ":" + strconv.Itoa(line)
}

func NewLogBuilder(cfg configuration.Log) (log.LoggerBuilder, error) {
	defaults := DefaultLoggerSettings()
	pCfg, err := ParseLogConfigWithDefaults(cfg, defaults)
	if err != nil {
		return log.LoggerBuilder{}, err
	}

	var logBuilder log.LoggerBuilder

	pCfg.SkipFrameBaselineAdjustment = skipFrameBaselineAdjustment

	msgFmt := logfmt.GetDefaultLogMsgFormatter()
	msgFmt.TimeFmt = TimestampFormat

	switch strings.ToLower(cfg.Adapter) {
	case "zerolog":
		logBuilder, err = newZerologAdapter(pCfg, msgFmt)
	case "bilog":
		logBuilder, err = newBilogAdapter(pCfg, msgFmt)
	default:
		return log.LoggerBuilder{}, errors.New("invalid logger config, unknown adapter")
	}

	switch {
	case err != nil:
		return log.LoggerBuilder{}, err
	case logBuilder.IsZero():
		return log.LoggerBuilder{}, errors.New("logger was not initialized")
	default:
		return logBuilder, nil
	}
}

// newLog creates a new logger with the given configuration
func NewLog(cfg configuration.Log) (logger log.Logger, err error) {
	var b log.LoggerBuilder
	b, err = NewLogBuilder(cfg)
	if err == nil {
		logger, err = b.Build()
		if err == nil {
			return logger, nil
		}
	}
	return log.Logger{}, err
}

var loggerKey = struct{}{}

func InitNodeLogger(ctx context.Context, cfg configuration.Log, nodeRef, nodeRole string) (context.Context, log.Logger) {
	inslog, err := NewLog(cfg)
	if err != nil {
		panic(err)
	}

	return initNodeLogger(ctx, inslog, nodeRef, nodeRole)
}

func InitGlobalNodeLogger(ctx context.Context, cfg configuration.Log, nodeRef, nodeRole string) (context.Context, log.Logger) {
	inslog, err := NewLog(cfg)
	if err != nil {
		panic(err)
	}

	ctx, logger := initNodeLogger(ctx, inslog, nodeRef, nodeRole)
	global.SetLogger(logger)
	return ctx, logger
}

func InitNodeLoggerByGlobal(nodeRef, nodeRole string) (context.Context, log.Logger) {
	ctx, logger := initNodeLogger(context.Background(), global.Logger(), nodeRef, nodeRole)
	global.SetLogger(logger)
	return ctx, logger
}

func initNodeLogger(ctx context.Context, inslog log.Logger, nodeRef, nodeRole string) (context.Context, log.Logger) {
	fields := map[string]interface{}{"loginstance": "node"}
	if nodeRef != "" {
		fields["nodeid"] = nodeRef
	}
	if nodeRole != "" {
		fields["role"] = nodeRole
	}
	inslog = inslog.WithFields(fields)

	ctx = SetLogger(ctx, inslog)
	return ctx, inslog
}


func TraceID(ctx context.Context) string {
	return trace.ID(ctx)
}

// FromContext returns logger from context.
func FromContext(ctx context.Context) log.Logger {
	return getLogger(ctx)
}

func Clean(ctx context.Context) (context.Context, error) {
	l, wasInContext := _getLogger(ctx)
	if wasInContext {
		var err error
		l, err = l.Copy().WithoutInheritedFields().Build()
		if err != nil {
			return nil, err
		}
	}
	return SetLogger(context.Background(), l), nil
}

// SetLogger returns context with provided insolar.Logger,
func SetLogger(ctx context.Context, l log.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

func UpdateLogger(ctx context.Context, fn func(log.Logger) (log.Logger, error)) context.Context {
	lOrig := FromContext(ctx)
	lNew, err := fn(lOrig)
	if err != nil {
		panic(err)
	}
	if lOrig == lNew {
		return ctx
	}
	return SetLogger(ctx, lNew)
}

// SetLoggerLevel and set logLevel on logger and returns context with the new logger
func WithLoggerLevel(ctx context.Context, logLevel log.Level) context.Context {
	if logLevel == log.NoLevel {
		return ctx
	}
	oldLogger := FromContext(ctx)
	b := oldLogger.Copy()
	if b.GetLogLevel() == logLevel {
		return ctx
	}
	logCopy, err := b.WithLevel(logLevel).Build()
	if err != nil {
		oldLogger.Error("failed to set log level: ", err.Error())
		return ctx
	}
	return SetLogger(ctx, logCopy)
}

// WithField returns context with logger initialized with provided field's key value and logger itself.
func WithField(ctx context.Context, key string, value string) (context.Context, log.Logger) {
	l := getLogger(ctx).WithField(key, value)
	return SetLogger(ctx, l), l
}

// WithFields returns context with logger initialized with provided fields map.
func WithFields(ctx context.Context, fields map[string]interface{}) (context.Context, log.Logger) {
	l := getLogger(ctx).WithFields(fields)
	return SetLogger(ctx, l), l
}

// WithTraceField returns context with logger initialized with provided traceid value and logger itself.
func WithTraceField(ctx context.Context, traceid string) (context.Context, log.Logger) {
	ctx, err := trace.SetID(ctx, traceid)
	if err != nil {
		getLogger(ctx).WithField("backtrace", string(debug.Stack())).Error(err)
	}
	return WithField(ctx, "traceid", traceid)
}

func TraceField(traceid string) logfmt.LogField {
	return logfmt.LogField{
		Name:  "traceid",
		Value: traceid,
	}
}

// ContextWithTrace returns only context with logger initialized with provided traceid.
func ContextWithTrace(ctx context.Context, traceid string) context.Context {
	ctx, _ = WithTraceField(ctx, traceid)
	return ctx
}

func getLogger(ctx context.Context) log.Logger {
	l, _ := _getLogger(ctx)
	return l
}

func _getLogger(ctx context.Context) (log.Logger, bool) {
	val := ctx.Value(loggerKey)
	if val == nil {
		return global.CopyForContext(), false
	}
	return val.(log.Logger), true
}

func GetLoggerLevel(ctx context.Context) log.Level {
	return getLogger(ctx).Copy().GetLogLevel()
}
