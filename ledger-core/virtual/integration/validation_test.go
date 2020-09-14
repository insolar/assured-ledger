// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_ValidationReport(t *testing.T) {
	defer commonTestUtils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		initState    = []byte("init state")
		initRef      = server.RandomLocalWithPulse()
		prevPulse    = server.GetPulse().PulseNumber
		objectGlobal = server.RandomGlobalWithPulse()
		class        = server.RandomGlobalWithPulse()
	)

	objDescriptor := descriptor.NewObject(objectGlobal, server.RandomLocalWithPulse(), class, []byte("new state"), false)
	validatedStateRef := reference.NewRecordOf(objDescriptor.HeadRef(), objDescriptor.StateID())

	// add checker
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCachedMemoryRequest.Set(func(request *payload.VCachedMemoryRequest) bool {
			assert.Equal(t, objectGlobal, request.Object)
			assert.Equal(t, validatedStateRef.GetLocal(), request.StateID)

			pl := &payload.VCachedMemoryResponse{
				Object:      request.Object,
				StateID:     request.StateID,
				CallStatus:  payload.CachedMemoryStateFound,
				Node:        server.GlobalCaller(),
				PrevStateID: payload.LocalReference{},
				Inactive:    false,
				Memory:      []byte("new state"),
			}
			server.SendPayload(ctx, pl)
			return false
		})
	}

	server.IncrementPulse(ctx)

	// send VStateReport and VObjectValidationReport
	{
		pl := &payload.VStateReport{
			Status: payload.StateStatusReady,
			Object: objectGlobal,
			AsOf:   prevPulse,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: initRef,
					Class:     class,
					State:     initState,
				},
			},
		}
		waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, pl)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitReport)

		validationReport := &payload.VObjectValidationReport{
			Object:    objectGlobal,
			In:        prevPulse,
			Validated: validatedStateRef,
		}
		waitValidationReport := server.Journal.WaitStopOf(&handlers.SMVObjectValidationReport{}, 1)
		server.SendPayload(ctx, validationReport)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitValidationReport)
	}

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VCachedMemoryRequest.Wait(ctx, 1))
	assert.Equal(t, 1, typedChecker.VCachedMemoryRequest.Count())

	mc.Finish()
}
