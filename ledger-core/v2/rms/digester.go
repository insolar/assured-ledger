// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type DigestProvider interface {
	GetDigest() cryptkit.Digest
	MustDigest() cryptkit.Digest
}

func NewDigestValue(digest cryptkit.Digest) DigestProvider {
	if digest.IsEmpty() {
		panic(throw.IllegalValue())
	}
	return digestValue{digest}
}

func NewDigestProvider(digest cryptkit.Digest) DigestProvider {
	if digest.IsEmpty() {
		panic(throw.IllegalValue())
	}
	return digestValue{digest}
}

type digestOnce struct {
	digest   cryptkit.Digest
	digestFn func([]byte) cryptkit.Digest
}

func (p *digestOnce) DigestBytes(b []byte) error {
	if !p.digest.IsEmpty() {
		return nil
	}
	p.digest = p.digestFn(b)
	p.digestFn = nil
	return nil
}

func (p *digestOnce) MustDigest() cryptkit.Digest {
	if d := p.GetDigest(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

func (p *digestOnce) GetDigest() cryptkit.Digest {
	return p.digest
}

/**********************************/

type digestValue struct {
	cryptkit.Digest
}

func (v digestValue) GetDigest() cryptkit.Digest {
	return v.Digest
}

func (v digestValue) MustDigest() cryptkit.Digest {
	if d := v.GetDigest(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}
