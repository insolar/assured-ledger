package logcommon

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/log/logcommon.EmbeddedLogger -s _mock.go -g

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

	FieldsOf(reflect.Value) logfmt.LogObjectMarshaller
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
