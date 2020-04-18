// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"
	"math"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l4/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const minHeadBatchWeight = 1 << 20

func NewController(pt uniproto.ProtocolType, factory nwapi.DeserializationFactory,
	receiveFn ReceiverFunc, resolverFn ResolverFunc,
) Service {
	c := &controller{pType: pt, factory: factory, timeCycle: 10 * time.Millisecond, receiveFn: receiveFn, resolverFn: resolverFn}
	c.sender.stages.InitStages(minHeadBatchWeight, [...]int{10, 50, 100})
	return c
}

type controller struct {
	pType      uniproto.ProtocolType
	factory    nwapi.DeserializationFactory
	receiveFn  ReceiverFunc
	resolverFn ResolverFunc
	timeCycle  time.Duration

	receiver packetReceiver
	sender   retrySender
	bodyRq   bodyRequester
	starter  protoStarter

	pulseCycle atomickit.Uint64

	stopSignal        synckit.ClosableSignalChannel
	maxSmallSize      uint
	maxHeadSize       uint // <= maxSmallSize
	maxIgnoreHeadSize uint // <= maxSmallSize
	localID           nwapi.ShortNodeID
}

// for initialization only
func (p *controller) SetTimings(timeCycle time.Duration, retryStages [retries.RetryStages]time.Duration) {
	switch {
	case timeCycle < time.Microsecond:
		panic(throw.IllegalValue())
	case !p.isConfigurable():
		panic(throw.IllegalState())
	}
	p.timeCycle = timeCycle

	var periods [retries.RetryStages]int
	last := 1
	for i, ts := range retryStages {
		n := int(ts / timeCycle)
		if n < last {
			panic(throw.IllegalValue())
		}
		last = n
		periods[i] = n
	}
	p.sender.stages.InitStages(minHeadBatchWeight, periods)
}

// for initialization only
func (p *controller) RegisterWith(regFn uniproto.RegisterControllerFunc) {
	switch {
	case p.pType == 0:
		panic(throw.IllegalState())
	case p.starter.ctl != nil:
		panic(throw.IllegalState())
	}
	p.starter.ctl = p
	regFn(p.pType, protoDescriptor, &p.starter, &p.receiver)
}

func (p *controller) ShipTo(to DeliveryAddress, shipment Shipment) error {
	if to.addrType == DirectAddress {
		switch {
		case to.nodeSelector == 0:
			return throw.IllegalValue()
		case to.dataSelector != 0:
			return throw.IllegalValue()
		}
		return p.send(nwapi.NewHostId(nwapi.HostID(to.nodeSelector)), 0, shipment)
	}

	addr, err := p.resolverFn(to.addrType, to.nodeSelector, to.dataSelector)
	if err != nil {
		return err
	}
	return p.send(addr, 0, shipment)
}

func (p *controller) ShipReturn(to ReturnAddress, shipment Shipment) error {
	if !to.IsValid() {
		return throw.IllegalValue()
	}
	return p.send(to.returnTo, to.returnID, shipment)
}

func (p *controller) PullBody(from ReturnAddress, rq ShipmentRequest) error {
	if !from.IsValid() {
		return throw.IllegalValue()
	}
	return p.sendBodyRq(from.returnTo, from.returnID, rq)
}

func (p *controller) isConfigurable() bool {
	return !p.starter.wasStarted()
}

