// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statevector

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/proofs"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

type Vector struct {
	Bitset member.StateBitset

	Trusted SubVector
	Doubted SubVector
}

type StateWithRank struct {
	StateSignature proofs.GlobulaStateSignature
	ExpectedRank   member.Rank
}

type SubVector struct {
	AnnouncementHash proofs.GlobulaAnnouncementHash
	StateWithRank
	DebugHash cryptkit.DigestHolder // TODO remove
}

func NewVector(bitset member.StateBitset, trusted SubVector, doubted SubVector) Vector {

	return Vector{bitset, trusted, doubted}
}

func NewSubVector(announcementHash proofs.GlobulaAnnouncementHash,
	stateVectorSignature proofs.GlobulaStateSignature, stateVectorHash cryptkit.DigestHolder,
	expectedRank member.Rank) SubVector {

	return SubVector{
		announcementHash,
		StateWithRank{stateVectorSignature, expectedRank},
		stateVectorHash,
	}
}

type CalcVector struct {
	Bitset member.StateBitset

	Trusted CalcSubVector
	Doubted CalcSubVector
}

type CalcSubVector struct {
	AnnouncementHash proofs.GlobulaAnnouncementHash
	CalcStateWithRank
}

func (v *CalcSubVector) Sign(signer cryptkit.DigestSigner) SubVector {
	return SubVector{v.AnnouncementHash,
		v.CalcStateWithRank.Sign(signer),
		v.StateHash,
	}
}

type CalcStateWithRank struct {
	StateHash    proofs.GlobulaStateHash
	ExpectedRank member.Rank
}

func (v *CalcStateWithRank) Sign(signer cryptkit.DigestSigner) StateWithRank {
	if v.StateHash == nil {
		return StateWithRank{}
	}
	return StateWithRank{v.StateHash.SignWith(signer).GetSignatureHolder(), v.ExpectedRank}
}
