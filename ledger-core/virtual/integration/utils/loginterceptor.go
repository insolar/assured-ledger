// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
)

type LogEventHandler = func(arg interface{})

type LogInterceptor struct {
	logcommon.EmbeddedLogger
	EventHandler LogEventHandler
}

func (lc *LogInterceptor) NewEventStruct(level logcommon.Level) func(i interface{}, marshallers []logfmt.LogFieldMarshaller) {
	return func(i interface{}, marshallers []logfmt.LogFieldMarshaller) {
		lc.EmbeddedLogger.NewEventStruct(level)(i, marshallers)
		if lc.EventHandler != nil {
			lc.EventHandler(i)
		}
	}
}

func (lc *LogInterceptor) NewEvent(level logcommon.Level) func(args []interface{}) {
	return func(args []interface{}) {
		lc.EmbeddedLogger.NewEvent(level)(args)
		if lc.EventHandler != nil {
			if len(args) > 0 {
				lc.EventHandler(args[0])
			}
		}
	}
}
