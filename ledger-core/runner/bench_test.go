package runner

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

func BenchmarkRunnerService(b *testing.B) {
	runner := NewService()
	_ = runner.Init()

	var (
		ctx = instestlogger.TestContext(b)

		object        = gen.UniqueGlobalRef()
		state         = gen.UniqueLocalRefWithPulse(object.GetLocal().Pulse())
		remoteObject  = gen.UniqueGlobalRef()
		class         = testwalletProxy.GetClass()
		wallet, _     = testwallet.New()
		defaultObject = insolar.MustSerialize(wallet)
		defaultArgs   = insolar.MustSerialize([]interface{}{remoteObject, uint32(100)})
	)

	executionContext := execution.Context{
		ObjectDescriptor: descriptor.NewObject(object, state, class, defaultObject, false),
		Context:          ctx,
		Request: &rms.VCallRequest{
			CallType:       rms.CallTypeMethod,
			CallSiteMethod: "GetBalance",
			Arguments:      rms.NewBytes(defaultArgs),
		},
		Sequence: 0,
		Object:   object,
	}

	b.ResetTimer()
	b.ReportAllocs()

	c := make(chan struct{})

	for i := 0; i < b.N; i++ {
		var (
			eventSink = runner.runPrepare(executionContext)
			runState  = &runState{eventSink, Start}
			sink      = runState.state
		)
		runner.runStart(sink, func() { c <- struct{}{} })
		<-c
	}
}
