// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"
	"math"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type ReceiverFunc func(nwapi.PayloadCompleteness, nwapi.Serializable) error

type Controller struct {
	pType     uniproto.ProtocolType
	factory   nwapi.DeserializationFactory
	receiveFn ReceiverFunc
	receiver  packetReceiver
	dedup     receiveDeduplicator
	smSender  retrySender
	starter   protoStarter

	pulseCycle atomickit.Uint64

	maxSmallSize      uint
	maxHeadSize       uint // <= maxSmallSize
	maxIgnoreHeadSize uint // <= maxSmallSize
}

func (p *Controller) RegisterWith(regFn uniproto.RegisterProtocolFunc) {
	switch {
	case p.pType == 0:
		panic(throw.IllegalState())
	case p.starter.ctl != nil:
		panic(throw.IllegalState())
	}
	p.starter.ctl = p
	desc := protoDescriptor
	desc.Receiver = &p.receiver
	regFn(p.pType, desc, &p.starter)
}

func (p *Controller) isActive() bool {
	return p.starter.isActive()
}

func (p *Controller) checkActive() error {
	if !p.isActive() {
		return throw.FailHere("inactive")
	}
	return nil
}

func (p *Controller) reportError(err error) {
	// TODO
	println(throw.ErrorWithStack(err))
}

func (p *Controller) nextPulseCycle(pn pulse.Number) bool {
	for {
		v := p.pulseCycle.Load()
		if pulse.Number(v) >= pn {
			return false
		}

		n := uint64(pn) | v&^math.MaxUint32 + 1<<32
		if p.pulseCycle.CompareAndSwap(v, n) {
			return true
		}
	}
}

func (p *Controller) getPulseCycle() (uint32, pulse.Number) {
	v := p.pulseCycle.Load()
	return uint32(v >> 32), pulse.Number(v)
}

func (p *Controller) receiveState(packet *uniproto.ReceivedPacket, payload *StatePacket) error {
	if err := p.checkActive(); err != nil {
		p.reportError(err)
		return err
	}

	peerID := packet.Header.SourceID

	if payload.BodyRq != 0 {
		dPeer, ok := packet.Peer.GetProtoInfo(p.pType).(*DeliveryPeer)
		if !ok {
			err := throw.RemoteBreach("peer is not a node")
			p.reportError(err)
			return err
		}

		shid := AsShipmentID(peerID, payload.BodyRq)
		if msg := p.smSender.get(shid); msg != nil && msg.markBodyRq() {
			// TODO "go" is insecure because sender can potentially flood us
			go p.smSender._sendBody(msg)
		} else {
			dPeer.addReject(payload.BodyRq)
		}
	}

	for _, id := range payload.AckList {
		shid := AsShipmentID(peerID, id)
		if msg := p.smSender.get(shid); msg != nil {
			msg.markAck()
		}
	}

	for _, id := range payload.BodyAckList {
		shid := AsShipmentID(peerID, id)
		if msg := p.smSender.get(shid); msg != nil {
			msg.markBodyAck()
		}
	}

	for _, id := range payload.RejectList {
		shid := AsShipmentID(peerID, id)
		if msg := p.smSender.get(shid); msg != nil {
			msg.markReject()
		}
	}

	return nil
}

func (p *Controller) receiveParcel(packet *uniproto.ReceivedPacket, payload *ParcelPacket) error {
	if err := p.checkActive(); err != nil {
		p.reportError(err)
		return err
	}

	dPeer, ok := packet.Peer.GetOrCreateProtoInfo(p.pType, p.createProtoPeer).(*DeliveryPeer)
	if !ok {
		err := throw.RemoteBreach("peer is not a node")
		p.reportError(err)
		return err
	}

	//if payload.RepeatedSend {
	//	// TODO collect stats
	//} else {
	//
	//}

	peerID := packet.Header.SourceID

	if payload.ReturnId != 0 {
		retID := AsShipmentID(peerID, payload.ReturnId)
		if msg := p.smSender.getHeads(retID); msg != nil {
			msg.markAck()
		} else if msg = p.smSender.getBodies(retID); msg != nil {
			msg.markBodyAck()
		}
	}

	if payload.ParcelType == nwapi.HeadOnlyPayload {
		dPeer.addAck(payload.ParcelId)
	} else {
		dPeer.addBodyAck(payload.ParcelId)
	}

	if !p.dedup.Add(DedupId(payload.ParcelId)) {
		return nil
	}

	// TODO in-proc unique node id
	err := p.receiveFn(payload.ParcelType, payload.Data)
	if err != nil {
		p.reportError(err)
	}
	return err
}

