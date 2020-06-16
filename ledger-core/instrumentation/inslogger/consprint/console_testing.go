// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package consprint

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewConsoleTestingPrinter(w logcommon.TestingLogger, config Config) logcommon.TestingLogger {
	if w == nil {
		panic(throw.IllegalValue())
	}

	if config.PartsOrder == nil {
		config.PartsOrder = []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.MessageFieldName,
			zerolog.CallerFieldName,
		}
	}

	cw := &testingConsoleWriter{config, w}
	cw.ConsoleWriter.Out = cw
	return cw
}

const (
	logLevel = 0
	errLevel = 1
	ftlLevel = 2
)

type testingConsoleWriter struct {
	mutex sync.Mutex
	level int

	ConsoleWriter zerolog.ConsoleWriter
	Testing logcommon.TestingLogger
}

func (p *testingConsoleWriter) Helper() {
	p.Testing.Helper()
}

func (p *testingConsoleWriter) Log(args ...interface{}) {
	p.log(fmt.Sprintln(args...), logLevel)
}

func (p *testingConsoleWriter) Error(args ...interface{}) {
	p.log(fmt.Sprintln(args...), errLevel)
}

func (p *testingConsoleWriter) Fatal(args ...interface{}) {
	p.log(fmt.Sprintln(args...), ftlLevel)
}

func (p *testingConsoleWriter) log(s string, level int) {
	p.mutex.Lock()
	p.mutex.Unlock()
	p.level = level
	_, _ = p.ConsoleWriter.Write([]byte(s))
}

func (p *testingConsoleWriter) Write(b []byte) (int, error) {
	switch p.level {
	case errLevel:
		p.Testing.Error(string(b))
	case ftlLevel:
		p.Testing.Fatal(string(b))
	default:
		p.Testing.Log(string(b))
	}
	return len(b), nil
}

