// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

type stateSender struct {
	stages retries.StagedController

	oob    chan rqShipment
	jobs   chan retryJob
	states chan stateJob

	marks  nodeRqMarks

	peers peerMap

	mutex    sync.Mutex
	requests map[ShipmentID]rqShipment
	suspends map[ShipmentID]struct{}
}

func (p *stateSender) setConfig(batchSize uint, cycle time.Duration, c SenderConfig) {
	switch {
	case c.MaxPostponedPerWorker <= 0:
		panic(throw.IllegalValue())
	case c.ParallelPeersPerWorker <= 0:
		panic(throw.IllegalValue())
	case c.ParallelWorkers <= 0:
		panic(throw.IllegalValue())
	}

	p.init(c.FastQueue, c.RetryQueue)
	p.stages.InitStages(batchSize, calcPeriods(cycle, c.RetryIntervals))
}

func (p *stateSender) init(oobQueue, jobQueue int) {
	switch {
	case oobQueue <= 0:
		panic(throw.IllegalValue())
	case jobQueue <= 0:
		panic(throw.IllegalValue())
	}

	p.oob = make(chan rqShipment, oobQueue)
	p.states = make(chan stateJob, oobQueue)
	p.jobs = make(chan retryJob, jobQueue)
	p.requests = make(map[ShipmentID]rqShipment)
	p.suspends = make(map[ShipmentID]struct{})
}

func (p *stateSender) Add(rq rqShipment) error {
	if err := p.put(rq); err != nil {
		return err
	}

	p.oob <- rq
	return nil
}

func (p *stateSender) put(rq rqShipment) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, ok := p.requests[rq.id]; ok {
		return throw.FailHere("duplicate request")
	}
	p.requests[rq.id] = rq
	return nil
}

func (p *stateSender) get(id ShipmentID) (rqShipment, retries.RetryState) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	rq, ok := p.requests[id]
	if !ok {
		return rq, retries.RemoveCompletely
	}
	if _, ok = p.suspends[id]; ok {
		return rq, retries.StopRetrying
	}
	return rq, retries.KeepRetrying
}

func (p *stateSender) RemoveRq(peer *DeliveryPeer, id ShortShipmentID) (rqShipment, bool) {
	return p.RemoveByID(AsShipmentID(uint32(peer.peerID), id))
}

func (p *stateSender) RemoveByID(sid ShipmentID) (rqShipment, bool) {
	p.mutex.Lock()
	rq, ok := p.requests[sid]
	if ok {
		delete(p.suspends, sid)
		delete(p.requests, sid)
	}
	p.mutex.Unlock()

	if !ok || rq.request.Cancel.IsCancelled() {
		return rqShipment{}, false
	}
	return rq, ok
}

func (p *stateSender) suspendRetry(id ShipmentID) func() {
	p.mutex.Lock()
	if _, ok := p.requests[id]; !ok {
		p.mutex.Unlock()
		return nil
	}

	p.suspends[id] = struct{}{}
	p.mutex.Unlock()

	return func() {
		p.mutex.Lock()
		delete(p.suspends, id)
		p.mutex.Unlock()
	}
}

func (p *stateSender) NextTimeCycle() {
	p.peers.MarkFlush()
	p.stages.NextCycle(p)
	p.peers.RunFlush(p)
}

func (p *stateSender) Retry(ids []retries.RetryID, repeatFn func(retries.RetryID)) {

	keepCount, removeStart := p.retry(ids)

	for _, id := range ids[keepCount:removeStart] {
		repeatFn(id)
	}

	if keepCount > 0 {
		p.jobs <- retryJob{ids: ids[:keepCount], repeatFn: repeatFn}
	}
}

