// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"crypto"
	"hash/crc32"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// GenerateUintShortID generate short ID for node without checking collisions
func GenerateUintShortID(ref reference.Global) uint32 {
	return crc32.ChecksumIEEE(ref.AsBytes())
}

func NewActiveNode(id reference.Global, role member.PrimaryRole, publicKey crypto.PublicKey, address string) nodeinfo.NetworkNode {
	return newMutableNode(id, role, publicKey, nodeinfo.Ready, address)
}

