// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"
	"math"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewController(pt uniproto.ProtocolType, regFn uniproto.RegisterControllerFunc,
	factory nwapi.DeserializationFactory, timeCycle time.Duration, receiveFn ReceiverFunc,
) Service {
	c := &controller{pType: pt, factory: factory, timeCycle: timeCycle, receiveFn: receiveFn}
	c.registerWith(regFn)
	return c
}

type controller struct {
	pType     uniproto.ProtocolType
	factory   nwapi.DeserializationFactory
	receiveFn ReceiverFunc
	timeCycle time.Duration

	receiver packetReceiver
	dedup    receiveDeduplicator
	sender   retrySender
	starter  protoStarter

	pulseCycle atomickit.Uint64

	stopSignal        synckit.ClosableSignalChannel
	maxSmallSize      uint
	maxHeadSize       uint // <= maxSmallSize
	maxIgnoreHeadSize uint // <= maxSmallSize
	localID           nwapi.ShortNodeID
}

func (p *controller) ShipTo(to DeliveryAddress, shipment Shipment) error {
	panic("implement me")
}

func (p *controller) ShipReturn(to ReturnAddress, shipment Shipment) error {
	panic("implement me")
}

func (p *controller) PullBody(from ReturnAddress, receiveFn ReceiverFunc) error {
	panic("implement me")
}

func (p *controller) registerWith(regFn uniproto.RegisterControllerFunc) {
	switch {
	case p.pType == 0:
		panic(throw.IllegalState())
	case p.starter.ctl != nil:
		panic(throw.IllegalState())
	}
	p.starter.ctl = p
	regFn(p.pType, protoDescriptor, &p.starter)
}

func (p *controller) isActive() bool {
	return p.starter.isActive()
}

func (p *controller) checkActive() error {
	if !p.isActive() {
		return throw.FailHere("inactive")
	}
	return nil
}

func (p *controller) reportError(err error) {
	// TODO
	println(throw.ErrorWithStack(err))
}

func (p *controller) nextPulseCycle(pn pulse.Number) bool {
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

func (p *controller) getPulseCycle() (uint32, pulse.Number) {
	v := p.pulseCycle.Load()
	return uint32(v >> 32), pulse.Number(v)
}

func (p *controller) receiveState(packet *uniproto.ReceivedPacket, payload *StatePacket) error {
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
		if msg := p.sender.get(shid); msg != nil && msg.markBodyRq() {
			// TODO "go" is insecure because sender can potentially flood us
			go p.sender._sendBody(msg)
		} else {
			dPeer.addReject(payload.BodyRq)
		}
	}

	for _, id := range payload.AckList {
		shid := AsShipmentID(peerID, id)
		if msg := p.sender.get(shid); msg != nil {
			msg.markAck()
		}
	}

	for _, id := range payload.BodyAckList {
		shid := AsShipmentID(peerID, id)
		if msg := p.sender.get(shid); msg != nil {
			msg.markBodyAck()
		}
	}

	for _, id := range payload.RejectList {
		shid := AsShipmentID(peerID, id)
		if msg := p.sender.get(shid); msg != nil {
			msg.markReject()
		}
	}

	return nil
}

func (p *controller) receiveParcel(packet *uniproto.ReceivedPacket, payload *ParcelPacket) error {
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
		if msg := p.sender.getHeads(retID); msg != nil {
			msg.markAck()
		} else if msg = p.sender.getBodies(retID); msg != nil {
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

	err := p.receiveFn(
		ReturnAddress{
			returnTo: packet.Peer.GetLocalUID(),
			returnId: payload.ParcelId,
		},
		payload.ParcelType,
		payload.Data)

	if err != nil {
		p.reportError(err)
	}
	return err
}

func (p *controller) createProtoPeer(peer uniproto.Peer) io.Closer {
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

func (p *controller) applySizePolicy(shipment *Shipment) uint {
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

func (p *controller) send(to nwapi.Address, returnId ShortShipmentID, shipment Shipment) {
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

		go p.sender._sendHead(msg, sendSize)
		return
	}

	if msg.isFireAndForget() {
		go msg.sendBody(false)
		return
	}

	msg.markBodyRq()
	go p.sender._sendBody(msg)
}

func (p *controller) onStarted() {
	p.maxSmallSize = p.starter.peers.MaxSmallPayloadSize()
	p.localID = p.starter.peers.LocalPeer().GetNodeID()

	if p.maxHeadSize > p.maxSmallSize || p.maxHeadSize == 0 {
		p.maxHeadSize = p.maxSmallSize
	}
	if p.maxIgnoreHeadSize > p.maxSmallSize || p.maxIgnoreHeadSize == 0 {
		p.maxIgnoreHeadSize = p.maxSmallSize
	}

	p.stopSignal = make(synckit.ClosableSignalChannel)
	go p.runWorker()
}

func (p *controller) onStopped() {
	close(p.stopSignal)
}

func (p *controller) runWorker() {
	ticker := time.Tick(p.timeCycle)
	for {
		select {
		case <-p.stopSignal:
			return
		case <-ticker:
			//
		}
		p.sender.NextTimeCycle()
	}
}
