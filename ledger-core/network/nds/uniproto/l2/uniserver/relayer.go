// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
)

// Relayer is a helper to handle packets which were requested and authorized for relay to another peer
type Relayer interface {
	// RelaySessionlessPacket should relay the given sessionless packet.
	RelaySessionlessPacket(*PeerManager, *uniproto.Packet, []byte) error
	// RelaySmallPacket should relay the given small packet.
	RelaySmallPacket(*PeerManager, *uniproto.Packet, []byte) error
	// RelayLargePacket should relay the given large packet.
	RelayLargePacket(*PeerManager, *uniproto.Packet, []byte, io.LimitedReader) error
}
