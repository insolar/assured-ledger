// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type pulseDataHolder interface {
	// PulseRange is nil for Future and Antique
	PulseRange() (pulse.Range, PulseSlotState)
	// PulseData is provided for Future, and empty for Antique
	PulseData() (pulse.Data, PulseSlotState)
	// PulseStartedAt returns time at which the pulse was started. Only valid for Present and Past
	PulseStartedAt() time.Time

	MakePresent(pr pulse.Range, pulseStart time.Time)
	MakePast()
}

var _ pulseDataHolder = &futurePulseDataHolder{}

type futurePulseDataHolder struct {
	mutex    sync.RWMutex
	expected pulse.Data
	pr       pulse.Range
	at       time.Time
	isPast   bool
}

func (p *futurePulseDataHolder) PulseData() (pulse.Data, PulseSlotState) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	switch {
	case p.pr == nil:
		switch {
		case p.expected.IsEmpty():
			return pulse.Data{}, 0
		case p.isPast:
			panic(throw.IllegalState())
		}
		return p.expected, Future
	case p.isPast:
		return p.pr.RightBoundData(), Past
	default:
		return p.pr.RightBoundData(), Present
	}
}

func (p *futurePulseDataHolder) PulseRange() (pulse.Range, PulseSlotState) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p._pulseRange()
}

func (p *futurePulseDataHolder) _pulseRange() (pulse.Range, PulseSlotState) {
	switch {
	case p.pr == nil:
		switch {
		case p.expected.IsEmpty():
			return nil, 0
		case p.isPast:
			panic(throw.IllegalState())
		}
		return p.expected.AsRange(), Future
	case p.isPast:
		return p.pr, Past
	default:
		return p.pr, Present
	}
}

func (p *futurePulseDataHolder) MakePresent(pr pulse.Range, pulseStart time.Time) {
	pr.RightBoundData().EnsurePulsarData()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, ps := p._pulseRange(); ps != Future {
		panic(throw.IllegalState())
	}
	p.pr = pr
	p.at = pulseStart
}

func (p *futurePulseDataHolder) MakePast() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, ps := p._pulseRange(); ps != Present {
		panic(throw.IllegalState())
	}
	p.isPast = true
}

func (p *futurePulseDataHolder) PulseStartedAt() time.Time {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if _, ps := p._pulseRange(); ps == Future {
		panic(throw.IllegalState())
	}
	return p.at
}

/************************************/

var _ pulseDataHolder = &presentPulseDataHolder{}

type presentPulseDataHolder struct {
	pr     pulse.Range
	at     time.Time
	isPast uint32 // atomic
}

func (p *presentPulseDataHolder) PulseData() (pulse.Data, PulseSlotState) {
	return p.pr.RightBoundData(), p.State()
}

func (p *presentPulseDataHolder) PulseRange() (pulse.Range, PulseSlotState) {
	return p.pr, p.State()
}

func (p *presentPulseDataHolder) State() PulseSlotState {
	if atomic.LoadUint32(&p.isPast) == 0 {
		return Present
	}
	return Past
}

func (p *presentPulseDataHolder) MakePresent(pulse.Range, time.Time) {
	panic(throw.IllegalState())
}

func (p *presentPulseDataHolder) MakePast() {
	atomic.StoreUint32(&p.isPast, 1)
}

func (p *presentPulseDataHolder) PulseStartedAt() time.Time {
	return p.at
}

/************************************/

var _ pulseDataHolder = &antiqueNoPulseDataHolder{}

type antiqueNoPulseDataHolder struct {
}

func (antiqueNoPulseDataHolder) PulseData() (pulse.Data, PulseSlotState) {
	return pulse.Data{}, Antique
}

func (antiqueNoPulseDataHolder) PulseRange() (pulse.Range, PulseSlotState) {
	return nil, Antique
}

func (antiqueNoPulseDataHolder) PulseStartedAt() time.Time {
	panic(throw.IllegalState())
}

func (antiqueNoPulseDataHolder) MakePresent(pulse.Range, time.Time) {
	panic(throw.IllegalState())
}

func (antiqueNoPulseDataHolder) MakePast() {
	panic(throw.IllegalState())
}
