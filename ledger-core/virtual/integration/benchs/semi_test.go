package benchs

import (
	"math/rand"
	"testing"
	"time"

	walletproxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func BenchmarkOnWallets(b *testing.B) {
	convlog.DisableTextConvLog()
	server, ctx := utils.NewServer(nil, b)
	defer server.Stop()

	wallets := make([]reference.Global, 0, 1000)

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, b, server)
	typedChecker.VCallResult.SetResend(false)

	pl := utils.GenerateVCallRequestConstructor(server)
	pl.SetClass(walletproxy.GetClass())

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1000)

	// create 1000 wallets to run get/set on them
	for i := 0; i < 1000; i++ {
		pl.SetCallSequence(uint32(i) + 1)
		rawPl := pl.Get()

		server.SendPayload(ctx, &rawPl)
		wallets = append(wallets, pl.GetObject())
	}

	testutils.WaitSignalsTimed(b, 10*time.Second, executeDone)

	b.Logf("created %d wallets", len(wallets))

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(wallets), func(i, j int) { wallets[i], wallets[j] = wallets[j], wallets[i] })

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

		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, b.N)

		for i := 0; i < b.N; i++ {
			pl.CallOutgoing.Set(server.BuildRandomOutgoingWithPulse())
			pl.Callee.Set(wallets[i%len(wallets)])

			b.StartTimer()
			server.SendPayload(ctx, &pl)
			<-resultSignal
			b.StopTimer()
		}

		testutils.WaitSignalsTimed(b, 10*time.Second, executeDone)
		testutils.WaitSignalsTimed(b, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	})

	b.Run("Set", func(b *testing.B) {
		resultSignal := make(synckit.ClosableSignalChannel, 1)
		typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
			resultSignal <- struct{}{}
			return false
		})

		pl := *utils.GenerateVCallRequestMethod(server)
		pl.CallSiteMethod = "Accept"

		b.StopTimer()
		b.ResetTimer()

		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, b.N)

		for i := 0; i < b.N; i++ {
			pl.CallOutgoing.Set(server.BuildRandomOutgoingWithPulse())
			pl.Callee.Set(wallets[i%len(wallets)])

			b.StartTimer()
			server.SendPayload(ctx, &pl)
			<-resultSignal
			b.StopTimer()
		}

		testutils.WaitSignalsTimed(b, 10*time.Second, executeDone)
		testutils.WaitSignalsTimed(b, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	})
}
