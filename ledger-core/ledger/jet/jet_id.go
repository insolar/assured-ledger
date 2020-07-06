// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"fmt"
	"math"
	"math/bits"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Tree = *PrefixTree

type ID uint16

func (v ID) MinLen() int {
	return bits.Len16(uint16(v))
}

func (v ID) AsPrefixed(prefixLen uint8) PrefixedID {
	if prefixLen > 16 {
		panic(throw.IllegalValue())
	}
	return (PrefixedID(prefixLen) << prefixedLenOfs) | PrefixedID(v)
}

func (v ID) AsLeg(prefixLen uint8, createdAt pulse.Number) LegID {
	return v.AsPrefixed(prefixLen).AsLeg(createdAt)
}

func (v ID) AsDrop(createdAt pulse.Number) DropID {
	if !createdAt.IsTimePulse() {
		panic(throw.IllegalValue())
	}
	return DropID(createdAt) | (DropID(v)<<32)
}

/***************************************************************/

type PrefixedID uint32 // JetPrefix + 5bit length

const (
	prefixedLenBits = 8
	prefixedLenOfs = 32 - prefixedLenBits
)

const (
	GenesisPrefixedID PrefixedID = 1<<prefixedLenOfs
	UnknownPrefixedID PrefixedID = 0
)

func (v PrefixedID) ID() ID {
	return ID(v)
}

func (v PrefixedID) Prefix() Prefix {
	return Prefix(v.ID())
}

func (v PrefixedID) PrefixLength() (uint8, bool) {
	if n := uint8(v >> prefixedLenOfs); n > 0 {
		return n - 1, true
	}
	return 0, false
}

func (v PrefixedID) HasLength() bool {
	_, ok := v.PrefixLength()
	return ok
}

func (v PrefixedID) AsLeg(createdAt pulse.Number) LegID {
	if !createdAt.IsTimePulse() {
		panic(throw.IllegalValue())
	}
	return LegID(createdAt) | (LegID(v)<<32)
}

func (v PrefixedID) String() string {
	if n, ok := v.PrefixLength(); ok {
		return fmt.Sprintf("0x%02X[%d]", v.Prefix(), n)
	} else {
		return fmt.Sprintf("0x%02X[]", v.Prefix())
	}
}

/***************************************************************/

type LegID uint64 // PrefixedID + Split/Merge Pulse

func (v LegID) IsValid() bool {
	_, ok := v.PrefixedID().PrefixLength()
	return ok && pulse.IsValidAsPulseNumber(int(v&math.MaxUint32))
}

func (v LegID) PrefixedID() PrefixedID {
	return PrefixedID(v>>32)
}

func (v LegID) CreatedAt() pulse.Number {
	return pulse.OfUint32(uint32(v))
}

func (v LegID) String() string {
	return fmt.Sprintf("%v@%d", v.PrefixedID(), v.CreatedAt())
}

func (v LegID) AsDrop() DropID {
	return v.PrefixedID().ID().AsDrop(v.CreatedAt())
}

/***************************************************************/

type DropID uint64 // ID + current Pulse

func (v DropID) IsValid() bool {
	_, ok := v.prefixedID().PrefixLength()
	return !ok && pulse.IsValidAsPulseNumber(int(v&math.MaxUint32))
}

func (v DropID) prefixedID() PrefixedID {
	return PrefixedID(v>>32)
}

func (v DropID) ID() ID {
	return v.prefixedID().ID()
}

func (v DropID) CreatedAt() pulse.Number {
	return pulse.OfUint32(uint32(v))
}

func (v DropID) String() string {
	return fmt.Sprintf("%v@%d", v.ID(), v.CreatedAt())
}

type Prefix uint32

