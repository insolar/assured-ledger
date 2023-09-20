package throw

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tracesLines() []StackTrace {
	return []StackTrace{
		stackTrace{captureStackByDebug(0, false), false},
		stackTrace{captureStackByDebug(0, true), true},
		stackTrace{captureStackByCallers(0, false), false},
		stackTrace{captureStackByCallers(0, true), true},
	}
}

func tracesCodesAndLines() []StackTrace {
	return []StackTrace{
		// first 2 entries MUST BE on the same line
		stackTrace{captureStackByDebug(0, false), false}, stackTrace{captureStackByDebug(0, false), false},
		stackTrace{captureStackByDebug(0, false), false},
		stackTrace{captureStackByDebug(0, false), false},
		stackTrace{captureStackByDebug(1, false), false},
	}
}

func TestCompareStackTraceExt(t *testing.T) {
	others := tracesCodesAndLines()
	shorter := others[len(others)-1]
	others = others[:len(others)-1]

	for i, o := range others {
		for j := i + 1; j < len(others)-1; j++ {
			assert.Equal(t, DifferentStack, CompareStackTraceExt(o, others[j], StrictMatch), i)
			assert.Equal(t, DifferentStack, CompareStackTraceExt(others[j], o, StrictMatch), i)
		}
		assert.Equal(t, SubsetStack, CompareStackTraceExt(shorter, o, StrictMatch), i)
		assert.Equal(t, SupersetStack, CompareStackTraceExt(o, shorter, StrictMatch), i)
	}

	for i, o := range others {
		assert.Equal(t, SubsetStack, CompareStackTraceExt(shorter, o, SameLine), i)
		assert.Equal(t, SupersetStack, CompareStackTraceExt(o, shorter, SameLine), i)
	}

	for i, o := range others {
		assert.Equal(t, SubsetStack, CompareStackTraceExt(shorter, o, SameMethod), i)
		assert.Equal(t, SupersetStack, CompareStackTraceExt(o, shorter, SameMethod), i)
	}

	assert.Equal(t, SupersetStack, CompareStackTraceExt(others[0], others[1], SameLine))
	assert.Equal(t, SubsetStack, CompareStackTraceExt(others[1], others[0], SameLine))
	assert.Equal(t, DifferentStack, CompareStackTraceExt(others[2], others[3], SameLine))
	assert.Equal(t, DifferentStack, CompareStackTraceExt(others[3], others[2], SameLine))

	assert.Equal(t, SupersetStack, CompareStackTraceExt(others[0], others[1], SameMethod))
	assert.Equal(t, SubsetStack, CompareStackTraceExt(others[1], others[0], SameMethod))
	assert.Equal(t, SupersetStack, CompareStackTraceExt(others[2], others[3], SameMethod))
	assert.Equal(t, SubsetStack, CompareStackTraceExt(others[3], others[2], SameMethod))
}

func TestCompareStackTrace(t *testing.T) {
	others := tracesLines()

	for i, o := range others {
		for j := i + 1; j < len(others); j++ {
			assert.Equal(t, DifferentStack, CompareStackTrace(o, others[j]))
			assert.Equal(t, DifferentStack, CompareStackTrace(others[j], o))
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
					require.Equal(t, SupersetStack, CompareStackTrace(full, full1))
					require.Equal(t, SubsetStack, CompareStackTrace(full1, full))
				}

				assert.Equal(t, EqualStack, CompareStackTrace(top, top))
				assert.Equal(t, EqualStack, CompareStackTrace(full, full))

				assert.Equal(t, StackTop, CompareStackTrace(top, full))
				assert.Equal(t, FullStack, CompareStackTrace(full, top))

				for i, o := range others {
					assert.Equal(t, DifferentStack, CompareStackTrace(o, top), i)
					assert.Equal(t, DifferentStack, CompareStackTrace(top, o), i)
					assert.Equal(t, DifferentStack, CompareStackTrace(o, full), i)
					assert.Equal(t, DifferentStack, CompareStackTrace(full, o), i)
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

func TestExtractStackTop(t *testing.T) {
	st := stackTrace{data: []byte(testStackDebug)}
	require.Equal(t, testStackDebugCall, ExtractStackTop(st, 0).StackTraceAsText())
	require.Equal(t, testStackValuableOnce, ExtractStackTop(st, 1).StackTraceAsText())
	require.Equal(t, testCreatedBy, ExtractStackTop(st, 99).StackTraceAsText())

	st.data = []byte(testStackValuableOnce)

	require.Equal(t, testStackValuableOnce, ExtractStackTop(st, 0).StackTraceAsText())
	require.Equal(t, testStackValuableOnce, ExtractStackTop(st, 1).StackTraceAsText())
	require.Equal(t, testStackValuableOnce, ExtractStackTop(st, 99).StackTraceAsText())

	st.limit = true

	require.Equal(t, testStackValuableOnce, ExtractStackTop(st, 0).StackTraceAsText())
	require.Equal(t, testStackValuableOnce, ExtractStackTop(st, 1).StackTraceAsText())
	require.Equal(t, testStackValuableOnce, ExtractStackTop(st, 99).StackTraceAsText())
}
