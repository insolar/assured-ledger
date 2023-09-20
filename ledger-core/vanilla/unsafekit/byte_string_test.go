package unsafekit

import (
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
)

// In accordance with unsafe Rule (6) this behavior is NOT guaranteed. Yet it is valid for gc compiler.
// So one should be careful with Wrap/Unwrap operations.
//go:nocheckptr
func TestRetentionByByteString(t *testing.T) {
	type testPtr [8912]byte // enforce off-stack allocation
	b := make([]byte, 2048)

	b[0] = 'A'
	b[1] = 'B'

	finMark := uint32(0)

	shd := unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&b)).Data)
	runtime.SetFinalizer((*testPtr)(shd), func(_ *testPtr) {
		atomic.StoreUint32(&finMark, 1)
	})

	w := WrapBytes(b)
	runtime.KeepAlive(b)
	b = nil

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	// Here, the original content of (b) is retained by
	// an internal Data reference of ByteString/string

	require.Equal(t, uint32(0), finMark)
	tp := (*testPtr)(shd)
	require.Equal(t, byte('A'), tp[0])
	require.Equal(t, byte('B'), tp[1])
	tp = nil
	require.Equal(t, byte('A'), w[0])
	require.Equal(t, byte('B'), w[1])
	w = ""

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	fm := atomic.LoadUint32(&finMark)
	require.Equal(t, uint32(1), fm)
}

// In accordance with unsafe Rule (6) this behavior is NOT guaranteed. Yet it is valid for gc compiler.
// So one should be careful with Wrap/Unwrap operations.
func TestRetentionBySlice(t *testing.T) {
	type testPtr [8912]byte // enforce off-stack allocation

	bb := &testPtr{'A', 'B'}
	finMark := uint32(0)
	runtime.SetFinalizer(bb, func(_ *testPtr) {
		atomic.StoreUint32(&finMark, 1)
	})

	var b []byte

	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh.Data = uintptr(unsafe.Pointer(bb))
	sh.Len = len(*bb)
	sh.Cap = len(*bb)
	sh = nil
	bb = nil

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	// Here, the original content of (bb) is retained by
	// an internal Data reference of slice

	require.Equal(t, uint32(0), finMark)
	require.Equal(t, byte('A'), b[0])
	require.Equal(t, byte('B'), b[1])
	b = nil

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	fm := atomic.LoadUint32(&finMark)
	require.Equal(t, uint32(1), fm)
}