func (p *controller) checkActive() error {
	if !p.starter.isActive() {
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

	if n := len(payload.BodyRq); n > 0 {
		dPeer, ok := packet.Peer.GetProtoInfo(p.pType).(*DeliveryPeer)
		if !ok {
			err := throw.RemoteBreach("peer is not a node")
			p.reportError(err)
			return err
		}

		for _, id := range payload.BodyRq {
			switch msg := p.sender.get(AsShipmentID(peerID, id)); {
			case msg == nil:
				dPeer.addReject(id)
			case msg.markBodyRq():
				p.sender.sendBodyNoWait(msg)
			}
		}
	}

	for _, id := range payload.BodyAckList {
		if msg := p.sender.get(AsShipmentID(peerID, id)); msg != nil {
			msg.markBodyAck()
		}
	}

	for _, id := range payload.AckList {
		if msg := p.sender.get(AsShipmentID(peerID, id)); msg != nil {
			msg.markAck()
		}
	}

	for _, id := range payload.RejectList {
		if msg := p.sender.get(AsShipmentID(peerID, id)); msg != nil {
			msg.markReject()
		}
	}

	return nil
}

func (p *controller) getDeliveryPeer(peer uniproto.Peer) (*DeliveryPeer, error) {
	dPeer, ok := peer.GetOrCreateProtoInfo(p.pType, p.createProtoPeer).(*DeliveryPeer)
	if ok {
		return dPeer, nil
	}
	err := throw.RemoteBreach("peer is not a node")
	p.reportError(err)
	return nil, err
}

func (p *controller) receiveParcel(packet *uniproto.ReceivedPacket, payload *ParcelPacket) error {
	if err := p.checkActive(); err != nil {
		p.reportError(err)
		return err
	}

	dPeer, err := p.getDeliveryPeer(packet.Peer)
	if err != nil {
		return err
	}

	//if payload.RepeatedSend {
	//	// TODO collect stats
	//} else {
	//
	//}

	peerID := packet.Header.SourceID

	if payload.ReturnID != 0 {
		retID := AsShipmentID(peerID, payload.ReturnID)
		if msg := p.sender.get(retID); msg != nil {
			// any reply is considered as a regular Ack
			// to keep Body request-able
			msg.markAck()
		}
	}

	receiveFn := p.receiveFn

	if payload.ParcelType == nwapi.HeadOnlyPayload {
		dPeer.addAck(payload.ParcelID)
		if !dPeer.dedup.Add(DedupID(payload.ParcelID)) {
			return nil
		}
	} else {
		dPeer.addBodyAck(payload.ParcelID)
		switch fn, ok := p.bodyRq.Remove(dPeer, payload.ParcelID); {
		case !ok:
			return nil
		case fn != nil:
			receiveFn = fn
		}
	}

	err = receiveFn(
		ReturnAddress{
			returnTo: packet.Peer.GetLocalUID(),
			returnID: payload.ParcelID,
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

func (p *controller) applySizePolicy(shipment *Shipment) (uint, error) {
	switch {
	case shipment.Head != nil:
		headSize := shipment.Head.ByteSize()
		if shipment.Body == nil {
			if headSize > p.maxHeadSize {
				shipment.Body = shipment.Head
				shipment.Head = nil
			}
			return headSize, nil
		}

		bodySize := shipment.Body.ByteSize()
		if bodySize > p.maxSmallSize {
			shipment.Policies |= largeBody
		}

		switch {
		case bodySize <= headSize:
			return 0, throw.IllegalValue()
		case headSize > p.maxHeadSize:
			shipment.Head = nil
		case bodySize <= p.maxHeadSize:
			shipment.Head = shipment.Body
			shipment.Body = nil
		case bodySize <= p.maxIgnoreHeadSize || shipment.Policies&ExpectedParcel != 0:
			shipment.Head = nil
		default:
			return headSize, nil
		}
		return bodySize, nil

	case shipment.Body != nil:
		bodySize := shipment.Body.ByteSize()
		switch {
		case bodySize <= p.maxHeadSize:
			shipment.Head = shipment.Body
			shipment.Body = nil
		case bodySize > p.maxSmallSize:
			shipment.Policies |= largeBody
		}
		return bodySize, nil

	default:
		return 0, throw.IllegalValue()
	}
}

func (p *controller) peer(to nwapi.Address) (*DeliveryPeer, error) {
	if err := p.checkActive(); err != nil {
		return nil, err
	}

	// This protocol is only allowed for peers added by consensus
	// It can't connect unknown peers.
	peer, err := p.starter.peers.ConnectedPeer(to)
	if err != nil {
		return nil, err
	}

	return p.getDeliveryPeer(peer)
}

func (p *controller) send(to nwapi.Address, returnId ShortShipmentID, shipment Shipment) error {
	dPeer, err := p.peer(to)
	if err != nil {
		return err
	}

	sendSize, err := p.applySizePolicy(&shipment)
	if err != nil {
		return err
	}

	cycle, pn := p.getPulseCycle()
	msg := &msgShipment{
		id:       dPeer.NextShipmentId(),
		peer:     dPeer,
		returnID: returnId,
		expires:  cycle + uint32(shipment.TTL),
		shipment: shipment,
	}

	switch {
	case shipment.PN == pn:
		//
	case shipment.PN > pn:
		return throw.IllegalState()
	case msg.expires <= cycle:
		// has expired
		return nil
	default:
		msg.expires--
	}

	switch {
	case msg.shipment.Head == nil:
		msg.markBodyRq()
		p.sender.sendBodyNoWait(msg)
	case msg.shipment.Body == nil && msg.isFireAndForget():
		p.sender.sendHeadNoRetryNoWait(msg)
	default:
		p.sender.sendHeadNoWait(msg, sendSize)
	}
	return nil
}

func (p *controller) sendBodyRq(to nwapi.Address, id ShortShipmentID, rq ShipmentRequest) error {
	dPeer, err := p.peer(to)
	if err != nil {
		return err
	}

	return p.bodyRq.Add(rqShipment{
		id:      AsShipmentID(uint32(dPeer.peerID), id),
		peer:    dPeer,
		request: rq,
	})
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
