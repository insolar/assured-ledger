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
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

func NewTestLogger(target logcommon.TestingRedirectTarget) log.Logger {
	return NewTestLoggerExt(target, "")
}

func NewTestLoggerExt(target logcommon.TestingRedirectTarget, adapter string) log.Logger {
	if target == nil {
		panic("illegal value")
	}
	logCfg := defaultLogConfig()
	if adapter != "" {
		logCfg.Adapter = adapter
	}

	l, err := newLogger(logCfg)
	if err != nil {
		panic(err)
	}
	return l.WithMetrics(logcommon.LogMetricsResetMode).
		WithFormat(logcommon.TextFormat).
		WithCaller(logcommon.CallerField).
		WithOutput(logcommon.TestingLoggerOutput{target}).
		MustBuild()
}

func SetTestOutput(target logcommon.TestingRedirectTarget) {
	global.SetLogger(NewTestLogger(target))
}
