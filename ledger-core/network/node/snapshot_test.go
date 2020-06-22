// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package node

import (
	"testing"

	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
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

func (m *Mutator) AddWorkingNode(n node2.NetworkNode) {
	if _, ok := m.refIndex[n.ID()]; ok {
		return
	}
	mutableNode := n.(MutableNode)
	m.addToIndex(mutableNode)
	listType := nodeStateToListType(mutableNode)
	m.snapshot.nodeList[listType] = append(m.snapshot.nodeList[listType], n)
	m.active = append(m.active, n)
}

func TestSnapshot_Copy(t *testing.T) {
	snapshot := NewSnapshot(pulse.MinTimePulse, nil)
	mutator := NewMutator(snapshot)
	ref1 := gen.UniqueGlobalRef()
	node1 := newMutableNode(ref1, node2.StaticRoleVirtual, nil, node2.Ready, "127.0.0.1:0", "")
	mutator.AddWorkingNode(node1)

	snapshot2 := snapshot.Copy()
	accessor := NewAccessor(snapshot2)

	ref2 := gen.UniqueGlobalRef()
	node2 := newMutableNode(ref2, node2.StaticRoleLightMaterial, nil, node2.Ready, "127.0.0.1:0", "")
	mutator.AddWorkingNode(node2)

	// mutator and accessor observe different copies of snapshot and don't affect each other
	assert.Equal(t, 2, len(mutator.GetActiveNodes()))
	assert.Equal(t, 1, len(accessor.GetActiveNodes()))
	assert.False(t, snapshot.Equal(snapshot2))
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

	genNodeCopy := func(reference reference.Global) node2.NetworkNode {
		return newMutableNode(reference, node2.StaticRoleLightMaterial,
			nil, node2.Ready, "127.0.0.1:0", "")
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
