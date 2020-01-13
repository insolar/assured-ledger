package reflectkit

import (
	"bytes"
	"io"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCodeOf(t *testing.T) {
	var v interface{}

	v = TestCodeOf
	require.Equal(t, reflect.ValueOf(v).Pointer(), CodeOf(v))
	require.Equal(t, reflect.ValueOf(v).Pointer(), CodeOf(reflect.ValueOf(v).Interface()))

	v = bytes.NewBufferString("abc").Len
	require.Equal(t, reflect.ValueOf(v).Pointer(), CodeOf(v))
	require.NotEqual(t, reflect.ValueOf(v).Pointer(), CodeOf(TestCodeOf))

	var vi io.Writer
	vi = bytes.NewBufferString("abc")
	v = vi.Write
	require.Equal(t, reflect.ValueOf(v).Pointer(), CodeOf(v))
	require.NotEqual(t, reflect.ValueOf(v).Pointer(), CodeOf(TestCodeOf))

	v = (*bytes.Buffer).Len
	require.Equal(t, reflect.ValueOf(v).Pointer(), CodeOf(v))
	require.NotEqual(t, reflect.ValueOf(v).Pointer(), CodeOf(TestCodeOf))

	v = io.Writer.Write
	require.Equal(t, reflect.ValueOf(v).Pointer(), CodeOf(v))
	require.NotEqual(t, reflect.ValueOf(v).Pointer(), CodeOf(TestCodeOf))
}

func TestCodeOfNegative(t *testing.T) {
	require.Panics(t, func() { CodeOf(nil) })
	require.Panics(t, func() { CodeOf(1) })
	require.Panics(t, func() { CodeOf(bytes.NewBufferString("abc")) })
	require.Panics(t, func() { CodeOf(uintptr(0)) })
	require.Panics(t, func() { CodeOf(reflect.ValueOf(TestCodeOf)) })
	require.Panics(t, func() { CodeOf(reflect.ValueOf(TestCodeOf).Pointer()) })
}

func TestCodeSpeed(t *testing.T) {
	samples := [5]interface{}{TestCodeOf, bytes.NewBufferString("abc").Len, (*bytes.Buffer).Len, io.Writer.Write,
		((io.Writer)(bytes.NewBufferString("abc"))).Write,
	}

	const count = int(1e8)
	startedAt := time.Now()
	for i := count; i > 0; i-- {
		if v := samples[i%len(samples)]; v == nil {
			runtime.KeepAlive(i)
		}
	}
	overheadDuration := time.Since(startedAt)

	startedAt = time.Now()
	for i := count; i > 0; i-- {
		if v := reflect.ValueOf(samples[i%len(samples)]).Pointer(); v+1 == 0 {
			runtime.KeepAlive(v)
		}
	}
	reflectDuration := time.Since(startedAt)

	startedAt = time.Now()
	for i := count; i > 0; i-- {
		if v := CodeOf(samples[i%len(samples)]); v+1 == 0 {
			runtime.KeepAlive(v)
		}
	}
	directDuration := time.Since(startedAt)

	t.Logf("overhead=%v reflect=%v direct=%v", overheadDuration, reflectDuration, directDuration)
}
