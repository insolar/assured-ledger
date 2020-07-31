// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package routing

import (
	"strconv"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	mock "github.com/insolar/assured-ledger/ledger-core/testutils/network"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nodeset"
)

func newNode(ref reference.Global, id int) *mutable.Node {
	address := "127.0.0.1:" + strconv.Itoa(id)
	result := mutable.NewTestNode(ref, member.PrimaryRoleUnknown, address)
	result.SetShortID(node.ShortNodeID(id))
	return result
}

func TestTable_Resolve(t *testing.T) {
	table := Table{}

	refs := gen.UniqueGlobalRefs(2)

	na := nodeset.NewAccessor(nodeset.NewSnapshot(pulse.MinTimePulse, []nodeinfo.NetworkNode{newNode(refs[0], 123)}))

	nodeKeeperMock := mock.NewNodeKeeperMock(t)
	nodeKeeperMock.GetAccessorMock.Return(na)
	nodeKeeperMock.GetLatestAccessorMock.Return(na)

	table.NodeKeeper = nodeKeeperMock

	h, err := table.Resolve(refs[0])
	require.NoError(t, err)
	assert.EqualValues(t, 123, h.ShortID)
	assert.Equal(t, "127.0.0.1:123", h.Address.String())

	_, err = table.Resolve(refs[1])
	assert.Error(t, err)
}
