// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package hostnetwork

// func TestHostNetwork_SendRequestPacket2(t *testing.T) {
// 	t.Skip("fix me")
// 	defer testutils.LeakTester(t)
// 	// response could be sent when test is already finished
// 	instestlogger.SetTestOutputWithErrorFilter(t, func(s string) bool {
// 		return !strings.Contains(s, "Failed to send response")
// 	})
//
// 	s := newHostSuite(t)
// 	defer s.Stop()
//
// 	wg := sync.WaitGroup{}
// 	wg.Add(1)
//
// 	handler := func(ctx context.Context, r network.ReceivedPacket) (network.Packet, error) {
// 		defer wg.Done()
// 		inslogger.FromContext(ctx).Info("handler triggered")
// 		ref, err := reference.GlobalFromString(id1)
// 		require.NoError(t, err)
// 		require.Equal(t, ref, r.GetSenderHost())
// 		// require.Equal(t, s.n1.PublicAddress(), r.GetSenderHost().String())
// 		return nil, nil
// 	}
//
// 	s.n2.RegisterRequestHandler(types.RPC, handler)
//
// 	s.Start()
//
// 	_, err := reference.GlobalFromString(id2)
// 	require.NoError(t, err)
// 	f, err := s.n1.SendRequestToHost(s.ctx1, types.RPC, &rms.RPCRequest{}, &nwapi.Address{})
// 	require.NoError(t, err)
// 	f.Cancel()
//
// 	wg.Wait()
// }
