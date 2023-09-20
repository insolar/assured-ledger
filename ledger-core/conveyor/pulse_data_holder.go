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
	BeatData() (BeatData, PulseSlotState)
	// PulseData is provided for Future, and empty for Antique
	PulseData() pulse.Data
	// PulseStartedAt returns time at which the pulse was started. Only valid for Present and Past
	PulseStartedAt() time.Time

	MakePresent(bd BeatData, pulseStart time.Time)
	MakePast()
}

var _ pulseDataHolder = &futurePulseDataHolder{}

type futurePulseDataHolder struct {
	mutex sync.RWMutex
	bd    BeatData
	at    time.Time
	state PulseSlotState
}

func (p *futurePulseDataHolder) PulseData() pulse.Data {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.bd.Range == nil {
		return pulse.Data{}
	}
	return p.bd.Range.RightBoundData()
}

func (p *futurePulseDataHolder) BeatData() (BeatData, PulseSlotState) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p._beatData()
}

func (p *futurePulseDataHolder) _beatData() (BeatData, PulseSlotState) {
	return p.bd, p.state
}

func (p *futurePulseDataHolder) MakePresent(bd BeatData, pulseStart time.Time) {
	bd.Range.RightBoundData().EnsurePulsarData()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.state > Future {
		panic(throw.IllegalState())
	}
	p.state = Present
	p.bd = bd
	p.at = pulseStart
}

func (p *futurePulseDataHolder) MakePast() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.state != Present {
		panic(throw.IllegalState())
	}
	p.state = Past
}

func (p *futurePulseDataHolder) PulseStartedAt() time.Time {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.state <= Future {
		panic(throw.IllegalState())
	}
	return p.at
}

/************************************/

var _ pulseDataHolder = &presentPulseDataHolder{}

type presentPulseDataHolder struct {
	bd     BeatData
	at     time.Time
	isPast uint32 // atomic
}

func (p *presentPulseDataHolder) PulseData() pulse.Data {
	return p.bd.Range.RightBoundData()
}

func (p *presentPulseDataHolder) BeatData() (BeatData, PulseSlotState) {
	return p.bd, p.State()
}

func (p *presentPulseDataHolder) State() PulseSlotState {
	if atomic.LoadUint32(&p.isPast) == 0 {
		return Present
	}
	return Past
}

func (p *presentPulseDataHolder) MakePresent(BeatData, time.Time) {
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

type antiqueNoPulseDataHolder struct {}

func (antiqueNoPulseDataHolder) PulseData() pulse.Data {
	return pulse.Data{}
}

func (antiqueNoPulseDataHolder) BeatData() (BeatData, PulseSlotState) {
	return BeatData{}, Antique
}

func (antiqueNoPulseDataHolder) PulseStartedAt() time.Time {
	panic(throw.IllegalState())
}

func (antiqueNoPulseDataHolder) MakePresent(BeatData, time.Time) {
	panic(throw.IllegalState())
}

func (antiqueNoPulseDataHolder) MakePast() {
	panic(throw.IllegalState())
}
