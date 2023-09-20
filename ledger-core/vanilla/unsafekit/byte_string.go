package unsafekit

import (
	"hash"
	"io"
	"reflect"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

// WARNING! The given array MUST be immutable
// WARNING! This method violates unsafe pointer-conversion rules.
// You MUST make sure that (b) stays alive while the resulting ByteString is in use.
func WrapBytes(b []byte) longbits.ByteString {
	if len(b) == 0 {
		return longbits.EmptyByteString
	}
	return wrapUnsafe(b)
}

// Expects struct ptr.
// WARNING! This method violates unsafe pointer-conversion rules.
// You MUST make sure that (v) stays alive while the resulting ByteString is in use.
func WrapStruct(v interface{}) longbits.ByteString {
	vt := reflect.ValueOf(v)
	if vt.Kind() != reflect.Ptr {
		panic("illegal value")
	}
	return wrapStruct(vt.Elem())
}

// Expects struct value. Reuses the value.
// WARNING! Further unwraps MUST NOT modify the content.
// WARNING! This method violates unsafe pointer-conversion rules.
// You MUST make sure that (v) stays alive while the resulting ByteString is in use.
func WrapValueStruct(v interface{}) longbits.ByteString {
	return wrapStruct(reflect.ValueOf(v))
}

func wrapStruct(vt reflect.Value) longbits.ByteString {
	if vt.Kind() != reflect.Struct {
		panic("illegal value")
	}
	if !vt.CanAddr() {
		panic("illegal value")
	}
	return wrapUnsafePtr(vt.Pointer(), vt.Type().Size())
}

func WrapOf(v interface{}, mt MMapType) longbits.ByteString {
	vt := reflect.ValueOf(v)
	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}
	if vt.Type() != mt.ReflectType() {
		panic("illegal value type")
	}
	if !vt.CanAddr() {
		panic("illegal value")
	}
	return wrapUnsafePtr(vt.Pointer(), vt.Type().Size())
}

func UnwrapAs(v longbits.ByteString, mt MMapType) interface{} {
	if mt.Size() != len(v) {
		panic("illegal value")
	}
	return reflect.NewAt(mt.ReflectType(), _unwrapUnsafe(v)).Interface()
}

func Hash(v longbits.ByteString, h hash.Hash) hash.Hash {
	unwrapUnsafe(v, func(b []byte) uintptr {
		_, _ = h.Write(b)
		return 0
	})
	return h
}

func WriteTo(v longbits.ByteString, w io.Writer) (n int64, err error) {
	unwrapUnsafe(v, func(b []byte) uintptr {
		nn := 0
		nn, err = w.Write(b)
		n = int64(nn)
		return 0
	})
	return
}

// WARNING! This method violates unsafe pointer-conversion rules.
// You MUST make sure that (b) stays alive while the resulting ByteString is in use.
func wrapUnsafe(b []byte) longbits.ByteString {
	pSlice := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	var res longbits.ByteString
	pString := (*reflect.StringHeader)(unsafe.Pointer(&res))

	pString.Data = pSlice.Data
	pString.Len = pSlice.Len

	return res
}

// WARNING! This method violates unsafe pointer-conversion rules.
// You MUST make sure that (p) stays alive while the resulting ByteString is in use.
func wrapUnsafePtr(p uintptr, size uintptr) longbits.ByteString {
	var res longbits.ByteString
	pString := (*reflect.StringHeader)(unsafe.Pointer(&res))

	pString.Data = p
	pString.Len = int(size)

	return res
}

// WARNING! This method violates unsafe pointer-conversion rules.
// You MUST make sure that (p) stays alive while the resulting Pointer is in use.
func _unwrapUnsafe(s longbits.ByteString) unsafe.Pointer {
	if len(s) == 0 {
		return nil
	}
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	return unsafe.Pointer(sh.Data)
}

func _unwrapUnsafeUintptr(s longbits.ByteString) uintptr {
	if len(s) == 0 {
		return 0
	}
	return uintptr(_unwrapUnsafe(s))
}

// nolint
func unwrapUnsafe(s longbits.ByteString, fn func([]byte) uintptr) uintptr {
	return KeepAliveWhile(unsafe.Pointer(&s), func(p unsafe.Pointer) uintptr {
		pString := (*reflect.StringHeader)(p)

		var b []byte
		pSlice := (*reflect.SliceHeader)(unsafe.Pointer(&b))

		pSlice.Data = pString.Data
		pSlice.Len = pString.Len
		pSlice.Cap = pString.Len

		r := fn(b)
		//*pSlice = reflect.SliceHeader{}

		return r
	})
}
