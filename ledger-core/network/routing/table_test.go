// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package routing

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTable_Resolve(t *testing.T) {
	table := Table{}

	refs := gen.UniqueGlobalRefs(2)

	vf := cryptkit.NewSignatureVerifierFactoryMock(t)
	vf.CreateSignatureVerifierWithPKSMock.Return(nil)
	pop := census.NewOnlinePopulationMock(t)

	ip, _ := endpoints.NewIPAddress("127.0.0.1:123")
	ep := endpoints.NewOutboundMock(t)
	ep.GetIPAddressMock.Return(ip)

	nds := profiles.NewStaticProfileMock(t)
	nds.GetDefaultEndpointMock.Return(ep)

	nd := profiles.NewActiveNodeMock(t)
	nd.GetNodeIDMock.Return(123)
	nd.GetStaticMock.Return(nds)

	snap := beat.NewNodeSnapshotMock(t)
	snap.GetPopulationMock.Return(pop)
	snap.FindNodeByRefMock.When(refs[0]).Then(nd)
	snap.FindNodeByRefMock.When(refs[1]).Then(nil)

	nodeKeeperMock := beat.NewAppenderMock(t)
	nodeKeeperMock.FindAnyLatestNodeSnapshotMock.Return(snap)

	table.PulseHistory = nodeKeeperMock

	h, err := table.Resolve(refs[0])
	require.NoError(t, err)
	assert.EqualValues(t, 123, h.ShortID)
	assert.Equal(t, "127.0.0.1:123", h.Address.String())

	_, err = table.Resolve(refs[1])
	assert.Error(t, err)
}
