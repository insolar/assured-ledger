// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PulseSlotState uint8

const (
	_ PulseSlotState = iota
	Future
	Present
	Past
	Antique // non-individual past
)

func NewPresentPulseSlot(pulseManager *PulseDataManager, pr pulse.Range) PulseSlot {
	return PulseSlot{
		pulseManager: pulseManager,
		pulseData: &presentPulseDataHolder{
			pr: pr,
			at: time.Unix(int64(pr.RightBoundData().Timestamp), 0),
		},
	}
}

func NewPastPulseSlot(pulseManager *PulseDataManager, pr pulse.Range) PulseSlot {
	return PulseSlot{
		pulseManager: pulseManager,
		pulseData: &presentPulseDataHolder{
			pr:     pr,
			at:     time.Unix(int64(pr.RightBoundData().Timestamp), 0),
			isPast: 1,
		},
	}
}

type PulseSlot struct {
	pulseManager *PulseDataManager
	pulseData    pulseDataHolder
}

func (p *PulseSlot) State() PulseSlotState {
	_, ps := p.pulseData.PulseRange()
	return ps
}

func (p *PulseSlot) PulseData() pulse.Data {
	pd, _ := p.pulseData.PulseData()
	if pd.IsEmpty() {
		// possible incorrect injection for SM in the Antique slot
		panic("illegal state - not initialized")
	}
	return pd
}

func (p *PulseSlot) PulseRange() (pulse.Range, PulseSlotState) {
	pr, st := p.pulseData.PulseRange()
	if pr == nil {
		// possible incorrect injection for SM in the Antique slot
		panic("illegal state - not initialized")
	}
	return pr, st
}

func (p *PulseSlot) PulseStartedAt() time.Time {
	return p.pulseData.PulseStartedAt()
}

func (p *PulseSlot) CurrentPulseNumber() pulse.Number {
	pr, st := p.PulseRange()
	if st == Present {
		return pr.RightBoundData().PulseNumber
	}
	pn, _ := p.pulseManager.GetPresentPulse()
	return pn
}

func (p *PulseSlot) CurrentPulseData() pulse.Data {
	pr, st := p.PulseRange()
	if st == Present {
		return pr.RightBoundData()
	}
	pn, _ := p.pulseManager.GetPresentPulse()
	if pd, ok := p.pulseManager.GetPulseData(pn); ok {
		return pd
	}
	panic(throw.Impossible())
}

func (p *PulseSlot) isAcceptedFutureOrPresent(pn pulse.Number) (isFuture, isAccepted bool) {
	presentPN, futurePN := p.pulseManager.GetPresentPulse()
	if ps, ok := p._isAcceptedPresent(presentPN, pn); ps != Future {
		return false, ok
	}
	return true, pn >= futurePN
}

func (p *PulseSlot) _isAcceptedPresent(presentPN, pn pulse.Number) (PulseSlotState, bool) {
	var isProhibited bool

	switch pr, ps := p.pulseData.PulseRange(); {
	case ps != Present:
		return ps, false
	case pn == presentPN:
		return ps, true
	case pn < pr.LeftBoundNumber():
		// pn belongs to Past or Antique for sure
		return Past, false
	case pr.IsSingular():
		return ps, false
	case !pr.EnumNonArticulatedNumbers(func(n pulse.Number, prevDelta, nextDelta uint16) bool {
		switch {
		case n == pn:
		case pn.IsEqOrOut(n, prevDelta, nextDelta):
			return pn < n // stop search, as EnumNumbers from smaller to higher pulses
		default:
			// this number is explicitly prohibited by a known pulse data
			isProhibited = true
			// and stop now
		}
		return true
	}):
		// we've seen neither positive nor negative match
		if pr.IsArticulated() {
			// Present range is articulated, so anything that is not wrong - can be valid as present
			return ps, true
		}
		fallthrough
	case isProhibited: // Present range is articulated, so anything that is not wrong - can be valid
		return ps, false
	default:
		// we found a match in a range of the present slot
		return ps, true
	}
}

func (p *PulseSlot) HasPulseData(pn pulse.Number) bool {
	if pr, _ := p.pulseData.PulseRange(); pr != nil {
		return true
	}
	return p.pulseManager.HasPulseData(pn)
}
