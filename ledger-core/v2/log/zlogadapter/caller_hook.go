//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package zlogadapter

import (
	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
)

type callerHook struct {
	callerSkipFrameCount int
}

func newCallerHook(skipFrameCount int) callerHook {
	return callerHook{callerSkipFrameCount: skipFrameCount + 1}
}

func (ch callerHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	switch level {
	case zerolog.NoLevel, zerolog.Disabled:
		return
	default:
		fileName, funcName, line := logadapter.GetCallInfo(ch.callerSkipFrameCount)
		fileName = zerolog.CallerMarshalFunc(fileName, line)

		e.Str(zerolog.CallerFieldName, fileName)
		e.Str(logadapter.FuncFieldName, funcName)
	}
}
