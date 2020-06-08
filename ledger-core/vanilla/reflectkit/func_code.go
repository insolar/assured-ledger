// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reflectkit

import (
	"reflect"
	"unsafe"
)

// A faster version (25-35%) to get comparable identity of function's code.
// Same to reflect.ValueOf(<func>).Pointer(), but doesn't force heap-escape of the argument.
// NB! ONLY works for hard-coded functions. Functions created with reflect will have same code identity.
func CodeOf(v interface{}) uintptr {
	if v == nil {
		panic("illegal value")
	}

	ptr, kind := unwrapIface(v)
	if kind != uint8(reflect.Func)|kindDirectIface {
		panic("illegal value")
	}
	if ptr == nil {
		return 0
	}

	// Non-nil func value points at data block.
	// First word of data block is actual code.
	return *(*uintptr)(ptr)
}

// const kindMask = (1 << 5) - 1
const kindDirectIface = 1 << 5

func unwrapIface(v interface{}) (word unsafe.Pointer, kind uint8) {
	type rtype struct {
		_, _    uintptr
		_       uint32
		_, _, _ uint8
		kind    uint8
	}

	type emptyInterface struct {
		typ  *rtype
		word unsafe.Pointer
	}

	iface := (*emptyInterface)(unsafe.Pointer(&v))
	return iface.word, iface.typ.kind
}

// A faster version (25-35%) to check for interface nil
func IsNil(v interface{}) bool {
	if v == nil {
		return true
	}
	ptr, kind := unwrapIface(v)
	return ptr == nil && kind&kindDirectIface != 0
}


