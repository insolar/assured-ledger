// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l1"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

func TestServer(t *testing.T) {
	var protocols apinetwork.UnifiedProtocolSet
	protocols.SignatureSizeHint = 32

	protocols.Protocols[0].SupportedPackets[0] = apinetwork.ProtocolPacketDescriptor{
		LengthBits: 14,
	}

	const Server1 = "127.0.0.1:10001"
	const Server2 = "127.0.0.1:10002"

	ups1 := NewUnifiedProtocolServer(&protocols)
	ups1.SetConfig(ServerConfig{
		BindingAddress: Server1,
		UdpMaxSize:     1400,
		PeerLimit:      -1,
	})

	ups1.StartListen()
	ups1.SetMode(AllowAll)

	pm1 := ups1.PeerManager()
	pm1.AddHostId(pm1.Local().GetPrimary(), 1)

	ups2 := NewUnifiedProtocolServer(&protocols)
	ups2.SetConfig(ServerConfig{
		BindingAddress: Server2,
		UdpMaxSize:     1400,
		PeerLimit:      -1,
	})

	ups2.StartNoListen()

	pm2 := ups2.PeerManager()
	pm2.AddHostId(pm2.Local().GetPrimary(), 2)

	conn21, err := pm2.ConnectTo(apinetwork.NewHostPort(Server1))
	require.NoError(t, err)
	require.NotNil(t, conn21)
	require.NoError(t, conn21.EnsureConnect())

	testStr := longbits.Zero(apinetwork.LargePacketBaselineWithoutSignatureSize)

	require.NoError(t, conn21.UseSessionful(int64(testStr.FixedByteSize()), true, func(t l1.OutTransport) error {
		return t.Send(testStr)
	}))
	require.NoError(t, conn21.UseSessionful(int64(testStr.FixedByteSize()), true, func(t l1.OutTransport) error {
		return t.Send(testStr)
	}))

	testStr = longbits.Zero(32768)
	require.NoError(t, conn21.UseSessionful(int64(testStr.FixedByteSize()), true, func(t l1.OutTransport) error {
		return t.Send(testStr)
	}))

	time.Sleep(5 * time.Second)
}
