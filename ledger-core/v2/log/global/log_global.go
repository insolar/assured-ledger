// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package global

import (
	stdlog "log"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logwriter"

	"github.com/pkg/errors"
)

var globalLogger = struct {
	mutex   sync.RWMutex
	output  logwriter.ProxyLoggerOutput
	logger  *log.Logger
	defInit func() (log.LoggerBuilder, error)
}{
	defInit: _initDefaultWithBilog,
}

func _initDefaultWithBilog() (log.LoggerBuilder, error) {
	zc := logcommon.Config{}

	var err error
	zc.BareOutput, err = logoutput.OpenLogBareOutput(logoutput.StdErrOutput, "")
	if err != nil {
		return log.LoggerBuilder{}, err
	}
	if zc.BareOutput.Writer == nil {
		panic("output is nil")
	}

	zc.Output = logcommon.OutputConfig{
		Format: logcommon.TextFormat,
	}
	zc.MsgFormat = logfmt.GetDefaultLogMsgFormatter()
	zc.Instruments.CallerMode = logcommon.CallerField
	zc.Instruments.MetricsMode = logcommon.LogMetricsWriteDelayField | logcommon.LogMetricsTimestamp

	b := log.NewBuilder(bilog.NewFactory(nil, false), zc, log.InfoLevel)
	b = b.WithField("loginstance", "global_default")
	return b, nil
}

func initDefault() {
	switch b, err := globalLogger.defInit(); {
	case err != nil:
		panic(errors.Wrap(err, "default global logger initializer has failed"))
	case b.IsZero():
		panic("default global logger initializer has returned zero builder")
	default:
		switch logger, err := b.Build(); {
		case err != nil:
			panic(errors.Wrap(err, "default global logger builder has failed"))
		default:
			globalLogger.logger = &logger
		}
	}
}

func Logger() log.Logger {
	globalLogger.mutex.RLock()
	l := globalLogger.logger
	globalLogger.mutex.RUnlock()

	if l != nil {
		return *l
	}

	globalLogger.mutex.Lock()
	defer globalLogger.mutex.Unlock()
	if globalLogger.logger == nil {
		initDefault()
	}

	return *globalLogger.logger
}

func CopyForContext() log.Logger {
	return Logger()
}

func SaveLogger() func() {
	return SaveLoggerAndFilter(false)
}

