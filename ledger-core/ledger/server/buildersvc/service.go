// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
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
	storageFactoryFn StorageSnapshotFactoryFunc, plashLimit int,
) *serviceImpl {
	if plashLimit <= 0 {
		panic(throw.IllegalValue())
	}

	return &serviceImpl{
		allocStrategy: allocStrategy,
		merklePair: merklePair,
		storageFactoryFn: storageFactoryFn,
		plashLimiter: make(chan pulse.Number, plashLimit),
	}
}

type serviceImpl struct {
	allocStrategy jetalloc.MaterialAllocationStrategy
	merklePair    cryptkit.PairDigester

	storageFactoryFn StorageSnapshotFactoryFunc

	mapMutex sync.RWMutex
	lastPN   pulse.Number
	plashes  map[pulse.Number]*plashAssistant
	plashLimiter chan pulse.Number
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

func (p *serviceImpl) AppendToDropSummary(id jet.DropID, summary lineage.LineSummary) {

}

func (p *serviceImpl) FinalizeDropSummary(jet.DropID) catalog.DropReport {
	return catalog.DropReport{}
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
		nextReady: smsync.NewConditionalBool(false, "plash.nextReady"),
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
	pa.writer = bw

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
				exactID: jetPID,
				dropID: jetPID.AsDrop(pn),
				writer: bw,
			}
			if len(jets) > 1 {
				da.merkle = pa.merkle.ForkSequence()
			}
		}
		pa.dropAssists[jetID] = da
	}

	// TODO write down shared data?

	var prevPlash *plashAssistant

	switch {
	case p.plashes == nil:
		p.plashes = map[pulse.Number]*plashAssistant{}
	case p.lastPN.IsUnknown():
	default:
		if prevPlash = p.plashes[p.lastPN]; prevPlash == nil {
			panic(throw.Impossible())
		}
	}

	p.addAndExpire(pn)
	p.plashes[pn] = pa
	p.lastPN = pn
	pa.status.Store(plashStarted)

	if prevPlash != nil {
		go prevPlash.setNextPlash(pa)
	}

	return pa, result
}

func (p *serviceImpl) addAndExpire(pn pulse.Number) {
	select {
	case p.plashLimiter <- pn:
		return
	default:
	}

	expiredPN := <- p.plashLimiter
	p.plashLimiter <- pn

	delete(p.plashes, expiredPN)
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

