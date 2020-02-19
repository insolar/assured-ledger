// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package args

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

func IsPowerOfTwo(x uint) bool {
	return x&(x-1) == 0
}
