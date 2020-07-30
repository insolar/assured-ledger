// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

var _ nodeinfo.NetworkNode = profileNode{}
type profileNode struct {
	n profiles.ActiveNode
}

func (v profileNode) GetStatic() profiles.StaticProfile {
	return v.n.GetStatic()
}

func (v profileNode) GetNodeID() node.ShortNodeID {
	return v.n.GetNodeID()
}

func (v profileNode) PublicKey() crypto.PublicKey {
	nip := v.n.GetStatic()
	store := nip.GetPublicKeyStore()
	return store.(*ECDSAPublicKeyStore).publicKey
}

func (v profileNode) GetDeclaredPower() member.Power {
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

func (v profileNode) GetSignature() cryptkit.SignedDigestHolder {
	return v.n.GetStatic().GetBriefIntroSignedDigest()
}

