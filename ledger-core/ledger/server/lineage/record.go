// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type Record struct {
	Excerpt catalog.Excerpt
	RegRecord  *rms.LRegisterRequest

	RegistrarSignature cryptkit.SignedDigest
}

func (v Record) Equal(record Record) bool {
	return v.RegRecord != nil && v.RegRecord.Equal(record.RegRecord)
}

func (v Record) GetRecordRef() reference.Holder {
	if v.RegRecord == nil {
		return nil
	}
	return v.RegRecord.AnticipatedRef.Get()
}

