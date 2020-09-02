// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/merkler"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type StorageSnapshotFactoryFunc = func(pulse.Number) bundle.SnapshotWriter

var _ Service = &serviceImpl{}
var _ ReadService = &serviceImpl{}

func newService(allocStrategy jetalloc.MaterialAllocationStrategy, merklePair cryptkit.PairDigester,
	storageFactoryFn StorageSnapshotFactoryFunc,
) *serviceImpl {
	return &serviceImpl{
		allocStrategy: allocStrategy,
		merklePair: merklePair,
		storageFactoryFn: storageFactoryFn,
	}
}

type serviceImpl struct {
	allocStrategy jetalloc.MaterialAllocationStrategy
	merklePair    cryptkit.PairDigester

	storageFactoryFn StorageSnapshotFactoryFunc

	mapMutex sync.RWMutex
	lastPN   pulse.Number
	plashes  map[pulse.Number]*plashAssistant
}

func (p *serviceImpl) DropReadDirty(id jet.DropID, fn func(reader bundle.DirtyReader) error) error {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	pa := p.get(id.CreatedAt())

	switch {
	case pa == nil:
		return throw.E("unknown plash", struct { jet.DropID }{ id })
	case pa.dropAssists[id.ID()] == nil:
		return throw.E("unknown drop", struct { jet.DropID }{ id })
	}

	return fn(pa.dirtyReader)
}

func (p *serviceImpl) AppendToDrop(id jet.DropID, future AppendFuture, bundle lineage.UpdateBundle) {
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

func (p *serviceImpl) CreatePlash(pr pulse.Range, treePrev, tree jet.Tree, population census.OnlinePopulation) (PlashAssistant, []jet.ExactID) {
	if tree == nil || tree.IsEmpty() {
		// missing value or genesis tree
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

	return p.createPlash(pr, *tree, population)
}

func (p *serviceImpl) createPlash(pr pulse.Range, tree jet.PrefixTree, population census.OnlinePopulation) (PlashAssistant, []jet.ExactID) {
	localNodeID := population.GetLocalProfile().GetNodeID()
	if localNodeID.IsAbsent() {
		panic(throw.IllegalValue())
	}

	pd := pr.RightBoundData()
	pd.EnsurePulsarData()
	pn := pd.PulseNumber

	pa := &plashAssistant{
		pulseData: pd,
		tree:      tree,
		population: population,
		dropAssists: map[jet.ID]*dropAssistant{},
		merkle: merkler.NewForkingCalculator(p.merklePair, cryptkit.Digest{}),
	}

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

	sw := p.storageFactoryFn(pn)
	pa.dirtyReader = sw.DirtyReader()
	bw := bundle.NewWriter(sw) // NB! MUST be one writer per storage

	result := jets[:0]
	for _, jetPID := range jets {

		jetID := jetPID.ID()
		var da *dropAssistant

		switch assignedNodeID := jet2nodes[jetID]; {
		case assignedNodeID.IsAbsent():
			panic(throw.IllegalState())

		case localNodeID == assignedNodeID:
			result = append(result, jetPID)
			da = &dropAssistant{
				nodeID: assignedNodeID,
				dropID: jetPID.AsDrop(pn),
			}
			if len(jets) > 1 {
				da.merkle = pa.merkle.ForkSequence()
			}
			da.writer = bw
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


func (p *serviceImpl) CreateGenesis(pr pulse.Range, population census.OnlinePopulation) (PlashAssistant, jet.ExactID) {
	pd := pr.RightBoundData()
	pd.EnsurePulsarData()

	p.mapMutex.Lock()
	defer p.mapMutex.Unlock()

	pn := pd.PulseNumber
	if len(p.plashes) > 0 {
		panic(throw.E("duplicate genesis", struct { PN pulse.Number }{ pn }))
	}

	pa, jets := p.createPlash(pr, jet.NewPrefixTree(true), population)

	switch len(jets) {
	case 0:
		return pa, jet.UnknownExactID
	case 1:
		return pa, jets[0]
	}
	panic(throw.Impossible())
}

