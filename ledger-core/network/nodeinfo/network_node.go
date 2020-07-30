// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeinfo

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type NetworkNode = profiles.ActiveNode

func NodeAddr(n NetworkNode) string {
	return n.GetStatic().GetDefaultEndpoint().GetIPAddress().String()
}

func NodeRef(n NetworkNode) reference.Global {
	return n.GetStatic().GetExtension().GetReference()
}

func NodeRole(n NetworkNode) member.PrimaryRole {
	return n.GetStatic().GetPrimaryRole()
}

func NodeSignedDigest(n NetworkNode) cryptkit.SignedDigestHolder {
	return n.GetStatic().GetBriefIntroSignedDigest()
}
