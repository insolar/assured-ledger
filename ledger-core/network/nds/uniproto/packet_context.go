package uniproto

import "github.com/insolar/assured-ledger/ledger-core/network/nwapi"

var _ nwapi.DeserializationContext = packetContext{}

type packetContext struct {
	pf nwapi.DeserializationFactory
}

func (v packetContext) GetPayloadFactory() nwapi.DeserializationFactory {
	return v.pf
}