func (p *stateSender) retry(ids []retries.RetryID) (keepCount, removeStart int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return retries.Segregate(ids, func(id retries.RetryID) retries.RetryState {
		sid := ShipmentID(id)
		switch rq, ok := p.requests[sid]; {
		case !ok:
			return retries.RemoveCompletely
		case rq.isExpired() || !rq.peer.isValid():
			delete(p.requests, sid)
			delete(p.suspends, sid)
			return retries.RemoveCompletely
		}
		if _, ok := p.suspends[sid]; ok {
			return retries.StopRetrying
		}
		return retries.KeepRetrying
	})
}

func (p *stateSender) CheckState(retries.RetryID) (r retries.RetryState) {
	return retries.KeepRetrying
}

func (p *stateSender) Remove(ids []retries.RetryID) {
	p.mutex.Lock()
	for _, id := range ids {
		sid := ShipmentID(id)
		delete(p.requests, sid)
		delete(p.suspends, sid)
	}
	p.mutex.Unlock()
}

func (p *stateSender) startWorker(parallel, postponed int) {
	if p.jobs == nil {
		panic(throw.IllegalState())
	}
	worker := newRetryRqWorker(p, parallel, postponed)
	go worker.runRetry()
}

func (p *stateSender) AddPeer(peer *DeliveryPeer) {
	p.peers.Put(peer)
}

type stateJob struct {
	peer   *DeliveryPeer
	packet StatePacket
}

func (p *stateSender) AddState(peer *DeliveryPeer, packet StatePacket) {
	p.states <- stateJob{peer, packet}
}

func (p *stateSender) stop() {
	close(p.oob)
}

/***********************************/

type peerMap struct {
	mutex    sync.Mutex
	running  bool
	peers    map[nwapi.LocalUniqueID]*DeliveryPeer
	pendings []*DeliveryPeer
}

func (p *peerMap) Put(peer *DeliveryPeer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.peers == nil {
		p.peers = make(map[nwapi.LocalUniqueID]*DeliveryPeer)
	}
	p.peers[peer.uid] = peer
}

func (p *peerMap) MarkFlush() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.pendings == nil {
		p.pendings = make([]*DeliveryPeer, 0, len(p.peers))
	} else {
		p.pendings = p.pendings[:0]
	}

	for id, peer := range p.peers {
		peer.dedup.TrimBuffer()
		switch isValid, hasUpdates := peer.markFlush(); {
		case !isValid:
			delete(p.peers, id)
		case hasUpdates:
			p.pendings = append(p.pendings, peer)
		}
	}
}

func (p *peerMap) RunFlush(sender *stateSender) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.running || len(p.pendings) == 0 {
		return
	}
	pendings := p.pendings
	p.pendings = nil
	p.running = true
	go p._runFlush(sender, pendings)
}

func (p *peerMap) _runFlush(sender *stateSender, pendings []*DeliveryPeer) {
	defer func() {
		p.mutex.Lock()
		p.running = false
		if p.pendings == nil {
			// recycle the slice
			p.pendings = pendings
		}
		p.mutex.Unlock()
	}()

	for _, peer := range pendings {
		packet := peer.flushState()
		if packet != nil {
			sender.AddState(peer, *packet)
		}
	}
}

type nodeRqMarks struct {
	mutex sync.Mutex
	marks map[uint32]*retryRqWorker
}

func (p *nodeRqMarks) unmark(id ShipmentID) {
	p.unmarkNode(id.NodeID())
}

func (p *nodeRqMarks) mark(id ShipmentID, w *retryRqWorker) (bool, *retryRqWorker) {
	return p.markNode(id.NodeID(), w)
}

func (p *nodeRqMarks) markNode(nid uint32, w *retryRqWorker) (bool, *retryRqWorker) {
	if w == nil {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.marks == nil {
		p.marks = map[uint32]*retryRqWorker{}
	} else if owner, ok := p.marks[nid]; ok {
		return false, owner
	}

	p.marks[nid] = w
	return true, w
}

func (p *nodeRqMarks) unmarkNode(nid uint32) {
	p.mutex.Lock()
	delete(p.marks, nid)
	p.mutex.Unlock()
}
