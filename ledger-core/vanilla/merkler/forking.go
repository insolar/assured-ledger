package merkler

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ cryptkit.ForkingDigester = &ForkingCalculator{}

func NewForkingCalculator(digester cryptkit.PairDigester, unbalancedStub cryptkit.Digest) ForkingCalculator {
	if digester == nil {
		panic("illegal value")
	}
	return ForkingCalculator{NewStackedCalculator(digester, unbalancedStub, nil)}
}

type ForkingCalculator struct {
	StackedCalculator
}

func (p *ForkingCalculator) ForkSequence() cryptkit.ForkingDigester {
	cp := p.ForkCalculator()
	return &cp
}

func (p *ForkingCalculator) ForkCalculator() ForkingCalculator {
	if p.finished || p.digester == nil || p.traceFn != nil {
		panic(throw.IllegalState())
	}
	cp := *p
	cp.treeLevels = append(make([]treeLevel, 0, cap(p.treeLevels)), p.treeLevels...)
	return cp
}
