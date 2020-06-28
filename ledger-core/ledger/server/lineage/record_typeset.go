// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type RecordTypeSet struct {
	ofs  RecordType
	mask longbits.BitSliceLSB
}

func (v RecordTypeSet) Has(typ RecordType) bool {
	if typ < v.ofs {
		return false
	}
	typ -= v.ofs
	if typ >= RecordType(len(v.mask)) {
		return false
	}
	return v.mask.BitBool(int(typ))
}

func (v RecordTypeSet) IsZero() bool {
	return v.ofs == 0 && v.mask == nil
}
