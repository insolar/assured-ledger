// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"
	"log"
	"math"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

const minHeadBatchWeight = 1 << 20

func NewController(pt uniproto.ProtocolType, factory nwapi.DeserializationFactory,
	receiveFn ReceiverFunc, resolverFn ResolverFunc, logger uniserver.MiniLogger,
) *Controller {
	switch {
	case factory == nil:
		panic(throw.IllegalValue())
	case receiveFn == nil:
		panic(throw.IllegalValue())
	case logger == nil:
		panic(throw.IllegalValue())
	}

	c := &Controller{pType: pt, factory: factory, timeCycle: 10 * time.Millisecond,
		receiveFn: receiveFn, resolverFn: resolverFn, logger: logger}

	c.senderConfig = SenderWorkerConfig{1, 5, 100}
	c.sender.init(10, 10)
	c.sender.stages.InitStages(minHeadBatchWeight, [...]int{10, 50, 100})

	c.staterConfig = SenderWorkerConfig{1, 5, 100}
	c.stater.init(10, 10)
	c.stater.stages.InitStages(1, [...]int{5, 10, 50})

	c.receiver.ctl = c
	return c
}

type Controller struct {
	pType      uniproto.ProtocolType
	factory    nwapi.DeserializationFactory
	receiveFn  ReceiverFunc
	resolverFn ResolverFunc
	logger     uniserver.MiniLogger

	timeCycle time.Duration

	starter  protoStarter
	receiver packetReceiver
	sender   msgSender
	stater   stateSender

	senderConfig SenderWorkerConfig
	staterConfig SenderWorkerConfig

	pulseCycle atomickit.Uint64

	stopSignal   synckit.ClosableSignalChannel
	maxSmallSize uint
	maxHeadSize  uint // <= maxSmallSize
}

// for initialization only
func (p *Controller) SetConfig(c Config) {
	switch {
	case !p.isConfigurable():
		panic(throw.IllegalState())
	case c.TimeCycle <= 0:
		panic(throw.IllegalValue())
	}

	p.timeCycle = c.TimeCycle
	p.sender.setConfig(c.MessageBatchSize, c.TimeCycle, c.MessageSender)
	p.stater.setConfig(1, c.TimeCycle, c.StateSender)

	p.senderConfig = c.MessageSender.SenderWorkerConfig
	p.staterConfig = c.StateSender.SenderWorkerConfig
}

// for initialization only
func (p *Controller) RegisterWith(regFn uniproto.RegisterControllerFunc) {
	switch {
	case p.pType == 0:
		panic(throw.IllegalState())
	case p.starter.ctl != nil:
		panic(throw.IllegalState())
	}
	p.starter.ctl = p
	regFn(p.pType, protoDescriptor, &p.starter, &p.receiver)
}

func (p *Controller) NewFacade() Service {
	return facade{p}
}

func (p *Controller) shipTo(to DeliveryAddress, shipment Shipment) error {
	addr := to.ResolveWith(p.resolverFn)
	return p.send(addr, 0, shipment)
}

func (p *Controller) shipReturn(to ReturnAddress, shipment Shipment) error {
	if !to.IsValid() {
		return throw.IllegalValue()
	}

	return p.send(to.returnTo, to.returnID, shipment)
}

func (p *Controller) pullBody(from ReturnAddress, rq ShipmentRequest) error {
	if !from.IsValid() {
		return throw.IllegalValue()
	}

	switch cycle, _ := p.getPulseCycle(); {
	case cycle > from.expires:
		//
	case !from.canPull:
		//
	default:
		return p.sendBodyRq(from, rq)
	}

	if fn := rq.ReceiveFn; fn != nil {
		go func() {
			if err := fn(from, false, nil); err != nil {
				p.reportError(err)
			}
		}()
	}
	return nil
}

func (p *Controller) rejectBody(from ReturnAddress) error {
	if !from.IsValid() {
		return throw.IllegalValue()
	}
	if from.canPull {
		return p.rejectBodyRq(from)
	}
	return nil
}

func (p *Controller) isConfigurable() bool {
	return !p.starter.wasStarted()
}

func (p *Controller) checkActive() error {
	if !p.starter.isActive() {
		return throw.FailHere("inactive")
	}
	return nil
}

func (p *Controller) reportError(err error) {
	p.logger.LogError(err)
}

func (p *Controller) nextPulseCycle(pn pulse.Number) (bool, uint32) {
	for {
		v := p.pulseCycle.Load()
		if pulse.Number(v) >= pn {
			return false, uint32(v >> 32)
		}

		n := uint64(pn) | v&^math.MaxUint32 + 1<<32
		if p.pulseCycle.CompareAndSwap(v, n) {
			return true, uint32(n >> 32)
		}
	}
}

