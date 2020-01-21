package reflectkit

import (
	"bytes"
	"io"
	"reflect"
	"runtime"
	"testing"

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

func BenchmarkCodeOf(t *testing.B) {
	samples := [5]interface{}{TestCodeOf, bytes.NewBufferString("abc").Len, (*bytes.Buffer).Len, io.Writer.Write,
		((io.Writer)(bytes.NewBufferString("abc"))).Write,
	}

	count := t.N * 100000000
	for i := count; i > 0; i-- {
		if v := samples[i%len(samples)]; v == nil {
			runtime.KeepAlive(i)
		}
	}

	t.Run("reflect", func(b *testing.B) {
		for i := count; i > 0; i-- {
			if v := reflect.ValueOf(samples[i%len(samples)]).Pointer(); v+1 == 0 {
				runtime.KeepAlive(v)
			}
		}
	})

	t.Run("direct", func(b *testing.B) {
		for i := count; i > 0; i-- {
			if v := CodeOf(samples[i%len(samples)]); v+1 == 0 {
				runtime.KeepAlive(v)
			}
		}
	})
}
