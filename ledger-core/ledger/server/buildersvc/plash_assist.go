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
)

type plashAssistant struct {
	pulseData   pulse.Data
	tree        jet.PrefixTree
	population  census.OnlinePopulation
	calc        jetalloc.MaterialAllocationCalculator
	dirtyReader bundle.DirtyReader
	dropAssists map[jet.ID]*dropAssistant

	nextPlash   *plashAssistant
	nextReady   smsync.BoolConditionalLink

	status    atomickit.Uint32
	commit    sync.Mutex // LOCK: Spans across methods
	merkle    merkler.ForkingCalculator
}

func (p *plashAssistant) setNextPlash(next *plashAssistant) {
	if p.nextPlash != nil {
		panic(throw.IllegalState())
	}
	p.nextPlash = next
	smachine.ApplyAdjustmentAsync(p.nextReady.NewValue(true))
}

func (p *plashAssistant) GetNextPlashReadySync() smachine.SyncLink {
	return p.nextReady.SyncLink()
}

func (p *plashAssistant) GetNextPlash() PlashAssistant {
	if pa := p.nextPlash; pa != nil {
		return pa
	}
	panic(throw.IllegalState())
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
	if !p.status.CompareAndSwap(plashPendingPulse, plashClosed) {
		panic(throw.IllegalState())
	}
	// TODO mark storage(s) as readonly?
	p.commit.Unlock()
}

func (p *plashAssistant) appendToDrop(id jet.DropID, future AppendFuture, bundle lineage.UpdateBundle) error {
	assist, ok := p.dropAssists[id.ID()]
	switch {
	case !ok:
		return throw.E("unknown drop", struct { jet.DropID }{ id })
	case assist == nil:
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

// EXTREME LOCK WARNING!
// This method is under locks of: (1) DropWriter, (2) plashAssistant, (3) dropAssistant.
//nolint:unparam
func (p *plashAssistant) _updateMerkle(_ jet.DropID, indices []ledger.DirectoryIndex, digests []cryptkit.Digest) ([]ledger.Ordinal, error) {
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
