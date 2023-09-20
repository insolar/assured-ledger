package logwriter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
)

var _ logcommon.LogLevelWriter = &testLevelWriter{}

type testLevelWriter struct {
	TestWriterStub
}

func (p *testLevelWriter) Write([]byte) (int, error) {
	panic("unexpected")
}

func (p *testLevelWriter) LogLevelWrite(level logcommon.Level, b []byte) (int, error) {
	return p.TestWriterStub.Write([]byte(level.String() + string(b)))
}

func TestAdapter_fatal_close_on_no_flush(t *testing.T) {
	tw := TestWriterStub{}
	tw.NoFlush = true

	writer := NewAdapter(&tw, false, nil, nil)
	writer.setState(adapterPanicOnFatal)

	var err error

	require.Equal(t, 0, tw.FlushCount)
	err = writer.DirectFlushFatal()
	require.NoError(t, err)

	require.Equal(t, 1, tw.FlushCount)
	require.Equal(t, 1, tw.CloseCount)

	require.PanicsWithValue(t, "fatal lock", func() {
		_ = writer.Flush()
	})
	require.Equal(t, 1, tw.FlushCount)
}

func TestAdapter_fatal(t *testing.T) {
	tw := TestWriterStub{}
	writer := NewAdapter(&tw, false, nil, nil)
	writer.setState(adapterPanicOnFatal)

	var err error

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("NORM must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "NORM must pass")

	require.Equal(t, 0, tw.FlushCount)
	err = writer.Flush()
	require.NoError(t, err)
	require.Equal(t, 1, tw.FlushCount)

	require.False(t, writer.IsFatal())
	require.True(t, writer.SetFatal())
	require.True(t, writer.IsFatal())
	require.False(t, writer.SetFatal())
	require.True(t, writer.IsFatal())

	require.Panics(t, func() {
		_, _ = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must NOT pass\n"))
	})
	require.Panics(t, func() {
		_ = writer.Flush()
	})
	require.Equal(t, 1, tw.FlushCount)
	err = writer.DirectFlushFatal()
	require.NoError(t, err)
	require.Equal(t, 2, tw.FlushCount)

	require.True(t, writer.IsClosed())
	require.Equal(t, 0, tw.CloseCount)
	require.Panics(t, func() {
		_ = writer.Close()
	})
	require.Equal(t, 0, tw.CloseCount)
	require.True(t, writer.IsClosed())

	assert.NotContains(t, tw.String(), "must NOT pass")

	_, err = writer.DirectLevelWrite(logcommon.WarnLevel, []byte("DIRECT must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "DIRECT must pass")
}

func TestAdapter_fatal_close(t *testing.T) {
	tw := TestWriterStub{}
	writer := NewAdapter(&tw, false, nil, nil)
	writer.setState(adapterPanicOnFatal)

	require.True(t, writer.SetFatal())

	require.False(t, writer.IsClosed())
	require.Equal(t, 0, tw.CloseCount)
	require.Panics(t, func() {
		_ = writer.Close()
	})
	require.Equal(t, 1, tw.CloseCount)
	require.True(t, writer.IsClosed())
}

func TestAdapter_fatal_direct_close(t *testing.T) {
	tw := TestWriterStub{}
	writer := NewAdapter(&tw, false, nil, nil)
	writer.setState(adapterPanicOnFatal)

	var err error

	require.True(t, writer.SetFatal())

	require.Equal(t, 0, tw.CloseCount)
	err = writer.DirectClose()
	require.NoError(t, err)
	require.Equal(t, 1, tw.CloseCount)
	require.True(t, writer.IsClosed())
	err = writer.DirectClose()
	require.Error(t, err) // underlying's error
	require.Equal(t, 2, tw.CloseCount)

	require.Panics(t, func() {
		_ = writer.Close()
	})
	require.Equal(t, 2, tw.CloseCount)
	require.True(t, writer.IsClosed())
}

func TestAdapter_fatal_flush(t *testing.T) {
	tw := TestWriterStub{}
	writer := NewAdapter(&tw, false, nil, nil)
	writer.setState(adapterPanicOnFatal)

	var err error

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "must pass")

	require.False(t, writer.IsFatal())
	require.Equal(t, 0, tw.FlushCount)
	err = writer.DirectFlushFatal()
	require.NoError(t, err)
	require.Equal(t, 1, tw.FlushCount)
	require.True(t, writer.IsFatal())
}

func TestAdapter_fatal_flush_helper(t *testing.T) {
	tw := TestWriterStub{}
	flushHelperCount := 0
	writer := NewAdapter(&tw, false, nil, func() error {
		flushHelperCount++
		return nil
	})
	writer.setState(adapterPanicOnFatal)

	var err error

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "must pass")

	require.False(t, writer.IsFatal())
	require.Equal(t, 0, tw.FlushCount)
	require.Equal(t, 0, flushHelperCount)
	err = writer.Flush()
	require.NoError(t, err)
	require.Equal(t, 1, tw.FlushCount)
	require.Equal(t, 0, flushHelperCount)
	err = writer.Flush()
	require.NoError(t, err)
	require.Equal(t, 2, tw.FlushCount)
	require.Equal(t, 0, flushHelperCount)

	err = writer.DirectFlushFatal()
	require.NoError(t, err)
	require.Equal(t, 3, tw.FlushCount)
	require.Equal(t, 1, flushHelperCount)
	require.True(t, writer.IsFatal())

	err = writer.DirectFlushFatal()
	require.Error(t, err)
	require.Equal(t, 3, tw.FlushCount)
	require.Equal(t, 1, flushHelperCount)
}

func TestAdapter_close(t *testing.T) {
	tw := TestWriterStub{}
	writer := NewAdapter(&tw, false, nil, nil)

	var err error

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "must pass")

	require.False(t, writer.IsClosed())
	require.Equal(t, 0, tw.CloseCount)
	err = writer.Close()
	require.NoError(t, err)
	require.Equal(t, 1, tw.CloseCount)
	require.True(t, writer.IsClosed())
	require.False(t, writer.SetClosed())
	require.True(t, writer.IsClosed())
	err = writer.Close()
	require.Error(t, err)

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must NOT pass\n"))
	require.Error(t, err)

	assert.NotContains(t, tw.String(), "must NOT pass")

	err = writer.DirectClose()
	require.Error(t, err)
	require.Equal(t, 2, tw.CloseCount)

	_, err = writer.DirectLevelWrite(logcommon.WarnLevel, []byte("DIRECT must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "DIRECT must pass")
}

func TestAdapter_close_protect(t *testing.T) {
	tw := TestWriterStub{}
	writer := NewAdapter(&tw, true, nil, nil)

	var err error

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "must pass")

	require.False(t, writer.IsClosed())
	require.Equal(t, 0, tw.CloseCount)
	err = writer.Close()
	require.NoError(t, err)
	require.Equal(t, 0, tw.CloseCount)
	require.True(t, writer.IsClosed())
	require.False(t, writer.SetClosed())
	require.True(t, writer.IsClosed())
	err = writer.Close()
	require.Error(t, err)
	require.Equal(t, 0, tw.CloseCount)

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must NOT pass\n"))
	require.Error(t, err)

	assert.NotContains(t, tw.String(), "must NOT pass")
}

func TestAdapter_direct_close_protect(t *testing.T) {
	tw := TestWriterStub{}
	writer := NewAdapter(&tw, true, nil, nil)

	var err error

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "must pass")

	require.False(t, writer.IsClosed())
	require.Equal(t, 0, tw.CloseCount)
	err = writer.DirectClose()
	require.NoError(t, err)
	require.Equal(t, 0, tw.CloseCount)
	require.True(t, writer.IsClosed())
	require.False(t, writer.SetClosed())
	require.True(t, writer.IsClosed())
	err = writer.DirectClose()
	require.NoError(t, err)
	require.Equal(t, 0, tw.CloseCount)

	_, err = writer.LogLevelWrite(logcommon.WarnLevel, []byte("must NOT pass\n"))
	require.Error(t, err)

	assert.NotContains(t, tw.String(), "must NOT pass")
}

func TestAdapter_level_write(t *testing.T) {
	tw := testLevelWriter{}
	writer := NewAdapter(&tw, false, nil, nil)

	_, err := writer.LogLevelWrite(logcommon.WarnLevel, []byte(" must pass\n"))
	require.NoError(t, err)
	assert.Contains(t, tw.String(), "warn must pass")
}
