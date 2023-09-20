package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
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

type RecordTypeSet struct {
	ofs  RecordType
	mask longbits.BitSliceLSB
}

func (v RecordTypeSet) Has(typ RecordType) bool {
	if typ < v.ofs {
		return false
	}
	typ -= v.ofs
	if (typ >> 3) >= RecordType(len(v.mask)) {
		return false
	}
	return v.mask.BitBool(int(typ))
}

func (v RecordTypeSet) IsZero() bool {
	return v.ofs == 0 && v.mask == nil
}

