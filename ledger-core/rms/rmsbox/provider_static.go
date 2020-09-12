// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewDigestValue(digest cryptkit.Digest) DigestProvider {
	if digest.IsEmpty() {
		panic(throw.IllegalValue())
	}
	return digestValue{digest}
}

type digestValue struct {
	digest cryptkit.Digest
}

func (v digestValue) GetDigestMethod() cryptkit.DigestMethod {
	return v.digest.GetDigestMethod()
}

func (v digestValue) GetDigestSize() int {
	return v.digest.FixedByteSize()
}

func (v digestValue) GetDigest() cryptkit.Digest {
	return v.digest
}

/*****************************/

func NewRefValue(digest cryptkit.Digest, t reference.Template) ReferenceProvider {
	switch {
	case digest.IsEmpty():
		panic(throw.IllegalValue())
	case t.IsZero():
		panic(throw.IllegalValue())
	}

	rv := refValue{ref: t.AsMutable()}
	rv.ref.SetHash(reference.CopyToLocalHash(digest))
	return rv
}

type refValue struct {
	ref reference.MutableTemplate
}

func (v refValue) GetReference() reference.Global {
	return v.ref.MustGlobal()
}

func (v refValue) TryPullReference() reference.Global {
	return v.ref.MustGlobal()
}
