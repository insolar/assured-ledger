// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package validation

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"gotest.tools/assert"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_ObjectValidationReport(t *testing.T) {
	defer commonTestUtils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		prevPulse    = server.GetPulse().PulseNumber
		objectGlobal = server.RandomGlobalWithPulse()
		class        = server.RandomGlobalWithPulse()
	)

	objDescriptor := descriptor.NewObject(objectGlobal, server.RandomLocalWithPulse(), class, []byte("new state"), false)
	validatedStateRef := reference.NewRecordOf(objDescriptor.HeadRef(), objDescriptor.StateID())

	// add checker
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	{
		typedChecker.VCachedMemoryRequest.Set(func(request *rms.VCachedMemoryRequest) bool {
			assert.Equal(t, objectGlobal, request.Object)
			assert.Equal(t, validatedStateRef.GetLocal(), request.StateID)

			pl := &rms.VCachedMemoryResponse{
				Object:     request.Object,
				StateID:    request.StateID,
				CallStatus: rms.CachedMemoryStateFound,
				Node:       rms.NewReference(server.GlobalCaller()),
				Inactive:   false,
				Memory:     rms.NewBytes([]byte("new state")),
			}
			server.SendPayload(ctx, pl)
			return false
		})
	}

	server.IncrementPulse(ctx)

	// send VObjectValidationReport
	{
		validationReport := &rms.VObjectValidationReport{
			Object:    rms.NewReference(objectGlobal),
			In:        prevPulse,
			Validated: rms.NewReference(validatedStateRef),
		}
		waitValidationReport := server.Journal.WaitStopOf(&handlers.SMVObjectValidationReport{}, 1)
		server.SendPayload(ctx, validationReport)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitValidationReport)
	}

	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VCachedMemoryRequest.Wait(ctx, 1))
	assert.Equal(t, 1, typedChecker.VCachedMemoryRequest.Count())

	mc.Finish()
}
