// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package validation

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_ObjectValidationReport(t *testing.T) {
	defer commonTestUtils.LeakTester(t)

	testCases := []struct {
		name                  string
		validatedIsEqualDirty bool
	}{
		{name: "get memory from cache", validatedIsEqualDirty: false},
		{name: "use memory from VStateReport", validatedIsEqualDirty: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
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

			// prepare cache
			{
				err := server.MemoryCache.Set(ctx, validatedStateRef, objDescriptor)
				require.Nil(t, err)
			}

			server.IncrementPulse(ctx)

			// add typedChecker
			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			{
				typedChecker.VCachedMemoryResponse.Set(func(response *rms.VCachedMemoryResponse) bool {
					assert.Equal(t, objectGlobal, response.Object.GetValue())
					assert.Equal(t, validatedStateRef, response.StateID.GetValue())
					assert.Equal(t, []byte("new state"), response.Memory.GetBytes())
					return false
				})
			}

			// send VObjectValidationReport and VStateReport
			{
				report := &rms.VStateReport{
					AsOf:   prevPulse,
					Status: rms.StateStatusReady,
					Object: rms.NewReference(objectGlobal),
					ProvidedContent: &rms.VStateReport_ProvidedContentBody{
						LatestDirtyState: &rms.ObjectState{
							Reference: rms.NewReferenceLocal(gen.UniqueLocalRefWithPulse(prevPulse)),
							Class:     rms.NewReference(class),
							State:     rms.NewBytes([]byte("dirty state")),
						},
						LatestValidatedState: &rms.ObjectState{
							Reference: rms.NewReferenceLocal(gen.UniqueLocalRefWithPulse(prevPulse)),
							Class:     rms.NewReference(class),
							State:     rms.NewBytes([]byte("dirty state")),
						},
					},
				}
				if testCase.validatedIsEqualDirty {
					report.ProvidedContent.LatestDirtyState.Reference = rms.NewReference(validatedStateRef)
					report.ProvidedContent.LatestDirtyState.State = rms.NewBytes([]byte("new state"))
				}
				waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
				server.SendPayload(ctx, report)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitReport)

				validationReport := &rms.VObjectValidationReport{
					Object:    rms.NewReference(objectGlobal),
					In:        server.GetPulse().PulseNumber,
					Validated: rms.NewReference(validatedStateRef),
				}
				waitValidationReport := server.Journal.WaitStopOf(&handlers.SMVObjectValidationReport{}, 1)
				server.SendPayload(ctx, validationReport)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitValidationReport)
			}

			// send VCachedMemoryRequest
			{
				executeDone := server.Journal.WaitStopOf(&handlers.SMVCachedMemoryRequest{}, 1)
				pl := &rms.VCachedMemoryRequest{
					Object:  rms.NewReference(objectGlobal),
					StateID: rms.NewReference(validatedStateRef),
				}
				server.SendPayload(ctx, pl)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, typedChecker.VCachedMemoryResponse.Wait(ctx, 1))

				assert.Equal(t, 1, typedChecker.VCachedMemoryResponse.Count())
			}

			mc.Finish()
		})
	}
}
