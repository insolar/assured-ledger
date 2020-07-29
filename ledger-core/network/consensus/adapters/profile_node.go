// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

var _ nodeinfo.NetworkNode = profileNode{}
type profileNode struct {
	n profiles.ActiveNode
}

func (v profileNode) GetReference() reference.Global {
	return v.n.GetStatic().GetExtension().GetReference()
}

func (v profileNode) GetNodeID() node.ShortNodeID {
	return v.n.GetNodeID()
}

func (v profileNode) GetPrimaryRole() member.PrimaryRole {
	return v.n.GetStatic().GetPrimaryRole()
}

func (v profileNode) PublicKey() crypto.PublicKey {
	nip := v.n.GetStatic()
	store := nip.GetPublicKeyStore()
	return store.(*ECDSAPublicKeyStore).publicKey
}

func (v profileNode) Address() string {
	return v.n.GetStatic().GetDefaultEndpoint().GetNameAddress().String()
}

func (v profileNode) GetPower() member.Power {
	if v.n.GetOpMode().IsPowerless() {
		return 0
	}
	return v.n.GetDeclaredPower()
}

func (v profileNode) IsJoiner() bool {
	return v.n.IsJoiner()
}

func (v profileNode) IsPowered() bool {
	return v.n.IsPowered()
}

func (v profileNode) GetSignature() ([]byte, cryptography.Signature) {
	sd := v.n.GetStatic().GetBriefIntroSignedDigest()

	// TODO use cryptkit.SignedDigestHolder
	return longbits.AsBytes(sd.GetDigestHolder()),
		cryptography.SignatureFromBytes(longbits.AsBytes(sd.GetSignatureHolder()))
}

