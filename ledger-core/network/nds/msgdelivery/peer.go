// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"math"
	"math/bits"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type DeliveryPeer struct {
	ctl      *Controller
	uid      nwapi.LocalUniqueID
	nextSSID atomickit.Uint32 // ShortShipmentID
	peerID   nwapi.ShortNodeID
	isDead   atomickit.OnceFlag

	dedup receiveDeduplicator

	mutex    sync.Mutex
	canFlush bool
	prepared StatePacket
	peer     uniproto.Peer
}

func (p *DeliveryPeer) NextShipmentID() ShipmentID {
	id := p.nextSSID.Add(1)
	for id == 0 {
		// receiveDeduplicator can't handle zero
		id = p.nextSSID.Add(1)
	}
	return AsShipmentID(uint32(p.peerID), ShortShipmentID(id))
}

func (p *DeliveryPeer) Close() error {
	p.isDead.DoSet(func() { go p._close() })
	return nil
}

func (p *DeliveryPeer) init() {
	p.uid = p.peer.GetLocalUID().AsLocalUID()
	p.ctl.stater.AddPeer(p)
}

func (p *DeliveryPeer) _close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.prepared = StatePacket{}
	p.canFlush = false
	// p.ctl = nil
	// p.peer = nil
}

func (p *DeliveryPeer) isValid() bool {
	return p != nil && !p.isDead.IsSet()
}

func (p *DeliveryPeer) _addToStatePacket(needed int, fn func(*StatePacket)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch n := p.prepared.remainingSpace(maxStatePacketDataSize); {
	case n > needed:
		if !p.canFlush {
			fn(&p.prepared)
			return
		}
		fallthrough
	case n == needed:
		fn(&p.prepared)
		packet := p.prepared
		p.canFlush = false
		p.prepared = StatePacket{}
		p.ctl.stater.AddState(p, packet)
	default:
		packet := p.prepared
		p.canFlush = false
		p.prepared = StatePacket{}
		fn(&p.prepared)
		p.ctl.stater.AddState(p, packet)
	}
}

func (p *DeliveryPeer) addReject(id ShortShipmentID) {
	p._addToStatePacket(2, func(packet *StatePacket) {
		packet.RejectList = append(packet.RejectList, id)
	})
}

func (p *DeliveryPeer) addBodyAck(id ShortShipmentID) {
	p._addToStatePacket(2, func(packet *StatePacket) {
		packet.BodyAckList = append(packet.BodyAckList, id)
	})
}

func (p *DeliveryPeer) addBodyRq(id ShortShipmentID) {
	p._addToStatePacket(2, func(packet *StatePacket) {
		packet.BodyRqList = append(packet.BodyRqList, id)
	})
}

func (p *DeliveryPeer) addAck(id ShortShipmentID) {
	p._addToStatePacket(1, func(packet *StatePacket) {
		packet.AckList = append(packet.AckList, id)
	})
}

func (p *DeliveryPeer) sendBodyRq(id ShortShipmentID) {
	if id == 0 {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	if p.prepared.isEmpty() {
		p.mutex.Unlock()
		return
	}

	packet := p.prepared
	p.prepared = StatePacket{}
	packet.BodyRqList = append(packet.BodyRqList, id)
	p.mutex.Unlock()

	p.sendState(packet)
}

func (p *DeliveryPeer) flushState() *StatePacket {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if !p.canFlush || p.prepared.isEmpty() {
		return nil
	}
	packet := p.prepared
	p.canFlush = false
	p.prepared = StatePacket{}
	return &packet
}

func (p *DeliveryPeer) sendState(packet StatePacket) {
	if !p.isValid() {
		return
	}

	template, dataSize, writeFn := packet.PreparePacket()
	_, template.PulseNumber = p.ctl.getPulseCycle()

	p._setPacketForSend(&template)
	if err := p.peer.SendPreparedPacket(p.ctl.stateOutType, &template.Packet, dataSize, writeFn, nil); err != nil {
		p.ctl.reportError(err)
	}
}

func (p *DeliveryPeer) sendParcel(msg *msgShipment, isBody, isRepeated bool) {
	packet := ParcelPacket{ParcelID: msg.id.ShortID(), ReturnID: msg.returnID, RepeatedSend: isRepeated}

	packet.ParcelType = nwapi.CompletePayload

	if !p.assignAndCheckTTL(msg, &packet) {
		msg.markExpired()
		return
	}

	switch {
	case isBody:
		packet.Data = msg.shipment.Body
		p._sendParcel(p.ctl.parcelOutType, packet, msg.canSendBody)
		return
	case msg.shipment.Body == nil:
		packet.Data = msg.shipment.Head
	case msg.shipment.Head == nil:
		packet.Data = msg.shipment.Body
	default:
		packet.ParcelType = nwapi.HeadOnlyPayload
		packet.Data = msg.shipment.Head
		packet.BodyScale = uint8(bits.Len(msg.shipment.Body.ByteSize()))
		packet.TTLCycles = msg.shipment.TTL
	}

	p._sendParcel(p.ctl.parcelOutType, packet, msg.canSendHead)
}

func (p *DeliveryPeer) sendLargeParcel(msg *msgShipment, isRepeated bool) {
	packet := ParcelPacket{ParcelID: msg.id.ShortID(), ReturnID: msg.returnID, RepeatedSend: isRepeated, ParcelType: nwapi.CompletePayload}
	packet.Data = msg.shipment.Body

	if !p.assignAndCheckTTL(msg, &packet) {
		msg.markExpired()
		return
	}

	p._sendParcel(uniproto.SessionfulLarge, packet, msg.canSendBody)
}

func (p *DeliveryPeer) assignAndCheckTTL(msg *msgShipment, packet *ParcelPacket) bool {
	cycle, pn := p.ctl.getPulseCycle()

	if msg.expires < cycle {
		return false
	}

	if msg.shipment.Policies&ExactPulse == 0 {
		msg.shipment.Policies |= ExactPulse
		msg.shipment.PN = pn
		if cycle = msg.expires - cycle; cycle > math.MaxUint8 {
			cycle = math.MaxUint8
		}
		msg.shipment.TTL = uint8(cycle)
	}

	packet.PulseNumber = msg.shipment.PN
	return true
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
	template.Header.SourceID = uint32(p.ctl.getLocalID())
	template.Header.ReceiverID = uint32(p.peerID)
	template.Header.TargetID = uint32(p.peerID)
	template.Header.SetRelayRestricted(true)
}

func (p *DeliveryPeer) markFlush() (isValid, hasUpdates bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isDead.IsSet() {
		return false, false
	}
	p.canFlush = !p.prepared.isEmpty()
	return true, p.canFlush
}
