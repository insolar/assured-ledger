// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package instestlogger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewTestLogger(target logcommon.TestingLogger, suppressTestError bool) log.Logger {
	if !suppressTestError {
		return NewTestLoggerWithErrorFilter(target, nil)
	}

	return NewTestLoggerWithErrorFilter(target, func(string) bool {	return false })
}

func NewTestLoggerWithErrorFilter(target logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc) log.Logger {
	return newTestLoggerExt(target, filterFn, inslogger.DefaultTestLogConfig(), false)
}

func newTestLoggerExt(target logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc, logCfg configuration.Log, ignoreCmd bool) log.Logger {
	if target == nil {
		panic("illegal value")
	}

	echoAll := false
	emuMarks := false
	prettyPrintJSON := false

	if !ignoreCmd {
		readTestLogConfig(&logCfg, &echoAll, &emuMarks, &prettyPrintJSON)
	}

	outputType, err := inslogger.ParseOutput(logCfg.OutputType)
	if err != nil {
		panic(err)
	}

	isConsoleOutput := outputType.IsConsole()
	if isConsoleOutput {
		prettyPrintJSON = false // is only needed for file output
	}

	l, err := inslogger.NewLogBuilder(logCfg)
	if err != nil {
		panic(err)
	}

	var echoTo io.Writer
	if echoAll && !isConsoleOutput {
		echoTo = os.Stderr
	}

	out := l.GetOutput()

	name := ""
	if namer, ok := target.(interface{ Name() string }); ok {
		name = namer.Name()
	}

	switch l.GetFormat() {
	case logcommon.JSONFormat:
		if prettyPrintJSON {
			if emuMarks {
				emulateTestText(out, target, time.Now)
			}
			out = ConvertJSONConsoleOutput(out)
		} else if emuMarks {
			emulateTestJSON(out, target, time.Now)
		}
		target = ConvertJSONTestingOutput(target)
		echoTo = ConvertJSONConsoleOutput(echoTo)
	case logcommon.TextFormat:
		if emuMarks {
			emulateTestText(out, target, time.Now)
		}
	default:
		panic(throw.Unsupported())
	}

	if name != "" {
		l = l.WithField("testname", name)
	}

	return l.WithMetrics(logcommon.LogMetricsResetMode | logcommon.LogMetricsTimestamp).
		WithCaller(logcommon.CallerField).
		WithOutput(&logcommon.TestingLoggerOutput{
			Testing: target,
			Output: out,
			EchoTo: echoTo,
			ErrorFilterFn: filterFn}).
		MustBuild()
}

func SetTestOutputWithErrorFilter(target logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc) {
	global.SetLogger(NewTestLoggerWithErrorFilter(target, filterFn))
}

// deprecated
func SetTestOutputWithIgnoreAllErrors(target logcommon.TestingLogger) {
	global.SetLogger(NewTestLogger(target, true))
}

func SetTestOutput(target logcommon.TestingLogger) {
	global.SetLogger(NewTestLogger(target, false))
}

func SetTestOutputWithCfg(target logcommon.TestingLogger, cfg configuration.Log) {
	global.SetLogger(newTestLoggerExt(target, nil, cfg, false))
}

func SetTestOutputWithStub(suppressLogError bool) (teardownFn func(pass bool)) {
	emu := &stubT{}
	global.SetLogger(NewTestLogger(emu, suppressLogError))
	return emu.cleanup
}

// TestContext returns context with initalized log field "testname" equal t.Name() value.
func TestContext(target logcommon.TestingLogger) context.Context {
	if !global.IsInitialized() {
		SetTestOutput(target)
	}
	// ctx, _ := inslogger.WithField(context.Background(), "testname", t.Name())
	return context.Background()
}

