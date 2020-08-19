// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package beatstor

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type snapshot struct {
	beat.Beat

	refIndex  map[reference.Global]member.Index
	addrIndex map[string]member.Index
}

func (p snapshot) GetPopulation() census.OnlinePopulation {
	return p.Online
}

func (p snapshot) GetPulseNumber() pulse.Number {
	return p.PulseNumber
}

func (p snapshot) FindNodeByAddr(address string) profiles.ActiveNode {
	idx, ok := p.addrIndex[address]
	if !ok {
		return nil
	}
	// NB! local node is not allowed to address itself
	return p.Online.GetProfile(idx)
}

func (p snapshot) FindNodeByRef(ref reference.Global) profiles.ActiveNode {
	idx, ok := p.refIndex[ref]
	if !ok {
		return nil
	}
	n := p.Online.GetProfile(idx)
	if n == nil && idx == 0 {
		return p.Online.GetLocalProfile()
	}
	return n
}

func (p snapshot) addAddr(endpoint endpoints.Outbound, idx member.Index) {
	p.addrIndex[endpoint.GetNameAddress().String()] = idx
}

func (p snapshot) IsOperational() bool {
	return p.Beat.IsFromPulsar()
}

func newSnapshot(pu beat.Beat) snapshot {
	local := pu.Online.GetLocalProfile()
	nodes := pu.Online.GetProfiles()

	n := len(nodes)
	if n <= 1 {
		// this is a special case for both joiner (before consensus local node is not listed) and one node population
		localRef := nodeinfo.NodeRef(local)
		return snapshot{
			Beat:      pu,
			refIndex:  map[reference.Global]member.Index{localRef:0},
		}
	}

	result := snapshot{
		Beat:      pu,
		refIndex:  make(map[reference.Global]member.Index, n),
		addrIndex: make(map[string]member.Index, n - 1), // local node is not allowed to address itself
	}

	for _, node := range pu.Online.GetProfiles() {
		ref := nodeinfo.NodeRef(node)
		idx := node.GetIndex()
		result.refIndex[ref] = idx
		if local.GetIndex() == idx {
			// NB! local node is not allowed to address itself
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