func SaveLoggerAndFilter(includeFilter bool) func() {
	globalLogger.mutex.RLock()
	defer globalLogger.mutex.RUnlock()

	loggerCopy := globalLogger.logger
	outputCopy := globalLogger.output.GetTarget()
	globalAdapterCopy := getGlobalLogAdapter()

	globalFilterCopy := log.MinLevel
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
		// as we use global.Logger() - the copy will follow any redirection made on the global.Logger()
		tickLogger, err := Logger().Copy().WithCaller(logcommon.NoCallerField).Build()
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

func TrySetDefaultInitializer(initFn func() (log.LoggerBuilder, error)) bool {
	if initFn == nil {
		panic("illegal value")
	}
	globalLogger.mutex.Lock()
	defer globalLogger.mutex.Unlock()

	globalLogger.defInit = initFn
	return globalLogger.logger == nil
}

func setLogger(logger log.Logger) error {
	eb := logger.Embeddable().Copy()
	adapter := eb.GetGlobalLogAdapter()
	if adapter == nil {
		return errors.New("log adapter doesn't support global filter")
	}

	loggerLevel := eb.GetTemplateLevel()
	switch lastAdapter := getGlobalLogAdapter(); {
	case lastAdapter == adapter:
		//
	case lastAdapter != nil:
		adapter.SetGlobalLoggerFilter(lastAdapter.GetGlobalLoggerFilter())
	default:
		lvl := eb.GetTemplateLevel()
		adapter.SetGlobalLoggerFilter(lvl)
		loggerLevel = logcommon.MinLevel
	}

	b := log.NewBuilderWithTemplate(eb, loggerLevel)

	output := eb.GetLoggerOutput()
	b = b.WithOutput(&globalLogger.output)

	logger, err := b.Build()
	if err != nil {
		return err
	}

	setGlobalLogAdapter(adapter)
	globalLogger.logger = &logger

	globalLogger.output.SetTarget(output)
	return nil
}

func SetLogger(logger log.Logger) {
	globalLogger.mutex.Lock()
	defer globalLogger.mutex.Unlock()

	if err := setLogger(logger); err != nil {
		stdlog.Println("warning: unable to update global logger, ", err)
		panic("unable to update global logger")
	}
}

// SetTextLevel lets log level for global logger
func SetTextLevel(level string) error {
	lvl, err := log.ParseLevel(level)
	if err != nil {
		return err
	}

	SetLevel(lvl)
	return nil
}

func SetLevel(level log.Level) {
	globalLogger.mutex.Lock()
	defer globalLogger.mutex.Unlock()

	if globalLogger.logger == nil {
		initDefault()
	}

	b := globalLogger.logger.Copy()
	if b.GetLogLevel() == level {
		return
	}

	logger := b.WithLevel(level).MustBuild()
	globalLogger.logger = &logger
}

func SetFilter(level log.Level) error {
	if ga := getGlobalLogAdapter(); ga != nil {
		ga.SetGlobalLoggerFilter(level)
		return nil
	}
	return errors.New("not supported")
}

func GetFilter() log.Level {
	if ga := getGlobalLogAdapter(); ga != nil {
		return ga.GetGlobalLoggerFilter()
	}
	return log.MinLevel
}

//func EnforceOutput(outFn func())

/*
We use EmbeddedLog functions here to avoid SkipStackFrame corrections
*/

func g() logcommon.EmbeddedLogger {
	return Logger().Embeddable()
}

func Event(level log.Level, args ...interface{}) {
	if fn := g().NewEvent(level); fn != nil {
		fn(args)
	}
}

func Eventf(level log.Level, fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(level); fn != nil {
		fn(fmt, args)
	}
}

func Eventm(level log.Level, arg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := g().NewEventStruct(level); fn != nil {
		fn(arg, fields)
	}
}

func Debug(args ...interface{}) {
	if fn := g().NewEvent(log.DebugLevel); fn != nil {
		fn(args)
	}
}

func Debugf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(log.DebugLevel); fn != nil {
		fn(fmt, args)
	}
}

func Debugm(arg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := g().NewEventStruct(log.DebugLevel); fn != nil {
		fn(arg, fields)
	}
}

func Info(args ...interface{}) {
	if fn := g().NewEvent(log.InfoLevel); fn != nil {
		fn(args)
	}
}

func Infof(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(log.InfoLevel); fn != nil {
		fn(fmt, args)
	}
}

func Infom(arg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := g().NewEventStruct(log.InfoLevel); fn != nil {
		fn(arg, fields)
	}
}

func Warn(args ...interface{}) {
	if fn := g().NewEvent(log.WarnLevel); fn != nil {
		fn(args)
	}
}

func Warnf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(log.WarnLevel); fn != nil {
		fn(fmt, args)
	}
}

func Warnm(arg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := g().NewEventStruct(log.WarnLevel); fn != nil {
		fn(arg, fields)
	}
}

func Error(args ...interface{}) {
	if fn := g().NewEvent(log.ErrorLevel); fn != nil {
		fn(args)
	}
}

func Errorm(arg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := g().NewEventStruct(log.ErrorLevel); fn != nil {
		fn(arg, fields)
	}
}

func Errorf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(log.ErrorLevel); fn != nil {
		fn(fmt, args)
	}
}

func Fatal(args ...interface{}) {
	if fn := g().NewEvent(log.FatalLevel); fn != nil {
		fn(args)
	}
}

func Fatalm(arg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := g().NewEventStruct(log.FatalLevel); fn != nil {
		fn(arg, fields)
	}
}

func Fatalf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(log.FatalLevel); fn != nil {
		fn(fmt, args)
	}
}

func Panic(args ...interface{}) {
	if fn := g().NewEvent(log.PanicLevel); fn != nil {
		fn(args)
	}
}

func Panicm(arg interface{}, fields ...logfmt.LogFieldMarshaller) {
	if fn := g().NewEventStruct(log.PanicLevel); fn != nil {
		fn(arg, fields)
	}
}

func Panicf(fmt string, args ...interface{}) {
	if fn := g().NewEventFmt(log.PanicLevel); fn != nil {
		fn(fmt, args)
	}
}

func Flush() {
	g().EmbeddedFlush("Global logger flush")
}
