package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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

//nolint:unused
func (p *digestProvider) setDigest(digest cryptkit.Digest, setFn func(cryptkit.Digest)) {
	switch {
	case digest.IsEmpty():
		panic(throw.IllegalValue())
	case p.ready.DoDiscardByOne(func(bool) {
		if p.digester != nil && p.digester.GetDigestMethod() != digest.GetDigestMethod() {
			panic(throw.IllegalValue())
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
		digest := fn(p.digester)
		if p.digester != nil && p.digester.GetDigestMethod() != digest.GetDigestMethod() {
			panic(throw.IllegalValue())
		}
		if digest.IsZero() {
			digest = cryptkit.NewZeroSizeDigest(digest.GetDigestMethod())
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

//nolint:unused
func (p *digestProvider) tryCancel(setFn func(cryptkit.Digest)) bool {
	return p.ready.DoDiscardByOne(func(bool) {
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

func (p *digestProvider) GetDigestMethod() cryptkit.DigestMethod {
	switch {
	case p.isReady():
		return p.digest.GetDigestMethod()
	case p.ready.WasStarted():
		return p.digester.GetDigestMethod()
	}
	panic(throw.IllegalState())
}

func (p *digestProvider) GetDigestSize() int {
	switch {
	case p.isReady():
		return p.digest.FixedByteSize()
	case p.ready.WasStarted():
		return p.digester.GetDigestSize()
	}
	panic(throw.IllegalState())
}
