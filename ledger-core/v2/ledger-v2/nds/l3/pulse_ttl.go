// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l3

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type PulseTTL struct {
	StartFrom pulse.Number
	TTL       uint8
}

func (v PulseTTL) IsZero() bool {
	return v.StartFrom.IsUnknown()
}

type PulseOrdinal uint32

type LocalTTL struct {
	EOL PulseOrdinal
	PulseTTL
}

func (v LocalTTL) IsZero() bool {
	return v.StartFrom.IsUnknown()
}

func (v LocalTTL) IsAlreadyExpired() bool {
	return v.EOL == 0
}

func (v LocalTTL) IsExpired(now PulseOrdinal) bool {
	return v.IsAlreadyExpired() || v.EOL < now
}

func NewExpiryManager(maxTTL uint8) *ExpiryManager {
	return &ExpiryManager{maxTTL: maxTTL}
}

type ExpiryManager struct {
	mutex  sync.RWMutex
	oldest pulse.Number

	latest pulse.Number
	next   pulse.Number

	counter       atomickit.Uint32
	pulseOrdinals map[pulse.Number]pulseOrd
	maxTTL        uint8
}

type pulseOrd struct {
	ord  PulseOrdinal
	next pulse.Number
}

func (p *ExpiryManager) Localize(ttl PulseTTL) LocalTTL {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.pulseOrdinals == nil {
		panic(throw.IllegalState())
	}

	if ttl.TTL > p.maxTTL {
		ttl.TTL = p.maxTTL
	}
	result := LocalTTL{PulseTTL: ttl, EOL: PulseOrdinal(p.counter.Load())}

	switch {
	case ttl.StartFrom >= p.next:
		result.StartFrom = p.next
		result.EOL += PulseOrdinal(ttl.TTL)
	case ttl.StartFrom >= p.latest:
		switch result.StartFrom = p.latest; {
		case ttl.StartFrom != p.latest:
			// wrong pulse - leave minimal TTL
		default:
			result.EOL += PulseOrdinal(ttl.TTL)
		}
	case ttl.StartFrom < p.oldest:
		switch n := int(ttl.TTL) - len(p.pulseOrdinals); {
		case n <= 0:
			// expired
			result.EOL = 0
		default:
			// it seems this node was started recently and doesn't have all pulses
			// so, have to be pessimistic
			result.EOL += PulseOrdinal(n) - 1
		}
	default:
		if ord, ok := p.pulseOrdinals[ttl.StartFrom]; ok {
			if eol := ord.ord + PulseOrdinal(ttl.TTL); eol < result.EOL {
				result.EOL = 0
			} else {
				result.EOL = eol
			}
			break
		}
		// wrong pulse - leave minimal TTL
	}
	return result
}

func (p *ExpiryManager) init(pd pulse.Data) {
	if p.pulseOrdinals != nil {
		panic(throw.IllegalState())
	}
	p.pulseOrdinals = make(map[pulse.Number]pulseOrd, int(p.maxTTL)+1)
	p.counter.Store(1)
	p.latest = pd.PulseNumber
	p.next = pd.NextPulseNumber()
	p.oldest = p.latest
}

func (p *ExpiryManager) GetOrdinal() PulseOrdinal {
	n := PulseOrdinal(p.counter.Load())
	if n != 0 {
		return n
	}
	panic(throw.IllegalState())
}

func (p *ExpiryManager) NextPulse(pd pulse.Data) {
	pd.EnsurePulsarData()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.pulseOrdinals == nil {
		p.init(pd)
		return
	}

	p.pulseOrdinals[p.latest] = pulseOrd{PulseOrdinal(p.counter.Add(1) - 1), pd.PulseNumber}
	p.latest = pd.PulseNumber
	p.next = pd.NextPulseNumber()

	if len(p.pulseOrdinals) <= int(p.maxTTL) {
		return
	}
	if ord, ok := p.pulseOrdinals[p.oldest]; ok {
		delete(p.pulseOrdinals, p.oldest)
		p.oldest = ord.next
		return
	}
	panic(throw.Impossible())
}
