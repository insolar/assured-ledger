package proofs

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type OriginalPulsarPacket interface {
	longbits.FixedReader
	pulse.DataHolder
	OriginalPulsarPacket()
}
