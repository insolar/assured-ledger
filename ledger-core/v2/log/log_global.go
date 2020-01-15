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

package log

import (
	stdlog "log"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/zlogadapter"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/critlog"
)

var globalLogger = struct {
	mutex   sync.RWMutex
	output  critlog.ProxyLoggerOutput
	logger  logcommon.Logger
	defInit func() (logcommon.LoggerBuilder, error)
}{
	defInit: _initDefaultWithZlog,
}

func _initDefaultWithZlog() (logcommon.LoggerBuilder, error) {
	zc := logadapter.Config{}

	var err error
	zc.BareOutput, err = logadapter.OpenLogBareOutput(logadapter.StdErrOutput, "")
	if err != nil {
		return nil, err
	}
	if zc.BareOutput.Writer == nil {
		panic("output is nil")
	}

	zc.Output = logadapter.OutputConfig{
		Format: logcommon.TextFormat,
	}
	zc.MsgFormat = logadapter.GetDefaultLogMsgFormatter()
	zc.Instruments.SkipFrameCountBaseline = zlogadapter.ZerologSkipFrameCount
	zc.Instruments.CallerMode = logcommon.CallerField
	zc.Instruments.MetricsMode = logcommon.LogMetricsWriteDelayField

	b := zlogadapter.NewBuilder(zc, logcommon.InfoLevel)
	b = b.WithField("loginstance", "global_default")
	return b, nil
}

func initDefaultGlobalLogger() {
	switch b, err := globalLogger.defInit(); {
	case err != nil:
		panic(errors.Wrap(err, "default global logger initializer has failed"))
	case b == nil:
		panic("default global logger initializer has returned nil")
	default:
		switch logger, err := b.Build(); {
		case err != nil:
			panic(errors.Wrap(err, "default global logger builder has failed"))
		case logger == nil:
			panic("default global logger builder has returned nil")
		default:
			globalLogger.logger = logger
		}
	}
}

func GlobalLogger() logcommon.Logger {
	globalLogger.mutex.RLock()
	l := globalLogger.logger
	globalLogger.mutex.RUnlock()

	if l != nil {
		return l
	}

	globalLogger.mutex.Lock()
	defer globalLogger.mutex.Unlock()
	if globalLogger.logger == nil {
		initDefaultGlobalLogger()
	}

	return globalLogger.logger
}

func CopyGlobalLoggerForContext() logcommon.Logger {
	return GlobalLogger()
}

func SaveGlobalLogger() func() {
	return SaveGlobalLoggerAndFilter(false)
}

func SaveGlobalLoggerAndFilter(includeFilter bool) func() {
	globalLogger.mutex.RLock()
	defer globalLogger.mutex.RUnlock()

	loggerCopy := globalLogger.logger
	outputCopy := globalLogger.output.GetTarget()
	globalAdapterCopy := getGlobalLogAdapter()

	globalFilterCopy := logcommon.MinLevel
	if includeFilter && globalAdapterCopy != nil {
		globalFilterCopy = globalAdapterCopy.GetGlobalLoggerFilter()
	}

	return func() {
		globalLogger.mutex.Lock()
		defer globalLogger.mutex.Unlock()

		globalLogger.logger = loggerCopy
		globalLogger.output.SetTarget(outputCopy)

		setGlobalLogAdapter(globalAdapterCopy)
		if includeFilter && globalAdapterCopy != nil {
			globalAdapterCopy.SetGlobalLoggerFilter(globalFilterCopy)
		}
	}
}

var globalTickerOnce sync.Once

func InitTicker() {
	globalTickerOnce.Do(func() {
		// as we use GlobalLogger() - the copy will follow any redirection made on the GlobalLogger()
		tickLogger, err := GlobalLogger().Copy().WithCaller(logcommon.NoCallerField).Build()
		if err != nil {
			panic(err)
		}

		go func() {
			for {
				// Tick between seconds
				time.Sleep(time.Second - time.Since(time.Now().Truncate(time.Second)))
				tickLogger.Debug("Logger tick")
			}
		}()
	})
}

func TrySetGlobalLoggerInitializer(initFn func() (logcommon.LoggerBuilder, error)) bool {
	if initFn == nil {
		panic("illegal value")
	}
	globalLogger.mutex.Lock()
	defer globalLogger.mutex.Unlock()

	globalLogger.defInit = initFn
	return globalLogger.logger == nil
}

