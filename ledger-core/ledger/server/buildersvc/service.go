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
	PreparePulseChange(/* callback */)
	CancelPulseChange()
	CommitPulseChange()

	CalculateJetDrop(reference.Holder) jet.DropID
	GetResolver() lineage.DependencyResolver
}

type AppendFuture interface {
	TrySetFutureResult(committed bool, allocatedBase uint32, err error) bool
}

type Service interface {
	CreatePlash(pulse.Range, jet.Tree, census.OnlinePopulation) (PlashAssistant, []jet.ExactID)
	AppendToDrop(jet.DropID, AppendFuture, *lineage.BundleResolver)
}

var _ Service = &serviceImpl{}

func NewService() Service {
	return &serviceImpl{}
}

type serviceImpl struct {
	allocationStrategy jetbalancer.MaterialAllocationStrategy
}

func (p *serviceImpl) AppendToDrop(id jet.DropID, future AppendFuture, resolver *lineage.BundleResolver) {
	panic("implement me")
}

func (p *serviceImpl) CreatePlash(pr pulse.Range, tree jet.Tree, population census.OnlinePopulation) (PlashAssistant, []jet.ExactID) {
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
		return pa, []jet.ExactID{jet.GenesisExactID}
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

	pn := pa.pulseData.PulseNumber
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
		da.pa = pa
		pa.dropAssists[jetID] = da
	}

	// TODO write down shared data?

	return pa, result
}

type plashAssistant struct {
	pulseData   pulse.Data
	tree        jet.PrefixTree
	population  census.OnlinePopulation
	calc        jetbalancer.MaterialAllocationCalculator
	dropAssists map[jet.ID]*dropAssistant
}

func (p *plashAssistant) PreparePulseChange() {
	panic("implement me")
}

func (p *plashAssistant) CancelPulseChange() {
	panic("implement me")
}

func (p *plashAssistant) CommitPulseChange() {
	panic("implement me")
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

func (p *plashAssistant) getDropAssistant(id jet.ID) *dropAssistant {
	if p.dropAssists == nil {
		panic(throw.IllegalState())
	}

	switch assist, ok := p.dropAssists[id]; {
	case !ok:
		return nil
	case assist == nil:
		panic(throw.IllegalState())
	default:
		return assist
	}
}

func (p *plashAssistant) addRecords(pid jet.ExactID, br *lineage.BundleResolver, future AppendFuture) bool {
	panic(throw.NotImplemented())
}

// TODO a configuration set for conveyor that provides adapters and input-SM mapper

type dropAssistant struct {
	nodeID node.ShortNodeID
	dropID jet.ExactID
	pa     *plashAssistant
}

func (p *dropAssistant) AddRecords(future AppendFuture, br *lineage.BundleResolver) bool {
	// TODO check if drop was properly initialized
	return p.pa.addRecords(p.dropID, br, future)
}

func (p *dropAssistant) isLocal() bool {
	return p.dropID != 0
}
