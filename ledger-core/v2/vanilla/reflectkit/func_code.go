//
//    Copyright 2020 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

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

	//const kindMask        = (1 << 5) - 1
	const kindDirectIface = 1 << 5

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
