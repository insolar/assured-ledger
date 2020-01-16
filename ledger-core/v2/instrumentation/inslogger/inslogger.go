//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package inslogger

import (
	"context"
	"errors"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

const TimestampFormat = "2006-01-02T15:04:05.000000000Z07:00"

const insolarPrefix = "github.com/insolar/assured-ledger/ledger-core/v2/"
const skipFrameBaselineAdjustment = 0

func init() {
	initZlog()

	// NB! initialize adapters' globals before the next call
	log.TrySetGlobalLoggerInitializer(func() (logcommon.LoggerBuilder, error) {
		holder := configuration.NewHolder().MustInit(false)
		logCfg := holder.Configuration.Log

		// enforce buffer-less for a non-configured logger
		logCfg.BufferSize = 0
		logCfg.LLBufferSize = -1

		return newLogger(logCfg)
	})
}

func fileLineMarshaller(file string, line int) string {
	var skip = 0
	if idx := strings.Index(file, insolarPrefix); idx != -1 {
		skip = idx + len(insolarPrefix)
	}
	return file[skip:] + ":" + strconv.Itoa(line)
}

func newLogger(cfg configuration.Log) (logcommon.LoggerBuilder, error) {
	defaults := DefaultLoggerSettings()
	pCfg, err := ParseLogConfigWithDefaults(cfg, defaults)
	if err != nil {
		return nil, err
	}

	var logBuilder logcommon.LoggerBuilder

	pCfg.SkipFrameBaselineAdjustment = skipFrameBaselineAdjustment

	msgFmt := logadapter.GetDefaultLogMsgFormatter()
	msgFmt.TimeFmt = TimestampFormat

	switch strings.ToLower(cfg.Adapter) {
	case "zerolog":
		logBuilder, err = newZerologAdapter(pCfg, msgFmt)
	default:
		return nil, errors.New("invalid logger config, unknown adapter")
	}

	switch {
	case err != nil:
		return nil, err
	case logBuilder == nil:
		return nil, errors.New("logger was not initialized")
	default:
		return logBuilder, nil
	}
}

// newLog creates a new logger with the given configuration
func NewLog(cfg configuration.Log) (logcommon.Logger, error) {
	switch b, err := newLogger(cfg); {
	case err != nil:
		return nil, err
	default:
		switch logger, err := b.Build(); {
		case err != nil:
			return nil, err
		case logger == nil:
			return nil, errors.New("logger builder has returned nil")
		default:
			return logger, nil
		}
	}
}

var loggerKey = struct{}{}

func InitNodeLogger(ctx context.Context, cfg configuration.Log, nodeRef, nodeRole string) (context.Context, logcommon.Logger) {
	inslog, err := NewLog(cfg)
	if err != nil {
		panic(err)
	}

	fields := map[string]interface{}{"loginstance": "node"}
	if nodeRef != "" {
		fields["nodeid"] = nodeRef
	}
	if nodeRole != "" {
		fields["role"] = nodeRole
	}
	inslog = inslog.WithFields(fields)

	ctx = SetLogger(ctx, inslog)
	log.SetGlobalLogger(inslog)

	return ctx, inslog
}

func TraceID(ctx context.Context) string {
	return utils.TraceID(ctx)
}

// FromContext returns logger from context.
func FromContext(ctx context.Context) logcommon.Logger {
	return getLogger(ctx)
}

// SetLogger returns context with provided insolar.Logger,
func SetLogger(ctx context.Context, l logcommon.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

func UpdateLogger(ctx context.Context, fn func(logcommon.Logger) (logcommon.Logger, error)) context.Context {
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

// SetLoggerLevel returns context with provided insolar.LogLevel and set logLevel on logger,
func WithLoggerLevel(ctx context.Context, logLevel logcommon.LogLevel) context.Context {
	if logLevel == logcommon.NoLevel {
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
func WithField(ctx context.Context, key string, value string) (context.Context, logcommon.Logger) {
	l := getLogger(ctx).WithField(key, value)
	return SetLogger(ctx, l), l
}

// WithFields returns context with logger initialized with provided fields map.
func WithFields(ctx context.Context, fields map[string]interface{}) (context.Context, logcommon.Logger) {
	l := getLogger(ctx).WithFields(fields)
	return SetLogger(ctx, l), l
}

// WithTraceField returns context with logger initialized with provided traceid value and logger itself.
func WithTraceField(ctx context.Context, traceid string) (context.Context, logcommon.Logger) {
	ctx, err := utils.SetInsTraceID(ctx, traceid)
	if err != nil {
		getLogger(ctx).WithField("backtrace", string(debug.Stack())).Error(err)
	}
	return WithField(ctx, "traceid", traceid)
}

// ContextWithTrace returns only context with logger initialized with provided traceid.
func ContextWithTrace(ctx context.Context, traceid string) context.Context {
	ctx, _ = WithTraceField(ctx, traceid)
	return ctx
}

func getLogger(ctx context.Context) logcommon.Logger {
	val := ctx.Value(loggerKey)
	if val == nil {
		return log.CopyGlobalLoggerForContext()
	}
	return val.(logcommon.Logger)
}

// TestContext returns context with initalized log field "testname" equal t.Name() value.
func TestContext(t *testing.T) context.Context {
	ctx, _ := WithField(context.Background(), "testname", t.Name())
	return ctx
}

func GetLoggerLevel(ctx context.Context) logcommon.LogLevel {
	return getLogger(ctx).Copy().GetLogLevel()
}
