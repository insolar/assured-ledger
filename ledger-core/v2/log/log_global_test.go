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

package log

import (
	"bytes"
	"os"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func capture(f func()) string {
	defer SaveGlobalLogger()()

	var buf bytes.Buffer

	gl, err := GlobalLogger().Copy().WithOutput(&buf).Build()
	if err != nil {
		panic(err)
	}
	SetGlobalLogger(gl)

	f()

	return buf.String()
}

func assertHelloWorld(t *testing.T, out string) {
	assert.Contains(t, out, "HelloWorld")
}

func TestLog_GlobalLogger_redirection(t *testing.T) {
	defer SaveGlobalLogger()()

	SetLogLevel(logcommon.InfoLevel)

	originalG := GlobalLogger()

	var buf bytes.Buffer
	newGL, err := GlobalLogger().Copy().WithOutput(&buf).WithBuffer(10, false).Build()
	require.NoError(t, err)

	SetGlobalLogger(newGL)
	newCopyLL, err := GlobalLogger().Copy().BuildLowLatency()
	require.NoError(t, err)

	originalG.Info("viaOriginalGlobal")
	newGL.Info("viaNewInstance")
	GlobalLogger().Info("viaNewGlobal")
	newCopyLL.Info("viaNewLLCopyOfGlobal")

	s := buf.String()
	require.Contains(t, s, "viaOriginalGlobal")
	require.Contains(t, s, "viaNewInstance")
	require.Contains(t, s, "viaNewGlobal")
	require.Contains(t, s, "viaNewLLCopyOfGlobal")
}

func TestLog_GlobalLogger(t *testing.T) {

	SetLogLevel(logcommon.DebugLevel)

	assertHelloWorld(t, capture(func() { Debug("HelloWorld") }))
	assertHelloWorld(t, capture(func() { Debugf("%s", "HelloWorld") }))

	assertHelloWorld(t, capture(func() { Info("HelloWorld") }))
	assertHelloWorld(t, capture(func() { Infof("%s", "HelloWorld") }))

	assertHelloWorld(t, capture(func() { Warn("HelloWorld") }))
	assertHelloWorld(t, capture(func() { Warnf("%s", "HelloWorld") }))

	assertHelloWorld(t, capture(func() { Error("HelloWorld") }))
	assertHelloWorld(t, capture(func() { Errorf("%s", "HelloWorld") }))

	assert.Panics(t, func() { capture(func() { Panic("HelloWorld") }) })
	assert.Panics(t, func() { capture(func() { Panicf("%s", "HelloWorld") }) })

	// cyclic run of this test changes loglevel, so revert it back
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// can't catch os.exit() to test Fatal
	// Fatal("HelloWorld")
	// Fatalln("HelloWorld")
	// Fatalf("%s", "HelloWorld")
}

func TestLog_GlobalLogger_Level(t *testing.T) {
	defer SaveGlobalLogger()()

	assert.NoError(t, SetLevel("error"))
	assert.Error(t, SetLevel("errorrr"))
}

func TestLog_GlobalLogger_FilterLevel(t *testing.T) {
	defer SaveGlobalLoggerAndFilter(true)()

	SetLogLevel(logcommon.DebugLevel)
	assert.NoError(t, SetGlobalLevelFilter(logcommon.DebugLevel))
	assertHelloWorld(t, capture(func() { Debug("HelloWorld") }))
	assert.NoError(t, SetGlobalLevelFilter(logcommon.InfoLevel))
	assert.Equal(t, "", capture(func() { Debug("HelloWorld") }))
}

func TestLog_GlobalLogger_Save(t *testing.T) {
	assert.NotNil(t, GlobalLogger()) // ensure initialization

	restoreFn := SaveGlobalLoggerAndFilter(true)
	level := GlobalLogger().Copy().GetLogLevel()
	filter := GetGlobalLevelFilter()

	if level != logcommon.PanicLevel {
		SetLogLevel(level + 1)
	} else {
		SetLogLevel(logcommon.DebugLevel)
	}
	assert.NotEqual(t, level, GlobalLogger().Copy().GetLogLevel())

	if filter != logcommon.PanicLevel {
		assert.NoError(t, SetGlobalLevelFilter(filter+1))
	} else {
		assert.NoError(t, SetGlobalLevelFilter(logcommon.DebugLevel))
	}
	assert.NotEqual(t, filter, GetGlobalLevelFilter())

	restoreFn()

	assert.Equal(t, level, GlobalLogger().Copy().GetLogLevel())
	assert.Equal(t, filter, GetGlobalLevelFilter())
}

func TestMain(m *testing.M) {
	l, err := GlobalLogger().Copy().WithFormat(logcommon.JSONFormat).WithLevel(logcommon.DebugLevel).Build()
	if err != nil {
		panic(err)
	}
	SetGlobalLogger(l)
	_ = SetGlobalLevelFilter(logcommon.DebugLevel)
	exitCode := m.Run()
	os.Exit(exitCode)
}
