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

	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestValidation_ObjectTranscriptReport_AfterConstructor(t *testing.T) {
	defer commontestutils.LeakTester(t)
	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	authService := authentication.NewServiceMock(t)
	authService.CheckMessageFromAuthorizedVirtualMock.Return(false, nil)
	server.ReplaceAuthenticationService(authService)

	server.Init(ctx)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VCachedMemoryRequest.Set(func(report *payload.VCachedMemoryRequest) bool {
		return false
	})

	callRequest := utils.GenerateVCallRequestConstructor(server)
	outgoing := callRequest.CallOutgoing
	objectRef := reference.NewSelf(outgoing.GetLocal())
	p := server.GetPulse().PulseNumber

	callRequestBin, err := callRequest.Marshal()
	require.NoError(t, err)

	pl := rms.VObjectTranscriptReport{
		AsOf:   p,
		Object: rms.NewReference(objectRef),
		ObjectTranscript: rms.VObjectTranscriptReport_Transcript{
			Entries: []rms.Any{
				{},
			},
		},
	}
	pl.ObjectTranscript.Entries[0].Set(
		&rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest{
			ObjectMemory: rms.NewReference(reference.NewRecordOf(objectRef, server.RandomLocalWithPulse())),
			Request:      callRequestBin,
		},
	)

	done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
	server.SendPayload(ctx, &pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())

	mc.Finish()
}