// +build 386 amd64 amd64p32 arm arm64 mips64le mipsle ppc64le wasm

package unsafekit

import "unsafe"

const BigEndian = false // unfortunately, sys.BigEndian is inaccessible

func init() {
	bytes := [4]byte{1, 2, 3, 4}
	v := *((*uint32)((unsafe.Pointer)(&bytes)))
	if v != 0x04030201 {
		panic("FATAL - expected LittleEndian memory architecture")
	}
}
