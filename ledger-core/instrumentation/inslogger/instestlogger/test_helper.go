// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package instestlogger

import (
	"fmt"
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
	if adapterOverride != "" {
		logCfg.Adapter = adapterOverride
	} else {
		readTestLogConfig(&logCfg, &echoAllCfg, &emuMarks)
	}

	outputType, err := inslogger.ParseOutput(logCfg.OutputType)
	if err != nil {
		panic(err)
	}
	isConsoleOutput := outputType.IsConsole()

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
		if emuMarks {
			emulateTestJSON(out, target)
		}
		target = ConvertJSONTestingOutput(target)
		echoTo = ConvertJSONConsoleOutput(echoTo)
	case logcommon.TextFormat:
		if emuMarks {
			emulateTestText(out, target)
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

type tb interface {
	Name() string
	Cleanup(func())
	Failed() bool
	Skipped() bool
}

func emulateTestText(out io.Writer, tLog interface{}) {
	t, ok := tLog.(tb)
	if !ok {
		return
	}

	startedAt := time.Now()
	_, _ = fmt.Fprintln(out, `=== RUN  `, t.Name())
	t.Cleanup(func() {
		d := time.Since(startedAt)
		resName := ""
		switch {
		case t.Failed():
			resName = "FAIL"
		case t.Skipped():
			resName = "SKIP"
		default:
			resName = "PASS"
		}
		_, _ = fmt.Fprintf(out, `--- %s: %s (%.2fs)\n`, resName, t.Name(), d.Seconds())
	})
}

func emulateTestJSON(out io.Writer, tLog interface{}) {
	t, ok := tLog.(tb)
	if !ok {
		return
	}

	packageName := "" // ? how to get it ?
	const timestampFormat = "2006-01-02T15:04:05.000000000Z"

	startedAt := time.Now()
	_, _ = fmt.Fprintf(out, `{"Time":"%s","Action":"run","Package":"%s","Test":"%s"}`+"\n",
		startedAt.Format(timestampFormat),
		packageName, t.Name())

	// TODO Is emulation of text output needed?
	// _, _ = fmt.Fprintf(out, `{"Time":"%s","Action":"output","Package":"%s","Test":"%s","Output":"=== RUN   %s\n"}`,
	// 	time.Now().Format(timestampFormat),
	// 	packageName, testName)

	t.Cleanup(func() {
		d := time.Since(startedAt)
		resName := ""
		switch {
		case t.Failed():
			resName = "fail"
		case t.Skipped():
			resName = "skip"
		default:
			resName = "pass"
		}
		_, _ = fmt.Fprintf(out, `{"Time":"%s","Action":"%s","Package":"%s","Test":"%s","Elapsed":%.3f}`+"\n",
			startedAt.Format(timestampFormat), resName,
			packageName, t.Name(), d.Seconds())
	})
}