func (p *Controller) createProtoPeer(peer uniproto.Peer) io.Closer {
	id := peer.GetNodeID()
	if id.IsAbsent() {
		return nil
	}

	dp := &DeliveryPeer{
		ctl:    p,
		peerID: id,
		peer:   peer,
	}
	return dp
}

func (p *Controller) applySizePolicy(shipment *Shipment) uint {
	switch {
	case shipment.Head != nil:
		headSize := shipment.Head.ByteSize()
		if shipment.Body == nil {
			if headSize > p.maxHeadSize {
				shipment.Body = shipment.Head
				shipment.Head = nil
			}
			return headSize
		}

		bodySize := shipment.Body.ByteSize()
		if bodySize > p.maxSmallSize {
			shipment.Policies |= largeBody
		}

		switch {
		case bodySize <= headSize:
			panic(throw.IllegalValue())
		case headSize > p.maxHeadSize:
			shipment.Head = nil
			return bodySize
		case bodySize <= p.maxHeadSize:
			shipment.Head = shipment.Body
			shipment.Body = nil
			return bodySize
		case bodySize <= p.maxIgnoreHeadSize || shipment.Policies&ExpectedParcel != 0:
			shipment.Head = nil
			return bodySize
		default:
			return headSize
		}
	case shipment.Body != nil:
		bodySize := shipment.Body.ByteSize()
		switch {
		case bodySize <= p.maxHeadSize:
			shipment.Head = shipment.Body
			shipment.Body = nil
		case bodySize > p.maxSmallSize:
			shipment.Policies |= largeBody
		}

		return bodySize
	default:
		panic(throw.IllegalValue())
	}
}

func (p *Controller) Send(to nwapi.Address, returnId ShortShipmentID, shipment Shipment) {
	if err := p.checkActive(); err != nil {
		panic(err)
	}

	sendSize := p.applySizePolicy(&shipment)

	// This protocol is only allowed for peers added by consensus
	// It can't connect unknown peers.
	peer, err := p.starter.peers.ConnectedPeer(to)
	if err != nil {
		panic(err)
	}

	dPeer, ok := peer.GetOrCreateProtoInfo(p.pType, p.createProtoPeer).(*DeliveryPeer)
	if !ok {
		panic(throw.E("peer is not a node"))
	}

	cycle, pn := p.getPulseCycle()
	msg := &msgShipment{
		id:       dPeer.NextShipmentId(),
		peer:     dPeer,
		returnId: returnId,
		expires:  cycle + uint32(shipment.TTL),
		shipment: shipment,
	}

	switch {
	case shipment.PN == pn:
		//
	case shipment.PN > pn:
		panic(throw.IllegalState())
	case msg.expires <= cycle:
		// has expired
		return
	default:
		msg.expires--
	}

	if msg.shipment.Head != nil {
		if msg.shipment.Body == nil && msg.isFireAndForget() {
			go msg.sendHead(false)
			return
		}

		go p.smSender._sendHead(msg, sendSize)
		return
	}

	if msg.isFireAndForget() {
		go msg.sendBody(false)
		return
	}

	msg.markBodyRq()
	go p.smSender._sendBody(msg)
}

func (p *Controller) onStarted() {
	p.maxSmallSize = p.starter.peers.MaxSmallPayloadSize()

	if p.maxHeadSize > p.maxSmallSize || p.maxHeadSize == 0 {
		p.maxHeadSize = p.maxSmallSize
	}
	if p.maxIgnoreHeadSize > p.maxSmallSize || p.maxIgnoreHeadSize == 0 {
		p.maxIgnoreHeadSize = p.maxSmallSize
	}
}

func (p *Controller) onStopped() {}
