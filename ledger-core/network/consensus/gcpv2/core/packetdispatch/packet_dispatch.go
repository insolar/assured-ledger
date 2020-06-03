// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package packetdispatch

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/phases"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/core/coreapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/core/population"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
)

type MemberPacketReceiver interface {
	GetNodeID() node.ShortNodeID
	CanReceivePacket(pt phases.PacketType) bool
	VerifyPacketAuthenticity(packetSignature cryptkit.SignedDigest, from endpoints.Inbound, strictFrom bool) error
	SetPacketReceived(pt phases.PacketType) bool
	DispatchMemberPacket(ctx context.Context, packet transport.PacketParser, from endpoints.Inbound, flags coreapi.PacketVerifyFlags,
		pd population.PacketDispatcher) error
}
