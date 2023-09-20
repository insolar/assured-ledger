package beatstor

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewInMemory(depth int) *Memory {
	if depth <= 0 {
		panic(throw.IllegalState())
	}

	return &Memory{ snapshots: make([]snapshot, depth)}
}

var _ beat.Appender = &Memory{}
type Memory struct {
	mutex     sync.RWMutex
	numbers   map[pulse.Number]uint32
	snapshots []snapshot
	nextPos   int
	expected  beat.Beat
}

func (p *Memory) TimeBeat(pn pulse.Number) (beat.Beat, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	switch n, err := p.get(pn); {
	case err != nil:
		return beat.Beat{}, err
	case !p.snapshots[n].IsOperational():
		return beat.Beat{}, throw.E("non-operational pulse", struct{ PN pulse.Number}{pn})
	default:
		return p.snapshots[n].Beat, nil
	}
}

func (p *Memory) get(pn pulse.Number) (int, error) {
	if n, ok := p.numbers[pn]; ok {
		return int(n), nil
	}
	return -1, throw.E("unknown pulse", struct{ PN pulse.Number}{pn})
}

func (p *Memory) LatestTimeBeat() (beat.Beat, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	switch n, err := p.latest(); {
	case err != nil:
		return beat.Beat{}, err
	default:
		return p.snapshots[n].Beat, nil
	}
}

func (p *Memory) latest() (int, error) {
	switch {
	case p.nextPos > 0:
		return p.nextPos - 1, nil
	case len(p.numbers) > 0:
		return len(p.snapshots) - 1, nil
	default:
		return -1, throw.E("no pulses")
	}
}

func (p *Memory) AddExpectedBeat(bt beat.Beat) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if err := p._checkBeat(bt); err != nil {
		return err
	}

	if !p.expected.IsZero() {
		panic(throw.IllegalState())
	}

	p.expected = bt
	return nil
}

func (p *Memory) AddCommittedBeat(bt beat.Beat) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if err := p._checkBeat(bt); err != nil {
		return err
	}

	if !p.expected.IsZero() {
		if err := p._checkExpected(bt); err != nil {
			return err
		}
		p.expected = beat.Beat{}
	}

	if oldest := p.snapshots[p.nextPos].PulseNumber; !oldest.IsUnknown() {
		delete(p.numbers, oldest)
	}
	p.snapshots[p.nextPos] = newSnapshot(bt)
	p.numbers[bt.PulseNumber] = uint32(p.nextPos)

	p.nextPos++
	if p.nextPos == len(p.snapshots) {
		p.nextPos = 0
	}

	return nil
}

func (p *Memory) EnsureLatestTimeBeat(bt beat.Beat) error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	errMsg := ""
	n, err := p.latest()
	if err != nil {
		return err
	}

	last := p.snapshots[n].Beat
	switch {
	case last.Data != bt.Data:
		errMsg = "inconsistent pulse data"
	case last.BeatSeq != bt.BeatSeq:
		errMsg = "inconsistent beat sequence"
	default:
		return nil
	}
	return throw.E(errMsg, struct { Beat1, Beat2 beat.Beat	}{last, bt})
}

func (p *Memory) GetNodeSnapshot(pn pulse.Number) (beat.NodeSnapshot, error) {
	switch n, err := p.get(pn); {
	case err != nil:
		return nil, err
	default:
		return p.snapshots[n], nil
	}
}

func (p *Memory) FindAnyLatestNodeSnapshot() beat.NodeSnapshot {
	switch n, err := p.latest(); {
	case err != nil:
		return nil
	default:
		return p.snapshots[n]
	}
}

func (p *Memory) FindLatestNodeSnapshot() beat.NodeSnapshot {
	switch n, err := p.latest(); {
	case err != nil:
		return nil
	case !p.snapshots[n].IsOperational():
		return nil
	default:
		return p.snapshots[n]
	}
}

func (p *Memory) _checkBeat(bt beat.Beat) error {
	if bt.PulseNumber.IsUnknown() {
		panic(throw.IllegalValue())
	}
	n, _ := p.latest()
	if n < 0 {
		return nil
	}

	latest := p.snapshots[n]

	switch {
	case latest.IsFromPulsar():
		if !bt.IsFromPulsar() {
			return throw.E("invalid pulse type", struct{ Beat beat.Beat }{bt})
		}
	case bt.IsFromPulsar():
		// any pulse number is acceptable after ephemeral
		return nil
	}

	if expectedPN := latest.NextPulseNumber(); expectedPN > bt.PulseNumber {
		return throw.E("incorrect pulse sequence",
			struct{ ExpectedPN, PN pulse.Number}{expectedPN, bt.PulseNumber})
	}
	return nil
}

func (p *Memory) _checkExpected(bt beat.Beat) error {
	errMsg := ""
	switch {
	case p.expected.PulseNumber != bt.PulseNumber:
		errMsg = "unexpected pulse number"
	case p.expected.BeatSeq != bt.BeatSeq:
		errMsg = "unexpected beat sequence"
	case p.expected.Online.GetIndexedCount() != bt.Online.GetIndexedCount():
		errMsg = "unexpected population size"
	default:
		return nil
	}
	return throw.E(errMsg, struct { Beat1, Beat2 beat.Beat	}{p.expected, bt})
}
