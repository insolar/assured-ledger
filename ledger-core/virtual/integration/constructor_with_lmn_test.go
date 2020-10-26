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

	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/recordchecker"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/vnlmn"
)

func TestVirtual_ConstructorWithLMN(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		plWrapper = utils.GenerateVCallRequestConstructor(server)

		outgoing  = plWrapper.GetOutgoing()
		objectRef = plWrapper.GetObject()

		runnerResult = []byte("123")
		pl           = plWrapper.Get()
		class        = pl.Callee.GetValue()
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCallResult.Set(func(res *rms.VCallResult) bool {
		assert.Equal(t, runnerResult, res.ReturnArguments.GetBytes())
		assert.Equal(t, objectRef, res.Callee.GetValue())
		assert.Equal(t, outgoing, res.CallOutgoing.GetValue())
		assert.Equal(t, rms.CallTypeConstructor, res.CallType)
		assert.Equal(t, pl.CallFlags, res.CallFlags)

		return false // no resend msg
	})

	checker := recordchecker.NewChecker(mc)
	{
		inb := checker.NewChainFromReference(rms.NewReference(objectRef)).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RLifelineStart{}),
			},
			recordchecker.ProduceResponse(ctx, server),
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RLineInboundRequest{}),
			},
			recordchecker.ProduceResponse(ctx, server),
		)
		inb.AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RInboundResponse{}),
			},
			recordchecker.ProduceResponse(ctx, server),
		)
		inb.AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RLineMemory{}),
			},
			recordchecker.ProduceResponse(ctx, server),
		).AddMessage(
			rms.LRegisterRequest{
				AnyRecordLazy: vnlmn.MustRecordToAnyRecordLazy(&rms.RLineActivate{}),
			},
			recordchecker.ProduceResponse(ctx, server),
		)
	}
	var (
		chainChecker = checker.GetReadView().GetObjectByReference(objectRef)
		err          error
	)
	typedChecker.LRegisterRequest.Set(func(request *rms.LRegisterRequest) bool {
		chainChecker, err = chainChecker.Feed(*request)
		require.NoError(t, err)
		require.NotNil(t, chainChecker)
		chainChecker.GetResponseProvider()(*request)
		return false
	})

	{
		requestResult := requestresult.New(runnerResult, outgoing)
		requestResult.SetActivate(class, []byte("some memory"))

		runnerMock.AddExecutionMock(outgoing).
			AddStart(nil, &execution.Update{
				Type:   execution.Done,
				Result: requestResult,
			})
	}

	server.SendPayload(ctx, &pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VCallResult.Count())

	mc.Finish()
}
