// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package args

import "sort"

const PrimeCount = 6550
const MaxPrime = 65581

func Prime(i int) int {
	switch n := len(primes); {
	case i >= n:
		return int(primesOverUint16[i-n])
	default:
		return int(primes[i])
	}
}

func Primes() PrimeList {
	return allPrimes
}

func OddPrimes() PrimeList {
	return oddPrimes
}

func IsolatedOddPrimes() PrimeList {
	return isolatedOddPrimes
}

var allPrimes = PrimeList{primes, primesOverUint16[0]}
var oddPrimes = PrimeList{primes[1:], primesOverUint16[0]}
var isolatedOddPrimes = PrimeList{isolatedPrimes[1:], isolatedPrimeOverUint16[0]}

type PrimeList struct {
	primes   []uint16
	maxPrime uint
}

func (m PrimeList) search(v int) uint {
	return uint(sort.Search(len(m.primes), func(i int) bool { return int(m.primes[i]) >= v }))
}

func (m PrimeList) Len() int {
	return len(m.primes) + 1
}

func (m PrimeList) Prime(i int) int {
	if len(m.primes) == i {
		return int(m.maxPrime)
	}
	return int(m.primes[i])
}

func (m PrimeList) Ceiling(v int) int {
	return m.Prime(int(m.search(v)))
}

func (m PrimeList) Floor(v int) int {
	switch n := m.search(v); {
	case n == 0:
		return int(m.primes[0])
	case n == uint(len(m.primes)):
		if v >= int(m.maxPrime) {
			return int(m.maxPrime)
		}
		return int(m.primes[len(m.primes)-1])
	case int(m.primes[n]) == v:
		return v
	default:
		return int(m.primes[n-1])
	}
}

func (m PrimeList) Nearest(v int) int {
	switch n := m.search(v); {
	case n == 0:
		return int(m.primes[0])
	case n == uint(len(m.primes)):
		return int(m.maxPrime)
	case (v - int(m.primes[n-1])) <= (int(m.primes[n]) - v):
		return int(m.primes[n-1])
	default:
		return int(m.primes[n])
	}
}

func (m PrimeList) Max() int {
	return int(m.maxPrime)
}

func (m PrimeList) Min() int {
	return int(m.primes[0])
}
