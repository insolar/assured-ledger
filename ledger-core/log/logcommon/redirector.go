// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"io"
)

var _ LoggerOutput = &TestingLoggerOutput{}

// this interface matches required  *testing.T
type TestingLogger interface {
	Helper()
	Log(...interface{})
	Error(...interface{})
	Fatal(...interface{})
}

type TestingLoggerWrapper interface {
	UnwrapTesting() TestingLogger
}

func IsBasedOn(t, lookFor TestingLogger) bool {
	for t != nil {
		if t == lookFor {
			return true
		}
		if w, ok := t.(TestingLoggerWrapper); ok {
			t = w.UnwrapTesting()
			continue
		}
		break
	}
	return false
}

type ErrorFilterFunc = func(string) bool

type TestingLoggerOutput struct {
	Output, EchoTo io.Writer
	Testing        TestingLogger
	InterceptFatal func([]byte) bool
	ErrorFilterFn  ErrorFilterFunc
	LogFiltered    bool
}

func (r *TestingLoggerOutput) Close() error {
	if closer, ok := r.EchoTo.(io.Closer); ok {
		_ = closer.Close()
	}
	if closer, ok := r.Output.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (r *TestingLoggerOutput) Flush() error {
	if flusher, ok := r.EchoTo.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
	if flusher, ok := r.Output.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

func (r *TestingLoggerOutput) Write(b []byte) (int, error) {
	if r.EchoTo != nil {
		_, _ = r.EchoTo.Write(b)
	}
	if r.Output != nil {
		return r.Output.Write(b)
	}

	return len(b), nil
}

func (r *TestingLoggerOutput) LogLevelWrite(level Level, b []byte) (int, error) {
	msg := string(b)
	switch level {
	case FatalLevel:
		if r.InterceptFatal == nil || !r.InterceptFatal(b) {
			defer r.Testing.Fatal(msg)
		} else {
			defer r.Testing.Error(msg)
		}
	case PanicLevel, ErrorLevel:
		if r.ErrorFilterFn == nil || r.ErrorFilterFn(msg) {
			defer r.Testing.Error(msg)
		} else if r.LogFiltered {
			defer r.Testing.Log(msg)
		}
	}

	if r.EchoTo != nil {
		_, _ = r.EchoTo.Write(b)
	}
	if r.Output != nil {
		return r.Output.Write(b)
	}
	return len(b), nil
}

func (r *TestingLoggerOutput) LowLatencyWrite(level Level, b []byte) (int, error) {
	//nolint
	go r.LogLevelWrite(level, b)
	return len(b), nil
}

func (r *TestingLoggerOutput) IsLowLatencySupported() bool {
	return true
}
