// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ DigestProvider = &digestProvider{}

type digestProvider struct {
	ready    atomickit.StartStopFlag // IsStarted() - has digester, IsStopped() - has digest
	digester cryptkit.DataDigester
	digest   cryptkit.Digest
}

func (p *digestProvider) isReady() bool {
	return p.ready.IsStopped()
}

func (p *digestProvider) setDigester(digester cryptkit.DataDigester, setFn func()) {
	switch {
	case digester == nil:
		panic(throw.IllegalValue())
	case p.ready.DoStart(func() {
		p.digester = digester
		if setFn != nil {
			setFn()
		}
	}):
	default:
		panic(throw.IllegalState())
	}
}

func (p *digestProvider) setDigest(digest cryptkit.Digest, setFn func(cryptkit.Digest)) {
	switch {
	case digest.IsEmpty():
		panic(throw.IllegalValue())
	case p.ready.DoDiscardByOne(func(bool) {
		switch {
		case p.digester == nil:
		case p.digester.GetDigestMethod() != digest.GetDigestMethod():
			panic(throw.IllegalValue())
		default:
			p.digester = nil
		}
		p.digest = digest
		if setFn != nil {
			setFn(digest)
		}
	}):
	case p.digest.Equals(digest):
	default:
		panic(throw.IllegalState())
	}
}

func (p *digestProvider) calcDigest(fn func(cryptkit.DataDigester) cryptkit.Digest, setFn func(cryptkit.Digest)) {
	switch {
	case fn == nil:
		panic(throw.IllegalValue())
	case p.ready.DoDiscardByOne(func(bool) {
		digester := p.digester
		p.digester = nil
		digest := fn(digester)
		switch {
		case digester == nil:
			//
		case digester.GetDigestMethod() != digest.GetDigestMethod():
			panic(throw.IllegalValue())
		}
		p.digest = digest
		if setFn != nil {
			setFn(digest)
		}
	}):
	default:
		panic(throw.IllegalState())
	}
}

func (p *digestProvider) tryCancel(setFn func(cryptkit.Digest)) bool {
	return p.ready.DoDiscardByOne(func(bool) {
		p.digester = nil
		digest := cryptkit.Digest{}
		p.digest = digest
		if setFn != nil {
			setFn(digest)
		}
	})
}

func (p *digestProvider) GetDigest() cryptkit.Digest {
	if p.isReady() {
		return p.digest
	}
	return cryptkit.Digest{}
}

func (p *digestProvider) MustDigest() cryptkit.Digest {
	if d := p.GetDigest(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}
