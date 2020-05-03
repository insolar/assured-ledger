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

type ReferenceDispenser interface {
	GetReference() reference.Global
	MustReference() reference.Global

	GetLocalReference() reference.Local
	MustLocalReference() reference.Local
}

var _ DigestDispenser = &refDispenser{}
var _ ReferenceDispenser = &refDispenser{}

type refDispenser struct {
	digestDispenser
	template reference.MutableTemplate
}

func (p *refDispenser) setDigest(digest cryptkit.Digest, setFn func(cryptkit.Digest)) {
	p.digestDispenser.setDigest(digest, func(digest cryptkit.Digest) {
		p.setRefByDigest(digest)
		if setFn != nil {
			setFn(digest)
		}
	})
}

func (p *refDispenser) calcDigest(fn func(cryptkit.DataDigester) cryptkit.Digest, setFn func(cryptkit.Digest)) {
	p.digestDispenser.calcDigest(fn, func(digest cryptkit.Digest) {
		p.setRefByDigest(digest)
		if setFn != nil {
			setFn(digest)
		}
	})
}

func (p *refDispenser) GetReference() reference.Global {
	if p.isReady() {
		return p.template.MustGlobal()
	}
	return reference.Global{}
}

func (p *refDispenser) GetLocalReference() reference.Local {
	if p.isReady() {
		return p.template.MustLocal()
	}
	return reference.Local{}
}

func (p *refDispenser) MustReference() reference.Global {
	if d := p.GetReference(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

func (p *refDispenser) MustLocalReference() reference.Local {
	if d := p.GetLocalReference(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

func (p *refDispenser) setRefByDigest(digest cryptkit.Digest) {
	p.template.SetHash(reference.CopyToLocalHash(digest))
}
