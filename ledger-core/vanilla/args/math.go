// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package args

import "math/bits"

func GreatestCommonDivisor(a, b int) int {
	if a == b {
		return a
	}

	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

func GreatestCommonDivisorUint64(a, b uint64) uint64 {
	if a == b {
		return a
	}

	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

func GreatestCommonDivisorInt64(a, b int64) int64 {
	if a == b {
		return a
	}

	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

func GCDListUint64(min, a uint64, b ...uint64) uint64 {
	for _, bb := range b {
		if a <= min || a <= 1 {
			break
		}
		a = GreatestCommonDivisorUint64(a, bb)
	}
	return a
}

func GCDListInt64(min, a int64, b ...int64) int64 {
	for _, bb := range b {
		if a <= min || a <= 1 {
			break
		}
		a = GreatestCommonDivisorInt64(a, bb)
	}
	return a
}

func GCDListInt(min, a int, b ...int) int {
	for _, bb := range b {
		if a <= min || a <= 1 {
			break
		}
		a = GreatestCommonDivisor(a, bb)
	}
	return a
}

// IsPowerOfTwo also returns true for (0)
func IsPowerOfTwo(x uint) bool {
	return x&(x-1) == 0
}

// CeilingPowerOfTwo returns 0 for (0)
func CeilingPowerOfTwo(x uint) uint {
	if IsPowerOfTwo(x) {
		return x
	}
	return 1 << uint(bits.Len(x))
}
