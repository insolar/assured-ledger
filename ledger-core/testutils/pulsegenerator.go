package testutils

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PulseGenerator struct {
	delta            uint16
	history          []beat.Beat
	population       census.OnlinePopulation
	entropyGenerator entropygenerator.EntropyGenerator
}

func NewPulseGenerator(delta uint16, online census.OnlinePopulation, entropyGenerator entropygenerator.EntropyGenerator) *PulseGenerator {
	pulseGenerator := &PulseGenerator{
		delta:      delta,
		population: online,
	}
	if entropyGenerator == nil {
		pulseGenerator.entropyGenerator = &entropygenerator.StandardEntropyGenerator{}
	} else {
		pulseGenerator.entropyGenerator = entropyGenerator
	}
	return pulseGenerator
}

func (g *PulseGenerator) GetLastPulseData() pulse.Data {
	return g.GetLastBeat().Data
}

func makePacket(delta uint16, b beat.Beat) pulsar.PulsePacket {
	var entropy rms.Entropy
	copy(entropy[:], b.PulseEntropy[:rms.EntropySize])
	return pulsar.PulsePacket{
		PulseNumber:      b.PulseNumber,
		Entropy:          entropy,
		NextPulseNumber:  b.PulseNumber + pulse.Number(delta),
		PrevPulseNumber:  b.PulseNumber - pulse.Number(delta),
		EpochPulseNumber: b.PulseNumber.AsEpoch(),
		PulseTimestamp:   b.StartedAt.UnixNano(),
		Signs:            map[string]pulsar.SenderConfirmation{},
	}
}

func (g *PulseGenerator) GetLastPulsePacket() pulsar.PulsePacket {
	return makePacket(g.delta, g.GetLastBeat())
}

func (g *PulseGenerator) CountBackPacket(count uint) pulsar.PulsePacket {
	if int(count) >= len(g.history) {
		panic(throw.New("count is too big"))
	}

	pd := g.history[len(g.history)-int(count)-1]

	return makePacket(g.delta, pd)
}

func (g *PulseGenerator) GetLastBeat() beat.Beat {
	if len(g.history) == 0 {
		panic(throw.IllegalState())
	}
	return g.history[len(g.history)-1]
}

func (g *PulseGenerator) GetPrevBeat() beat.Beat {
	if len(g.history) <= 1 {
		panic(throw.IllegalState())
	}
	return g.history[len(g.history)-2]
}

func (g *PulseGenerator) GetDelta() uint16 {
	return g.delta
}

func (g *PulseGenerator) generateEntropy() longbits.Bits256 {
	entropy := g.entropyGenerator.GenerateEntropy()
	return longbits.NewBits256FromBytes(entropy[:])
}

func (g *PulseGenerator) Generate() pulse.Data {
	var newBeat beat.Beat
	if len(g.history) == 0 {
		newBeat = beat.Beat{
			Data:   pulse.NewFirstPulsarData(g.delta, g.generateEntropy()),
			Online: g.population,
		}
	} else {
		newBeat = beat.Beat{
			Data:   g.GetLastBeat().CreateNextPulsarPulse(g.delta, g.generateEntropy),
			Online: g.population,
		}
	}
	newBeat.BeatSeq = uint32(len(g.history) + 1)
	newBeat.StartedAt = time.Now()

	g.history = append(g.history, newBeat)

	return g.GetLastBeat().Data
}
