// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// Based on Go runtime

package fastrand

import (
	"os"
	"sync/atomic"
	"time"
)

var seed uint64 // atomic

func init() {
	pidrand := 1597334677 * uint32(os.Getpid())

	nano := uint64(time.Now().UnixNano())
	cputicks := nano ^ (nano >> 32)

	seed = uint64(pidrand) | cputicks<<32
	if seed == 0 {
		seed = 1 << 32
	}
}

func Uint32() uint32 {
	// Implement xorshift64+: 2 32-bit xorshift sequences added together.
	// Shift triplet [17,7,16] was calculated as indicated in Marsaglia's
	// Xorshift paper: https://www.jstatsoft.org/article/view/v008i14/xorshift.pdf
	// This generator passes the SmallCrush suite, part of TestU01 framework:
	// http://simul.iro.umontreal.ca/testu01/tu01.html
	s := atomic.LoadUint64(&seed)
	s1, s0 := uint32(s), uint32(s>>32)
	s1 ^= s1 << 17
	s1 = s1 ^ s0 ^ s1>>7 ^ s0>>16
	atomic.StoreUint64(&seed, uint64(s0)|uint64(s1)<<32)
	return s0 + s1
}

func Intn(n uint32) uint32 {
	// This is similar to Uint32() % n, but faster.
	// See https://lemire.me/blog/2016/06/27/a-fast-alternative-to-the-modulo-reduction/
	return uint32(uint64(Uint32()) * uint64(n) >> 32)
}
