// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
<<<<<<< HEAD
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func setOf(x... RecordType) RecordTypeSet {
	if len(x) == 0 {
		return RecordTypeSet{}
	}

	bb := longbits.NewBitBuilder(longbits.LSB, 32)
	for _, mt := range x {
		if mt > maxRecordType {
			panic(throw.IllegalValue())
		}
		bb.SetBit(int(mt), 1)
	}
	skip, b := bb.TrimZeros()
	return RecordTypeSet{RecordType(skip<<3), b}
}

=======
)

>>>>>>> Further work
type RecordTypeSet struct {
	ofs  RecordType
	mask longbits.BitSliceLSB
}

func (v RecordTypeSet) Has(typ RecordType) bool {
	if typ < v.ofs {
		return false
	}
	typ -= v.ofs
<<<<<<< HEAD
	if (typ >> 3) >= RecordType(len(v.mask)) {
=======
	if typ >= RecordType(len(v.mask)) {
>>>>>>> Further work
		return false
	}
	return v.mask.BitBool(int(typ))
}

func (v RecordTypeSet) IsZero() bool {
	return v.ofs == 0 && v.mask == nil
}
<<<<<<< HEAD

=======
>>>>>>> Further work
