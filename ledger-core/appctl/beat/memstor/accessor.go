// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memstor

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Accessor struct {
	snapshot  *Snapshot
	refIndex  map[reference.Global]member.Index
	addrIndex map[string]member.Index
}

func (a *Accessor) GetPopulation() census.OnlinePopulation {
	return a.snapshot.population
}

func (a *Accessor) GetPulseNumber() pulse.Number {
	return a.snapshot.GetPulseNumber()
}

func (a *Accessor) FindNodeByAddr(address string) profiles.ActiveNode {
	idx, ok := a.addrIndex[address]
	if !ok {
		return nil
	}
	// NB! local node is not allowed to address itself
	return a.snapshot.population.GetProfile(idx)
}

func (a *Accessor) FindNodeByRef(ref reference.Global) profiles.ActiveNode {
	idx, ok := a.refIndex[ref]
	if !ok {
		return nil
	}
	n := a.snapshot.population.GetProfile(idx)
	if n == nil && idx == 0 {
		return a.snapshot.population.GetLocalProfile()
	}
	return n
}

func (a *Accessor) addAddr(endpoint endpoints.Outbound, idx member.Index) {
	a.addrIndex[endpoint.GetNameAddress().String()] = idx
}

func NewAccessor(snapshot *Snapshot) *Accessor {
	result := &Accessor{
		snapshot:  snapshot,
	}

	local := snapshot.population.GetLocalProfile()
	profiles := snapshot.population.GetProfiles()

	n := len(profiles)
	if n <= 1 {
		// this is a special case for both joiner (before consensus local node is not listed) and one node population
		localRef := nodeinfo.NodeRef(local)
		result.refIndex = map[reference.Global]member.Index{localRef:0}
		return result
	}

	result.refIndex = make(map[reference.Global]member.Index, n)
	result.addrIndex = make(map[string]member.Index, n - 1) // local node is not allowed to address itself

	for _, node := range snapshot.population.GetProfiles() {
		ref := nodeinfo.NodeRef(node)
		idx := node.GetIndex()
		result.refIndex[ref] = idx
		if local.GetIndex() == idx {
			// local node is not allowed to address itself
			continue
		}

		ns := node.GetStatic()

		result.addAddr(ns.GetDefaultEndpoint(), idx)
		for _, na := range ns.GetExtension().GetExtraEndpoints() {
			result.addAddr(na, idx)
		}
	}
	return result
}
