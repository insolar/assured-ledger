// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inspectsvc

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Service interface {
	InspectRecordSet(RegisterRequestSet) (InspectedRecordSet, error)
}

var _ Service = &serviceImpl{}

func NewService(crypto.RecordScheme) Service {
	return &serviceImpl{}
}

type serviceImpl struct {
	registrarRef reference.Holder
	signer cryptkit.DigestSigner
}

func (p *serviceImpl) InspectRecordSet(set RegisterRequestSet) (irs InspectedRecordSet, err error) {
	irs.Records = make([]lineage.Record, len(set.Requests))
	for i, r := range set.Requests {
		if i == 0 {
			err = p.inspectRecord(r, &irs.Records[0], &set.Excerpt)
		} else {
			err = p.inspectRecord(r, &irs.Records[i], nil)
		}
		if err != nil {
			return InspectedRecordSet{}, err
		}
	}
	return irs, nil
}

func (p *serviceImpl) inspectRecord(r *rms.LRegisterRequest, rec *lineage.Record, exr *catalog.Excerpt) error {
	switch {
	case r == nil:
		return throw.E("nil message")
	case r.AnticipatedRef.IsEmpty():
		return throw.E("empty record ref")
	case r.ProducedBy.IsEmpty():
		return throw.E("empty producer")
	}

	lrv := r.AnyRecordLazy.TryGetLazy()
	if lrv.IsEmpty() {
		return throw.E("empty record")
	}

	sv := p.getProducerSignatureVerifier(r.ProducedBy.Get())
	if sv == nil {
		return throw.E("unknown producer")
	}
	if sv.GetDigestSize() != r.ProducerSignature.FixedByteSize() {
		return throw.E("wrong signature length")
	}

	rd := sv.NewHasher().DigestOf(lrv).SumToDigest()
	rs := cryptkit.NewSignature(r.ProducerSignature.AsByteString(), sv.GetSignatureMethod())

	if !sv.IsValidDigestSignature(rd, rs) {
		return throw.E("signature mismatch")
	}

	if exr != nil {
		*rec = lineage.NewRegRecord(*exr, r)
	} else {
		excerpt, err := catalog.ReadExcerptFromLazy(lrv)
		if err != nil {
			return throw.W(err, "cant read excerpt")
		}
		*rec = lineage.NewRegRecord(excerpt, r)
	}

	p.applyRegistrarSignature(rd, rec)
	return nil
}

func (p *serviceImpl) applyRegistrarSignature(digest cryptkit.Digest, rec *lineage.Record) {
	rec.RegisteredBy = p.registrarRef
	signature := p.signer.SignDigest(digest)
	rec.RegistrarSignature = cryptkit.NewSignedDigest(digest, signature)
}

func (p *serviceImpl) getProducerSignatureVerifier(producer reference.Holder) cryptkit.DataSignatureVerifier {
	_ = producer
	panic(throw.NotImplemented())
}

func (p *serviceImpl) InspectRecap() (InspectedRecordSet, error) {
	panic(throw.NotImplemented())
}

