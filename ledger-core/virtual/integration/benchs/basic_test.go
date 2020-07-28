package benchs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	walletproxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func BenchmarkVCallRequestGetMethod(b *testing.B) {

	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class  = walletproxy.GetClass()
		object = reference.NewSelf(server.RandomLocalWithPulse())
	)

	walletMemory := insolar.MustSerialize(testwallet.Wallet{
		Balance: 1234567,
	})

	content := &payload.VStateReport_ProvidedContentBody{
		LatestDirtyState: &payload.ObjectState{
			Reference: reference.Local{},
			Class:     class,
			State:     walletMemory,
		},
	}

	report := &payload.VStateReport{
		Status:          payload.Ready,
		Object:          object,
		ProvidedContent: content,
	}

	wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	server.SendPayload(ctx, report)
	testutils.WaitSignalsTimed(b, 10*time.Second, wait)

	resultSignal := make(synckit.ClosableSignalChannel, 1)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, b, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		resultSignal <- struct{}{}
		return false
	})

	pl := payload.VCallRequest{
		CallType:            payload.CTMethod,
		CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
		Caller:              server.GlobalCaller(),
		Callee:              object,
		CallSiteDeclaration: class,
		CallSiteMethod:      "GetBalance",
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	b.StopTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pl.CallOutgoing = server.BuildRandomOutgoingWithPulse()
		msg := server.WrapPayload(&pl).Finalize()

		b.StartTimer()
		server.SendMessage(ctx, msg)
		testutils.WaitSignalsTimed(b, 10*time.Second, resultSignal)
		b.StopTimer()
	}
}

func BenchmarkTestAPIGetBalance(b *testing.B) {

	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class  = walletproxy.GetClass()
		object = reference.NewSelf(server.RandomLocalWithPulse())
	)

	walletMemory := insolar.MustSerialize(testwallet.Wallet{
		Balance: 1234567,
	})

	content := &payload.VStateReport_ProvidedContentBody{
		LatestDirtyState: &payload.ObjectState{
			Reference: reference.Local{},
			Class:     class,
			State:     walletMemory,
		},
	}

	report := &payload.VStateReport{
		Status:          payload.Ready,
		Object:          object,
		ProvidedContent: content,
	}

	wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	server.SendPayload(ctx, report)
	testutils.WaitSignalsTimed(b, 10*time.Second, wait)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code, _ := server.CallAPIGetBalance(ctx, object)
		require.Equal(b, 200, code)
	}
}

func BenchmarkTestAPIGetBalanceParallel(b *testing.B) {

	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()
	server.IncrementPulseAndWaitIdle(ctx)

	var (
		class  = walletproxy.GetClass()
		object = reference.NewSelf(server.RandomLocalWithPulse())
	)

	walletMemory := insolar.MustSerialize(testwallet.Wallet{
		Balance: 1234567,
	})

	content := &payload.VStateReport_ProvidedContentBody{
		LatestDirtyState: &payload.ObjectState{
			Reference: reference.Local{},
			Class:     class,
			State:     walletMemory,
		},
	}

	report := &payload.VStateReport{
		Status:          payload.Ready,
		Object:          object,
		ProvidedContent: content,
	}

	wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	server.SendPayload(ctx, report)
	testutils.WaitSignalsTimed(b, 10*time.Second, wait)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			code, _ := server.CallAPIGetBalance(ctx, object)
			require.Equal(b, 200, code)
		}
	})
}
