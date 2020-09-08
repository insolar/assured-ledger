// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

func TestAbort(t *testing.T) {
	runner := NewService()
	_ = runner.Init()

	var (
		ctx = instestlogger.TestContext(t)

		object        = gen.UniqueGlobalRef()
		remoteObject  = gen.UniqueGlobalRef()
		class         = testwalletProxy.GetClass()
		wallet, _     = testwallet.New()
		defaultObject = insolar.MustSerialize(wallet)
	)

	executionContext := execution.Context{
		ObjectDescriptor: descriptor.NewObject(object, reference.Local{}, class, defaultObject, false),
		Context:          ctx,
		Request: &payload.VCallRequest{
			CallType:       payload.CallTypeMethod,
			CallSiteMethod: "Transfer",
			Arguments:      insolar.MustSerialize([]interface{}{remoteObject, uint32(100)}),
		},
		Sequence: 0,
		Object:   object,
	}

	var (
		eventSink = runner.runPrepare(executionContext)
		runState  = &runState{eventSink, Start}
		sink      = runState.state
	)

	c := make(chan struct{}, 1)

	{
		runner.runStart(sink, func() { c <- struct{}{} })
		<-c

		ev := eventSink.GetEvent()
		assert.Equal(t, execution.OutgoingCall, ev.Type)

		ev = eventSink.GetEvent()
		assert.Nil(t, ev)
	}

	{
		runner.runAbort(sink, func() { c <- struct{}{} })
		<-c

		ev := eventSink.GetEvent()
		assert.Equal(t, execution.Abort, ev.Type)

		ev = eventSink.GetEvent()
		assert.NotNil(t, ev)
		assert.Equal(t, execution.Abort, ev.Type)
	}

	{ // double abort should be fine
		runner.runAbort(sink, func() { c <- struct{}{} })
		<-c
	}

	// we need some time to wait for background task will be stopped
	for i := 0; i < 100; i++ {
		if len(runner.eventSinkMap) == 0 {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	ev := eventSink.GetEvent()
	if assert.NotNil(t, ev) {
		assert.Equal(t, execution.Abort, ev.Type)
	}

	{ // 'even' after task stop
		runner.runAbort(sink, func() { c <- struct{}{} })
		<-c
	}

	assert.Equal(t, 0, len(runner.eventSinkMap))
	assert.Equal(t, 0, len(runner.eventSinkInProgressMap))
}
