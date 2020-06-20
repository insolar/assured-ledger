// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

type SynchronizationPoint struct {
	count int

	input  chan struct{}
	output chan struct{}
}

func (p *SynchronizationPoint) Synchronize() {
	p.input <- struct{}{}

	<-p.output
}

func (p *SynchronizationPoint) WaitAll(t *testing.T) {
	for i := 0; i < p.count; i++ {
		select {
		case <-p.input:
		case <-time.After(10 * time.Second):
			t.Fatal("timeout: failed to wait until all goroutines are synced")
		}
	}

	for i := 0; i < p.count; i++ {
		p.output <- struct{}{}
	}
}

func NewSynchronizationPoint(count int) *SynchronizationPoint {
	return &SynchronizationPoint{
		count: count,

		input:  make(chan struct{}, count),
		output: make(chan struct{}, 0),
	}
}

func TestDeduplication_Constructor_DuringExecution(t *testing.T) {
	t.Log("C4998")

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		isolation = contract.ConstructorIsolation()
		outgoing  = server.RandomLocalWithPulse()
		class     = gen.UniqueReference()
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Callee:         class,
		CallSiteMethod: "New",
		CallOutgoing:   outgoing,
	}

	synchronizeExecution := NewSynchronizationPoint(1)

	{
		requestResult := requestresult.New([]byte("123"), gen.UniqueReference())
		requestResult.SetActivate(gen.UniqueReference(), class, []byte("234"))

		executionMock := runnerMock.AddExecutionMock(calculateOutgoing(pl).String())
		executionMock.AddStart(func(ctx execution.Context) {
			synchronizeExecution.Synchronize()
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, t, server)
	typedChecker.VCallResult.SetResend(false)
	typedChecker.VDelegatedCallRequest.SetResend(true)
	typedChecker.VDelegatedCallResponse.SetResend(true)
	typedChecker.VDelegatedRequestFinished.SetResend(true)
	typedChecker.VStateReport.SetResend(true)

	{
		msg := server.WrapPayload(&pl).Finalize()
		server.SendMessage(ctx, msg)
	}

	server.WaitActiveThenIdleConveyor()
	server.IncrementPulse(ctx)

	synchronizeExecution.WaitAll(t)

	{
		msg := server.WrapPayload(&pl).Finalize()
		server.SendMessage(ctx, msg)
	}

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedCallRequest.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedCallResponse.Count())
		assert.Equal(t, 1, typedChecker.VDelegatedCallRequest.Count())
		assert.Equal(t, 1, typedChecker.VStateReport.Count())
	}
}

func TestDeduplication_SecondCallOfMethodDuringExecution(t *testing.T) {
	t.Log("C5095")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	oneExecutionEnded := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 2)

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	p1 := server.GetPulse().PulseNumber

	outgoing := server.RandomLocalWithPulse()
	class := gen.UniqueReference()
	object := gen.UniqueReference()

	server.IncrementPulseAndWaitIdle(ctx)

	report := &payload.VStateReport{
		Status: payload.Ready,
		AsOf:   p1,
		Object: object,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				State:       []byte("memory"),
				Deactivated: false,
			},
		},
	}
	server.SendPayload(ctx, report)

	releaseBlockedExecution := make(chan struct{}, 0)
	numberOfExecutions := 0
	{
		isolation := contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		runnerMock.AddExecutionClassify("SomeMethod", isolation, nil)

		newObjDescriptor := descriptor.NewObject(
			reference.Global{}, reference.Local{}, class, []byte(""), reference.Global{},
		)

		requestResult := requestresult.New([]byte("call result"), gen.UniqueReference())
		requestResult.SetAmend(newObjDescriptor, []byte("new memory"))

		executionMock := runnerMock.AddExecutionMock("SomeMethod")
		executionMock.AddStart(func(ctx execution.Context) {
			numberOfExecutions++
			<-releaseBlockedExecution
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		return false
	}).ExpectedCount(1)

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Callee:         object,
		CallSiteMethod: "SomeMethod",
		CallOutgoing:   outgoing,
	}
	server.SendPayload(ctx, &pl)
	server.SendPayload(ctx, &pl)

	testutils.WaitSignalsTimed(t, 10*time.Second, oneExecutionEnded)

	close(releaseBlockedExecution)

	testutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		assert.Equal(t, 1, numberOfExecutions)
		assert.Equal(t, 1, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

func TestDeduplication_SecondCallOfMethodAfterExecution(t *testing.T) {
	t.Log("C5096")
	t.Skip("https://insolar.atlassian.net/browse/PLAT-551")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	p1 := server.GetPulse().PulseNumber

	outgoing := server.RandomLocalWithPulse()
	class := gen.UniqueReference()
	object := gen.UniqueReference()

	server.IncrementPulseAndWaitIdle(ctx)

	report := &payload.VStateReport{
		Status: payload.Ready,
		AsOf:   p1,
		Object: object,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				State:       []byte("memory"),
				Deactivated: false,
			},
		},
	}
	server.SendPayload(ctx, report)

	numberOfExecutions := 0
	{
		isolation := contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
		runnerMock.AddExecutionClassify("SomeMethod", isolation, nil)

		newObjDescriptor := descriptor.NewObject(
			reference.Global{}, reference.Local{}, class, []byte(""), reference.Global{},
		)

		requestResult := requestresult.New([]byte("call result"), gen.UniqueReference())
		requestResult.SetAmend(newObjDescriptor, []byte("new memory"))

		executionMock := runnerMock.AddExecutionMock("SomeMethod")
		executionMock.AddStart(func(ctx execution.Context) {
			numberOfExecutions++
		}, &execution.Update{
			Type:   execution.Done,
			Result: requestResult,
		})
	}

	var firstResult *payload.VCallResult

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
		if firstResult != nil {
			require.Equal(t, firstResult, result)
		} else {
			firstResult = result
		}

		return false
	}).ExpectedCount(2)

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Callee:         object,
		CallSiteMethod: "SomeMethod",
		CallOutgoing:   outgoing,
	}

	server.SendPayload(ctx, &pl)
	server.PublisherMock.WaitCount(1, 10*time.Second)
	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	server.SendPayload(ctx, &pl)
	server.PublisherMock.WaitCount(2, 10*time.Second)
	assert.Equal(t, 2, typedChecker.VCallResult.Count())

	assert.Equal(t, 1, numberOfExecutions)

	mc.Finish()
}
