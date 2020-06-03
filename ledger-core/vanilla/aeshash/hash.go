// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package aeshash

import (
	"reflect"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/unsafekit"
)

func GoMapHash(v longbits.ByteString) uint32 {
	return uint32(ByteStr(v))
}

func GoMapHashWithSeed(v longbits.ByteString, seed uint32) uint32 {
	return uint32(StrWithSeed(string(v), uint(seed)))
}

// Hash hashes the given string using the algorithm used by Go's hash tables
func Str(s string) uint {
	return StrWithSeed(s, 0)
}

func StrWithSeed(s string, seed uint) uint {
	return uint(unsafekit.KeepAliveWhile(unsafe.Pointer(&s), func(p unsafe.Pointer) uintptr {
		sh := (*reflect.StringHeader)(p)
		return hash(sh.Data, sh.Len, seed)
	}))
}

// Hash hashes the given slice using the algorithm used by Go's hash tables
func Slice(b []byte) uint {
	return SliceWithSeed(b, 0)
}

func SliceWithSeed(b []byte, seed uint) uint {
	return uint(unsafekit.KeepAliveWhile(unsafe.Pointer(&b), func(p unsafe.Pointer) uintptr {
		sh := (*reflect.SliceHeader)(p)
		return hash(sh.Data, sh.Len, seed)
	}))
}

func ByteStr(v longbits.ByteString) uint {
	return Str(string(v))
}

func hash(data uintptr, len int, seed uint) uintptr {
	return aeshash(data, uintptr(seed), uintptr(len))
}

// pData, hSeed, sLen
func aeshash(p, h, s uintptr) uintptr
