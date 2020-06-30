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
	Excerpt  catalog.Excerpt
	RegReq   *rms.LRegisterRequest
	RecapRec *rms.RLineRecap

	RegistrarSignature cryptkit.SignedDigest
}

func (v Record) Equal(record Record) bool {
	return v.RegReq != nil && v.RegReq.Equal(record.RegReq)
}

func (v Record) GetRecordRef() reference.Holder {
	if v.RegReq == nil {
		return nil
	}
	return v.RegReq.AnticipatedRef.Get()
}

