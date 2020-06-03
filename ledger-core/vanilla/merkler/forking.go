// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package merkler

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"

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
	if p.finished || p.digester == nil || p.traceFn != nil {
		panic("illegal state")
	}

	cp := *p
	cp.treeLevels = append(make([]treeLevel, 0, cap(p.treeLevels)), p.treeLevels...)
	return &cp
}
