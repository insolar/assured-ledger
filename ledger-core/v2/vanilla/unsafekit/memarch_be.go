// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build mips mips64 ppc64 s390x

package unsafekit

import "unsafe"

const BigEndian = true

func init() {
	bytes := [4]byte{1, 2, 3, 4}
	v := *((*uint32)((unsafe.Pointer)(&bytes)))
	if v != 0x01020304 {
		panic("FATAL - expected BigEndian memory architecture")
	}
}
