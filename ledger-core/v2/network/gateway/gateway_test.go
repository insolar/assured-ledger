// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"context"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/certificate"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	pulse2 "github.com/insolar/assured-ledger/ledger-core/v2/pulse"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	testnet "github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"

	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
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
	nodekeeper.MoveSyncToActiveMock.Set(func(ctx context.Context, number pulse2.Number) {})
	gatewayer := testnet.NewGatewayerMock(t)
	// pm := mockPulseManager(t)

	ge := emtygateway(t)

	require.NotNil(t, ge)
	require.Equal(t, "NoNetworkState", ge.GetState().String())

	ge.Run(ctx, *pulse.EphemeralPulse)

	gatewayer.GatewayMock.Set(func() (g1 network.Gateway) {
		return ge
	})
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state node.NetworkState, pulse pulse.Pulse) {
		ge = ge.NewGateway(ctx, state)
	})
	gilreleased := false

	require.Equal(t, "CompleteNetworkState", ge.GetState().String())
	require.False(t, gilreleased)
	cref := gen.Reference()

	for _, state := range []node.NetworkState{node.NoNetworkState,
		node.JoinerBootstrap, node.CompleteNetworkState} {
		ge = ge.NewGateway(ctx, state)
		require.Equal(t, state, ge.GetState())
		ge.Run(ctx, *pulse.EphemeralPulse)
		au := ge.Auther()

		_, err := au.GetCert(ctx, cref)
		require.Error(t, err)

		_, err = au.ValidateCert(ctx, &certificate.Certificate{})
		require.Error(t, err)

	}

}

func TestDumbComplete_GetCert(t *testing.T) {
	t.Skip("fixme")
	ctx := context.Background()

	// nodekeeper := testnet.NewNodeKeeperMock(t)
	nodekeeper := testnet.NewNodeKeeperMock(t)
	nodekeeper.MoveSyncToActiveMock.Set(func(ctx context.Context, number pulse2.Number) {})

	gatewayer := testnet.NewGatewayerMock(t)

	// CR := testutils.NewContractRequesterMock(t)
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

	ge.Run(ctx, *pulse.EphemeralPulse)

	gatewayer.GatewayMock.Set(func() (r network.Gateway) { return ge })
	gatewayer.SwitchStateMock.Set(func(ctx context.Context, state node.NetworkState, pulse pulse.Pulse) {
		ge = ge.NewGateway(ctx, state)
	})
	gilreleased := false

	require.Equal(t, "CompleteNetworkState", ge.GetState().String())
	require.False(t, gilreleased)

	cref := gen.Reference()

	// CR.CallMock.Set(func(ctx context.Context, ref reference.Global, method string, argsIn []interface{}, p insolar.Number,
	// ) (r insolar.Reply, r2 reference.Global, r1 error) {
	// 	require.Equal(t, &cref, ref)
	// 	require.Equal(t, "GetNodeInfo", method)
	// 	repl, _ := insolar.Serialize(struct {
	// 		PublicKey string
	// 		Role      insolar.StaticRole
	// 	}{"LALALA", insolar.StaticRoleVirtual})
	// 	return &reply.CallMethod{
	// 		Result: repl,
	// 	}, nil, nil
	// })

	CM.GetCertificateMock.Set(func() (r node.Certificate) { return &certificate.Certificate{} })
	cert, err := ge.Auther().GetCert(ctx, cref)

	require.NoError(t, err)
	require.NotNil(t, cert)
	require.Equal(t, cert, &certificate.Certificate{})
}
