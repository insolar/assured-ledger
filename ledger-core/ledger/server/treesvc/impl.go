// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package treesvc

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Service = &treeSvc{}
// var _ managed.ComponentWithPulse = &treeSvc{}
type treeSvc struct {
	trees [3]jet.PrefixTree
	treeIdx uint8
}

// func (p *treeSvc) Init(holder managed.Holder) {
// 	panic("implement me")
// }
//
// func (p *treeSvc) Start(holder managed.Holder) {
// 	panic("implement me")
// }
//
// func (p *treeSvc) Stop(holder managed.Holder) {
// 	panic("implement me")
// }

func (p *treeSvc) pulseMigration(holder managed.Holder, p2 pulse.Range) {
	panic("implement me")
}

func (p *treeSvc) GetTrees() (prev, cur jet.PrefixTree) {
	panic("implement me")
}

func (p *treeSvc) advance() {
	p.treeIdx++
	if p.treeIdx == uint8(len(p.trees)) {
		p.treeIdx = 0
	}
}

func (p *treeSvc) isGenesis() bool {
	return p.treeIdx == 0 && p.trees[1].IsEmpty()
}

func (p *treeSvc) FinishGenesis(depth uint8, afterPulse pulse.Number) {
	if !p.isGenesis() {
		panic(throw.IllegalState())
	}
	p.trees[2].MakePerfect(depth)
	p.treeIdx++
}

func (p *treeSvc) SplitNext(id jet.DropID) {
	panic("implement me")
}
