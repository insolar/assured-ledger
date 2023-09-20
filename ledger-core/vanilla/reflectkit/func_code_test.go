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
	t.Run("baseline", func(b *testing.B) {
		for i := count; i > 0; i-- {
			if v := samples[i%len(samples)]; v == nil {
				runtime.KeepAlive(i)
			}
		}
	})

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

func TestIsNil(t *testing.T) {
	require.True(t, IsNil(nil))
	require.False(t, IsNil(""))
	require.False(t, IsNil(uintptr(0)))
	require.False(t, IsNil(0))
	require.False(t, IsNil(1))
	require.False(t, IsNil(struct{}{}))
	require.False(t, IsNil(&struct{}{}))

	var p *string
	require.True(t, IsNil(p))
	s := "x"
	p = &s
	var i interface{} = p
	require.False(t, IsNil(&s))
	require.False(t, IsNil(p))
	require.False(t, IsNil(i))

	p = nil
	i = p
	require.True(t, IsNil(p))
	require.True(t, IsNil(i))
}

func BenchmarkTestNil(t *testing.B) {
	var pn *string
	var n interface{} = pn

	t.Run("reflect", func(b *testing.B) {
		for i := b.N; i > 0; i-- {
			if !reflectIsNil(n) {
				b.Fail()
			}
		}
	})

	t.Run("direct", func(b *testing.B) {
		for i := b.N; i > 0; i-- {
			if !IsNil(n) {
				b.Fail()
			}
		}
	})
}

func reflectIsNil(v interface{}) bool {
	if v == nil {
		return true
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	}
	return false
}
