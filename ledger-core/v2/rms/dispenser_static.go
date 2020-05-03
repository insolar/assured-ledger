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

func NewDigestValue(digest cryptkit.Digest) DigestDispenser {
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

func NewRefValue(digest cryptkit.Digest, t reference.Template) ReferenceDispenser {
	if digest.IsEmpty() {
		panic(throw.IllegalValue())
	}
	t.IsZero()
	return refValue{digest}
}

type refValue struct {
	digest cryptkit.Digest
	ref    reference.Global
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
