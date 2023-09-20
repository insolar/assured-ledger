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
