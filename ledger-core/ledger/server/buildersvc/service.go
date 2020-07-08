// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dropstorage/cabinet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
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

func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {
	allocationStrategy jetalloc.MaterialAllocationStrategy

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
			if len(jets) > 1 {
				da.merkle = pa.merkle.ForkSequence()
			}
			// da.writer = ; // TODO writer factory

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
	calc        jetalloc.MaterialAllocationCalculator
	dropAssists map[jet.ID]*dropAssistant

	status   atomickit.Uint32
	commit   sync.Mutex // LOCK! Spans across methods
	merkle   merkler.ForkingCalculator
}

func (p *plashAssistant) PreparePulseChange(out chan<- cryptkit.Digest) {
	if p.status.Load() != plashStarted { // NB! Makes consecutive calls to fail, not to block
		panic(throw.IllegalState())
	}
	p.commit.Lock()
	if !p.status.CompareAndSwap(plashStarted, plashPendingPulse) {
		p.commit.Unlock()
		panic(throw.Impossible()) // race with multiple PreparePulseChange() calls?
	}

	forked := p.merkle.ForkCalculator()

	go func() {
		root := forked.FinishSequence()
		if out != nil {
			out <- root
		}
	}()
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

// EXTREME LOCK WARNING!
// This method is under locks of: (1) DropWriter, (2) plashAssistant, (3) dropAssistant.
func (p *plashAssistant) _updateMerkle(_ jet.DropID, indices []ledger.DirectoryIndex, digests []cryptkit.Digest) ([]ledger.Ordinal, error) {
	for i, ord := range indices {
		if ord.SectionID() != ledger.DefaultEntrySection {
			continue
		}
		p.merkle.AddNext(digests[i])
	}
	// TODO ordinals positions
	return nil, nil
}

// TODO a configuration set for conveyor that provides adapters and input-SM mapper

type dropAssistant struct {
	nodeID  node.ShortNodeID
	dropID  jet.DropID
	writer  cabinet.DropWriter

	mutex   sync.Mutex // LOCK! Is used under plashAssistant.commit lock
	merkle  cryptkit.ForkingDigester
}

func (p *dropAssistant) isLocal() bool {
	return p.dropID != 0
}

func (p *dropAssistant) append(pa *plashAssistant, future AppendFuture, bundle lineage.ResolvedBundle) error {
	writeBundle := make([]cabinet.WriteBundleEntry, 0, bundle.Count())
	digests := make([]cryptkit.Digest, 0, bundle.Count())

	bundle.Enum(func(record lineage.Record, dust lineage.DustMode) bool {
		br := record.AsBasicRecord()
		recPayloads := br.GetRecordPayloads()
		payloadCount := recPayloads.Count()

		bundleEntry := cabinet.WriteBundleEntry{
			Directory: ledger.DefaultEntrySection, // todo depends on record policy
			EntryKey:  record.GetRecordRef(),
			Payloads:  make([]cabinet.SectionPayload, 1+payloadCount),
		}

		if dust == lineage.DustRecord {
			bundleEntry.Directory = ledger.DefaultDustSection
		}

		bundleEntry.Payloads[0].Section = bundleEntry.Directory
		bundleEntry.Payloads[0].Payload = br.(cabinet.MarshalerTo)

		if payloadCount > 0 {
			if dust >= lineage.DustPayload {
				bundleEntry.Payloads[1].Section = ledger.DefaultDustSection
			} else {
				bundleEntry.Payloads[1].Section = ledger.DefaultDataSection
			}
			bundleEntry.Payloads[1].Payload = recPayloads.GetPayloadOrExtension(0)

			for i := 2; i <= payloadCount; i++ {
				secID := bundleEntry.Directory
				extID := ledger.ExtensionID(recPayloads.GetExtensionID(i - 1))
				if extID != ledger.SameAsBodyExtensionID {
					secID = p.extensionToSection(extID, secID)
				}
				bundleEntry.Payloads[i].Extension = extID
				bundleEntry.Payloads[i].Section = secID
				bundleEntry.Payloads[i].Payload = recPayloads.GetPayloadOrExtension(i - 1)
			}
		}

		bundleEntry.EntryFn = func(index ledger.DirectoryIndex, payloadLoc []ledger.StorageLocator) cabinet.MarshalerTo {
			entry := createCatalogEntry(index, payloadLoc, bundleEntry.Payloads, &record)
			return entry
		}

		digests = append(digests, record.RegistrarSignature.GetDigest())
		writeBundle = append(writeBundle, bundleEntry)
		return false
	})

	p.writer.WriteBundle(writeBundle, func(indices []ledger.DirectoryIndex) bool {
		// this closure is called later, after the bundle is completely written
		err := pa.commitDropUpdate(func() error {
			// EXTREME LOCK WARNING!
			// This section is under locks of: (1) DropWriter, (2) plashAssistant, and acquires (3) dropAssistant.
			return p.bundleProcessedByWriter(pa, indices, digests)
		})

		switch {
		case err != nil:
			future.TrySetFutureResult(nil, err)
			return false
		case future.TrySetFutureResult(indices, nil):
			return true
		default:
			panic(throw.Impossible())
		}
	})

	return nil
}

// EXTREME LOCK WARNING!
// This method is under locks of: (1) DropWriter, (2) plashAssistant, (3) dropAssistant.
func (p *dropAssistant) bundleProcessedByWriter(pa *plashAssistant, indices []ledger.DirectoryIndex, digests []cryptkit.Digest) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	positions, err := pa._updateMerkle(p.dropID, indices, digests)
	if err != nil {
		return err
	}
	p._updateMerkle(positions, indices, digests)
	return nil
}

