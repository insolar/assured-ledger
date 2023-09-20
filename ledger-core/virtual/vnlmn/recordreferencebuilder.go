package vnlmn

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

type RecordReferenceBuilder interface {
	AnticipatedRefFromWriterTo(reference.Global, pulse.Number, io.WriterTo) reference.Global
	AnticipatedRefFromBytes(reference.Global, pulse.Number, []byte) reference.Global
	AnticipatedRefFromRecord(reference.Global, pulse.Number, rms.BasicRecord) reference.Global
}

type DefaultRecordReferenceBuilder struct {
	RecordScheme   crypto.RecordScheme
	ProducerSigner cryptkit.DataSigner
	LocalNodeRef   reference.Holder
}

func NewRecordReferenceBuilder(rs crypto.RecordScheme, localNodeRef reference.Holder) *DefaultRecordReferenceBuilder {
	return &DefaultRecordReferenceBuilder{
		RecordScheme:   rs,
		ProducerSigner: cryptkit.AsDataSigner(rs.RecordDigester(), rs.RecordSigner()),
		LocalNodeRef:   localNodeRef,
	}
}

func (s DefaultRecordReferenceBuilder) AnticipatedRefFromWriterTo(object reference.Global, pn pulse.Number, w io.WriterTo) reference.Global {
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

func (s DefaultRecordReferenceBuilder) AnticipatedRefFromBytes(object reference.Global, pn pulse.Number, data []byte) reference.Global {
	buff := bytes.NewBuffer(data)
	return s.AnticipatedRefFromWriterTo(object, pn, buff)
}

func (s DefaultRecordReferenceBuilder) AnticipatedRefFromRecord(object reference.Global, pn pulse.Number, record rms.BasicRecord) reference.Global {
	sRec, ok := record.(rmsreg.GoGoSerializable)
	if !ok {
		panic(throw.IllegalValue())
	}

	var data = make([]byte, sRec.ProtoSize())
	_, err := sRec.MarshalTo(data)
	if err != nil {
		panic(throw.W(err, "Fail to serialize record"))
	}
	return s.AnticipatedRefFromBytes(object, pn, data)
}
