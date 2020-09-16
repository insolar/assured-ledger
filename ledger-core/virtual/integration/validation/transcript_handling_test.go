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

	callRequest := utils.GenerateVCallRequestConstructor(server)
	outgoing := callRequest.CallOutgoing
	objectRef := reference.NewSelf(outgoing.GetValue().GetLocal())
	p := server.GetPulse().PulseNumber

	pl := rms.VObjectTranscriptReport{
		AsOf:   p,
		Object: rms.NewReference(objectRef),
		ObjectTranscript: rms.VObjectTranscriptReport_Transcript{
			Entries: []rms.Any{
				{},
				{},
			},
		},
	}
	pl.ObjectTranscript.Entries[0].Set(
		&rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest{
			Request:      *callRequest,
		},
	)
	pl.ObjectTranscript.Entries[1].Set(
		&rms.VObjectTranscriptReport_TranscriptEntryIncomingResult{
		},
	)

	done := server.Journal.WaitStopOf(&handlers.SMVObjectTranscriptReport{}, 1)
	server.SendPayload(ctx, &pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, done)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	assert.Equal(t, 1, typedChecker.VObjectTranscriptReport.Count())

	mc.Finish()
}
