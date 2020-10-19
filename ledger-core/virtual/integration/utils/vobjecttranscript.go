// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

func GenerateIncomingTranscript(request rms.VCallRequest, stateBefore, stateAfter reference.Holder, incomingRef, resultRef reference.Global) rms.Transcript {
	transcript := rms.Transcript{Entries: []rms.Any{{}, {}}}
	t := &rms.Transcript_TranscriptEntryIncomingRequest{
		Incoming: rms.NewReference(incomingRef),
		Request:  request,
	}
	if stateBefore != nil {
		t.ObjectMemory.Set(stateBefore)
	}
	transcript.Entries[0].Set(t)
	result := &rms.Transcript_TranscriptEntryIncomingResult{
		IncomingResult: rms.NewReference(resultRef),
		ObjectState:    rms.NewReference(stateAfter),
		Reason:         request.CallOutgoing,
	}
	transcript.Entries[1].Set(result)

	return transcript
}
