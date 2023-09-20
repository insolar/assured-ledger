package errors

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/warning"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/phases"
)

func LimitExceeded(packetType phases.PacketType, sourceID node.ShortNodeID, sourceEndpoint endpoints.Inbound) error {
	err := fmt.Errorf(
		"packet type (%v) limit exceeded: from=%v(%v)",
		packetType,
		sourceID,
		sourceEndpoint,
	)

	if packetType == phases.PacketPhase3 {
		return warning.New(err)
	}

	return err
}

func UnknownPacketType(packetType phases.PacketType) error {
	err := fmt.Errorf("packet type (%v) is unknown", packetType)

	if packetType == phases.PacketPulsarPulse {
		return warning.New(err)
	}

	return err
}
