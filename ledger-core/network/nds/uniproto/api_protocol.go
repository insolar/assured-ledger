// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type Receiver interface {
	// ReceiveSmallPacket is called on small (non-excessive length) packets, (b) is exactly whole packet
	ReceiveSmallPacket(rp *ReceivedPacket, b []byte)
	// ReceiveLargePacket is called on large (excessive length) packets, (preRead) is a pre-read portion, that can be larger than a header, and (r) is configured for the remaining length.
	ReceiveLargePacket(rp *ReceivedPacket, preRead []byte, r io.LimitedReader) error
}

type Supporter interface {
	VerifyHeader(*Header, pulse.Number, cryptkit.DataSignatureVerifier) (cryptkit.DataSignatureVerifier, error)
	ToHostID(id uint32) nwapi.HostID
	GetLocalNodeID(id uint32) uint32 // fix naming either peer or host
}