func (p *Controller) getPulseCycle() (uint32, pulse.Number) {
	v := p.pulseCycle.Load()
	return uint32(v >> 32), pulse.Number(v)
}

func (p *Controller) receiveState(packet *uniproto.ReceivedPacket, payload *StatePacket) error {
	if err := p.checkActive(); err != nil {
		return err
	}

	peerID := packet.Header.SourceID

	if n := len(payload.BodyRqList); n > 0 {
		dPeer, ok := packet.Peer.GetProtoInfo(p.pType).(*DeliveryPeer)
		if !ok {
			return throw.RemoteBreach("peer is not a node")
		}

		for _, id := range payload.BodyRqList {
			switch msg := p.sender.get(AsShipmentID(peerID, id)); {
			case msg == nil:
				dPeer.addReject(id)
			case msg.markBodyRq():
				p.sender.sendBodyOnly(msg)
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
		if rq, ok := p.stater.RemoveByID(sid); ok {
			if fn := rq.requestRejectedFn(); fn != nil {
				go fn()
			}
		}
	}

	return nil
}

func (p *Controller) getDeliveryPeer(peer uniproto.Peer) (*DeliveryPeer, error) {
	dPeer, ok := peer.GetOrCreateProtoInfo(p.pType, p.createProtoPeer).(*DeliveryPeer)
	if ok {
		return dPeer, nil
	}
	return nil, throw.RemoteBreach("peer is not a node")
}

func (p *Controller) receiveParcel(packet *uniproto.ReceivedPacket, payload *ParcelPacket) error {
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

	duplicate := !dPeer.dedup.Add(DedupID(payload.ParcelID))
	// todo: remove me
	log.Printf("peerID: %d ParcelID: %d duplicate: %t", dPeer.peerID, payload.ParcelID, duplicate)

	ok := false
	if ok, _, retAddr.expires, err = p.adjustedExpiry(payload.PulseNumber, payload.TTLCycles, true); !ok {
		dPeer.addReject(payload.ParcelID)
		return err
	}

	if payload.ParcelType == nwapi.HeadOnlyPayload {
		if duplicate {
			dPeer.addReject(payload.ParcelID)
			return nil
		}
		dPeer.addAck(payload.ParcelID)
		retAddr.canPull = true
		return p.receiveFn(retAddr, payload.ParcelType, payload.Data)
	}

	var rq rqShipment
	switch rq, ok = p.stater.RemoveRq(dPeer, payload.ParcelID); {
	case ok:
		// Body ignores peer-based deduplication when served per-request
		if rq.isValid() {
			break
		}

		dPeer.addReject(payload.ParcelID)
		if fn := rq.requestRejectedFn(); fn != nil {
			fn()
		}
		return nil
	case duplicate:
		dPeer.addReject(payload.ParcelID)
		return nil
	}

	dPeer.addBodyAck(payload.ParcelID)

	receiveFn := rq.request.ReceiveFn
	if receiveFn == nil {
		receiveFn = p.receiveFn
	}
	return receiveFn(retAddr, payload.ParcelType, payload.Data)
}

func (p *Controller) onReceiveLargeParcelData(packet *uniproto.Packet, payload *ParcelPacket, dataFn func() error) error {
	sid := AsShipmentID(packet.Header.SourceID, payload.ParcelID)
	if fn := p.stater.suspendRetry(sid); fn != nil {
		defer fn()
	}

	return dataFn()
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
	dp.init()
	return dp
}

func (p *Controller) applySizePolicy(shipment *Shipment) (uint, error) {
	switch {
	case shipment.Head != nil:
		headSize := shipment.Head.ByteSize()
		switch {
		case headSize > p.maxHeadSize:
			return 0, throw.FailHere("head is too big")
		case shipment.Body == nil:
			return headSize, nil
		}

		bodySize := shipment.Body.ByteSize()
		if bodySize > p.maxSmallSize {
			shipment.Policies |= largeBody
		}

		switch {
		case bodySize <= headSize:
			return 0, throw.IllegalValue()
		case bodySize <= p.maxHeadSize:
			shipment.Head = shipment.Body
			shipment.Body = nil
		case shipment.Policies&ExpectedParcel != 0:
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

func (p *Controller) peer(to nwapi.Address) (*DeliveryPeer, error) {
	if err := p.checkActive(); err != nil {
		return nil, err
	}

	// This protocol is only allowed for peers added by consensus
	// It can't connect unknown peers.
	switch peer, err := p.starter.peers.ConnectedPeer(to); {
	case err != nil:
		return nil, err
	case peer != nil:
		return p.getDeliveryPeer(peer)
	default:
		// local
		return nil, nil
	}
}

func (p *Controller) adjustedExpiry(pn pulse.Number, ttl uint8, inbound bool) (bool, uint32, uint32, error) {
	cycle, cyclePN := p.getPulseCycle()
	switch {
	case cyclePN == pn:
		//
	case cyclePN < pn:
		return false, 0, 0, throw.IllegalState()
	case inbound:
		return false, 0, 0, throw.FailHere("past pulse")
	case pn == 0:
		// use current
	case ttl == 0:
		// expired
		return false, 0, 0, nil
	default:
		return true, cycle, cycle + uint32(ttl) - 1, nil
	}
	return true, cycle, cycle + uint32(ttl), nil
}

func (p *Controller) send(to nwapi.Address, returnID ShortShipmentID, shipment Shipment) error {
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
		returnID: returnID,
		shipment: shipment,
	}

	var currentCycle uint32
	switch ok, cur, expiry, err := p.adjustedExpiry(shipment.PN, shipment.TTL, false); {
	case err != nil:
		return err
	case !ok:
		// expired
		return nil
	default:
		currentCycle = cur
		msg.expires = expiry
	}

	if dPeer == nil {
		p.sendLoopback(msg)
		return nil
	}

	// ATTN! Avoid gaps in numbering
	msg.id = dPeer.NextShipmentID()

	switch {
	case msg.shipment.Head == nil:
		msg.markBodyRq()
		p.sender.sendBodyOnly(msg)
	case msg.shipment.Body == nil && msg.isFireAndForget():
		p.sender.sendHeadNoRetry(msg)
	default:
		p.sender.sendHead(msg, sendSize, currentCycle)
	}
	return nil
}

func (p *Controller) sendBodyRq(from ReturnAddress, rq ShipmentRequest) error {
	dPeer, err := p.peer(from.returnTo)
	switch {
	case err != nil:
		return err
	case dPeer == nil:
		// local address can't be here
		return throw.Impossible()
	}

	return p.stater.Add(rqShipment{
		id:      AsShipmentID(uint32(dPeer.peerID), from.returnID),
		expires: from.expires,
		peer:    dPeer,
		request: rq,
	})
}

func (p *Controller) rejectBodyRq(from ReturnAddress) error {
	dPeer, err := p.peer(from.returnTo)
	switch {
	case err != nil:
		return err
	case dPeer == nil:
		// local address can't be here
		return throw.Impossible()
	}

	sid := AsShipmentID(uint32(dPeer.peerID), from.returnID)
	if rq, _ := p.stater.RemoveByID(sid); !rq.isEmpty() {
		dPeer.addReject(from.returnID)
	}
	return nil
}

func (p *Controller) getLocalID() nwapi.ShortNodeID {
	return p.starter.peers.LocalPeer().GetNodeID()
}

func (p *Controller) onStarted() {
	p.maxSmallSize = p.starter.peers.MaxSmallPayloadSize()

	if p.maxHeadSize > p.maxSmallSize || p.maxHeadSize == 0 {
		p.maxHeadSize = p.maxSmallSize
	}
	p.stopSignal = make(synckit.ClosableSignalChannel)

	startWorkerByConfig(p.staterConfig, p.stater.startWorker)
	startWorkerByConfig(p.senderConfig, p.sender.startWorker)

	go p.runWorker()
}

func startWorkerByConfig(c SenderWorkerConfig, startFn func(parallel int, postponed int)) {
	for i := c.ParallelWorkers; i > 0; i-- {
		startFn(c.ParallelPeersPerWorker, c.MaxPostponedPerWorker)
	}
}

func (p *Controller) onStopped() {
	close(p.stopSignal)
}

func (p *Controller) runWorker() {
	ticker := time.NewTicker(p.timeCycle)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopSignal:
			p.sender.stop()
			p.stater.stop()
			return

		case <-ticker.C:
			//
		}
		p.stater.NextTimeCycle()
		p.sender.nextTimeCycle()
	}
}

func (p *Controller) sendLoopback(msg *msgShipment) {
	data := msg.shipment.Body
	if data == nil {
		data = msg.shipment.Head
	}

	peer := p.starter.peers.LocalPeer()
	retAddr := ReturnAddress{
		returnTo: peer.GetLocalUID(),
		expires:  msg.expires,
	}

	err := p.receiveFn(retAddr, nwapi.CompletePayload, data)
	if err != nil {
		p.reportError(err)
	}
}
