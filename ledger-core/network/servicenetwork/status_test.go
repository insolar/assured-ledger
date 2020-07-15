// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package servicenetwork

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/version"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network"

	testutils "github.com/insolar/assured-ledger/ledger-core/testutils/network"
)

func TestGetNetworkStatus(t *testing.T) {
	sn := &ServiceNetwork{}
	gwer := testutils.NewGatewayerMock(t)
	gw := testutils.NewGatewayMock(t)
	ins := node.NetworkState(1)
	gw.GetStateMock.Set(func() node.NetworkState { return ins })
	gwer.GatewayMock.Set(func() network.Gateway { return gw })
	sn.Gatewayer = gwer

	pa := appctl.NewPulseAccessorMock(t)
	pc := appctl.PulseChange{}
	pc.PulseNumber = 200000
	ppn := pc.PulseNumber
	pc.NextPulseDelta = 10
	pa.GetLatestPulseMock.Return(pc, nil)
	sn.PulseAccessor = pa

	nk := testutils.NewNodeKeeperMock(t)
	a := testutils.NewAccessorMock(t)
	activeLen := 1
	active := make([]node.NetworkNode, activeLen)
	a.GetActiveNodesMock.Set(func() []node.NetworkNode { return active })

	workingLen := 2
	working := make([]node.NetworkNode, workingLen)
	a.GetWorkingNodesMock.Set(func() []node.NetworkNode { return working })

	nk.GetAccessorMock.Set(func(pulse.Number) network.Accessor { return a })

	nn := testutils.NewNetworkNodeMock(t)
	nk.GetOriginMock.Set(func() node.NetworkNode { return nn })

	sn.NodeKeeper = nk

	ns := sn.GetNetworkStatus()
	require.Equal(t, ins, ns.NetworkState)

	require.Equal(t, nn, ns.Origin)

	require.Equal(t, activeLen, ns.ActiveListSize)

	require.Equal(t, workingLen, ns.WorkingListSize)

	require.Len(t, ns.Nodes, activeLen)

	require.Equal(t, ppn, ns.Pulse.PulseNumber)

	require.Equal(t, version.Version, ns.Version)

	pa.GetLatestPulseMock.Return(pc, errors.New("test"))
	ns = sn.GetNetworkStatus()
	require.Equal(t, ins, ns.NetworkState)

	require.Equal(t, nn, ns.Origin)

	require.Equal(t, activeLen, ns.ActiveListSize)

	require.Equal(t, workingLen, ns.WorkingListSize)

	require.Len(t, ns.Nodes, activeLen)

	require.Equal(t, pulsestor.GenesisPulse.PulseNumber, ns.Pulse.PulseNumber)

	require.Equal(t, version.Version, ns.Version)
}
