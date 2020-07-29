// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"

	"github.com/stretchr/testify/assert"
)

func NewMutator(snapshot *Snapshot) *Mutator {
	return &Mutator{Accessor: NewAccessor(snapshot)}
}

type Mutator struct {
	*Accessor
}

func (m *Mutator) AddWorkingNode(n nodeinfo.NetworkNode) {
	if _, ok := m.refIndex[n.GetReference()]; ok {
		return
	}
	if isWorkingNode(n) {
		m.snapshot.workingNodes = append(m.snapshot.workingNodes, n)
	}
	m.snapshot.activeNodes = append(m.snapshot.activeNodes, n)
	m.addToIndex(n)
}

func TestSnapshot_GetPulse(t *testing.T) {
	snapshot := NewSnapshot(10, nil)
	assert.EqualValues(t, 10, snapshot.GetPulse())
	snapshot = NewSnapshot(152, nil)
	assert.EqualValues(t, 152, snapshot.GetPulse())
}

func TestSnapshot_Equal(t *testing.T) {
	snapshot := NewSnapshot(10, nil)
	snapshot2 := NewSnapshot(11, nil)
	assert.False(t, snapshot.Equal(snapshot2))

	snapshot2.pulse = pulse.Number(10)

	genNodeCopy := func(reference reference.Global) nodeinfo.NetworkNode {
		return newMutableNode(nil, reference, member.PrimaryRoleLightMaterial,
			nil, nodeinfo.Ready, "127.0.0.1:0")
	}

	refs := gen.UniqueGlobalRefs(2)
	node1 := genNodeCopy(refs[0])
	node2 := genNodeCopy(refs[1])
	node3 := genNodeCopy(refs[1])

	mutator := NewMutator(snapshot)
	mutator.AddWorkingNode(node1)
	mutator.AddWorkingNode(node2)

	mutator2 := NewMutator(snapshot2)
	mutator2.AddWorkingNode(node1)
	mutator2.AddWorkingNode(node3)

	assert.True(t, snapshot.Equal(snapshot2))
}
