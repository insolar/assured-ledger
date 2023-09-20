package unsafekit

import (
	"runtime"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

// WARNING! You MUST make sure that (v) stays alive while the resulting longbits.ByteString is in use.
func WrapLocalRef(v *reference.Local) (r longbits.ByteString) {
	if v == nil {
		return ""
	}
	KeepAliveWhile((unsafe.Pointer)(v), func(p unsafe.Pointer) uintptr {
		r = wrapUnsafePtr(uintptr(p), unsafe.Sizeof(*v))
		return 0
	})
	return
}

// WARNING! You MUST make sure that (v) stays alive while the resulting longbits.ByteString is in use.
func WrapGlobalRef(v *reference.Global) (r longbits.ByteString) {
	if v == nil {
		return ""
	}
	KeepAliveWhile((unsafe.Pointer)(v), func(p unsafe.Pointer) uintptr {
		r = wrapUnsafePtr(uintptr(p), unsafe.Sizeof(*v))
		return 0
	})
	return
}

// WARNING! This function has different guarantees based on (s) origin:
// 1) When (s) is made by wrapping another type - it satisfies Unsafe Rule (1) Conversion of a *T1 to Pointer to *T2.
//    You are safe.
//
// 2) When (s) is made by wrapping []byte or string - it violates Unsafe Rule (6) Conversion of SliceHeader/StringHeader
//    And YOU MUST make sure that the origin stays alive while the result is in use.
//
func UnwrapAsLocalRef(s longbits.ByteString) *reference.Local {
	switch len(s) {
	case 0:
		return nil
	case reference.LocalBinarySize:
		r := (*reference.Local)(_unwrapUnsafe(s))
		runtime.KeepAlive(s)
		return r
	default:
		panic("illegal value")
	}
}

// WARNING! This function has different guarantees based on (s) origin:
// 1) When (s) is made by wrapping another type - it satisfies Unsafe Rule (1) Conversion of a *T1 to Pointer to *T2.
//    You are safe.
//
// 2) When (s) is made by wrapping []byte or string - it violates Unsafe Rule (6) Conversion of SliceHeader/StringHeader
//    And YOU MUST make sure that the origin stays alive while the result is in use.
//
func UnwrapAsGlobalRef(s longbits.ByteString) *reference.Global {
	switch len(s) {
	case 0:
		return nil
	case reference.GlobalBinarySize:
		r := (*reference.Global)(_unwrapUnsafe(s))
		runtime.KeepAlive(s)
		return r
	default:
		panic("illegal value")
	}
}
