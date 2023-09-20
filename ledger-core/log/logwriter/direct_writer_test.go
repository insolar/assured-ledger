package logwriter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
)

func TestFatalDirectWriter_mute_on_fatal(t *testing.T) {
	tw := TestWriterStub{}
	writer := NewDirectWriter(NewAdapter(&tw, false, nil, func() error {
		panic("fatal")
	}))
	// We don't want to lock the writer on fatal in tests.
	var err error

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("WARN must pass\n"))
	require.NoError(t, err)
	_, err = writer.LogLevelWrite(logcommon.ErrorLevel, []byte("ERROR must pass\n"))
	require.NoError(t, err)

	require.Equal(t, 0, tw.FlushCount)
	_, err = writer.LogLevelWrite(logcommon.PanicLevel, []byte("PANIC must pass\n"))
	require.NoError(t, err)
	require.Equal(t, 1, tw.FlushCount)

	require.PanicsWithValue(t, "fatal", func() {
		_, _ = writer.LogLevelWrite(logcommon.FatalLevel, []byte("FATAL must pass\n"))
	})
	require.Equal(t, 2, tw.FlushCount)
	require.Equal(t, 0, tw.CloseCount)

	// MUST hang. Tested by logwriter.Adapter
	//_, _ = writer.LogLevelWrite(logcommon.WarnLevel, []byte("WARN must NOT pass\n"))
	//_, _ = writer.LogLevelWrite(logcommon.ErrorLevel, []byte("ERROR must NOT pass\n"))
	//_, _ = writer.LogLevelWrite(logcommon.PanicLevel, []byte("PANIC must NOT pass\n"))
	testLog := tw.String()
	assert.Contains(t, testLog, "WARN must pass")
	assert.Contains(t, testLog, "ERROR must pass")
	assert.Contains(t, testLog, "FATAL must pass")
	//assert.NotContains(t, testLog, "must NOT pass")
}

func TestFatalDirectWriter_close_on_fatal_without_flush(t *testing.T) {
	tw := TestWriterStub{}
	tw.NoFlush = true

	writer := NewDirectWriter(NewAdapter(&tw, false, nil, nil))
	var err error

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("WARN must pass\n"))
	require.NoError(t, err)

	_, err = writer.LogLevelWrite(logcommon.FatalLevel, []byte("FATAL must pass\n"))
	require.NoError(t, err)
	require.Equal(t, 1, tw.FlushCount)
	require.Equal(t, 1, tw.CloseCount)

	// MUST hang. Tested by logwriter.Adapter
	//_, _ = writer.LogLevelWrite(logcommon.WarnLevel, []byte("WARN must NOT pass\n"))
	//_, _ = writer.LogLevelWrite(logcommon.ErrorLevel, []byte("ERROR must NOT pass\n"))
	//_, _ = writer.LogLevelWrite(logcommon.PanicLevel, []byte("PANIC must NOT pass\n"))
	testLog := tw.String()
	assert.Contains(t, testLog, "WARN must pass")
	assert.Contains(t, testLog, "FATAL must pass")
	//assert.NotContains(t, testLog, "must NOT pass")
}
