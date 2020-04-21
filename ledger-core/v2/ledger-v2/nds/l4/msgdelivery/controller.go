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
	c.bodyRq.stages.InitStages(maxStatePacketEntries, [...]int{5, 10, 50})
	return c
}

type controller struct {
	pType      uniproto.ProtocolType
	factory    nwapi.DeserializationFactory
	receiveFn  ReceiverFunc
	resolverFn ResolverFunc
	timeCycle  time.Duration

	starter  protoStarter
	receiver packetReceiver
	sender   msgSender
	bodyRq   rqSender

	pulseCycle atomickit.Uint64

	stopSignal        synckit.ClosableSignalChannel
	maxSmallSize      uint
	maxHeadSize       uint // <= maxSmallSize
	maxIgnoreHeadSize uint // <= maxSmallSize
	localID           nwapi.ShortNodeID
}

// for initialization only
func (p *controller) SetTimings(timeCycle time.Duration, sendStages, rqStages [retries.RetryStages]time.Duration) {
	switch {
	case timeCycle < time.Microsecond:
		panic(throw.IllegalValue())
	case !p.isConfigurable():
		panic(throw.IllegalState())
	}
	p.timeCycle = timeCycle

	p.sender.stages.InitStages(minHeadBatchWeight, calcPeriods(timeCycle, sendStages))
	p.bodyRq.stages.InitStages(maxStatePacketEntries, calcPeriods(timeCycle, rqStages))
}

func calcPeriods(timeCycle time.Duration, stages [retries.RetryStages]time.Duration) (periods [retries.RetryStages]int) {
	last := 1
	for i, ts := range stages {
		n := int(ts / timeCycle)
		if n < last {
			panic(throw.IllegalValue())
		}
		last = n
		periods[i] = n
	}
	return
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

// TODO isolation facade
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
	if cycle, _ := p.getPulseCycle(); cycle > from.expires {
		if rq.ReceiveFn != nil {
			// avoid possible deadlock of the caller
			go func() {
				if err := rq.ReceiveFn(from, false, nil); err != nil {
					p.reportError(err)
				}
			}()
		}
		return nil
	}

	return p.sendBodyRq(from, rq)
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
		return err
	}

	peerID := packet.Header.SourceID

	if n := len(payload.BodyRq); n > 0 {
		dPeer, ok := packet.Peer.GetProtoInfo(p.pType).(*DeliveryPeer)
		if !ok {
			return throw.RemoteBreach("peer is not a node")
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
		sid := AsShipmentID(peerID, id)
		if msg := p.sender.get(sid); msg != nil {
			msg.markReject()
		}
		if rq, ok := p.bodyRq.RemoveByID(sid); ok {
			rq.requestRejected()
		}

	}

	return nil
}

func (p *controller) getDeliveryPeer(peer uniproto.Peer) (*DeliveryPeer, error) {
	dPeer, ok := peer.GetOrCreateProtoInfo(p.pType, p.createProtoPeer).(*DeliveryPeer)
	if ok {
		return dPeer, nil
	}
	return nil, throw.RemoteBreach("peer is not a node")
}

func (p *controller) receiveParcel(packet *uniproto.ReceivedPacket, payload *ParcelPacket) error {
	if err := p.checkActive(); err != nil {
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

	retAddr := ReturnAddress{
		returnTo: packet.Peer.GetLocalUID(),
		returnID: payload.ParcelID,
	}

	receiveFn := p.receiveFn
	duplicate := !dPeer.dedup.Add(DedupID(payload.ParcelID))

	if payload.ParcelType == nwapi.HeadOnlyPayload {
		if duplicate {
			dPeer.addReject(payload.ParcelID)
			return nil
		}
		ok, expiry, err := p.adjustedExpiry(payload.PulseNumber, payload.TTLCycles, true)
		if !ok {
			dPeer.addReject(payload.ParcelID)
			return err
		}
		retAddr.expires = expiry

		dPeer.addAck(payload.ParcelID)
	} else {
		// ignores peer-based deduplication as served per-request

		rq, ok := p.bodyRq.RemoveRq(dPeer, payload.ParcelID)
		if !ok {
			dPeer.addReject(payload.ParcelID)
			return nil
		}

		retAddr.expires = rq.expires
		switch {
		case rq.isExpired():
			dPeer.addReject(payload.ParcelID)
			rq.requestRejected()
			return nil
		case rq.request.ReceiveFn != nil:
			receiveFn = rq.request.ReceiveFn
		}

		dPeer.addBodyAck(payload.ParcelID)
	}

	return receiveFn(retAddr, payload.ParcelType, payload.Data)
}

func (p *controller) receiveParcelBeforeData(packet *uniproto.Packet, payload *ParcelPacket, dataFn func() error) error {
	sid := AsShipmentID(packet.Header.SourceID, payload.ParcelID)
	if fn := p.bodyRq.suspendRetry(sid); fn != nil {
		defer fn()
	}

	return dataFn()
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

// TODO support loopback
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

func (p *controller) adjustedExpiry(pn pulse.Number, ttl uint8, inbound bool) (bool, uint32, error) {
	cycle, cyclePN := p.getPulseCycle()
	switch {
	case cyclePN == pn:
		//
	case cyclePN < pn:
		return false, 0, throw.IllegalState()
	case inbound:
		return false, 0, throw.IllegalValue()
	case pn != 0: //
		cycle--
	}
	return true, cycle + uint32(ttl), nil
}

func (p *controller) send(to nwapi.Address, returnId ShortShipmentID, shipment Shipment) error {
	dPeer, err := p.peer(to)
	if err != nil {
		return err
	}

	switch pn := shipment.PN; {
	case pn.IsUnknown():
		if shipment.Policies&ExactPulse != 0 {
			return throw.IllegalValue()
		}
	case !pn.IsTimePulse():
		return throw.IllegalValue()
	}

	sendSize, err := p.applySizePolicy(&shipment)
	if err != nil {
		return err
	}

	msg := &msgShipment{
		peer:     dPeer,
		returnID: returnId,
		shipment: shipment,
	}

	switch ok, expiry, err := p.adjustedExpiry(shipment.PN, shipment.TTL, false); {
	case err != nil:
		return err
	case !ok:
		// expired
		return nil
	default:
		msg.expires = expiry
	}

	// ATTN! Avoid gaps in numbering
	msg.id = dPeer.NextShipmentId()

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

func (p *controller) sendBodyRq(from ReturnAddress, rq ShipmentRequest) error {
	dPeer, err := p.peer(from.returnTo)
	if err != nil {
		return err
	}

	return p.bodyRq.Add(rqShipment{
		id:      AsShipmentID(uint32(dPeer.peerID), from.returnID),
		expires: from.expires,
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
		p.bodyRq.NextTimeCycle()
		p.sender.NextTimeCycle()
	}
}
