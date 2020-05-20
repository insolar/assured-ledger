// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"io"
	"sync"
)

var _ LoggerOutput = &TestingLoggerOutput{}

// this interface matches required  *testing.T
type TestingRedirectTarget interface {
	Helper()
	Log(...interface{})
	Error(...interface{})
	Fatal(...interface{})
}

type TestingLoggerOutput struct {
	Mutex sync.Mutex
	Target TestingRedirectTarget
}

func (r *TestingLoggerOutput) Close() error {
	if closer, ok := r.Target.(io.Closer); ok {
		r.Mutex.Lock()
		defer r.Mutex.Unlock()

		return closer.Close()
	}
	return nil
}

func (r *TestingLoggerOutput) Flush() error {

	if flusher, ok := r.Target.(interface{ Flush() error }); ok {
		r.Mutex.Lock()
		defer r.Mutex.Unlock()

		return flusher.Flush()
	}
	return nil
}

func (r *TestingLoggerOutput) Write(b []byte) (int, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.Target.Log(string(b))
	return len(b), nil
}

func (r *TestingLoggerOutput) LogLevelWrite(level Level, b []byte) (int, error) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

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

func (r *TestingLoggerOutput) LowLatencyWrite(level Level, b []byte) (int, error) {
	//nolint
	go r.LogLevelWrite(level, b)
	return len(b), nil
}

func (r *TestingLoggerOutput) IsLowLatencySupported() bool {
	return true
}
