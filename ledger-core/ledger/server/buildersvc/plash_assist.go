// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/ctlsec"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/merkler"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	_ = iota
	plashStarted
	plashPendingPulse
	plashClosed
	plashSummarized
)

type plashAssistant struct {
	pulseData   pulse.Data
	tree        jet.PrefixTree
	population  census.OnlinePopulation
	calc        jetalloc.MaterialAllocationCalculator
	dropAssists map[jet.ID]*dropAssistant
	dirtyReader bundle.DirtyReader
	writer      bundle.Writer

	nextPlash *plashAssistant
	nextReady smsync.BoolConditionalLink

	status atomickit.Uint32
	commit sync.Mutex // LOCK: Spans across methods
	merkle merkler.ForkingCalculator
}

func (p *plashAssistant) setNextPlash(next *plashAssistant) {
	if p.nextPlash != nil {
		panic(throw.IllegalState())
	}
	p.nextPlash = next
	smachine.ApplyAdjustmentAsync(p.nextReady.NewValue(true))
}

// EXTREME LOCK WARNING!
// This method is under locks of: (1) bundle writer, (2) plashAssistant, (3) dropAssistant.
func (p *plashAssistant) _updateMerkle(indices []ledger.DirectoryIndex, digests []cryptkit.Digest) ([]ledger.Ordinal, error) {
	ords := make([]ledger.Ordinal, 0, len(indices))
	for i, ord := range indices {
		if ord.SectionID() != ledger.DefaultEntrySection {
			continue
		}
		ords = append(ords, ledger.Ordinal(p.merkle.Count()))
		p.merkle.AddNext(digests[i])
	}
	return ords, nil
}

func (p *plashAssistant) PreparePulseChange(outFn conveyor.PreparePulseCallbackFunc) {
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
		if outFn == nil {
			return
		}

		state := conveyor.PreparedState{}
		state.NodeState = root
		outFn(state)
	}()
}

func (p *plashAssistant) CancelPulseChange() {
	if !p.status.CompareAndSwap(plashPendingPulse, plashStarted) {
		panic(throw.IllegalState())
	}
	p.commit.Unlock()
}

func (p *plashAssistant) CommitPulseChange() {
	// plash will not accept writes anymore
	if !p.status.CompareAndSwap(plashPendingPulse, plashClosed) {
		panic(throw.IllegalState())
	}

	p.commit.Unlock()
	// NB! Underlying writer must NOT be marked read-only here as summary has to be written also

	go p.startSummary()
}

func (p *plashAssistant) getDropAssist(id jet.DropID) (*dropAssistant, error) {
	switch assist, ok := p.dropAssists[id.ID()]; {
	case !ok:
		return nil, throw.E("unknown drop", struct{ jet.DropID }{id})
	case assist == nil:
		return nil, throw.E("drop is not local", struct{ jet.DropID }{id})
	default:
		return assist, nil
	}
}

func (p *plashAssistant) appendToDrop(id jet.DropID, future AppendFuture, bundle lineage.UpdateBundle) error {
	assist, err := p.getDropAssist(id)
	if err != nil {
		return err
	}
	if err = p.waitNotPending(); err != nil {
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
		case 0:
			return throw.E("plash not ready")
		default:
			return throw.E("plash closed")
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
	case 0:
		return throw.E("plash not ready")
	default:
		return throw.E("plash closed")
	}

	return fn()
}

func (p *plashAssistant) IsGenesis() bool {
	return p.tree.IsEmpty()
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
	switch da, ok := p.dropAssists[jetID]; {
	case !ok:
		panic(throw.Impossible())
	case da != nil:
		return da.dropID
	default:
		return 0
	}
}

func (p *plashAssistant) GetNextReadySync() smachine.SyncLink {
	return p.nextReady.SyncLink()
}

func (p *plashAssistant) getNextPlash() *plashAssistant {
	if pa := p.nextPlash; pa != nil {
		return pa
	}
	panic(throw.IllegalState())
}

func (p *plashAssistant) CalculateNextDrops(id jet.DropID) []jet.DropID {
	assist, err := p.getDropAssist(id)
	if err != nil {
		panic(err)
	}
	return p.getNextPlash().getDropsOfJet(assist.exactID)
}

func (p *plashAssistant) getDropsOfJet(id jet.ExactID) (result []jet.DropID) {
	prefix, pLen := p.tree.GetPrefix(id.ID().AsPrefix())

	pn := p.pulseData.PulseNumber

	if pLen <= id.BitLen() {
		// jet was merged or remains same - there will be only one descendant
		result = []jet.DropID{ prefix.AsID().AsDrop(pn) }
	} else {
		// jet was split. NB! Genesis can split into more than 2 jets.
		splitDepth := pLen - id.BitLen()
		if splitDepth > 16 {
			panic(throw.Impossible())
		}

		splitPrefix := jet.Prefix(1<<splitDepth)
		result = make([]jet.DropID, splitPrefix)

		// generate ids for all sub-jets
		for splitPrefix > 0 {
			splitPrefix--
			subJetPrefix := (splitPrefix<<pLen)|prefix
			result[splitPrefix] = subJetPrefix.AsID().AsDrop(pn)
		}
	}

	// double check
	for _, subID := range result {
		if _, err := p.getDropAssist(subID); err != nil {
			panic(err)
		}
	}

	return result
}

func (p *plashAssistant) appendToDropSummary(id jet.DropID, summary lineage.LineSummary) error {
	return nil
}

func (p *plashAssistant) finalizeDropSummary(id jet.DropID) (catalog.DropReport, error) {
	return catalog.DropReport{}, nil
}

func (p *plashAssistant) startSummary() {
	sw := &ctlsec.PlashSummaryWriter{}
	sw.ReadCatalog(p.dirtyReader, len(p.dropAssists))

	_ = p.writer.WriteBundle(sw, nil)
}

func (p *plashAssistant) finalizeSummary()  {
	// // underlying writer will be marked read only as soon as all writing bundles are completed / discarded
	// // but CommitPulseChange will be released immediately after putting the closure into the write chain
	// // and any further calls to WaitWriteBundles() or to WriteBundle() will wait for this closure to complete.
	// p.writer.WaitWriteBundlesAsync(nil, func(bool) {
	// 	if err := p.writer.MarkReadOnly(); err != nil {
	// 		panic(throw.W(err, "failed to mark storage as read-only"))
	// 	}
	// })
}

