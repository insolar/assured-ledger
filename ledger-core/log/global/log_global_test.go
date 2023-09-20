package global

import (
	"bytes"
	"os"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func capture(f func()) string {
	defer SaveLogger()()

	var buf bytes.Buffer

	gl, err := Logger().Copy().WithOutput(&buf).Build()
	if err != nil {
		panic(err)
	}
	SetLogger(gl)

	f()

	return buf.String()
}

func assertHelloWorld(t *testing.T, out string) {
	assert.Contains(t, out, "HelloWorld")
}

func TestLog_GlobalLogger_redirection(t *testing.T) {
	defer SaveLogger()()

	SetLevel(log.InfoLevel)

	originalG := Logger()

	var buf bytes.Buffer
	newGL, err := Logger().Copy().WithOutput(&buf).WithBuffer(10, false).Build()
	require.NoError(t, err)

	SetLogger(newGL)
	newCopyLL, err := Logger().Copy().BuildLowLatency()
	require.NoError(t, err)

	originalG.Info("viaOriginalGlobal")
	newGL.Info("viaNewInstance")
	Logger().Info("viaNewGlobal")
	newCopyLL.Info("viaNewLLCopyOfGlobal")

	s := buf.String()
	require.Contains(t, s, "viaOriginalGlobal")
	require.Contains(t, s, "viaNewInstance")
	require.Contains(t, s, "viaNewGlobal")
	require.Contains(t, s, "viaNewLLCopyOfGlobal")
}

func TestLog_GlobalLogger(t *testing.T) {

	SetLevel(log.DebugLevel)

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
	defer SaveLogger()()

	assert.NoError(t, SetTextLevel("error"))
	assert.Error(t, SetTextLevel("errorrr"))
}

func TestLog_GlobalLogger_FilterLevel(t *testing.T) {
	defer SaveLoggerAndFilter(true)()

	SetLevel(log.DebugLevel)
	assert.NoError(t, SetFilter(log.DebugLevel))
	assertHelloWorld(t, capture(func() { Debug("HelloWorld") }))
	assert.NoError(t, SetFilter(log.InfoLevel))
	assert.Equal(t, "", capture(func() { Debug("HelloWorld") }))
}

func TestLog_GlobalLogger_Save(t *testing.T) {
	assert.NotNil(t, Logger()) // ensure initialization

	restoreFn := SaveLoggerAndFilter(true)
	level := Logger().Copy().GetLogLevel()
	filter := GetFilter()

	if level != log.PanicLevel {
		SetLevel(level + 1)
	} else {
		SetLevel(log.DebugLevel)
	}
	assert.NotEqual(t, level, Logger().Copy().GetLogLevel())

	if filter != log.PanicLevel {
		assert.NoError(t, SetFilter(filter+1))
	} else {
		assert.NoError(t, SetFilter(log.DebugLevel))
	}
	assert.NotEqual(t, filter, GetFilter())

	restoreFn()

	assert.Equal(t, level, Logger().Copy().GetLogLevel())
	assert.Equal(t, filter, GetFilter())
}

func TestMain(m *testing.M) {
	l, err := Logger().Copy().WithFormat(logcommon.JSONFormat).WithLevel(log.DebugLevel).Build()
	if err != nil {
		panic(err)
	}
	SetLogger(l)
	_ = SetFilter(log.DebugLevel)
	exitCode := m.Run()
	os.Exit(exitCode)
}
