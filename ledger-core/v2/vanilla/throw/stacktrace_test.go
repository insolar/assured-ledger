package throw

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"runtime"
	"testing"
)

func TestCaptureStack(t *testing.T) {
	// TODO proper tests
	//fmt.Printf("%s\n==============\n", captureStackByDebug(0, false))
	//fmt.Printf("%s\n==============\n", captureStackByDebug(1, false))
	//fmt.Printf("%s\n==============\n", captureStackByDebug(0, true))
	//fmt.Printf("%s\n==============\n", captureStackByDebug(99, true))
	//
	//fmt.Println()
	//
	//fmt.Printf("%s\n==============\n", captureStackByCallers(0, false))
	//fmt.Printf("%s\n==============\n", captureStackByCallers(1, false))
	//fmt.Printf("%s\n==============\n", captureStackByCallers(0, true))

	fmt.Printf("%s\n==============\n", captureStack(0, false))
	fmt.Printf("%s\n==============\n", captureStack(0, true))
	fmt.Printf("%s\n==============\n", captureStack(99, true))
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

func Test_compareStackTrace(t *testing.T) {
	others := []StackTrace{
		stackTrace{captureStackByDebug(0, false), false},
		stackTrace{captureStackByDebug(0, true), true},
		stackTrace{captureStackByCallers(0, false), false},
		stackTrace{captureStackByCallers(0, true), true},
	}

	for i, o := range others {
		for j := i + 1; j < len(others); j++ {
			assert.Equal(t, DifferentTrace, CompareStackTrace(o, others[j]))
			assert.Equal(t, DifferentTrace, CompareStackTrace(others[j], o))
		}
	}

	for n, tt := range []struct {
		fn0, fn1 stackCaptureFunc
	}{
		{captureStackByDebug, captureStackByDebug},
		{captureStackByCallers, captureStackByCallers},
		{captureStackByCallers, captureStackByDebug},
		// Not supported - { captureStackByDebug, captureStackByCallers },
	} {
		for i := 0; i <= 1; i++ {
			var top, full StackTrace
			getFn := getTraces
			skipFrames := 1
			if i != 0 {
				getFn = getDeferTraces
				skipFrames++ // to avoid catching different line of getFn calls
			}
			top, full = getFn(0, tt.fn0, tt.fn1)
			_, full1 := getFn(skipFrames, tt.fn0, tt.fn1)

			t.Run(fmt.Sprint(n, i), func(t *testing.T) {
				{
					require.Equal(t, SupersetTrace, CompareStackTrace(full, full1))
					require.Equal(t, SubsetTrace, CompareStackTrace(full1, full))
				}

				assert.Equal(t, EqualTrace, CompareStackTrace(top, top))
				assert.Equal(t, EqualTrace, CompareStackTrace(full, full))

				assert.Equal(t, TopTrace, CompareStackTrace(top, full))
				assert.Equal(t, FullTrace, CompareStackTrace(full, top))

				for i, o := range others {
					assert.Equal(t, DifferentTrace, CompareStackTrace(o, top), i)
					assert.Equal(t, DifferentTrace, CompareStackTrace(top, o), i)
					assert.Equal(t, DifferentTrace, CompareStackTrace(o, full), i)
					assert.Equal(t, DifferentTrace, CompareStackTrace(full, o), i)
				}
			})
		}
	}
}

type stackCaptureFunc = func(skipFrames int, limitFrames bool) []byte

func getTraces(skipFrames int, fn0, fn1 stackCaptureFunc) (StackTrace, StackTrace) {
	skipFrames++
	return stackTrace{fn0(skipFrames, true), true}, stackTrace{fn1(skipFrames, false), false}
}

func getDeferTraces(skipFrames int, fn0, fn1 stackCaptureFunc) (st0 StackTrace, st1 StackTrace) {
	skipFrames++
	defer func() {
		st0, st1 = stackTrace{fn0(skipFrames, true), true}, stackTrace{fn1(skipFrames, false), false}
	}()
	runtime.Gosched() // just to do something
	return
}
