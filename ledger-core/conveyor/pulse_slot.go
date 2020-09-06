// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
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

// NewPresentPulseSlot is for test use only
func NewPresentPulseSlot(pulseManager *PulseDataManager, pr pulse.Range) PulseSlot {
	return PulseSlot{
		pulseManager: pulseManager,
		pulseData: &presentPulseDataHolder{
			bd: BeatData{Range: pr},
			at: time.Unix(int64(pr.RightBoundData().Timestamp), 0),
		},
	}
}

// NewPastPulseSlot is for test use only
func NewPastPulseSlot(pulseManager *PulseDataManager, pr pulse.Range) PulseSlot {
	return PulseSlot{
		pulseManager: pulseManager,
		pulseData: &presentPulseDataHolder{
			bd:     BeatData{Range: pr},
			at:     time.Unix(int64(pr.RightBoundData().Timestamp), 0),
			isPast: 1,
		},
	}
}

type PulseSlot struct {
	pulseManager *PulseDataManager
	pulseData    pulseDataHolder
	pulseChanger PulseChangerFunc
}

func (p *PulseSlot) State() PulseSlotState {
	_, ps := p.pulseData.BeatData()
	return ps
}

func (p *PulseSlot) PulseNumber() pulse.Number {
	return p.PulseData().PulseNumber
}

func (p *PulseSlot) PulseData() pulse.Data {
	pd := p.pulseData.PulseData()
	if pd.IsEmpty() {
		// possible incorrect injection for SM in the Antique slot
		panic(throw.IllegalState())
	}
	return pd
}

func (p *PulseSlot) PrevOperationPulseNumber() pulse.Number {
	bd, _ := p.pulseData.BeatData()

	if bd.Range == nil {
		// this is a backup
		// TODO reconsider if this is a viable option
		pd := p.pulseData.PulseData()
		if pn, ok := pd.PulseNumber.TryPrev(pd.PrevPulseDelta); ok {
			return pn
		}
		return pulse.Unknown
	}

	switch prd := bd.Range.LeftPrevDelta(); {
	case prd == 0:
		// first pulse, it has no prev
	case bd.Range.IsArticulated():
		// there is no data about an immediate previous pulse
	default:
		if pn, ok := bd.Range.LeftBoundNumber().TryPrev(prd); ok {
			return pn
		}
	}

	return pulse.Unknown
}

func (p *PulseSlot) PulseRange() (pulse.Range, PulseSlotState) {
	bd, st := p.pulseData.BeatData()
	if bd.Range == nil {
		// possible incorrect injection for SM in the Antique slot
		panic(throw.IllegalState())
	}
	return bd.Range, st
}

func (p *PulseSlot) BeatData() (BeatData, PulseSlotState) {
	bd, st := p.pulseData.BeatData()
	if bd.Range == nil {
		// possible incorrect injection for SM in the Antique slot
		panic(throw.IllegalState())
	}
	return bd, st
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

	switch bd, ps := p.pulseData.BeatData(); {
	case ps != Present:
		return ps, false
	case pn == presentPN:
		return ps, true
	case pn < bd.Range.LeftBoundNumber():
		// pn belongs to Past or Antique for sure
		return Past, false
	case bd.Range.IsSingular():
		return ps, false
	case !bd.Range.EnumNonArticulatedNumbers(func(n pulse.Number, prevDelta, nextDelta uint16) bool {
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
		if bd.Range.IsArticulated() {
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
	if bd, _ := p.pulseData.BeatData(); bd.Range != nil {
		return true
	}
	return p.pulseManager.HasPulseData(pn)
}

func (p *PulseSlot) postMigrate(prevState PulseSlotState, holder smachine.SlotMachineHolder) {
	if fn := p.pulseManager.pulseMigrateFn; fn != nil {
		fn(prevState, p, holder)
	}
}

var stubChanger PulseChangerFunc = func(outFn PreparePulseCallbackFunc) {
	if outFn != nil {
		// TODO temporary hack
		outFn(beat.AckData{})
	}
}

func (p *PulseSlot) prepareMigrate(outFn PreparePulseCallbackFunc) {
	if p.pulseChanger == nil {
		p.pulseChanger = stubChanger
	}

	if outFn != nil {
		p.pulseChanger(outFn)
	}
}

func (p *PulseSlot) cancelMigrate() {
	p.pulseChanger(nil)
}

// PulseChangerFunc is invoked with non-nil PreparePulseCallbackFunc when pulse change is preparing, and with nil when pulse change was cancelled.
type PulseChangerFunc func(PreparePulseCallbackFunc)

func (p *PulseSlot) SetPulseChanger(fn PulseChangerFunc) error {
	switch {
	case fn == nil:
		panic(throw.IllegalValue())
	case p.pulseChanger != nil:
		return throw.IllegalState()
	}
	p.pulseChanger = fn
	return nil
}
