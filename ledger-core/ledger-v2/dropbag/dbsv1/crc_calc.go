// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dbsv1

import (
	"fmt"
	"hash"
)

func calcCrc32(hasher hash.Hash32, b []byte) hash.Hash32 {
	switch n, err := hasher.Write(b); {
	case err != nil:
		panic(err)
	case n != len(b):
		panic(fmt.Errorf("internal error, crc calc failed: written=%d expected=%d", n, len(b)))
	}
	return hasher
}

func addCrc32(hash hash.Hash32, x uint32) {
	// byte order is according to crc32.appendUint32
	if n, err := hash.Write([]byte{byte(x >> 24), byte(x >> 16), byte(x >> 8), byte(x)}); err != nil || n != 4 {
		panic(fmt.Errorf("crc calc failure: %d, %v", n, err))
	}
}
