// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"math/bits"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type DeliveryPeer struct {
	nextSSID atomickit.Uint32 // ShortShipmentID
	ctl      *controller
	peerID   nwapi.ShortNodeID
	isDead   atomickit.OnceFlag

	mutex    sync.Mutex
	prepared StatePacket
	peer     uniproto.Peer
}

func (p *DeliveryPeer) NextShipmentId() ShipmentID {
	id := p.nextSSID.Add(1)
	for id == 0 {
		id = p.nextSSID.Add(1)
	}
	return AsShipmentID(uint32(p.peerID), ShortShipmentID(id))
}

func (p *DeliveryPeer) Close() error {
	p.isDead.DoSet(func() { go p._close() })
	return nil
}

func (p *DeliveryPeer) _close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.prepared = StatePacket{}
	//p.peer.SetProtoInfo(p.ctl.pType, nil)
	p.ctl = nil
	p.peer = nil
}

func (p *DeliveryPeer) isValid() bool {
	return !p.isDead.IsSet()
}

func (p *DeliveryPeer) addToStatePacket(needed int, fn func(*StatePacket)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch n := p.prepared.remainingSpace(maxStatePacketDataSize); {
	case n > needed:
		fn(&p.prepared)
	case n == needed:
		fn(&p.prepared)
		packet := p.prepared
		p.prepared = StatePacket{}
		// TODO "go" is insecure because sender can potentially flood us
		go p.sendState(packet)
	default:
		packet := p.prepared
		p.prepared = StatePacket{}
		fn(&p.prepared)
		// TODO "go" is insecure because sender can potentially flood us
		go p.sendState(packet)
	}
}

func (p *DeliveryPeer) addReject(id ShortShipmentID) {
	p.addToStatePacket(2, func(packet *StatePacket) {
		packet.RejectList = append(packet.RejectList, id)
	})
}

func (p *DeliveryPeer) addBodyAck(id ShortShipmentID) {
	p.addToStatePacket(2, func(packet *StatePacket) {
		packet.BodyAckList = append(packet.BodyAckList, id)
	})
}

func (p *DeliveryPeer) addAck(id ShortShipmentID) {
	p.addToStatePacket(1, func(packet *StatePacket) {
		packet.AckList = append(packet.AckList, id)
	})
}

func (p *DeliveryPeer) sendBodyRq(id ShortShipmentID) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if id == 0 && p.prepared.isEmpty() {
		return
	}

	packet := p.prepared
	p.prepared = StatePacket{}
	packet.BodyRq = append(packet.BodyRq, id)
	p.sendState(packet)
}

func (p *DeliveryPeer) sendState(packet StatePacket) {
	if !p.isValid() {
		return
	}
	template, dataSize, writeFn := packet.PreparePacket()
	p._setPacketForSend(&template)
	if err := p.peer.SendPreparedPacket(uniproto.SmallAny, &template.Packet, dataSize, writeFn, nil); err != nil {
		p.ctl.reportError(err)
	}
}

func (p *DeliveryPeer) sendParcel(msg *msgShipment, isBody, isRepeated bool) {
	packet := ParcelPacket{ParcelID: msg.id.ShortID(), ReturnID: msg.returnID, RepeatedSend: isRepeated}

	packet.ParcelType = nwapi.CompletePayload

	switch {
	case isBody:
		packet.Data = msg.shipment.Body
		p._sendParcel(uniproto.Any, packet, msg.canSendBody)
		return
	case msg.shipment.Body == nil:
		packet.Data = msg.shipment.Head
	case msg.shipment.Head == nil:
		packet.Data = msg.shipment.Body
	default:
		packet.ParcelType = nwapi.HeadOnlyPayload
		packet.Data = msg.shipment.Head
		packet.BodyScale = uint8(bits.Len(msg.shipment.Body.ByteSize()))
	}
	p._sendParcel(uniproto.Any, packet, msg.canSendHead)
}

func (p *DeliveryPeer) sendLargeParcel(msg *msgShipment, isRepeated bool) {
	packet := ParcelPacket{ParcelID: msg.id.ShortID(), ReturnID: msg.returnID, RepeatedSend: isRepeated, ParcelType: nwapi.CompletePayload}
	p._sendParcel(uniproto.SessionfulLarge, packet, msg.canSendBody)
}

func (p *DeliveryPeer) _sendParcel(tp uniproto.OutType, parcel ParcelPacket, checkFn func() bool) {
	if !p.isValid() {
		return
	}
	if parcel.Data == nil {
		panic(throw.IllegalValue())
	}

	template, dataSize, writeFn := parcel.PreparePacket()
	p._setPacketForSend(&template)
	if err := p.peer.SendPreparedPacket(tp, &template.Packet, dataSize, writeFn, checkFn); err != nil {
		p.ctl.reportError(err)
	}
}

func (p *DeliveryPeer) _setPacketForSend(template *uniproto.PacketTemplate) {
	template.Header.SourceID = uint32(p.ctl.localID)
	template.Header.ReceiverID = uint32(p.peerID)
	template.Header.TargetID = uint32(p.peerID)
}
