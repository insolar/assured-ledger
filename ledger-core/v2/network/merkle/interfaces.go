// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package merkle

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
)

type OriginHash []byte

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/merkle.Calculator -o ../../testutils/merkle -s _mock.go -g
type Calculator interface {
	GetPulseProof(*PulseEntry) (OriginHash, *PulseProof, error)
	GetGlobuleProof(*GlobuleEntry) (OriginHash, *GlobuleProof, error)
	GetCloudProof(*CloudEntry) (OriginHash, *CloudProof, error)

	IsValid(Proof, OriginHash, crypto.PublicKey) bool
}

type Proof interface {
	hash([]byte, *merkleHelper) []byte
	signature() cryptography.Signature
}
