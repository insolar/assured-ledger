// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/network/mutable"
	"github.com/insolar/assured-ledger/ledger-core/version"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network"

	testutils "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func TestGetNetworkStatus(t *testing.T) {
	sn := &ServiceNetwork{}
	gwer := testutils.NewGatewayerMock(t)
	gw := testutils.NewGatewayMock(t)
	ins := network.State(1)
	gw.GetStateMock.Set(func() network.State { return ins })
	gwer.GatewayMock.Set(func() network.Gateway { return gw })
	sn.Gatewayer = gwer

	pc := beat.Beat{}
	pc.PulseNumber = 200000
	ppn := pc.PulseNumber
	pc.NextPulseDelta = 10

	nk := testutils.NewNodeKeeperMock(t)
	a := testutils.NewAccessorMock(t)
	activeLen := 1
	active := make([]nodeinfo.NetworkNode, activeLen)
	a.GetActiveNodesMock.Return(active)
	a.GetPulseNumberMock.Return(pc.PulseNumber)

	workingLen := 2
	working := make([]nodeinfo.NetworkNode, workingLen)
	a.GetWorkingNodesMock.Return(working)

	nk.GetLatestAccessorMock.Return(a)

	ref := gen.UniqueGlobalRef()
	nn := mutable.NewTestNode(ref, member.PrimaryRoleNeutral, "")
	nk.GetLocalNodeReferenceMock.Return(ref)
	nk.GetOriginMock.Return(nn)

	sn.NodeKeeper = nk

	ns := sn.GetNetworkStatus()
	require.Equal(t, ins, ns.NetworkState)

	require.Equal(t, nn, ns.Origin)

	require.Equal(t, activeLen, ns.ActiveListSize)

	require.Equal(t, workingLen, ns.WorkingListSize)

	require.Len(t, ns.Nodes, activeLen)

	require.Equal(t, ppn, ns.PulseNumber)

	require.Equal(t, version.Version, ns.Version)

	ns = sn.GetNetworkStatus()
	require.Equal(t, ins, ns.NetworkState)

	require.Equal(t, nn, ns.Origin)

	require.Equal(t, activeLen, ns.ActiveListSize)

	require.Equal(t, workingLen, ns.WorkingListSize)

	require.Len(t, ns.Nodes, activeLen)

	require.Equal(t, pulse.Number(200000), ns.PulseNumber)

	require.Equal(t, version.Version, ns.Version)
}
