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
