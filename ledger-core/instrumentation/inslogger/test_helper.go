// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inslogger

import (
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewTestLogger(target logcommon.TestingLogger, suppressTestError bool) log.Logger {
	return newTestLoggerExt(target, suppressTestError, "")
}

func newTestLoggerExt(target logcommon.TestingLogger, suppressTestError bool, adapterOverride string) log.Logger {
	if target == nil {
		panic("illegal value")
	}

	logCfg := defaultTestLogConfig()
	if adapterOverride != "" {
		logCfg.Adapter = adapterOverride
	}

	l, err := newLogger(logCfg)
	if err != nil {
		panic(err)
	}

	switch l.GetFormat() {
	case logcommon.JSONFormat:
		target = ConvertJSONTestingOutput(target)
	case logcommon.TextFormat:
		// leave as is
	default:
		panic(throw.Unsupported())
	}

	return l.WithMetrics(logcommon.LogMetricsResetMode | logcommon.LogMetricsTimestamp).
		WithCaller(logcommon.CallerField).
		WithOutput(&logcommon.TestingLoggerOutput{Testing: target, Output: l.GetOutput(), SuppressTestError: suppressTestError}).
		MustBuild()
}

func SetTestOutput(target logcommon.TestingLogger, suppressLogError bool) {
	global.SetLogger(NewTestLogger(target, suppressLogError))
}
