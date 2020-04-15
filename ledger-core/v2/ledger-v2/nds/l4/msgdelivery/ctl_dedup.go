// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type DedupId uint32

const maxReceiveWindow = 1 << 16

type receiveDeduplicator struct {
	pn            pulse.Number
	maxReceived   DedupId
	received      map[DedupId]struct{}
	minReceived   DedupId // >= minContinuous
	excepts       map[DedupId]struct{}
	minContinuous DedupId

	prevCount int
}

func (p *receiveDeduplicator) Has(id DedupId) bool {
	switch {
	case id > p.minReceived:
		if id >= p.maxReceived {
			return id == p.maxReceived
		}
		return p.hasReceived(id)
	case id <= p.minContinuous:
		return id != 0
	case id == p.minReceived:
		return true
	default:
		return !p.hasExcept(id)
	}
}

func (p *receiveDeduplicator) hasReceived(id DedupId) bool {
	_, ok := p.received[id]
	return ok
}

func (p *receiveDeduplicator) hasExcept(id DedupId) bool {
	_, ok := p.excepts[id]
	return ok
}

func (p *receiveDeduplicator) Add(id DedupId) bool {
	switch {
	case id >= p.maxReceived:
		switch {
		case id > p.maxReceived+1:
			if p.minReceived+maxReceiveWindow < id {
				panic(throw.IllegalValue()) // TODO force relocation to continuous area
			}
			//
		case id == p.minContinuous+1:
			p.minReceived = id
			p.minContinuous = id
		case id == p.minReceived+1:
			p.minReceived = id
		}
		p.maxReceived = id
		return true

	case id <= p.minContinuous:
		return false

	case id < p.minReceived:
		switch {
		case id == p.minContinuous+1:
			switch len(p.excepts) {
			case 1:
				p.minContinuous = p.minReceived
				p.excepts = nil
				return true
			case 0:
				//
			default:
				for next := id + 1; next < p.minReceived; next++ {
					if p.hasExcept(next) {
						delete(p.excepts, id)
						p.minContinuous = next - 1
						return true
					}
				}
			}

			panic(throw.Impossible())

		case !p.hasExcept(id):
			return false
		default:
			delete(p.excepts, id)
			return true
		}

	//case id == 0: //  covered by id <= p.minContinuous
	//	panic(throw.Impossible())

	case id == p.minReceived:
		return false
	case id == p.minReceived+1:
		p.minReceived = id
	case p.hasReceived(id):
		return false
	default:
	}

	delta := uint64(p.maxReceived - p.minReceived)
	if delta <= 1 {
		panic(throw.Impossible())
	}
	delta -= 2

	switch n := uint64(len(p.received)); {
	case n == delta:
		// full house
		if p.minReceived == p.minContinuous {
			p.minContinuous = p.maxReceived
		}
		p.minReceived = p.maxReceived
		p.received = nil
		return true

	case n > delta:
		panic(throw.Impossible())

	case id > p.minReceived:
		p.received[id] = struct{}{}

	default:
		for next := id + 1; next < p.maxReceived; next++ {
			if !p.hasReceived(next) {
				p.received[id] = struct{}{}
				p.minReceived = next - 1
				break
			}
		}
		panic(throw.Impossible())
	}

	// TODO estimate and relocated p.received to p.excepts
	return true
}

func (p *receiveDeduplicator) Reset( /* minCutoff DedupId */ ) {
	const minCutoff = 0
	//switch {
	//case minCutoff == 0 || minCutoff >= p.maxReceived:
	p.excepts = nil
	p.minReceived = minCutoff
	p.maxReceived = minCutoff
	p.minContinuous = minCutoff
	//case minCutoff <= p.minContinuous:
	//	//
	//}

	const minCount = 32
	n := len(p.received)
	switch h := p.prevCount >> 1; {
	case n > h:
		//
	case h < minCount>>1:
		p.prevCount = minCount
	default:
		n = h
	}

	if p.prevCount <= minCount {
		p.received = nil
	} else {
		p.received = make(map[DedupId]struct{}, n)
	}
	p.prevCount = n
}
