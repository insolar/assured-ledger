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

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

// TODO: temporarily test for debug
func TestVirtual_VCachedMemoryRequest(t *testing.T) {
	defer commontestutils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	runnerMock := logicless.NewServiceMock(ctx, mc, nil)
	server.ReplaceRunner(runnerMock)
	server.Init(ctx)

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	// init object
	{
		// need for correct handle state report (should from prev pulse)
		server.IncrementPulse(ctx)
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)
	}

	firstOutgoing := server.BuildRandomOutgoingWithPulse()
	{
		methodResult := requestresult.New([]byte("execution"), gen.UniqueGlobalRef())
		methodResult.SetAmend(descriptor.NewObject(reference.Global{}, reference.Local{}, gen.UniqueGlobalRef(), []byte(""), false), []byte("new memory"))

		runnerMock.AddExecutionMock(firstOutgoing.String()).AddStart(nil,
			&execution.Update{
				Type:   execution.Done,
				Result: methodResult,
			},
		)
		runnerMock.AddExecutionClassify(firstOutgoing.String(), tolerableFlags(), nil)
	}

	{
		typedChecker.VCachedMemoryResponse.Set(func(response *payload.VCachedMemoryResponse) bool {
			assert.Equal(t, objectGlobal, response.Object)
			assert.Equal(t, payload.CachedMemoryStateFound, response.CallStatus)
			assert.Equal(t, false, response.Inactive)
			assert.Equal(t, []byte("new memory"), response.Memory)
			return false
		})
		typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {
			assert.Equal(t, objectGlobal, result.Callee)
			assert.Equal(t, []byte("execution"), result.ReturnArguments)
			return false
		})
	}

	{
		executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

		pl := utils.GenerateVCallRequestMethod(server)
		pl.Callee = objectGlobal
		pl.CallOutgoing = firstOutgoing
		pl.CallFlags = payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)

		server.SendPayload(ctx, pl)

		commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	}

	{
		memoryRequestDone := server.Journal.WaitStopOf(&handlers.SMVCachedMemoryRequest{}, 1)

		msg := payload.VCachedMemoryRequest{Object: objectGlobal}
		server.SendPayload(ctx, &msg)

		commontestutils.WaitSignalsTimed(t, 10*time.Second, memoryRequestDone)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
		assert.Equal(t, 1, typedChecker.VCachedMemoryResponse.Count())
	}

	mc.Finish()
}
