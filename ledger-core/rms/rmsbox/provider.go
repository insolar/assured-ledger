// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type DigestProvider interface {
	cryptkit.BasicDigester
	GetDigest() cryptkit.Digest
}

type ReferenceProvider interface {
	GetReference() reference.Global
	TryPullReference() reference.Global
}
