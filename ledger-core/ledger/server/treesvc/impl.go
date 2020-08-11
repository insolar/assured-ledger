// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package treesvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewEmpty() *LocalTree {
	return &LocalTree{}
}

func New(tree jet.PrefixTree, pn pulse.Number) *LocalTree {
	if tree.IsEmpty() != pn.IsUnknown() {
		panic(throw.IllegalValue())
	}

	return &LocalTree{
		treePrev: tree,
		treeCurr: tree,
		treeNext: tree,
		last: pn,
	}
}

var _ Service = &LocalTree{}

var _ managed.ComponentWithPulse = &LocalTree{}
type LocalTree struct {
	mutex sync.RWMutex

	treePrev jet.PrefixTree
	treeCurr jet.PrefixTree
	treeNext jet.PrefixTree
	last     pulse.Number
}

func (p *LocalTree) Init(managed.Holder) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.treePrev.Init()
	p.treeCurr.Init()
	p.treeNext.Init()
}

func (p *LocalTree) Start(managed.Holder) {
}

func (p *LocalTree) Stop(managed.Holder) {
}

func (p *LocalTree) PulseMigration(_ managed.Holder, pr pulse.Range) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	pn := pr.RightBoundData().PulseNumber

	if p.isGenesis() {
		// genesis can only be switched explicitly
		return
	}
	p.advance(pn)
}

func (p *LocalTree) advance(pn pulse.Number) {
	p.last = pn
	p.treePrev, p.treeCurr = p.treeCurr, p.treeNext
}

func (p *LocalTree) GetTrees(pn pulse.Number) (prev, cur jet.PrefixTree, ok bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.isGenesis() || p.last.IsUnknownOrEqualTo(pn) {
		return p.treePrev, p.treeCurr, true
	}

	return jet.PrefixTree{}, jet.PrefixTree{}, false
}

func (p *LocalTree) isGenesis() bool {
	return p.treeCurr.IsEmpty()
}

func (p *LocalTree) FinishGenesis(depth uint8, lastGenesisPulse pulse.Number) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case !lastGenesisPulse.IsTimePulse():
		panic(throw.IllegalValue())
	case !p.isGenesis():
		panic(throw.IllegalState())
	}
	p.treeNext.MakePerfect(depth)
	p.advance(lastGenesisPulse)
}

func (p *LocalTree) SplitNext(id jet.DropID) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case p.treeCurr.IsEmpty():
		panic(throw.IllegalState())
	case id.CreatedAt() != p.last: // make sure that changes are actual
		panic(throw.IllegalValue())
	}
	pfx, pln := p.treeCurr.GetPrefix(id.ID().AsPrefix()) // get current prefix of the jet
	p.treeNext.Split(pfx, pln) // this can only be applied once - guarantees that jet can only be split once per pulse
}

