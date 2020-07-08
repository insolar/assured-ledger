// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetbalancer"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dropstorage/cabinet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PlashAssistant interface {
	PreparePulseChange(/* callback */)
	CancelPulseChange()
	CommitPulseChange()

	CalculateJetDrop(reference.Holder) jet.DropID
	GetResolver() lineage.DependencyResolver
}

type AppendFuture interface {
	TrySetFutureResult(allocations []catalog.DirectoryIndex, err error) bool
}

type Service interface {
	CreatePlash(pulse.Range, jet.Tree, census.OnlinePopulation) (PlashAssistant, []jet.ExactID)
	AppendToDrop(jet.DropID, AppendFuture, lineage.ResolvedBundle)
}

var _ Service = &serviceImpl{}

func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {
	allocationStrategy jetbalancer.MaterialAllocationStrategy

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
	if population == nil {
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

	pa.calc = p.allocationStrategy.CreateCalculator(pa.pulseData.PulseEntropy, population)
	jet2nodes := pa.calc.AllocationOfJets(jets, pn)

	if len(jet2nodes) != len(jets) {
		panic(throw.IllegalState())
	}

	localNodeID := node.ShortNodeID(0)
	var nodeMap map[node.ShortNodeID]*dropAssistant

	result := jets[:0]

	for _, jetPID := range jets {

		jetID := jetPID.ID()
		var da *dropAssistant

		switch assignedNodeID := jet2nodes[jetID]; {
		case localNodeID.IsAbsent():
			panic(throw.IllegalState())

		case localNodeID == assignedNodeID:
			result = append(result, jetPID)
			da = &dropAssistant{}
			da.nodeID = assignedNodeID
			da.dropID = jetPID.AsDrop(pn)

		case nodeMap == nil:
			nodeMap = map[node.ShortNodeID]*dropAssistant{}
			fallthrough
		default:
			if da = nodeMap[assignedNodeID]; da != nil {
				break
			}
			da = &dropAssistant{}
			da.nodeID = assignedNodeID
			nodeMap[assignedNodeID] = da
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

const (
	_ = iota
	plashStarted
	plashPendingPulse
	plashClosed
)

type plashAssistant struct {
	pulseData   pulse.Data
	tree        jet.PrefixTree
	population  census.OnlinePopulation
	calc        jetbalancer.MaterialAllocationCalculator
	dropAssists map[jet.ID]*dropAssistant

	status      atomickit.Uint32
	commit      sync.Mutex // LOCK! Spans across methods
}

func (p *plashAssistant) PreparePulseChange() {
	if p.status.Load() != plashStarted { // NB! Makes consecutive calls to fail, not to block
		panic(throw.IllegalState())
	}
	p.commit.Lock()
	if !p.status.CompareAndSwap(plashStarted, plashPendingPulse) {
		p.commit.Unlock()
		panic(throw.Impossible()) // race with multiple PreparePulseChange() calls?
	}
}

func (p *plashAssistant) CancelPulseChange() {
	if !p.status.CompareAndSwap(plashPendingPulse, plashStarted) {
		panic(throw.IllegalState())
	}
	p.commit.Unlock()
}

func (p *plashAssistant) CommitPulseChange() {
	if !p.status.CompareAndSwap(plashPendingPulse, plashClosed) {
		panic(throw.IllegalState())
	}
	p.commit.Unlock()
}

func (p *plashAssistant) appendToDrop(id jet.DropID, future AppendFuture, bundle lineage.ResolvedBundle) error {
	assist := p.dropAssists[id.ID()]
	switch {
	case assist == nil:
		return throw.E("unknown drop", struct { jet.DropID }{ id })
	case !assist.isLocal():
		return throw.E("drop is not local", struct { jet.DropID }{ id })
	}

	if err := p.waitNotPending(); err != nil {
		return throw.WithDetails(err, struct{ jet.DropID }{id})
	}
	return assist.append(p, future, bundle)
}

func (p *plashAssistant) waitNotPending() error {
	for {
		switch p.status.Load() {
		case plashStarted:
			return nil
		case plashPendingPulse:
			p.commit.Lock()
			// NB! blocks until pending is finished
			p.commit.Unlock()
		case plashClosed:
			return throw.E("plash closed")
		default:
			return throw.E("plash not ready")
		}
	}
}

func (p *plashAssistant) commitDropUpdate(fn func() error) error {
	p.commit.Lock()
	defer p.commit.Unlock()

	switch p.status.Load() {
	case plashStarted:
		//
	case plashPendingPulse:
		panic(throw.Impossible())
	case plashClosed:
		return throw.E("plash closed")
	default:
		return throw.E("plash not ready")
	}

	return fn()
}

func (p *plashAssistant) GetResolver() lineage.DependencyResolver {
	panic("implement me")
}

func (p *plashAssistant) CalculateJetDrop(holder reference.Holder) jet.DropID {
	switch {
	case reference.IsEmpty(holder):
		panic(throw.IllegalValue())
	case len(p.dropAssists) == 0:
		panic(throw.IllegalValue())
	}

	base := holder.GetBase()
	jetPrefix := p.calc.AllocationOfLine(base)
	jetPrefix, _ = p.tree.GetPrefix(jetPrefix)

	jetID := jetPrefix.AsID()
	switch da := p.dropAssists[jetID]; {
	case da == nil:
		panic(throw.Impossible())
	case da.isLocal():
		return jetID.AsDrop(p.pulseData.PulseNumber)
	}

	return 0
}


// TODO a configuration set for conveyor that provides adapters and input-SM mapper

const (
	_ = iota
	dropStarted

)

type dropAssistant struct {
	mutex   sync.Mutex // LOCK! Is used under plashAssistant.commit lock
	status  atomickit.Uint32 // Started, PulsePending, Closed
	nodeID  node.ShortNodeID
	dropID  jet.DropID
	writer  cabinet.DropWriter
}

func (p *dropAssistant) isLocal() bool {
	return p.dropID != 0
}

func (p *dropAssistant) append(pa *plashAssistant, future AppendFuture, bundle lineage.ResolvedBundle) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	writeBundle := make([]cabinet.WriteBundleEntry, 0, bundle.Count())

	bundle.Enum(func(record lineage.Record, dust lineage.DustMode) bool {
		br := record.AsBasicRecord()
		recPayloads := br.GetRecordPayloads()
		payloadCount := recPayloads.Count()

		bundleEntry := cabinet.WriteBundleEntry{
			Directory: catalog.DefaultEntrySection, // todo depends on record policy
			EntryKey:  record.GetRecordRef(),
			Payloads:  make([]cabinet.SectionPayload, 1+payloadCount),
			Entry: func(index catalog.DirectoryIndex, payloadLoc []catalog.StorageLocator) cabinet.MarshalerTo {
				return createCatalogEntry(index, payloadLoc, &record)
			},
		}

		if dust == lineage.DustRecord {
			bundleEntry.Directory = catalog.DefaultDustSection
		}

		bundleEntry.Payloads[0].Section = bundleEntry.Directory
		bundleEntry.Payloads[0].Payload = br.(cabinet.MarshalerTo)

		if payloadCount > 0 {
			if dust >= lineage.DustPayload {
				bundleEntry.Payloads[1].Section = catalog.DefaultDustSection
			} else {
				bundleEntry.Payloads[1].Section = catalog.DefaultDataSection
			}
			bundleEntry.Payloads[1].Payload = recPayloads.GetPayloadOrExtension(0)

			for i := 2; i <= payloadCount; i++ {
				secID := bundleEntry.Directory
				if extID := catalog.ExtensionID(recPayloads.GetExtensionID(i - 1)); extID != catalog.SameAsBodyExtensionID {
					secID = p.extensionToSection(extID, secID)
				}
				bundleEntry.Payloads[i].Section = secID
				bundleEntry.Payloads[i].Payload = recPayloads.GetPayloadOrExtension(i - 1)
			}
		}
		writeBundle = append(writeBundle, bundleEntry)
		return false
	})

	p.writer.WriteBundle(writeBundle, func(indices []catalog.DirectoryIndex) {

	})

	// return pa.commitDropUpdate(func() error {
	// 	batch.CommitBatch()
	// })
	panic(throw.NotImplemented())
}

func (p *dropAssistant) extensionToSection(_ catalog.ExtensionID, defSecID catalog.SectionID) catalog.SectionID {
	return defSecID
}

func createCatalogEntry(catalog.DirectoryIndex, []catalog.StorageLocator, *lineage.Record) *catalog.Entry {
	entry := &catalog.Entry{}

	return entry
}
