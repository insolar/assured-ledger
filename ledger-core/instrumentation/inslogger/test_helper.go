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
	return NewTestLoggerExt(target, suppressTestError, "bilog")
}

func NewTestLoggerExt(target logcommon.TestingLogger, suppressTestError bool, adapter string) log.Logger {
	if target == nil {
		panic("illegal value")
	}

	logCfg := defaultLogConfig()
	if adapter != "" {
		logCfg.Adapter = adapter
	}
	logCfg.Level = logcommon.DebugLevel.String()
	logCfg.Formatter = logcommon.TextFormat.String()

	l, err := newLogger(logCfg)
	if err != nil {
		panic(err)
	}

	return l.WithMetrics(logcommon.LogMetricsResetMode | logcommon.LogMetricsTimestamp).
		WithCaller(logcommon.CallerField).
		WithOutput(&logcommon.TestingLoggerOutput{Testing: target, Output: l.GetOutput(), SuppressTestError: suppressTestError}).
		MustBuild()
}

func SetTestOutput(target logcommon.TestingLogger, suppressLogError bool) {
	global.SetLogger(NewTestLogger(target, suppressLogError))
}
