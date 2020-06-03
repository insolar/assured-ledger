// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package refmap

import (
	"hash/fnv"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/unsafekit"
)

func hash32(v longbits.ByteString, seed uint32) uint32 {
	// FNV-1a has a better avalanche property vs FNV-1
	// use of 64 bits improves distribution
	h := fnv.New64a()
	if seed != 0 {
		_, _ = h.Write([]byte{byte(seed), byte(seed >> 8), byte(seed >> 16), byte(seed >> 24)})
	}
	unsafekit.Hash(v, h)
	sum := h.Sum64()
	return uint32(sum) ^ uint32(sum>>32)
}
