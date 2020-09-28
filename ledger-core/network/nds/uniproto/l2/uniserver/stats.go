// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import "github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"

// TODO
type Stats struct {
	ReceivedSessionless TransportStats

	ReceivedSessionfulSmallPackets TransportStats
	ReceivedSessionfulLargePackets TransportStats
}

type TransportStats struct {
	Count      atomickit.Uint64
	Size       atomickit.Uint64
	ConnectIn  atomickit.Uint32
	ConnectOut atomickit.Uint32
	ForcedOut  atomickit.Uint32
}
