package throw

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCaptureStack(t *testing.T) {
	// TODO proper tests
	// fmt.Printf("%s\n==============\n", captureStackByDebug(0, false))
	// fmt.Printf("%s\n==============\n", captureStackByDebug(1, false))
	// fmt.Printf("%s\n==============\n", captureStackByDebug(0, true))
	// fmt.Printf("%s\n==============\n", captureStackByDebug(99, true))
	//
	// fmt.Println()
	//
	// fmt.Printf("%s\n==============\n", captureStackByCallers(0, false))
	// fmt.Printf("%s\n==============\n", captureStackByCallers(1, false))
	// fmt.Printf("%s\n==============\n", captureStackByCallers(0, true))

	s := string(captureStack(0, false))
	require.Contains(t, s, "TestCaptureStack")
	require.Contains(t, s, "created by testing.(*T).Run")

	s = string(captureStack(0, true))
	require.Contains(t, s, "TestCaptureStack")
	require.NotContains(t, s, "created by testing.(*T).Run")

	s = string(captureStack(99, true))
	require.NotContains(t, string(captureStack(99, true)), "TestCaptureStack")
	require.Contains(t, s, "created by testing.(*T).Run")
}

func BenchmarkCaptureStack(b *testing.B) {
	b.Run("captureStackByDebug-full", func(b *testing.B) {
		for i := b.N; i > 0; i-- {
			v := captureStackByDebug(0, false)
			runtime.KeepAlive(v)
		}
	})

	b.Run("captureStackByCallers-full", func(b *testing.B) {
		for i := b.N; i > 0; i-- {
			v := captureStackByCallers(0, false)
			runtime.KeepAlive(v)
		}
	})

	b.Run("captureStackByDebug-top", func(b *testing.B) {
		for i := b.N; i > 0; i-- {
			v := captureStackByDebug(0, true)
			runtime.KeepAlive(v)
		}
	})

	b.Run("captureStackByCallers-top", func(b *testing.B) {
		for i := b.N; i > 0; i-- {
			v := captureStackByCallers(0, true)
			runtime.KeepAlive(v)
		}
	})
}
