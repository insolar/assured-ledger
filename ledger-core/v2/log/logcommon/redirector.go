// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"io"
)

var _ LoggerOutput = TestingLoggerOutput{}

// this interface matches required  *testing.T
type TestingRedirectTarget interface {
	Helper()
	Log(...interface{})
	Error(...interface{})
	Fatal(...interface{})
}

type TestingLoggerOutput struct {
	Target TestingRedirectTarget
}

func (r TestingLoggerOutput) Close() error {
	if closer, ok := r.Target.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (r TestingLoggerOutput) Flush() error {
	if flusher, ok := r.Target.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

func (r TestingLoggerOutput) Write(b []byte) (int, error) {
	r.Target.Log(string(b))
	return len(b), nil
}

func (r TestingLoggerOutput) LogLevelWrite(level Level, b []byte) (int, error) {
	msg := string(b)
	switch level {
	case FatalLevel:
		r.Target.Fatal(msg)
	case PanicLevel, ErrorLevel:
		r.Target.Error(msg)
	default:
		r.Target.Log(msg)
	}
	return len(b), nil
}

func (r TestingLoggerOutput) LowLatencyWrite(level Level, b []byte) (int, error) {
	return r.LogLevelWrite(level, b)
}

func (r TestingLoggerOutput) IsLowLatencySupported() bool {
	return true
}
