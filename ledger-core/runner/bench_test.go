// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
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
		remoteObject  = gen.UniqueGlobalRef()
		class         = testwalletProxy.GetClass()
		wallet, _     = testwallet.New()
		defaultObject = insolar.MustSerialize(wallet)
	)

	executionContext := execution.Context{
		ObjectDescriptor: descriptor.NewObject(object, reference.Local{}, class, defaultObject, false),
		Context:          ctx,
		Request: &rms.VCallRequest{
			CallType:       rms.CallTypeMethod,
			CallSiteMethod: "GetBalance",
			Arguments:      insolar.MustSerialize([]interface{}{remoteObject, uint32(100)}),
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
