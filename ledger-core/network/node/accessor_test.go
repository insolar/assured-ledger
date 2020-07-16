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
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"

	"github.com/stretchr/testify/assert"
)

func TestAccessor(t *testing.T) {
	t.Skip("FIXME")

	node := newMutableNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, nil, nodeinfo.Ready, "127.0.0.1:0", "")

	node2 := newMutableNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, nil, nodeinfo.Joining, "127.0.0.1:0", "")
	node2.SetShortID(11)

	node3 := newMutableNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, nil, nodeinfo.Leaving, "127.0.0.1:0", "")
	node3.SetShortID(10)

	node4 := newMutableNode(gen.UniqueGlobalRef(), member.PrimaryRoleVirtual, nil, nodeinfo.Undefined, "127.0.0.1:0", "")

	snapshot := NewSnapshot(pulse.MinTimePulse, []nodeinfo.NetworkNode{node, node2, node3, node4})
	accessor := NewAccessor(snapshot)
	assert.Equal(t, 4, len(accessor.GetActiveNodes()))
	assert.Equal(t, 1, len(accessor.GetWorkingNodes()))
	assert.NotNil(t, accessor.GetWorkingNode(node.ID()))
	assert.Nil(t, accessor.GetWorkingNode(node2.ID()))
	assert.NotNil(t, accessor.GetActiveNode(node2.ID()))
	assert.NotNil(t, accessor.GetActiveNode(node3.ID()))
	assert.NotNil(t, accessor.GetActiveNode(node4.ID()))

	assert.NotNil(t, accessor.GetActiveNodeByShortID(10))
	assert.NotNil(t, accessor.GetActiveNodeByShortID(11))
	assert.Nil(t, accessor.GetActiveNodeByShortID(12))
}
