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
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func BenchmarkOnWallets(b *testing.B) {
	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()

	resultSignal := make(synckit.ClosableSignalChannel, 1)

	wallets := make([]reference.Global, 0, 1000)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, b, server)
	typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
		if result.CallType == rms.CallTypeConstructor {
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

	pl := *utils.GenerateVCallRequestConstructor(server)
	pl.Callee = walletproxy.GetClass()
	pl.CallSiteMethod = "New"

	// create 1000 wallets to run get/set on them
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
		typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
			resultSignal <- struct{}{}
			return false
		})

		pl := *utils.GenerateVCallRequestMethod(server)
		pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
		pl.CallSiteMethod = "GetBalance"

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
		typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
			resultSignal <- struct{}{}
			return false
		})

		pl := *utils.GenerateVCallRequestMethod(server)
		pl.CallSiteMethod = "Accept"

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
