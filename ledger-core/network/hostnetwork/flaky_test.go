package hostnetwork

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
)

func TestHostNetwork_SendRequestPacket2(t *testing.T) {
	defer testutils.LeakTester(t)
	// response could be sent when test is already finished
	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
		return !strings.Contains(s, "Failed to send response")
	})

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
		return nil, nil
	}

	s.n2.RegisterRequestHandler(types.RPC, handler)

	s.Start()

	ref, err := reference.GlobalFromString(id2)
	require.NoError(t, err)
	f, err := s.n1.SendRequest(s.ctx1, types.RPC, &rms.RPCRequest{}, ref)
	require.NoError(t, err)
	f.Cancel()

	wg.Wait()
}
