// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package errors

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/warning"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/phases"
)

func LimitExceeded(packetType phases.PacketType, sourceID insolar.ShortNodeID, sourceEndpoint endpoints.Inbound) error {
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
