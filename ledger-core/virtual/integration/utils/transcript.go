package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

func BuildIncomingTranscript(request rms.VCallRequest, stateBefore, stateAfter reference.Holder) rms.Transcript {
	transcript := rms.Transcript{Entries: []rms.Any{{}, {}}}
	t := &rms.Transcript_TranscriptEntryIncomingRequest{
		Request: request,
	}
	if stateBefore != nil {
		t.ObjectMemory.Set(stateBefore)
	}
	transcript.Entries[0].Set(t)
	result := &rms.Transcript_TranscriptEntryIncomingResult{
		ObjectState: rms.NewReference(stateAfter),
		Reason:      request.CallOutgoing,
	}
	transcript.Entries[1].Set(result)

	return transcript
}

