// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmntestapp

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RecordBuilder struct {
	RS crypto.RecordScheme
	RT reference.Template
	DS cryptkit.DataSigner
	PS reference.Holder
}

func (v RecordBuilder) ApplySignature(req *rms.LRegisterRequest) {
	switch {
	case req.AnticipatedRef.IsEmpty():
		panic(throw.IllegalValue())
	case req.OverrideRecordType != 0:
		// override fields are only allowed for overridden types
	case !req.OverrideRootRef.IsEmpty():
		panic(throw.IllegalValue())
	case !req.OverridePrevRef.IsEmpty():
		panic(throw.IllegalValue())
	case !req.OverrideReasonRef.IsEmpty():
		panic(throw.IllegalValue())
	}

	rlv := req.TryGetLazy()
	if rlv.IsEmpty() {
		panic(throw.IllegalValue())
	}

	digester := v.RS.RecordDigester().NewHasher()
	digester.DigestOf(rlv)
	if req.OverrideRecordType != 0 {
		rc := rms.LRegisterRequest{
			OverrideRecordType: req.OverrideRecordType,
			OverrideRootRef: req.OverrideRootRef,
			OverridePrevRef: req.OverridePrevRef,
			OverrideReasonRef: req.OverrideReasonRef,
		}
		b, err := rc.Marshal()
		if err != nil {
			panic(err)
		}
		_, sz, err := protokit.DecodePolymorphFromBytes(b, true)
		if err != nil {
			panic(err)
		}
		digester.DigestBytes(b[sz:])
	}
	digest := digester.SumToDigest()

	sign := v.DS.SignDigest(digest)
	req.ProducerSignature.Set(sign)
	req.ProducedBy.Set(v.PS)
}

func (v RecordBuilder) MakeRequest(record rms.BasicRecord) *rms.LRegisterRequest {
	req := &rms.LRegisterRequest{}
	if err := req.SetAsLazy(record); err != nil {
		panic(err)
	}

	rlv := req.TryGetLazy()
	if rlv.IsEmpty() {
		panic(throw.IllegalValue())
	}

	digester := v.RS.RefDataDigester().NewHasher()
	digest := digester.DigestOf(rlv).SumToDigest()
	ref := v.RT.WithHash(reference.CopyToLocalHash(digest))
	req.AnticipatedRef.Set(ref)

	return req
}

func (v RecordBuilder) MakeSet(records ...*rms.LRegisterRequest) (r inspectsvc.RegisterRequestSet) {
	if len(records) == 0 {
		return
	}

	r.Requests = records

	for _, req := range records {
		switch {
		case req.AnticipatedRef.IsEmpty():
			panic(throw.IllegalValue())
		case req.ProducedBy.IsEmpty():
			v.ApplySignature(req)
		}
	}

	r0 := r.Requests[0]
	rlv := r0.TryGetLazy()
	if rlv.IsEmpty() {
		panic(throw.IllegalValue())
	}

	var err error
	r.Excerpt, err = catalog.ReadExcerptFromLazy(rlv)
	if err != nil {
		panic(err)
	}

	if r0.OverrideRecordType != 0 {
		r.Excerpt.RecordType = r0.OverrideRecordType

		if !r0.OverrideRootRef.IsEmpty() {
			r.Excerpt.RootRef = r0.OverrideRootRef
		}
		if !r0.OverridePrevRef.IsEmpty() {
			r.Excerpt.PrevRef = r0.OverrideRootRef
		}
		if !r0.OverrideReasonRef.IsEmpty() {
			r.Excerpt.ReasonRef = r0.OverrideReasonRef
		}
	}

	return r
}
