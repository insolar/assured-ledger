package instestlogger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/prettylog"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logwriter"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func newTestLogger(target logcommon.TestingLogger, suppressTestError bool) log.Logger {
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

	isQuiet := false
	isConsoleOutput := outputType.IsConsole()
	if isConsoleOutput {
		// only needed for file output
		prettyPrintJSON = false
		emuMarks = false

		if _, ok := target.(interface{ StartTimer() }); ok {
			// this is testing.B
			isQuiet = true
		}
	}

	l, err := inslogger.NewLogBuilder(logCfg)
	if err != nil {
		panic(err)
	}

	var echoTo io.Writer
	if echoAll && !isConsoleOutput {
		echoTo = os.Stderr
	}

	if emuMarks && isGlobalBasedOn(target) {
		emuMarks = false // avoid multiple marks per test
	}

	name := ""
	if namer, ok := target.(interface{ Name() string }); ok {
		name = namer.Name()
	}

	var out io.Writer
	if !isQuiet {
		out = l.GetOutput()
		switch l.GetFormat() {
		case logcommon.JSONFormat:
			if prettyPrintJSON {
				if emuMarks {
					emulateTestText(out, target, time.Now)
				}
				out = prettylog.ConvertJSONConsoleOutput(out)
			} else if emuMarks {
				emulateTestJSON(out, target, time.Now)
			}
			target = prettylog.ConvertJSONTestingOutput(target)
			echoTo = prettylog.ConvertJSONConsoleOutput(echoTo)
		case logcommon.TextFormat:
			if emuMarks {
				emulateTestText(out, target, time.Now)
			}
		default:
			panic(throw.Unsupported())
		}
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

func isGlobalBasedOn(target logcommon.TestingLogger) bool {
	if !global.IsInitialized() {
		return false
	}
	// check global to avoid multiple marks for the same testing.T
	lOut := global.Logger().Embeddable().Copy().GetLoggerOutput()

	if plo, ok := lOut.(*logwriter.ProxyLoggerOutput); ok {
		lOut = plo.GetTarget()
	}

	tlo, ok := lOut.(*logcommon.TestingLoggerOutput)
	return ok && logcommon.IsBasedOn(tlo.Testing, target)
}

func SetTestOutputWithErrorFilter(target logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc) {
	global.SetLogger(NewTestLoggerWithErrorFilter(target, filterFn))
}

// deprecated
func SetTestOutputWithIgnoreAllErrors(target logcommon.TestingLogger) {
	global.SetLogger(newTestLogger(target, true))
}

func SetTestOutput(target logcommon.TestingLogger) {
	global.SetLogger(newTestLogger(target, false))
}

func SetTestOutputWithCfg(target logcommon.TestingLogger, cfg configuration.Log) {
	global.SetLogger(newTestLoggerExt(target, nil, cfg, false))
}

func SetTestOutputWithStub() (teardownFn func(pass bool)) {
	emu := &stubT{}
	global.SetLogger(newTestLogger(emu, false))
	return emu.cleanup
}

// TestContext returns context with initialized log field "testname" equal t.Name() value.
func TestContext(target logcommon.TestingLogger) context.Context {
	if !global.IsInitialized() {
		SetTestOutput(target)
	}
	// ctx, _ := inslogger.WithField(context.Background(), "testname", t.Name())
	return context.Background()
}

