// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmntestapp

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewRecordBuilderFromDependencies(deps injector.DependencyInjector) RecordBuilder {
	var ps crypto.PlatformScheme
	var ref reference.Global
	deps.MustInject(&ps)
	deps.MustInjectByID(insapp.LocalNodeRefInjectionID, &ref)
	rs := ps.RecordScheme()

	return RecordBuilder{
		RecordScheme:   rs,
		ProducerSigner: cryptkit.AsDataSigner(rs.RecordDigester(), rs.RecordSigner()),
		ProducerRef:    ref,
	}
}

type RecordBuilder struct {
	RecordScheme   crypto.RecordScheme
	RefTemplate    reference.Template
	ProducerSigner cryptkit.DataSigner
	ProducerRef    reference.Holder

	Requests []*rms.LRegisterRequest
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
	if rlv.IsZero() {
		panic(throw.IllegalValue())
	}

	rs := v.RecordScheme.RecordDigester()
	digester := rs.NewHasher()
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

	sign := v.ProducerSigner.SignDigest(digest)
	req.ProducerSignature.Set(sign)
	req.ProducedBy.Set(v.ProducerRef)
}

func (v RecordBuilder) MakeLineStart(record rms.BasicRecord) (RecordBuilder, *rms.LRegisterRequest) {
	initRq := v.buildRequest(record)
	v.RefTemplate = reference.NewRefTemplate(initRq.AnticipatedRef.Get(), v.RefTemplate.LocalHeader().Pulse())
	v.Requests = []*rms.LRegisterRequest{initRq}
	return v, initRq
}

func (v RecordBuilder) BuildRequest(record rms.BasicRecord) *rms.LRegisterRequest {
	return v.buildRequest(record)
}

func (v *RecordBuilder) Add(record rms.BasicRecord) *rms.LRegisterRequest {
	req := v.buildRequest(record)
	v.Requests = append(v.Requests, req)
	return req
}

func (v RecordBuilder) buildRequest(record rms.BasicRecord) *rms.LRegisterRequest {
	req := &rms.LRegisterRequest{}
	if err := req.SetAsLazy(record); err != nil {
		panic(err)
	}

	rlv := req.TryGetLazy()
	if rlv.IsZero() {
		panic(throw.IllegalValue())
	}

	digester := v.RecordScheme.ReferenceDigester().NewHasher()
	digest := digester.DigestOf(rlv).SumToDigest()
	localHash := reference.CopyToLocalHash(digest)

	ref := v.RefTemplate.WithHash(localHash)
	req.AnticipatedRef.Set(ref)

	return req
}

func (v RecordBuilder) MakeSet() inspectsvc.RegisterRequestSet {
	return v.BuildSet(v.Requests...)
}

func (v RecordBuilder) BuildSet(requests ...*rms.LRegisterRequest) (r inspectsvc.RegisterRequestSet) {
	if len(requests) == 0 {
		return
	}

	r.Requests = requests

	for _, req := range requests {
		switch {
		case req.AnticipatedRef.IsEmpty():
			panic(throw.IllegalValue())
		case req.ProducedBy.IsEmpty():
			v.ApplySignature(req)

		}
	}

	r0 := r.Requests[0]
	rlv := r0.AnyRecordLazy.TryGetLazy()
	if rlv.IsZero() {
		panic(throw.IllegalValue())
	}

	var err error
	r.Excerpt, err = catalog.ReadExcerptFromLazy(rlv)
	if err != nil {
		panic(err)
	}

	r.Excerpt.ProducerSignature = r0.ProducerSignature

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
