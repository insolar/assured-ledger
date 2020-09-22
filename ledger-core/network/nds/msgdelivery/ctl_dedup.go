// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type DedupID uint32

// TODO  https://insolar.atlassian.net/browse/PLAT-826
var (
	maxReceiveExceptions = 1 << 8
	maxReceiveWindow     = 1 << 16
	minReceiveWindow     = maxReceiveWindow - maxReceiveExceptions>>1

	// TODO https://insolar.atlassian.net/browse/PLAT-819
	// overlapLimit = ^DedupID(0) - maxReceiveWindow - 1
)

type receiveDeduplicator struct {
	mutex         sync.Mutex
	maxReceived   DedupID // >= minReceived
	received      map[DedupID]struct{}
	minReceived   DedupID // >= maxContinuous
	excepts       map[DedupID]struct{}
	maxContinuous DedupID

	peakSize int
}

func (p *receiveDeduplicator) Has(id DedupID) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case id > p.minReceived:
		if id < p.maxReceived {
			return p.hasReceived(id)
		}
		return id == p.maxReceived
	case id <= p.maxContinuous:
		return id != 0
	case id == p.minReceived:
		return true
	default:
		return !p.hasExcept(id)
	}
}

func (p *receiveDeduplicator) hasReceived(id DedupID) bool {
	_, ok := p.received[id]
	return ok
}

func (p *receiveDeduplicator) hasExcept(id DedupID) bool {
	_, ok := p.excepts[id]
	return ok
}

func (p *receiveDeduplicator) Add(id DedupID) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case id >= p.maxReceived:
		switch {
		case id == p.maxReceived:
			return false
		case id > p.maxReceived+1:
			//
		case id == p.maxContinuous+1:
			p.maxContinuous = id
			fallthrough
		case id == p.minReceived+1:
			p.maxReceived = id
			p.minReceived = id
			return true
		}
		p.maxReceived = id

	case id <= p.maxContinuous:
		return false

	case id < p.minReceived:
		return p._addToExcepts(id)

	//case id == 0: //  covered by id <= p.maxContinuous
	//	panic(throw.Impossible())

	case id == p.minReceived:
		return false

	case id == p.minReceived+1:
		if p.hasReceived(id) {
			panic(throw.Impossible())
		}
		if p.maxContinuous == p.minReceived {
			p._addToReceivedAndTrim()
			p.maxContinuous = p.minReceived
		} else {
			p._addToReceivedAndTrim()
		}

		return true

	case p.hasReceived(id):
		return false
	}

	if p.received == nil {
		p.received = make(map[DedupID]struct{}, p.peakSize>>2)
		p.received[id] = struct{}{}
		p.updatePeakSize(1)
		return true
	}

	p.received[id] = struct{}{}
	n := len(p.received)
	p.updatePeakSize(n)
	maxReceived := int(p.maxReceived)
	minReceived := int(p.minReceived)

	if n > minReceiveWindow && minReceived+maxReceiveWindow < maxReceived {
		p._forceMinReceived(DedupID(maxReceived - minReceiveWindow))
	}
	return true

}

func (p *receiveDeduplicator) _addToExcepts(id DedupID) bool {
	switch {
	case id == p.maxContinuous+1:
		switch len(p.excepts) {
		case 1:
			p.maxContinuous = p.minReceived
			p.excepts = nil
			return true
		case 0:
			panic(throw.Impossible())
		default:
			p._trimExceptsAfter(id)
			return true
		}

	case !p.hasExcept(id):
		return false
	default:
		delete(p.excepts, id)
		return true
	}
}

func (p *receiveDeduplicator) _trimExceptsAfter(id DedupID) {
	for next := id + 1; next < p.minReceived; next++ {
		if p.hasExcept(next) {
			delete(p.excepts, id)
			p.maxContinuous = next - 1
			return
		}
	}
	panic(throw.Impossible())
}

func (p *receiveDeduplicator) _addToReceivedAndTrim() {
	delta := uint64(p.maxReceived - p.minReceived)
	if delta <= 1 {
		panic(throw.Impossible())
	}
	delta--

	switch n := uint64(len(p.received)); {
	case n == delta:
		// full house
		p.minReceived = p.maxReceived
		p.received = nil
		return

	case n > delta:
		panic(throw.Impossible())
	}

	p.minReceived++
	if !p.hasReceived(p.minReceived) {
		return
	}

	p.minReceived++
	for {
		delete(p.received, p.minReceived)
		p.minReceived++
		switch {
		case p.minReceived == p.maxReceived:
			p.received = nil
			return
		case !p.hasReceived(p.minReceived):
			return
		}
	}
}

func (p *receiveDeduplicator) updatePeakSize(n int) {
	if p.peakSize < n {
		p.peakSize = n
	}
}

func (p *receiveDeduplicator) _putExcept(id DedupID) {
	if p.excepts == nil {
		p.excepts = make(map[DedupID]struct{})
	}
	p.excepts[id] = struct{}{}
}

func (p *receiveDeduplicator) _forceMinReceived(min DedupID) {
	for i := min - 1; i > p.minReceived; i-- {
		if p.hasReceived(i) {
			delete(p.received, i)
			continue
		}
		p._putExcept(i)
	}

	if p.hasReceived(min) {
		delete(p.received, min)
		// find last received
		for min++; min < p.maxReceived; min++ {
			if !p.hasReceived(min) {
				min--
				break
			}
			delete(p.received, min)
		}
	} else {
		p._putExcept(min)

		// find first received
		for min++; min < p.maxReceived; min++ {
			if p.hasReceived(min) {
				delete(p.received, min)
				break
			}
			p.excepts[min] = struct{}{}
		}
	}
	p.minReceived = min

	switch {
	case p.maxContinuous >= p.minReceived:
		panic(throw.Impossible())
	case len(p.excepts) <= maxReceiveExceptions:
		return
	}

	for p.maxContinuous++; p.maxContinuous < p.minReceived; p.maxContinuous++ {
		delete(p.excepts, p.maxContinuous)
		if len(p.excepts) < maxReceiveExceptions {
			return
		}
	}

	if len(p.excepts) != 0 {
		panic(throw.Impossible())
	}
	p.excepts = nil
}

func (p *receiveDeduplicator) TrimBuffer() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	n := len(p.received)
	switch {
	case n == 0:
		p.received = nil
		p.peakSize = 0
		return
	case n >= p.peakSize>>1:
		return
	case p.peakSize <= maxReceiveExceptions:
		return
	}

	p.peakSize = n
	m := make(map[DedupID]struct{}, n)
	for k, v := range p.received {
		m[k] = v
	}
	p.received = m
}
