// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package instestlogger

import (
	"io"
	"os"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewTestLogger(target logcommon.TestingLogger, suppressTestError bool) log.Logger {
	return newTestLoggerExt(target, suppressTestError, false, "")
}

func newTestLoggerExt(target logcommon.TestingLogger, suppressTestError, echoAll bool, adapterOverride string) log.Logger {
	if target == nil {
		panic("illegal value")
	}

	logCfg := inslogger.DefaultTestLogConfig()

	echoAllCfg := false
	emuMarks := false
	prettyPrintJSON := false
	if adapterOverride != "" {
		logCfg.Adapter = adapterOverride
	} else {
		readTestLogConfig(&logCfg, &echoAllCfg, &emuMarks, &prettyPrintJSON)
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
	if (echoAll || echoAllCfg) && !isConsoleOutput {
		echoTo = os.Stderr
	}

	out := l.GetOutput()

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

	return l.WithMetrics(logcommon.LogMetricsResetMode | logcommon.LogMetricsTimestamp).
		WithCaller(logcommon.CallerField).
		WithOutput(&logcommon.TestingLoggerOutput{
			Testing: target,
			Output: out,
			EchoTo: echoTo,
			SuppressTestError: suppressTestError}).
		MustBuild()
}


func SetTestOutput(target logcommon.TestingLogger, suppressLogError bool) {
	global.SetLogger(NewTestLogger(target, suppressLogError))
}
