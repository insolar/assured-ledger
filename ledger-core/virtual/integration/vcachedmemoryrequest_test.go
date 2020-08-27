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

	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

// TODO: temporarily test for debug
func TestVirtual_VCachedMemoryRequest(t *testing.T) {
	defer commontestutils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	server.Init(ctx)

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
	)

	// init object
	{
		// need for correct handle state report (should from prev pulse)
		server.IncrementPulse(ctx)
		Method_PrepareObject(ctx, server, payload.StateStatusReady, objectGlobal, prevPulse)
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCachedMemoryResponse.Set(func(response *payload.VCachedMemoryResponse) bool {
		assert.Equal(t, objectGlobal, response.Object)
		assert.Equal(t, payload.CachedMemoryStateFound, response.CallStatus)
		assert.Equal(t, false, response.Inactive)
		assert.Equal(t, []byte("implement me"), response.Memory)
		return false
	})

	executeDone := server.Journal.WaitStopOf(&handlers.SMVCachedMemoryRequest{}, 1)

	msg := payload.VCachedMemoryRequest{Object: objectGlobal}
	server.SendPayload(ctx, &msg)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
	assert.Equal(t, 1, typedChecker.VCachedMemoryResponse.Count())

	mc.Finish()
}
