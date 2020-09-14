// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package validation

import (
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_ObjectValidationReport(t *testing.T) {
	defer commonTestUtils.LeakTester(t)

	testCases := []struct {
		name       string
		unknownKey bool
		emptyCache bool
	}{
		// {name: "cache is empty", unknownKey: true, empty: true},
		// {name: "unknown key", unknownKey: true},
		{name: "known key", unknownKey: false},
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

			if !testCase.emptyCache {
				err := server.MemoryCache.Set(ctx, validatedStateRef, objDescriptor)
				require.Nil(t, err)
			}

			server.IncrementPulse(ctx)

			// send VObjectValidationReport
			{
				ref := rms.NewReference(validatedStateRef)
				if testCase.unknownKey {
					ref = rms.NewReference(server.RandomGlobalWithPulse())
				}

				validationReport := &rms.VObjectValidationReport{
					Object:    rms.NewReference(objectGlobal),
					In:        prevPulse,
					Validated: ref,
				}
				waitValidationReport := server.Journal.WaitStopOf(&handlers.SMVObjectValidationReport{}, 1)
				server.SendPayload(ctx, validationReport)
				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitValidationReport)
			}

			mc.Finish()
		})
	}
}
