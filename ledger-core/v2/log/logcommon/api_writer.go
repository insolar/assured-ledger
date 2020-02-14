// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"io"
)

//type LoggerOutputGetter interface {
//	GetLoggerOutput() LoggerOutput
//}

type LoggerOutput interface {
	LogLevelWriter
	LowLatencyWrite(Level, []byte) (int, error)
	IsLowLatencySupported() bool
}

type LogLevelWriter interface {
	io.WriteCloser
	LogLevelWrite(Level, []byte) (int, error)
	Flush() error
}

type LogFlushFunc func() error
