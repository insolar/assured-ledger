// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

/*
	This interface provides methods with -1 call levels.
	DO NOT USE directly, otherwise WithCaller() functionality will be broken.
*/type EmbeddedLogger interface {
	NewEventStruct(level Level) func(interface{}, []logfmt.LogFieldMarshaller)
	NewEvent(level Level) func(args []interface{})
	NewEventFmt(level Level) func(fmt string, args []interface{})

	// Does flushing of an underlying buffer. Implementation and factual output may vary.
	EmbeddedFlush(msg string)

	Is(Level) bool
	Copy() EmbeddedLoggerBuilder
}

type EmbeddedLoggerBuilder interface {
	Template
	GetGlobalLogAdapter() GlobalLogAdapter
	GetLoggerOutput() LoggerOutput
}

type EmbeddedLoggerOptional interface {
	WithFields(fields map[string]interface{}) EmbeddedLogger
	WithField(name string, value interface{}) EmbeddedLogger
}