func setGlobalLogger(b logcommon.LoggerBuilder) error {
	var adapter logcommon.GlobalLogAdapter
	if f, ok := b.(logcommon.GlobalLogAdapterFactory); ok {
		adapter = f.GetGlobalLogAdapter()
	} else {
		return errors.New("log adapter doesn't support global filter")
	}

	switch lastAdapter := getGlobalLogAdapter(); {
	case lastAdapter == adapter:
		//
	case lastAdapter != nil:
		adapter.SetGlobalLoggerFilter(lastAdapter.GetGlobalLoggerFilter())
	default:
		lvl := b.GetLogLevel()
		adapter.SetGlobalLoggerFilter(lvl)
		b = b.WithLevel(logcommon.MinLevel)
	}

	output := b.(logcommon.LoggerOutputGetter).GetLoggerOutput()
	b = b.WithOutput(&globalLogger.output)

	logger, err := b.Build()
	switch {
	case err != nil:
		return err
	case logger == nil:
		return errors.New("log adapter builder has returned nil")
	}

	setGlobalLogAdapter(adapter)
	globalLogger.logger = logger

	globalLogger.output.SetTarget(output)
	return nil
}

func SetGlobalLogger(logger logcommon.Logger) {
	if logger == nil {
		panic("illegal value")
	}

	globalLogger.mutex.Lock()
	defer globalLogger.mutex.Unlock()

	if globalLogger.logger == logger {
		return
	}
	if err := setGlobalLogger(logger.Copy()); err != nil {
		stdlog.Println("warning: unable to update global logger, ", err)
		panic("unable to update global logger")
	}
}

// SetLevel lets log level for global logger
func SetLevel(level string) error {
	lvl, err := logcommon.ParseLevel(level)
	if err != nil {
		return err
	}

	SetLogLevel(lvl)
	return nil
}

func SetLogLevel(level logcommon.LogLevel) {
	globalLogger.mutex.Lock()
	defer globalLogger.mutex.Unlock()

	if globalLogger.logger == nil {
		initDefaultGlobalLogger()
	}

	globalLogger.logger = globalLogger.logger.Level(level)
}

func SetGlobalLevelFilter(level logcommon.LogLevel) error {
	if ga := getGlobalLogAdapter(); ga != nil {
		ga.SetGlobalLoggerFilter(level)
		return nil
	}
	return errors.New("not supported")
}

func GetGlobalLevelFilter() logcommon.LogLevel {
	if ga := getGlobalLogAdapter(); ga != nil {
		return ga.GetGlobalLoggerFilter()
	}
	return logcommon.MinLevel
}

/*
We use EmbeddedLog functions here to avoid SkipStackFrame corrections
*/

func g() logcommon.EmbeddedLogger {
	return GlobalLogger().Embeddable()
}

func Event(level logcommon.LogLevel, args ...interface{}) {
	if fn := g().NewEvent(level); fn != nil {
		fn(args)
	}
}

func Eventf(level logcommon.LogLevel, fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(level); fn != nil {
		fn(fmt, args)
	}
}

func Debug(args ...interface{}) {
	if fn := g().NewEvent(logcommon.DebugLevel); fn != nil {
		fn(args)
	}
}

func Debugf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(logcommon.DebugLevel); fn != nil {
		fn(fmt, args)
	}
}

func Info(args ...interface{}) {
	if fn := g().NewEvent(logcommon.InfoLevel); fn != nil {
		fn(args)
	}
}

func Infof(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(logcommon.InfoLevel); fn != nil {
		fn(fmt, args)
	}
}

func Warn(args ...interface{}) {
	if fn := g().NewEvent(logcommon.WarnLevel); fn != nil {
		fn(args)
	}
}

func Warnf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(logcommon.WarnLevel); fn != nil {
		fn(fmt, args)
	}
}

func Error(args ...interface{}) {
	if fn := g().NewEvent(logcommon.ErrorLevel); fn != nil {
		fn(args)
	}
}

func Errorf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(logcommon.ErrorLevel); fn != nil {
		fn(fmt, args)
	}
}

func Fatal(args ...interface{}) {
	if fn := g().NewEvent(logcommon.FatalLevel); fn != nil {
		fn(args)
	}
}

func Fatalf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(logcommon.FatalLevel); fn != nil {
		fn(fmt, args)
	}
}

func Panic(args ...interface{}) {
	if fn := g().NewEvent(logcommon.PanicLevel); fn != nil {
		fn(args)
	}
}

func Panicf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(logcommon.PanicLevel); fn != nil {
		fn(fmt, args)
	}
}

func Flush() {
	g().EmbeddedFlush("Global logger flush")
}
