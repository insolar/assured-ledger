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
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Service interface {
	InspectRecordSet(RegisterRequestSet) (InspectedRecordSet, error)
}

var _ Service = &serviceImpl{}

func NewService(registrarRef reference.Holder, rs crypto.RecordScheme) Service {
	digester := rs.RecordDigester()
	signer := rs.RecordSigner()

	return &serviceImpl{
		registrarRef: reference.Copy(registrarRef),
		digester: digester,
		signer:   signer,
		localSV:  rs.SelfVerifier(),
		signatureMethod: digester.GetDigestMethod().SignedBy(signer.GetSigningMethod()),
	}
}

type serviceImpl struct {
	registrarRef    reference.Global
	digester        crypto.RecordDigester
	signer          cryptkit.DigestSigner
	localSV         cryptkit.SignatureVerifier
	signatureMethod cryptkit.SignatureMethod
}

func (p *serviceImpl) InspectRecordSet(set RegisterRequestSet) (irs InspectedRecordSet, err error) {
	if len(set.Requests) == 0 {
		return
	}

	irs.Records = make([]lineage.Record, len(set.Requests))

	if err = p.inspectRecord(set.Requests[0], &irs.Records[0], &set.Excerpt, nil); err != nil {
		return InspectedRecordSet{}, err
	}
	producer := set.Requests[0].ProducedBy.Get()

	for i := 1; i < len(set.Requests); i++ {
		err = p.inspectRecord(set.Requests[i], &irs.Records[i], nil, producer)
		if err != nil {
			return InspectedRecordSet{}, err
		}
	}
	return irs, nil
}

//nolint
func (p *serviceImpl) inspectRecord(req *rms.LRegisterRequest, rec *lineage.Record, exr *catalog.Excerpt, producer reference.Holder) error {
	switch {
	case req == nil:
		return throw.E("nil message")
	case req.AnticipatedRef.IsEmpty():
		return throw.E("empty record ref")
	case !req.ProducedBy.IsEmpty():
		producer = req.ProducedBy.Get()
	case producer == nil:
		return throw.E("empty producer")
	}

	lrv := req.AnyRecordLazy.TryGetLazy()
	if lrv.IsZero() {
		return throw.E("lazy record is required")
	}

	sv := p.getProducerSignatureVerifier(producer)
	if sv == nil {
		return throw.E("unknown producer")
	}
	if p.digester.GetDigestSize() != req.ProducerSignature.FixedByteSize() {
		return throw.E("wrong signature length")
	}

	rh := p.digester.NewDataAndRefHasher()
	rh.DigestOf(lrv)

	var refDigest cryptkit.Digest
 	rh, refDigest = p.digester.GetRefDigestAndContinueData(rh)

	if recRef := req.AnticipatedRef.Get().GetLocal(); !longbits.Equal(recRef.IdentityHash(), refDigest) {
		return throw.E("reference mismatched content")
	}

	// TODO make a separate function in RMS
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
		rh.DigestBytes(b[sz:])
	}

	dataDigest := rh.SumToDigest()

	rs := cryptkit.NewSignature(req.ProducerSignature.AsByteString(), p.signatureMethod)

	if !sv.IsValidDigestSignature(dataDigest, rs) {
		return throw.E("signature mismatch")
	}

	if exr != nil {
		*rec = lineage.NewRegRecord(*exr, req)
	} else {
		excerpt, err := catalog.ReadExcerptFromLazy(lrv)
		if err != nil {
			return throw.W(err, "cant read excerpt")
		}
		*rec = lineage.NewRegRecord(excerpt, req)
	}

	p.applyRegistrarSignature(dataDigest, rec)
	return nil
}

func (p *serviceImpl) applyRegistrarSignature(digest cryptkit.Digest, rec *lineage.Record) {
	rec.RegisteredBy = p.registrarRef
	signature := p.signer.SignDigest(digest)
	rec.RegistrarSignature = cryptkit.NewSignedDigest(digest, signature)
}

func (p *serviceImpl) getProducerSignatureVerifier(producer reference.Holder) cryptkit.SignatureVerifier {
	if p.registrarRef.Equal(producer) {
		return p.localSV
	}
	// TODO find node profile
	return nil
}

func (p *serviceImpl) InspectRecap() (InspectedRecordSet, error) {
	panic(throw.NotImplemented())
}

