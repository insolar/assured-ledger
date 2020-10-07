// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/merkler"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Service = &serviceImpl{}
var _ ReadService = &serviceImpl{}

func newService(allocStrategy jetalloc.MaterialAllocationStrategy, merklePair cryptkit.PairDigester,
	storageFactory StorageFactory, plashLimit int,
) *serviceImpl {
	switch {
	case plashLimit <= 0:
		panic(throw.IllegalValue())
	case storageFactory == nil:
		panic(throw.IllegalValue())
	}

	return &serviceImpl{
		allocStrategy:  allocStrategy,
		merklePair:     merklePair,
		storageFactory: storageFactory,
		plashLimiter:   make(chan pulse.Number, plashLimit),
	}
}

type serviceImpl struct {
	allocStrategy jetalloc.MaterialAllocationStrategy
	merklePair    cryptkit.PairDigester

	storageFactory StorageFactory

	mapMutex sync.RWMutex
	lastPN   pulse.Number
	plashes  map[pulse.Number]*plashAssistant
	plashLimiter chan pulse.Number
}

func (p *serviceImpl) DropReadDirty(id jet.DropID, fn func(reader bundle.DirtyReader) error) error {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	pa, err := p.getPlashAssist(id)
	if err != nil {
		return err
	}
	if _, err := pa.getDropAssist(id); err != nil {
		return err
	}

	return fn(pa.dirtyReader)
}

func (p *serviceImpl) getPlashAssist(id jet.DropID) (*plashAssistant, error) {
	pa := p.get(id.CreatedAt())
	if pa == nil {
		return nil, throw.E("unknown plash", struct{ jet.DropID }{id})
	}
	return pa, nil
}

func (p *serviceImpl) AppendToDrop(id jet.DropID, future AppendFuture, bundle lineage.UpdateBundle) {
	pa, err := p.getPlashAssist(id)

	if err == nil {
		err = pa.appendToDrop(id, future, bundle)
		if err == nil {
			return
		}
	}

	if future != nil {
		future.TrySetFutureResult(nil, err)
	}
}

func (p *serviceImpl) AppendToDropSummary(id jet.DropID, summary lineage.LineSummary) {
	pa, err := p.getPlashAssist(id)
	if err == nil {
		if err = pa.appendToDropSummary(id, summary); err == nil {
			return
		}
	}
	panic(err)
}

func (p *serviceImpl) FinalizeDropSummary(id jet.DropID) (report catalog.DropReport) {
	pa, err := p.getPlashAssist(id)
	if err == nil {
		if report, err = pa.finalizeDropSummary(id); err == nil {
			return report
		}
	}
	panic(err)
}

func (p *serviceImpl) get(pn pulse.Number) *plashAssistant {
	p.mapMutex.RLock()
	defer p.mapMutex.RUnlock()
	return p.plashes[pn]
}

func (p *serviceImpl) CreatePlash(cfg BasicPlashConfig, treePrev, tree jet.Tree) (PlashAssistant, conveyor.PulseChanger, []jet.ExactID) {
	if tree == nil || tree.IsEmpty() {
		// missing value or genesis tree
		panic(throw.IllegalValue())
	}

	pd := cfg.PulseRange.RightBoundData()
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

	return p.createPlash(cfg, *tree)
}

func (p *serviceImpl) createPlash(cfg BasicPlashConfig, tree jet.PrefixTree) (PlashAssistant, conveyor.PulseChanger, []jet.ExactID) {
	localNodeID := cfg.Population.GetLocalProfile().GetNodeID()
	if localNodeID.IsAbsent() {
		panic(throw.IllegalValue())
	}

	pd := cfg.PulseRange.RightBoundData()
	pd.EnsurePulsarData()
	pn := pd.PulseNumber

	pa := &plashAssistant{
		pulseData:   pd,
		tree:        tree,
		population:  cfg.Population,
		callbackFn:  cfg.CallbackFn,
		dropAssists: map[jet.ID]*dropAssistant{},
		merkle:      merkler.NewForkingCalculator(p.merklePair, cryptkit.Digest{}),
		nextReady:   smsync.NewConditionalBool(false, "plash.nextReady"),
	}

	{
		// TODO temporary hack
		b := make([]byte, p.merklePair.GetDigestSize())
		pa.merkle.AddNext(longbits.WrapBytes(b))
	}

	pa.tree.SetPropagate() // grants O(1) to find jet

	jetCount := pa.tree.Count()
	jets := make([]jet.ExactID, 0, jetCount)

	pa.tree.Enum(func(prefix jet.Prefix, depth uint8) bool {
		jets = append(jets, jet.ID(prefix).AsExact(depth))
		return false
	})

	pa.calc = p.allocStrategy.CreateCalculator(pa.pulseData.PulseEntropy, cfg.Population)
	jet2nodes := pa.calc.AllocationOfJets(jets, pn)

	if len(jet2nodes) != len(jets) {
		panic(throw.IllegalState())
	}

	sw := p.storageFactory.CreateSnapshotWriter(pn, ledger.DefaultDustSection)
	pa.dirtyReader = sw.DirtyReader()
	bw := bundle.NewWriter(sw) // NB! MUST be one writer per storage
	pa.writer = bw
	pa.writerCloseFn = func() error {
		return p.storageFactory.DepositReadOnlyWriter(sw)
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

	pa.init(len(result))

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
	if err := pa.writeStartAndSharedData(); err != nil {
		panic(err)
	}

	// TODO isolate PlashAssistant from conveyor.PulseChanger
	return pa, pa, result
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

func (p *serviceImpl) CreateGenesis(cfg BasicPlashConfig) (PlashAssistant, conveyor.PulseChanger, jet.ExactID) {
	pd := cfg.PulseRange.RightBoundData()
	pd.EnsurePulsarData()

	p.mapMutex.Lock()
	defer p.mapMutex.Unlock()

	pn := pd.PulseNumber
	if len(p.plashes) > 0 {
		panic(throw.E("duplicate genesis", struct { PN pulse.Number }{ pn }))
	}

	pa, pc, jets := p.createPlash(cfg, jet.NewPrefixTree(true))

	switch len(jets) {
	case 0:
		return pa, pc, jet.UnknownExactID
	case 1:
		return pa, pc, jets[0]
	}
	panic(throw.Impossible())
}

