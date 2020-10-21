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

type Prefix uint32

func (v Prefix) AsID() ID {
	return ID(v)
}

type ID uint16

const IDBitLen = 16

func (v ID) BitLen() int {
	return bits.Len16(uint16(v))
}

func (v ID) AsExact(bitLen uint8) ExactID {
	switch {
	case bitLen > 16:
		panic(throw.IllegalValue())
	case v >>bitLen != 0:
		panic(throw.IllegalValue())
	}

	return (ExactID(1 + bitLen) << exactLenOfs) | ExactID(v)
}

func (v ID) AsLeg(bitLen uint8, createdAt pulse.Number) LegID {
	return v.AsExact(bitLen).AsLeg(createdAt)
}

func (v ID) AsDrop(createdAt pulse.Number) DropID {
	if !createdAt.IsTimePulse() {
		panic(throw.IllegalValue())
	}
	return DropID(createdAt) | (DropID(v)<<32)
}

func (v ID) AsPrefix() Prefix {
	return Prefix(v)
}

/***************************************************************/

func NoLengthExactID(id ID) ExactID {
	return ExactID(id)
}

type ExactID uint32 // ID + MSB 8bit length

const (
	exactLenBits = 8
	exactLenOfs  = 32 - exactLenBits
)

const (
	GenesisExactID ExactID = 1<< exactLenOfs
	UnknownExactID ExactID = 0
)

func (v ExactID) IsZero() bool {
	return v == 0
}

func (v ExactID) ID() ID {
	return ID(v)
}

func (v ExactID) BitLen() uint8 {
	n, ok := v.bitLength()
	if !ok {
		panic(throw.IllegalState())
	}
	return n
}

func (v ExactID) bitLength() (uint8, bool) {
	if n := uint8(v >> exactLenOfs); n > 0 {
		return n - 1, true
	}
	return 0, false
}

func (v ExactID) HasLength() bool {
	_, ok := v.bitLength()
	return ok
}

func (v ExactID) AsDrop(pn pulse.Number) DropID {
	if v == UnknownExactID {
		panic(throw.IllegalState())
	}
	return v.ID().AsDrop(pn)
}

func (v ExactID) AsLeg(createdAt pulse.Number) LegID {
	if v == UnknownExactID {
		panic(throw.IllegalState())
	}
	if !createdAt.IsTimePulse() {
		panic(throw.IllegalValue())
	}
	return LegID(createdAt) | (LegID(v)<<32)
}

func (v ExactID) String() string {
	if n, ok := v.bitLength(); ok {
		return fmt.Sprintf("0x%02X[%d]", v.ID(), n)
	}
	return fmt.Sprintf("0x%02X[]", v.ID())
}

/***************************************************************/

type LegID uint64 // ExactID + Split/Merge Pulse

func (v LegID) IsValid() bool {
	return v.ExactID().HasLength() && pulse.IsValidAsPulseNumber(int(v&math.MaxUint32))
}

func (v LegID) ExactID() ExactID {
	return ExactID(v>>32)
}

func (v LegID) CreatedAt() pulse.Number {
	return pulse.OfUint32(uint32(v))
}

func (v LegID) String() string {
	return fmt.Sprintf("%v@%v", v.ExactID(), v.CreatedAt())
}

func (v LegID) AsDrop() DropID {
	return v.ExactID().ID().AsDrop(v.CreatedAt())
}

/***************************************************************/

type DropID uint64 // ID + current Pulse

func (v DropID) IsValid() bool {
	return !v.exactID().HasLength() && pulse.IsValidAsPulseNumber(int(v&math.MaxUint32))
}

func (v DropID) exactID() ExactID {
	return ExactID(v>>32)
}

func (v DropID) ID() ID {
	return v.exactID().ID()
}

func (v DropID) CreatedAt() pulse.Number {
	return pulse.OfUint32(uint32(v))
}

func (v DropID) String() string {
	return fmt.Sprintf("%v@%d", v.ID(), v.CreatedAt())
}

