// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package zlog

import (
	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"
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
		fileName, funcName, line := logoutput.GetCallerInfo(ch.callerSkipFrameCount)
		fileName = zerolog.CallerMarshalFunc(fileName, line)

		e.Str(zerolog.CallerFieldName, fileName)
		e.Str(logoutput.FuncFieldName, funcName)
	}
}
