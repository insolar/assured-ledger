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

var _ DigestProvider = &refProvider{}
var _ ReferenceProvider = &refProvider{}

type refProvider struct {
	digestProvider
	template reference.MutableTemplate
}

func (p *refProvider) setDigest(digest cryptkit.Digest, setFn func(cryptkit.Digest)) {
	p.digestProvider.setDigest(digest, func(digest cryptkit.Digest) {
		p.setRefByDigest(digest)
		if setFn != nil {
			setFn(digest)
		}
	})
}

func (p *refProvider) calcDigest(fn func(cryptkit.DataDigester) cryptkit.Digest, setFn func(cryptkit.Digest)) {
	p.digestProvider.calcDigest(fn, func(digest cryptkit.Digest) {
		p.setRefByDigest(digest)
		if setFn != nil {
			setFn(digest)
		}
	})
}

func (p *refProvider) GetReference() reference.Global {
	if p.isReady() {
		return p.template.MustGlobal()
	}
	return reference.Global{}
}

func (p *refProvider) GetRecordReference() reference.Local {
	if p.isReady() {
		return p.template.MustRecord()
	}
	return reference.Local{}
}

func (p *refProvider) MustReference() reference.Global {
	if d := p.GetReference(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

func (p *refProvider) MustRecordReference() reference.Local {
	if d := p.GetRecordReference(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

func (p *refProvider) setRefByDigest(digest cryptkit.Digest) {
	p.template.SetHash(reference.CopyToLocalHash(digest))
}
