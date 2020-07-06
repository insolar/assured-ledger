// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jetbalancer

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type MaterialAllocationStrategy interface {
	CreateCalculator(entropy longbits.FixedReader, population census.OnlinePopulation) MaterialAllocationCalculator
}

type MaterialAllocationCalculator interface {
	AllocationOfLine(reference.Local) jet.DropID

	// AllocationOfJets MUST receive list of all jets, otherwise allocation may be incorrect.
	// This requirements for all jets is intended to support balanced distribution algorithms.
	AllocationOfJets([]jet.PrefixedID) map[jet.PrefixedID]node.ShortNodeID
}

// // TODO make abstract
// assignmentFn := population.GetRolePopulation(member.PrimaryRoleLightMaterial).GetAssignmentByPower
// metricFn := jetbalancer.MetricCalc{}.CalcMetric

// metric := metricFn(uint64(jetPID.ID().AsDrop(pn)), pa.pulseData.PulseEntropy)
// assignedNode, _ := assignmentFn(metric, 0)


