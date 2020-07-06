// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetbalancer"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PlashAssistant interface {
	CalculateJetDrop(reference.Holder) jet.DropID
	CreateJetDropAssistant(id jet.ID) JetDropAssistant
}

type AddFuture interface {
	TrySetCommitted(committed bool, allocatedBase uint32) bool
}

type JetDropAssistant interface {
	AddRecords(future AddFuture, br *lineage.BundleResolver) bool
	GetResolver() lineage.DependencyResolver
}

type Service interface {
	CreatePlash(pulse.Range, jet.Tree, census.OnlinePopulation) (PlashAssistant, []jet.PrefixedID)
}

var _ Service = &serviceImpl{}

func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {
	allocationStrategy jetbalancer.MaterialAllocationStrategy
}

func (p *serviceImpl) CreatePlash(pr pulse.Range, tree jet.Tree, population census.OnlinePopulation) (PlashAssistant, []jet.PrefixedID) {
	if population == nil {
		panic(throw.IllegalValue())
	}

	pa := &plashAssistant{
		pulseData: pr.RightBoundData(),
		population: population,
		dropAssists: map[jet.ID]*dropAssistant{},
	}

	pa.pulseData.EnsurePulsarData()

	if tree == nil || tree.IsEmpty() {
		pa.tree.Init()

		// TODO genesis
		return pa, []jet.PrefixedID{jet.GenesisPrefixedID}
	}

	pa.tree = *tree
	pa.tree.SetPropagate() // grants O(1) to find jet

	jetCount := pa.tree.Count()
	jets := make([]jet.PrefixedID, 0, jetCount)

	pa.tree.Enum(func(prefix jet.Prefix, depth uint8) bool {
		jets = append(jets, jet.ID(prefix).AsPrefixed(depth))
		return false
	})

	pa.calc = p.allocationStrategy.CreateCalculator(pa.pulseData.PulseEntropy, population)
	jet2nodes := pa.calc.AllocationOfJets(jets)

	localNodeID := node.ShortNodeID(0)
	var nodeMap map[node.ShortNodeID]*dropAssistant

	result := jets[:0]

	for jetPID, assignedNodeID := range jet2nodes {
		var da *dropAssistant

		switch {
		case localNodeID == assignedNodeID:
			result = append(result, jetPID)
			da = &dropAssistant{}
			da.nodeID = assignedNodeID
			da.dropID = jetPID
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
		pa.dropAssists[jetPID.ID()] = da
	}

	return pa, result
}

type plashAssistant struct {
	pulseData  pulse.Data
	tree       jet.PrefixTree
	population census.OnlinePopulation
	calc       jetbalancer.MaterialAllocationCalculator
	dropAssists map[jet.ID]*dropAssistant
}

func (p *plashAssistant) CalculateJetDrop(holder reference.Holder) jet.DropID {
	switch {
	case reference.IsEmpty(holder):
		panic(throw.IllegalValue())
	case len(p.dropAssists) == 0:
		panic(throw.IllegalValue())
	}

	base := holder.GetBase()
	dropID := p.calc.AllocationOfLine(base)

	switch da := p.dropAssists[dropID.ID()]; {
	case da == nil:
		panic(throw.Impossible())
	case dropID.CreatedAt() != p.pulseData.PulseNumber:
		panic(throw.Impossible())
	case da.IsLocal():
		return dropID
	}

	return 0
}

func (p *plashAssistant) CreateJetDropAssistant(id jet.ID) JetDropAssistant {
	if p.dropAssists == nil {
		panic(throw.IllegalState())
	}

	switch assist, ok := p.dropAssists[id]; {
	case !ok:
		return nil
	case assist == nil: // to avoid interface-nil
		panic(throw.IllegalState())
	default:
		return assist
	}
}

type dropAssistant struct {
	nodeID node.ShortNodeID
	dropID jet.PrefixedID
}

func (p *dropAssistant) AddRecords(future AddFuture, br *lineage.BundleResolver) bool {
	panic("implement me")
}

func (p *dropAssistant) GetResolver() lineage.DependencyResolver {
	panic("implement me")
}

func (p *dropAssistant) IsLocal() bool {

}
