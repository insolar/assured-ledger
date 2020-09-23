// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmn

import (
	"bytes"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RecordReferenceBuilderService interface {
	AnticipatedRefFromWriterTo(reference.Global, pulse.Number, io.WriterTo) reference.Global
	AnticipatedRefFromBytes(reference.Global, pulse.Number, []byte) reference.Global
	AnticipatedRefFromRecord(reference.Global, pulse.Number, rms.BasicRecord) reference.Global
}

type RecordReferenceBuilder struct {
	RecordScheme   crypto.RecordScheme
	ProducerSigner cryptkit.DataSigner
	LocalNodeRef   reference.Holder
}

func NewRecordReferenceBuilder(rs crypto.RecordScheme, localNodeRef reference.Holder) *RecordReferenceBuilder {
	return &RecordReferenceBuilder{
		RecordScheme:   rs,
		ProducerSigner: cryptkit.AsDataSigner(rs.RecordDigester(), rs.RecordSigner()),
		LocalNodeRef:   localNodeRef,
	}
}

func (s RecordReferenceBuilder) AnticipatedRefFromWriterTo(object reference.Global, pn pulse.Number, w io.WriterTo) reference.Global {
	digester := s.RecordScheme.ReferenceDigester().NewHasher()
	digest := digester.DigestOf(w).SumToDigest()
	localHash := reference.CopyToLocalHash(digest)

	var refTemplate reference.Template

	if object.IsEmpty() {
		refTemplate = reference.NewSelfRefTemplate(pn, reference.SelfScopeLifeline)
	} else {
		refTemplate = reference.NewRefTemplate(object, pn)
	}

	return refTemplate.WithHash(localHash)
}

func (s RecordReferenceBuilder) AnticipatedRefFromBytes(object reference.Global, pn pulse.Number, data []byte) reference.Global {
	buff := bytes.NewBuffer(data)
	return s.AnticipatedRefFromWriterTo(object, pn, buff)
}

func (s RecordReferenceBuilder) AnticipatedRefFromRecord(object reference.Global, pn pulse.Number, record rms.BasicRecord) reference.Global {
	sRec, ok := record.(rmsreg.GoGoSerializable)
	if !ok {
		panic(throw.IllegalValue())
	}

	var data = make([]byte, 0)
	_, err := sRec.MarshalTo(data)
	if err != nil {
		panic(throw.W(err, "Fail to serialize record"))
	}
	return s.AnticipatedRefFromBytes(object, pn, data)
}
