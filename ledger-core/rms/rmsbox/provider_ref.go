package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ ReferenceProvider = &refProvider{}
var _ ReferenceProvider = &rawValueProvider{}

type refProvider struct {
	digestProvider *digestProvider
	template       reference.MutableTemplate
	pullFn         func()
}

func (p *refProvider) SetLocalHash(hash reference.LocalHash) {
	if p.isReady() {
		panic(throw.IllegalState())
	}
	p.template.SetHash(hash)
}

func (p *refProvider) SetZeroValue() {
	if p.isReady() {
		panic(throw.IllegalState())
	}
	p.template.SetZeroValue()
}

func (p *refProvider) setRefByDigest(digest cryptkit.Digest) {
	if digest.FixedByteSize() == 0 {
		p.template.SetZeroValue()
		return
	}
	p.template.SetHash(reference.CopyToLocalHash(digest))
}

func (p *refProvider) isReady() bool {
	return p.digestProvider != nil && p.digestProvider.isReady()
}

func (p *refProvider) getReference() reference.Global {
	if !p.template.HasHash() {
		digest := p.digestProvider.GetDigest()
		if digest.IsZero() {
			return reference.Global{}
		}
		p.setRefByDigest(digest)
	}
	return p.template.MustGlobal()
}

func (p *refProvider) GetReference() reference.Global {
	if p.isReady() {
		return p.getReference()
	}
	return reference.Global{}
}

func (p *refProvider) TryPullReference() reference.Global {
	if !p.isReady() && p.pullFn != nil {
		p.pullFn()
	}
	return p.getReference()
}

type rawValueProvider struct {
	val reference.Global
}

func (r rawValueProvider) GetReference() reference.Global {
	return r.val
}

func (r rawValueProvider) TryPullReference() reference.Global {
	return r.val
}

func NewRawValueProvider(global reference.Global) ReferenceProvider {
	return &rawValueProvider{val: global}
}
