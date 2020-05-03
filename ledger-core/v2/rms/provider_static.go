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

func NewDigestValue(digest cryptkit.Digest) DigestProvider {
	if digest.IsEmpty() {
		panic(throw.IllegalValue())
	}
	return digestValue{digest}
}

type digestValue struct {
	digest cryptkit.Digest
}

func (v digestValue) GetDigest() cryptkit.Digest {
	return v.digest
}

func (v digestValue) MustDigest() cryptkit.Digest {
	if d := v.GetDigest(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}

/*****************************/

func NewRefValue(digest cryptkit.Digest, t reference.Template) ReferenceProvider {
	switch {
	case digest.IsEmpty():
		panic(throw.IllegalValue())
	case t.IsZero():
		panic(throw.IllegalValue())
	}

	rv := refValue{digest: digest, ref: t.AsMutable()}
	rv.ref.SetHash(reference.CopyToLocalHash(digest))
	return rv
}

type refValue struct {
	digest cryptkit.Digest
	ref    reference.MutableTemplate
}

func (v refValue) GetReference() reference.Global {
	return v.ref.MustGlobal()
}

func (v refValue) MustReference() reference.Global {
	return v.ref.MustGlobal()
}

func (v refValue) GetRecordReference() reference.Local {
	return v.ref.MustRecord()
}

func (v refValue) MustRecordReference() reference.Local {
	return v.ref.MustRecord()
}

func (v refValue) GetDigest() cryptkit.Digest {
	return v.digest
}

func (v refValue) MustDigest() cryptkit.Digest {
	if d := v.GetDigest(); !d.IsEmpty() {
		return d
	}
	panic(throw.IllegalState())
}
