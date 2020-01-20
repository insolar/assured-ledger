//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

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
