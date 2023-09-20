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
