package logoutput

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCallerInfo(t *testing.T) {
	fl1, fn1, ln1 := getCallerInfo1(0)
	fl2, fn2, ln2 := getCallerInfo2(0)
	require.Equal(t, fl1, fl2)
	require.Equal(t, fn1, fn2)
	require.Equal(t, ln1+1, ln2)
}

func BenchmarkGetCallerInfo(b *testing.B) {
	b.Run("Caller+FuncForPC", func(b *testing.B) {
		for i := b.N; i > 0; i-- {
			fl, fn, ln := getCallerInfo1(0)
			if fl == "" && fn == "" && ln == 0 {
				runtime.KeepAlive(fl)
			}
		}
	})

	b.Run("Callers+FuncForPC", func(b *testing.B) {
		for i := b.N; i > 0; i-- {
			fl, fn, ln := getCallerInfo2(0)
			if fl == "" && fn == "" && ln == 0 {
				runtime.KeepAlive(fl)
			}
		}
	})
}
