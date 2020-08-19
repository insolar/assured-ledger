// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memstor

import (
	"fmt"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// NewNodeNetwork create active node component
func NewNodeNetwork(_ configuration.Transport, certificate nodeinfo.Certificate) (beat.NodeNetwork, error) {
	nodeKeeper := NewNodeKeeper(certificate.GetNodeRef(), certificate.GetRole())
	return nodeKeeper, nil
}

// NewNodeKeeper create new NodeKeeper
func NewNodeKeeper(localRef reference.Holder, localRole member.PrimaryRole) beat.NodeKeeper {
	return &nodekeeper{
		localRef:  reference.Copy(localRef),
		localRole: localRole,
		storage:   NewMemoryStorage(),
	}
}

type nodekeeper struct {
	localRef  reference.Global
	localRole member.PrimaryRole

	mutex sync.RWMutex

	storage  *MemoryStorage
	last     beat.NodeSnapshot
	// expected int
}

func (nk *nodekeeper) GetLocalNodeReference() reference.Holder {
	return nk.localRef
}

func (nk *nodekeeper) GetLocalNodeRole() member.PrimaryRole {
	return nk.localRole
}

func (nk *nodekeeper) GetNodeSnapshot(pn pulse.Number) beat.NodeSnapshot {
	la := nk.FindAnyLatestNodeSnapshot()
	if la != nil && la.GetPulseNumber() == pn {
		return la
	}

	s, err := nk.storage.ForPulseNumber(pn)
	if err != nil {
		panic(fmt.Sprintf("GetNodeSnapshot(%d): %s", pn, err.Error()))
	}
	return NewAccessor(s)
}

func (nk *nodekeeper) FindAnyLatestNodeSnapshot() beat.NodeSnapshot {
	nk.mutex.RLock()
	defer nk.mutex.RUnlock()

	if nk.last == nil {
		return nil
	}
	return nk.last
}

func (nk *nodekeeper) AddExpectedBeat(beat.Beat) error {
	// n := beat.Online.GetIndexedCount()
	// // inslogger.FromContext(ctx).Debugf("AddExpectedBeat, nodes: %d", n)
	//
	// nk.mutex.Lock()
	// defer nk.mutex.Unlock()
	// nk.expected = n
	return nil
}

func (nk *nodekeeper) AddCommittedBeat(beat beat.Beat) error {
	if nodeinfo.NodeRef(beat.Online.GetLocalProfile()) != nk.localRef {
		panic(throw.IllegalValue())
	}

	nk.mutex.Lock()
	defer nk.mutex.Unlock()

	snapshot := NewSnapshot(beat.PulseNumber, beat.Online)
	accessor := NewAccessor(snapshot)

	if err := nk.storage.Append(snapshot); err != nil {
		return err
	}
	nk.last = accessor

	return nil
}
