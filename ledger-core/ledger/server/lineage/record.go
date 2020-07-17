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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewRegRecord(excerpt catalog.Excerpt, req *rms.LRegisterRequest) Record {
	return Record{
		Excerpt:            excerpt,
		RecRef:             req.AnticipatedRef.Get(),
		ProducedBy: 		req.ProducedBy.Get(),
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
		// ProducedBy: 		recap.ProducedBy.Get(), // TODO
		recapRec:           recap,
	}
}

type Record struct {
	Excerpt            catalog.Excerpt
	RecRef             reference.Holder
	RecapRef           reference.Holder
	ProducedBy         reference.Holder
	RegisteredBy	   reference.Holder
	RegistrarSignature cryptkit.SignedDigest

	regReq     *rms.LRegisterRequest
	recapRec   *rms.RLineRecap
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
}

func (v Record) IsValid() bool {
	switch {
	case reference.IsEmpty(v.GetRecordRef()):
		return false
	case v.regReq != nil:
		return !v.regReq.AnyRecordLazy.IsZero()
	case v.recapRec != nil:
		return true
	default:
		panic(throw.IllegalState())
	}
}

func (v Record) AsBasicRecord() rms.BasicRecord {
	switch {
	case v.regReq != nil:
		return &v.regReq.AnyRecordLazy
	case v.recapRec != nil:
		return v.recapRec
	default:
		panic(throw.IllegalState())
	}
}
