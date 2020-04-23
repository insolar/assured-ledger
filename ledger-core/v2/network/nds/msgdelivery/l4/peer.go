// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"math"
	"math/bits"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
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

func (p *DeliveryPeer) NextShipmentId() ShipmentID {
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
	//p.ctl = nil
	//p.peer = nil
}

func (p *DeliveryPeer) isValid() bool {
	return p != nil && !p.isDead.IsSet()
}

func (p *DeliveryPeer) addToStatePacket(needed int, fn func(*StatePacket)) {
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
	p.addToStatePacket(2, func(packet *StatePacket) {
		packet.RejectList = append(packet.RejectList, id)
	})
}

func (p *DeliveryPeer) addBodyAck(id ShortShipmentID) {
	p.addToStatePacket(2, func(packet *StatePacket) {
		packet.BodyAckList = append(packet.BodyAckList, id)
	})
}

func (p *DeliveryPeer) addBodyRq(id ShortShipmentID) {
	p.addToStatePacket(2, func(packet *StatePacket) {
		packet.BodyRq = append(packet.BodyRq, id)
	})
}

func (p *DeliveryPeer) addAck(id ShortShipmentID) {
	p.addToStatePacket(1, func(packet *StatePacket) {
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
	packet.BodyRq = append(packet.BodyRq, id)
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
	if err := p.peer.SendPreparedPacket(uniproto.SmallAny, &template.Packet, dataSize, writeFn, nil); err != nil {
		p.ctl.reportError(err)
	}
}

func (p *DeliveryPeer) sendParcel(msg *msgShipment, isBody, isRepeated bool) {
	packet := ParcelPacket{ParcelID: msg.id.ShortID(), ReturnID: msg.returnID, RepeatedSend: isRepeated}

	packet.ParcelType = nwapi.CompletePayload
	cycle, pn := p.ctl.getPulseCycle()

	if msg.expires < cycle {
		msg.markExpired()
		return
	}

	if msg.shipment.Policies&ExactPulse == 0 {
		packet.PulseNumber = pn
		if cycle = msg.expires - cycle; cycle > math.MaxUint8 {
			cycle = math.MaxUint8
		}
	} else {
		packet.PulseNumber = msg.shipment.PN
		cycle = uint32(msg.shipment.TTL)
	}

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
		packet.TTLCycles = uint8(cycle)
	}
	p._sendParcel(uniproto.Any, packet, msg.canSendHead)
}

func (p *DeliveryPeer) sendLargeParcel(msg *msgShipment, isRepeated bool) {
	packet := ParcelPacket{ParcelID: msg.id.ShortID(), ReturnID: msg.returnID, RepeatedSend: isRepeated, ParcelType: nwapi.CompletePayload}
	packet.Data = msg.shipment.Body

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
