// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build !windows

package hostnetwork

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

func TestHostNetwork_SendRequestPacket2(t *testing.T) {
	defer testutils.LeakTester(t)

	// TODO: PLAT-376 this test or network/pool should be fixed

	s := newHostSuite(t)
	defer s.Stop()

	wg := sync.WaitGroup{}
	wg.Add(1)

	handler := func(ctx context.Context, r network.ReceivedPacket) (network.Packet, error) {
		defer wg.Done()
		inslogger.FromContext(ctx).Info("handler triggered")
		ref, err := reference.GlobalFromString(id1)
		require.NoError(t, err)
		require.Equal(t, ref, r.GetSender())
		require.Equal(t, s.n1.PublicAddress(), r.GetSenderHost().Address.String())
		return s.n2.BuildResponse(ctx, r, &packet.RPCResponse{}), nil
	}

	s.n2.RegisterRequestHandler(types.RPC, handler)

	s.Start()

	ref, err := reference.GlobalFromString(id2)
	require.NoError(t, err)
	f, err := s.n1.SendRequest(s.ctx1, types.RPC, &packet.RPCRequest{}, ref)
	require.NoError(t, err)
	f.Cancel()

	wg.Wait()
}
