// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
)

type DeliveryPeer struct {
	nextSSID atomickit.Uint32 // ShortShipmentID
	peerID   nwapi.ShortNodeID
	outbound *peerProxy
}

func (p *DeliveryPeer) NextShipmentId() ShipmentID {
	id := p.nextSSID.Add(1)
	for id == 0 {
		id = p.nextSSID.Add(1)
	}
	return AsShipmentID(uint32(p.peerID), ShortShipmentID(id))
}

func (p *DeliveryPeer) Close() error {
	p.outbound.invalidate()
	return nil
}

func (p *DeliveryPeer) sendReject(id ShortShipmentID) {

}

func (p *DeliveryPeer) sendBodyAck(id ShortShipmentID) {

}

func (p *DeliveryPeer) sendAck(id ShortShipmentID) {

}

/**********************************/

type peerProxy struct {
	peer   *DeliveryPeer // keep alive
	isDead atomickit.OnceFlag
}

func (p *peerProxy) isValid() bool {
	return !p.isDead.IsSet()
}

func (p *peerProxy) invalidate() bool {
	return p.isDead.DoSet(func() {
		p.peer = nil
	})
}
