// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/pulse"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	testnet "github.com/insolar/assured-ledger/ledger-core/testutils/network"

	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func emtygateway(t *testing.T) network.Gateway {
	// todo use mockPulseManager(t)
	return newNoNetwork(&Base{})
}

func TestSwitch(t *testing.T) {
	t.Skip("fixme")
	ctx := context.Background()

	// nodekeeper := testnet.NewNodeKeeperMock(t)
	nodekeeper := testnet.NewNodeKeeperMock(t)
	nodekeeper.MoveSyncToActiveMock.Set(func(ctx context.Context, number pulse.Number) {})
	gatewayer := testnet.NewGatewayerMock(t)
	// pm := mockPulseManager(t)

	ge := emtygateway(t)

	require.NotNil(t, ge)
	require.Equal(t, "NoNetworkState", ge.GetState().String())

	ge.Run(ctx, pulsestor.EphemeralPulse)

	gatewayer.GatewayMock.Set(func() (g1 network.Gateway) {
		return ge
	})
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state nodeinfo.NetworkState, pulse network.NetworkedPulse) {
		ge = ge.NewGateway(ctx, state)
	})

	require.Equal(t, "CompleteNetworkState", ge.GetState().String())
	cref := gen.UniqueGlobalRef()

	for _, state := range []nodeinfo.NetworkState{nodeinfo.NoNetworkState,
		nodeinfo.JoinerBootstrap, nodeinfo.CompleteNetworkState} {
		ge = ge.NewGateway(ctx, state)
		require.Equal(t, state, ge.GetState())
		ge.Run(ctx, pulsestor.EphemeralPulse)
		au := ge.Auther()

		_, err := au.GetCert(ctx, cref)
		require.Error(t, err)

		_, err = au.ValidateCert(ctx, &mandates.Certificate{})
		require.Error(t, err)

	}

}

func TestDumbComplete_GetCert(t *testing.T) {
	t.Skip("fixme")
	ctx := context.Background()

	nodekeeper := testnet.NewNodeKeeperMock(t)
	nodekeeper.MoveSyncToActiveMock.Set(func(ctx context.Context, number pulse.Number) {})

	gatewayer := testnet.NewGatewayerMock(t)

	CM := testutils.NewCertificateManagerMock(t)
	ge := emtygateway(t)
	// pm := mockPulseManager(t)

	// ge := newNoNetwork(gatewayer, pm,
	//	nodekeeper, CR,
	//	testutils.NewServiceMock(t),
	//	testnet.NewHostNetworkMock(t),
	//	CM)

	require.NotNil(t, ge)
	require.Equal(t, "NoNetworkState", ge.GetState().String())

	ge.Run(ctx, pulsestor.EphemeralPulse)

	gatewayer.GatewayMock.Set(func() (r network.Gateway) { return ge })
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state nodeinfo.NetworkState, pulse network.NetworkedPulse) {
		ge = ge.NewGateway(ctx, state)
	})

	require.Equal(t, "CompleteNetworkState", ge.GetState().String())

	cref := gen.UniqueGlobalRef()

	CM.GetCertificateMock.Set(func() (r nodeinfo.Certificate) { return &mandates.Certificate{} })
	cert, err := ge.Auther().GetCert(ctx, cref)

	require.NoError(t, err)
	require.NotNil(t, cert)
	require.Equal(t, cert, &mandates.Certificate{})
}
