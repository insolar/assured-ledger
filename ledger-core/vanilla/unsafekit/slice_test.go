package unsafekit

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnwrapAsSlice(t *testing.T) {
	if BigEndian {
		t.SkipNow()
	}

	bytes := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}

	s := WrapBytes(bytes)
	st := MustMMapSliceType(reflect.TypeOf([]uint32(nil)), false)
	slice := UnwrapAsSliceOf(s, st).([]uint32)
	require.Equal(t, []uint32{0x03020100, 0x07060504, 0x0B0A0908}, slice)
	bytes[0] = 0xFF
	bytes[11] = 0xEE
	require.Equal(t, []uint32{0x030201FF, 0x07060504, 0xEE0A0908}, slice)

	runtime.KeepAlive(bytes)
}

func TestUnwrapEmptyAsSlice(t *testing.T) {
	st := MustMMapSliceType(reflect.TypeOf([]struct{}{}), false)
	slice := make([]struct{}, 10)

	wrapped := WrapSliceOf(slice, st)
	unwrapped := UnwrapAsSliceOf(wrapped, st).([]struct{})
	require.Equal(t, 0, len(unwrapped))

	slice = make([]struct{}, 0, 1)
	wrapped = WrapSliceOf(slice, st)
	unwrapped = UnwrapAsSliceOf(wrapped, st).([]struct{})
	require.Equal(t, 0, len(unwrapped))

	slice = nil
	wrapped = WrapSliceOf(slice, st)
	unwrapped = UnwrapAsSliceOf(wrapped, st).([]struct{})
	require.Equal(t, 0, len(unwrapped))
}
