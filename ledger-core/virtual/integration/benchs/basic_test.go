package benchs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func BenchmarkVCallRequestGetMethod(b *testing.B) {
	convlog.DisableTextConvLog()
	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()

	var (
		prevPulse = server.GetPulse()
		class     = server.RandomGlobalWithPulse()
		object    = server.RandomGlobalWithPulse()
	)

	server.IncrementPulseAndWaitIdle(ctx)

	walletMemory := insolar.MustSerialize(testwallet.Wallet{
		Balance: 1234567,
	})

	content := &rms.VStateReport_ProvidedContentBody{
		LatestDirtyState: &rms.ObjectState{
			Class: rms.NewReference(class),
			State: rms.NewBytes(walletMemory),
		},
	}

	report := &rms.VStateReport{
		AsOf:            prevPulse.GetPulseNumber(),
		Status:          rms.StateStatusReady,
		Object:          rms.NewReference(object),
		ProvidedContent: content,
	}

	wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	server.SendPayload(ctx, report)
	testutils.WaitSignalsTimed(b, 10*time.Second, wait)

	resultSignal := make(synckit.ClosableSignalChannel, 1)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, b, server)
	typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
		resultSignal <- struct{}{}
		return false
	})

	pl := *utils.GenerateVCallRequestMethod(server)
	pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
	pl.Callee.Set(object)
	pl.CallSiteMethod = "GetBalance"

	b.StopTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pl.CallOutgoing.Set(server.BuildRandomOutgoingWithPulse())
		msg := server.WrapPayload(&pl).Finalize()

		b.StartTimer()
		server.SendMessage(ctx, msg)
		testutils.WaitSignalsTimed(b, 10*time.Second, resultSignal)
		b.StopTimer()
	}
}

func BenchmarkVCallRequestAcceptMethod(b *testing.B) {
	convlog.DisableTextConvLog()
	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()

	var (
		prevPulse = server.GetPulse()
		class     = server.RandomGlobalWithPulse()
		object    = server.RandomGlobalWithPulse()
	)
	server.IncrementPulseAndWaitIdle(ctx)

	walletMemory := insolar.MustSerialize(testwallet.Wallet{
		Balance: 1234567,
	})

	content := &rms.VStateReport_ProvidedContentBody{
		LatestDirtyState: &rms.ObjectState{
			Class: rms.NewReference(class),
			State: rms.NewBytes(walletMemory),
		},
	}

	report := &rms.VStateReport{
		AsOf:            prevPulse.GetPulseNumber(),
		Status:          rms.StateStatusReady,
		Object:          rms.NewReference(object),
		ProvidedContent: content,
	}

	wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	server.SendPayload(ctx, report)
	testutils.WaitSignalsTimed(b, 10*time.Second, wait)

	resultSignal := make(synckit.ClosableSignalChannel, 1)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, b, server)
	typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
		resultSignal <- struct{}{}
		return false
	})

	pl := *utils.GenerateVCallRequestMethod(server)
	pl.Callee.Set(object)
	pl.CallSiteMethod = "Accept"

	b.StopTimer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pl.CallOutgoing.Set(server.BuildRandomOutgoingWithPulse())
		msg := server.WrapPayload(&pl).Finalize()

		b.StartTimer()
		server.SendMessage(ctx, msg)
		testutils.WaitSignalsTimed(b, 10*time.Second, resultSignal)
		b.StopTimer()
	}
}

func BenchmarkVCallRequestConstructor(b *testing.B) {
	convlog.DisableTextConvLog()
	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()

	resultSignal := make(synckit.ClosableSignalChannel, 1)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, b, server)
	typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
		resultSignal <- struct{}{}
		return false
	})

	pl := *utils.GenerateVCallRequestConstructor(server)
	pl.Callee.Set(server.RandomGlobalWithPulse())
	pl.CallSiteMethod = "New"
	pl.CallOutgoing.Set(server.BuildRandomOutgoingWithPulse())

	b.ReportAllocs()
	b.StopTimer()
	b.ResetTimer()

	b.Run("reflect marshaller", func(b *testing.B) {
		b.ReportAllocs()
		insconveyor.DisableLogStepInfoMarshaller = true
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			pl.CallOutgoing.Set(server.BuildRandomOutgoingWithPulse())
			msg := server.WrapPayload(&pl).Finalize()

			b.StartTimer()
			server.SendMessage(ctx, msg)
			testutils.WaitSignalsTimed(b, 10*time.Second, resultSignal)
			b.StopTimer()
		}
	})

	b.Run("code marshaller", func(b *testing.B) {
		b.ReportAllocs()
		insconveyor.DisableLogStepInfoMarshaller = false
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			pl.CallOutgoing.Set(server.BuildRandomOutgoingWithPulse())
			msg := server.WrapPayload(&pl).Finalize()

			b.StartTimer()
			server.SendMessage(ctx, msg)
			testutils.WaitSignalsTimed(b, 10*time.Second, resultSignal)
			b.StopTimer()
		}
	})

}

func BenchmarkTestAPIGetBalance(b *testing.B) {
	convlog.DisableTextConvLog()
	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()

	var (
		prevPulse = server.GetPulse()
		class     = server.RandomGlobalWithPulse()
		object    = server.RandomGlobalWithPulse()
	)
	server.IncrementPulseAndWaitIdle(ctx)

	walletMemory := insolar.MustSerialize(testwallet.Wallet{
		Balance: 1234567,
	})

	content := &rms.VStateReport_ProvidedContentBody{
		LatestDirtyState: &rms.ObjectState{
			Class: rms.NewReference(class),
			State: rms.NewBytes(walletMemory),
		},
	}

	report := &rms.VStateReport{
		AsOf:            prevPulse.GetPulseNumber(),
		Status:          rms.StateStatusReady,
		Object:          rms.NewReference(object),
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
	convlog.DisableTextConvLog()
	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()

	var (
		prevPulse = server.GetPulse()
		class     = server.RandomGlobalWithPulse()
		object    = server.RandomGlobalWithPulse()
	)
	server.IncrementPulseAndWaitIdle(ctx)

	walletMemory := insolar.MustSerialize(testwallet.Wallet{
		Balance: 1234567,
	})

	content := &rms.VStateReport_ProvidedContentBody{
		LatestDirtyState: &rms.ObjectState{
			Class: rms.NewReference(class),
			State: rms.NewBytes(walletMemory),
		},
	}

	report := &rms.VStateReport{
		AsOf:            prevPulse.GetPulseNumber(),
		Status:          rms.StateStatusReady,
		Object:          rms.NewReference(object),
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
