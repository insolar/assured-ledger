// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jetid

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type JetPrefix = Prefix

type ShortJetId uint32 // JetPrefix + 5bit length

const bitsShortJetIdLen = 5

func (v ShortJetId) Prefix() JetPrefix {
	return JetPrefix(v & ((^ShortJetId(0)) >> bitsShortJetIdLen))
}

func (v ShortJetId) PrefixLength() (uint8, bool) {
	if n := uint8(v >> (32 - bitsShortJetIdLen)); n > 0 {
		return n - 1, true
	}
	return 0, false
}

func (v ShortJetId) HasLength() bool {
	_, ok := v.PrefixLength()
	return ok
}

func (v ShortJetId) String() string {
	if n, ok := v.PrefixLength(); ok {
		return fmt.Sprintf("0x%02X[%d]", v.Prefix(), n)
	} else {
		return fmt.Sprintf("0x%02X[]", v.Prefix())
	}
}

type FullJetId uint64 // ShortJetId + LastSplitPulse

func (v FullJetId) IsValid() bool {
	_, ok := ShortJetId(v).PrefixLength()
	return ok && pulse.IsValidAsPulseNumber(int(v>>32))
}

func (v FullJetId) ShortId() ShortJetId {
	return ShortJetId(v)
}

func (v FullJetId) CreatedAt() pulse.Number {
	return pulse.OfUint32(uint32(v >> 32))
}

func (v FullJetId) String() string {
	return fmt.Sprintf("%v@%d", v.ShortId(), v.CreatedAt())
}
