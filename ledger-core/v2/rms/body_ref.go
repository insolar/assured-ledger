// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ GoGoSerializable = &BodyWithReference{}
var _ ReferenceProvider = &BodyWithReference{}
var _ reference.Holder = &BodyWithReference{}

type BodyWithReference struct {
	bodyWithDigest
	template reference.MutableTemplate
}

type bodyWithDigest = BodyWithDigest

func (p *BodyWithReference) ProtoSize() int {
	p.digest.calcDigest(p._digestDataWithRef, nil)
	return p._protoSize()
}

func (p *BodyWithReference) _digestDataWithRef(digester cryptkit.DataDigester) cryptkit.Digest {
	d := p._digestData(digester)

	if d.FixedByteSize() == 0 {
		p.template.SetZeroValue()
		return d
	}

	hash := reference.CopyToLocalHash(d)
	p.template.SetHash(hash)
	return d
}

func (p *BodyWithReference) GetReference() reference.Global {
	if p.digest.isReady() {
		return p.template.MustGlobal()
	}
	return reference.Global{}
}

func (p *BodyWithReference) GetRecordReference() reference.Local {
	if p.digest.isReady() {
		return p.template.MustRecord()
	}
	return reference.Local{}
}

func (p *BodyWithReference) MustReference() reference.Global {
	if d := p.GetReference(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

func (p *BodyWithReference) MustRecordReference() reference.Local {
	if d := p.GetRecordReference(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

func (p *BodyWithReference) GetLocal() reference.Local {
	return p.template.GetLocal()
}

func (p *BodyWithReference) GetBase() reference.Local {
	return p.template.GetBase()
}

func (p *BodyWithReference) IsEmpty() bool {
	return !p.template.HasHash()
}

func (p *BodyWithReference) GetScope() reference.Scope {
	return p.template.GetScope()
}

func (p *BodyWithReference) Equal(o *BodyWithReference) bool {
	switch {
	case p == o:
		return true
	case o == nil || p == nil:
		return false
	case longbits.Equal(p.GetDigest(), o.GetDigest()):
		return true
	case p.template.HasHash() == o.template.HasHash():
		return p.template == o.template
	}
	return false
}
