// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package routing

import (
	"strconv"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	node2 "github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/node"
)

func newNode(ref reference.Global, id int) node2.NetworkNode {
	address := "127.0.0.1:" + strconv.Itoa(id)
	result := node.NewNode(ref, node2.StaticRoleUnknown, nil, address, "")
	result.(node.MutableNode).SetShortID(node2.ShortNodeID(id))
	return result
}

func TestTable_Resolve(t *testing.T) {
	table := Table{}

	refs := gen.UniqueGlobalRefs(2)
	puls := pulsestor.GenesisPulse
	nodeKeeperMock := mock.NewNodeKeeperMock(t)
	nodeKeeperMock.GetAccessorMock.Set(func(p1 pulse.Number) network.Accessor {
		n := newNode(refs[0], 123)
		return node.NewAccessor(node.NewSnapshot(puls.PulseNumber, []node2.NetworkNode{n}))
	})

	pulseAccessorMock := appctl.NewPulseAccessorMock(t)
	pulseAccessorMock.GetLatestPulseMock.Return(puls, nil)

	table.PulseAccessor = pulseAccessorMock
	table.NodeKeeper = nodeKeeperMock

	h, err := table.Resolve(refs[0])
	require.NoError(t, err)
	assert.EqualValues(t, 123, h.ShortID)
	assert.Equal(t, "127.0.0.1:123", h.Address.String())

	_, err = table.Resolve(refs[1])
	assert.Error(t, err)
}
