// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type DigestProvider interface {
	GetDigest() cryptkit.Digest
	MustDigest() cryptkit.Digest
}

type ReferenceProvider interface {
	GetReference() reference.Global
	MustReference() reference.Global

	GetRecordReference() reference.Local
	MustRecordReference() reference.Local
}

type ReferencableRecord interface {
	InitFieldMap(reset bool)
	GetFieldMap() *insproto.FieldMap
	GetBody() *RecordBody
}

func ReferenceProviderFor(r ReferencableRecord, digester cryptkit.DataDigester, header reference.LocalHeader) ReferenceProvider {
	r.InitFieldMap(false)
	fm := r.GetFieldMap()

	var rd *referenceDispenser
	if fm.Callback != nil {
		rd = fm.Callback.(*referenceDispenser)
		rd.checkDigester(digester)
	} else {
		rd := &referenceDispenser{}
		rd.setDigester(digester)
		fm.Callback = rd
	}
	return rd.createRefProvider(header)
}

type referenceDispenser struct {
	dp digestProvider
	dm cryptkit.DigestMethod
}

func (p *referenceDispenser) OnMessage(fieldMap *insproto.FieldMap) {
	// fieldMap.Message
	p.dp.calcDigest(func(digester cryptkit.DataDigester) cryptkit.Digest {
		return digester.DigestBytes(fieldMap.Message)
	}, nil)
}

func (p *referenceDispenser) setDigester(digester cryptkit.DataDigester) {
	p.dp.setDigester(digester, nil)
	p.dm = digester.GetDigestMethod()
}

func (p *referenceDispenser) checkDigester(digester cryptkit.DataDigester) {
	if p.dm != digester.GetDigestMethod() {
		panic(throw.IllegalValue())
	}
}

func (p *referenceDispenser) createRefProvider(header reference.LocalHeader) ReferenceProvider {

}
