// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package benchs

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	walletproxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func BenchmarkOnWallets(b *testing.B) {
	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()

	resultSignal := make(synckit.ClosableSignalChannel, 1)

	wallets := make([]reference.Global, 0, 1000)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, b, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		if result.CallType == payload.CTConstructor {
			var (
				err             error
				ref             reference.Global
				contractCallErr *foundation.Error
			)
			err = foundation.UnmarshalMethodResultSimplified(result.ReturnArguments, &ref, &contractCallErr)
			require.NoError(b, err)
			require.Nil(b, contractCallErr)
			wallets = append(wallets, ref)
		}
		resultSignal <- struct{}{}
		return false
	})

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Caller:         server.GlobalCaller(),
		Callee:         walletproxy.GetClass(),
		CallSiteMethod: "New",
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}

	for i := 0; i < 1000; i++ {
		pl.CallOutgoing = server.BuildRandomOutgoingWithPulse()
		msg := server.WrapPayload(&pl).Finalize()

		server.SendMessage(ctx, msg)
		testutils.WaitSignalsTimed(b, 10*time.Second, resultSignal)
	}

	b.Logf("created %d wallets", len(wallets))

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(
		len(wallets),
		func(i, j int) {
			wallets[i], wallets[j] = wallets[j], wallets[i]
		},
	)

	b.StopTimer()
	b.ResetTimer()
	b.Run("Get", func(b *testing.B) {
		resultSignal := make(synckit.ClosableSignalChannel, 1)
		typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
			resultSignal <- struct{}{}
			return false
		})

		pl := payload.VCallRequest{
			CallType:       payload.CTMethod,
			CallFlags:      payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			Caller:         server.GlobalCaller(),
			CallSiteMethod: "GetBalance",
			Arguments:      insolar.MustSerialize([]interface{}{}),
		}

		b.StopTimer()
		b.ResetTimer()
		walletNum := 0
		for i := 0; i < b.N; i++ {
			pl.CallOutgoing = server.BuildRandomOutgoingWithPulse()
			pl.Callee = wallets[walletNum%len(wallets)]
			walletNum++
			msg := server.WrapPayload(&pl).Finalize()
			b.StartTimer()
			server.SendMessage(ctx, msg)
			testutils.WaitSignalsTimed(b, 10*time.Second, resultSignal)
			b.StopTimer()
		}
	})

	b.Run("Set", func(b *testing.B) {
		typedChecker := server.PublisherMock.SetTypedChecker(ctx, b, server)
		typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
			resultSignal <- struct{}{}
			return false
		})

		pl := payload.VCallRequest{
			CallType:       payload.CTMethod,
			CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			Caller:         server.GlobalCaller(),
			CallSiteMethod: "Accept",
			Arguments:      insolar.MustSerialize([]interface{}{uint32(10)}),
		}

		b.StopTimer()
		b.ResetTimer()

		walletNum := 0
		for i := 0; i < b.N; i++ {
			pl.CallOutgoing = server.BuildRandomOutgoingWithPulse()
			pl.Callee = wallets[walletNum%len(wallets)]
			walletNum++

			msg := server.WrapPayload(&pl).Finalize()

			b.StartTimer()
			server.SendMessage(ctx, msg)
			testutils.WaitSignalsTimed(b, 10*time.Second, resultSignal)
			b.StopTimer()
		}
	})
}
