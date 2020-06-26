// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
<<<<<<< HEAD
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewRegRecord(excerpt catalog.Excerpt, req *rms.LRegisterRequest) Record {
	return Record{
		Excerpt:            excerpt,
		RecRef:             req.AnticipatedRef.Get(),
		regReq:             req,
	}
}

func NewRecapRecord(excerpt catalog.Excerpt, recRef reference.Holder, recap *rms.RLineRecap) Record {
	switch {
	case recRef == nil:
		panic(throw.IllegalValue())
	case recap == nil:
		panic(throw.IllegalValue())
	}

	return Record{
		Excerpt:            excerpt,
		RecRef:             recRef,
		recapRec:           recap,
	}
}

type Record struct {
	Excerpt  catalog.Excerpt
	RecRef   reference.Holder
	RegistrarSignature cryptkit.SignedDigest

	regReq   *rms.LRegisterRequest
	recapRec *rms.RLineRecap
}

func (v Record) Equal(record Record) bool {
	return v.regReq != nil && v.regReq.Equal(record.regReq)
}

func (v Record) GetRecordRef() reference.Holder {
	if v.RecRef != nil {
		return v.RecRef
	}
	if v.regReq != nil {
		return v.regReq.AnticipatedRef.Get()
	}
	return nil
=======
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type Record struct {
	Excerpt catalog.Excerpt
	RegRecord  *rms.LRegisterRequest

	RegistrarSignature cryptkit.SignedDigest
}

func (v Record) Equal(record Record) bool {

>>>>>>> Further work
}

