// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ GoGoSerializable = &BodyWithReference{}
var _ ReferenceDispenser = &BodyWithReference{}

type BodyWithReference struct {
	bodyWithDigest
	template reference.MutableTemplate
}

type bodyWithDigest = BodyWithDigest

func (p *BodyWithReference) ProtoSize() int {
	p.digest.calcDigest(p._digestDataWithRef)
	return p._protoSize()
}

func (p *BodyWithReference) _digestDataWithRef(digester cryptkit.DataDigester) cryptkit.Digest {
	d := p._digestData(digester)
	p.template.SetHash(reference.CopyToLocalHash(d))
	return d
}

func (p *BodyWithReference) GetReference() reference.Global {
	if p.digest.isReady() {
		return p.template.MustGlobal()
	}
	return reference.Global{}
}

func (p *BodyWithReference) GetLocalReference() reference.Local {
	if p.digest.isReady() {
		return p.template.MustLocal()
	}
	return reference.Local{}
}

func (p *BodyWithReference) MustReference() reference.Global {
	if d := p.GetReference(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

func (p *BodyWithReference) MustLocalReference() reference.Local {
	if d := p.GetLocalReference(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}
