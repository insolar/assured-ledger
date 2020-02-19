// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package unsafekit

const PtrSize = 4 << (^uintptr(0) >> 63) // unsafe.Sizeof(uintptr(0)) but an ideal const

type MemoryModelSupported uint8

const (
	_ MemoryModelSupported = iota
	LittleEndianSupported
	BigEndianSupported
	EndianIndependent // LittleEndianSupported | BigEndianSupported
)

func IsCompatibleMemoryModel(v MemoryModelSupported) bool {
	switch v {
	case EndianIndependent:
		return true
	case LittleEndianSupported:
		return !BigEndian
	case BigEndianSupported:
		return BigEndian
	default:
		panic("illegal value")
	}
}
