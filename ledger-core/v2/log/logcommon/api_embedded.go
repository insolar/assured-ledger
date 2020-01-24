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
