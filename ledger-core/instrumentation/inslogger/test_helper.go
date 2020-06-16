// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inslogger

import (
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
)

func NewTestLogger(target logcommon.TestingLogger, suppressTestError bool) log.Logger {
	if !suppressTestError {
		return NewTestLoggerWithErrorFilter(target, nil)
	}

	return NewTestLoggerWithErrorFilter(target, func(string) bool {	return false })
}

func NewTestLoggerWithErrorFilter(target logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc) log.Logger {
	return NewTestLoggerExt(target, filterFn, "")
}

func NewTestLoggerExt(target logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc, adapter string) log.Logger {
	if target == nil {
		panic("illegal value")
	}

	logCfg := defaultLogConfig()
	if adapter != "" {
		logCfg.Adapter = adapter
	}
	logCfg.Level = logcommon.DebugLevel.String()

	l, err := newLogger(logCfg)
	if err != nil {
		panic(err)
	}

	return l.WithMetrics(logcommon.LogMetricsResetMode | logcommon.LogMetricsTimestamp).
		WithCaller(logcommon.CallerField).
		WithOutput(&logcommon.TestingLoggerOutput{Testing: target, Output: l.GetOutput(), ErrorFilterFn: filterFn}).
		MustBuild()
}

func SetTestOutput(target logcommon.TestingLogger, suppressLogError bool) {
	global.SetLogger(NewTestLogger(target, suppressLogError))
}

func SetTestOutputWithErrorFilter(target logcommon.TestingLogger, filterFn logcommon.ErrorFilterFunc) {
	global.SetLogger(NewTestLoggerWithErrorFilter(target, filterFn))
}