// EXTREME LOCK WARNING!
// This method is under locks of: (1) DropWriter, (2) plashAssistant, (3) dropAssistant.
func (p *dropAssistant) _updateMerkle(_ []ledger.Ordinal, indices []ledger.DirectoryIndex, digests []cryptkit.Digest) {
	if p.merkle == nil {
		// there is only one drop in plash, so there is no need for a secondary merkle
		return
	}

	for i, ord := range indices {
		if ord.SectionID() != ledger.DefaultEntrySection {
			continue
		}
		p.merkle.AddNext(digests[i])
		// TODO pre-calculate sparse merkle proof for this drop by plash
	}
}

func (p *dropAssistant) extensionToSection(_ ledger.ExtensionID, defSecID ledger.SectionID) ledger.SectionID {
	// TODO extension mapping
	return defSecID
}

func createCatalogEntry(idx ledger.DirectoryIndex, loc []ledger.StorageLocator, payloads []cabinet.SectionPayload, rec *lineage.Record) *catalog.Entry {
	entry := &catalog.Entry{
		RecordType:         rec.Excerpt.RecordType,
		BodyLoc:            loc[0],
		RecordBodyHash:     rec.Excerpt.RecordBodyHash,
		Ordinal: 			idx.Ordinal(),
		PrevRef:			rec.Excerpt.PrevRef,
		RootRef:			rec.Excerpt.RootRef,
		ReasonRef:			rec.Excerpt.ReasonRef,
		RedirectRef:		rec.Excerpt.RedirectRef,
		RejoinRef:			rec.Excerpt.RejoinRef,
		RecapRef: 			rms.NewReference(rec.RecapRef),

		ProducerSignature:  rec.Excerpt.ProducerSignature,
		ProducedBy:         rms.NewReference(rec.ProducedBy),

		RegistrarSignature: rms.NewRaw(rec.RegistrarSignature.GetSignature()).AsBinary(),
		RegisteredBy:       rms.NewReference(rec.RegisteredBy),
	}

	if n := len(loc); n > 1 {
		entry.PayloadLoc = loc[1]
		if n > 2 {
			entry.ExtensionLoc.Ext = make([]rms.ExtLocator, n - 2)
			for i := 2; i < n; i++ {
				entry.ExtensionLoc.Ext[i - 2] = rms.ExtLocator{
					ExtensionID: payloads[i].Extension,
					PayloadLoc:  loc[i],
				}
			}
		}
	}

	return entry
}
