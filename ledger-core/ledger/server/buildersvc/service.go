// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/merkler"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PlashAssistant interface {
	PreparePulseChange(out chan<- cryptkit.Digest)
	CancelPulseChange()
	CommitPulseChange()

	CalculateJetDrop(reference.Holder) jet.DropID
	GetResolver() lineage.DependencyResolver
}

type AppendFuture interface {
	TrySetFutureResult(allocations []ledger.DirectoryIndex, err error) bool
}

type Service interface {
	CreatePlash(pulse.Range, jet.Tree, census.OnlinePopulation) (PlashAssistant, []jet.ExactID)
	AppendToDrop(jet.DropID, AppendFuture, lineage.ResolvedBundle)
}

var _ Service = &serviceImpl{}

func NewService(allocStrategy jetalloc.MaterialAllocationStrategy, merklePair cryptkit.PairDigester) Service {
	return &serviceImpl{
		allocStrategy: allocStrategy,
		merklePair: merklePair,
	}
}

type serviceImpl struct {
	allocStrategy jetalloc.MaterialAllocationStrategy
	merklePair    cryptkit.PairDigester

	mapMutex sync.RWMutex
	lastPN   pulse.Number
	plashes  map[pulse.Number]*plashAssistant
}

func (p *serviceImpl) AppendToDrop(id jet.DropID, future AppendFuture, bundle lineage.ResolvedBundle) {
	pa := p.get(id.CreatedAt())
	var err error
	switch {
	case pa != nil:
		err = pa.appendToDrop(id, future, bundle)
		if err == nil || future == nil {
			return
		}
	case future == nil:
		return
	default:
		err = throw.E("unknown plash", struct { jet.DropID }{ id })
	}

	future.TrySetFutureResult(nil, err)
}

func (p *serviceImpl) get(pn pulse.Number) *plashAssistant {
	p.mapMutex.RLock()
	defer p.mapMutex.RUnlock()
	return p.plashes[pn]
}

func (p *serviceImpl) CreatePlash(pr pulse.Range, tree jet.Tree, population census.OnlinePopulation) (PlashAssistant, []jet.ExactID) {

	localNodeID := population.GetLocalProfile().GetNodeID()
	if localNodeID.IsAbsent() {
		panic(throw.IllegalValue())
	}

	pd := pr.RightBoundData()
	pd.EnsurePulsarData()

	p.mapMutex.Lock()
	defer p.mapMutex.Unlock()

	pn := pd.PulseNumber
	switch {
	case p.plashes[pn] != nil:
		panic(throw.E("duplicate plash", struct { PN pulse.Number }{ pn }))
	case pn <= p.lastPN:
		panic(throw.E("retrograde plash", struct { PN, LastPN pulse.Number }{ pn, p.lastPN }))
	}

	pa := &plashAssistant{
		pulseData: pd,
		population: population,
		dropAssists: map[jet.ID]*dropAssistant{},
		merkle: merkler.NewForkingCalculator(p.merklePair, cryptkit.Digest{}),
	}

	if tree == nil || tree.IsEmpty() {
		pa.tree.Init()

		// TODO genesis
		panic(throw.NotImplemented())
		// return pa, []jet.ExactID{jet.GenesisExactID}
	}

	pa.tree = *tree
	pa.tree.SetPropagate() // grants O(1) to find jet

	jetCount := pa.tree.Count()
	jets := make([]jet.ExactID, 0, jetCount)

	pa.tree.Enum(func(prefix jet.Prefix, depth uint8) bool {
		jets = append(jets, jet.ID(prefix).AsExact(depth))
		return false
	})

	pa.calc = p.allocStrategy.CreateCalculator(pa.pulseData.PulseEntropy, population)
	jet2nodes := pa.calc.AllocationOfJets(jets, pn)

	if len(jet2nodes) != len(jets) {
		panic(throw.IllegalState())
	}

	result := jets[:0]
	for _, jetPID := range jets {

		jetID := jetPID.ID()
		var da *dropAssistant

		switch assignedNodeID := jet2nodes[jetID]; {
		case assignedNodeID.IsAbsent():
			panic(throw.IllegalState())

		case localNodeID == assignedNodeID:
			result = append(result, jetPID)
			da = &dropAssistant{}
			da.nodeID = assignedNodeID
			da.dropID = jetPID.AsDrop(pn)
			if len(jets) > 1 {
				da.merkle = pa.merkle.ForkSequence()
			}
			// da.writer = ; // TODO writer factory
		}
		pa.dropAssists[jetID] = da
	}

	// TODO write down shared data?

	if p.plashes == nil {
		p.plashes = map[pulse.Number]*plashAssistant{}
	}
	p.plashes[pn] = pa
	p.lastPN = pn
	pa.status.Store(plashStarted)

	return pa, result
}

// TODO a configuration set for conveyor that provides adapters and input-SM mapper
