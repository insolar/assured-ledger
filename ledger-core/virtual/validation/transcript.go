package validation

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type Transcript struct {
	Entries []TranscriptEntry
}

func NewTranscript() Transcript {
	return Transcript{
		Entries: make([]TranscriptEntry, 0),
	}
}

func (t *Transcript) Add(e TranscriptEntry) {
	t.Entries = append(t.Entries, e)
}

func (t *Transcript) GetRMSTranscript() (rms.VObjectTranscriptReport_Transcript, error) {
	objectTranscript := rms.VObjectTranscriptReport_Transcript{}
	for _, entry := range t.Entries {
		rmsEntry := new(rms.Any)

		switch typedEntry := interface{}(entry.Custom).(type) {
		case TranscriptEntryIncomingRequest:
			rmsEntry.Set(
				&rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest{
					ObjectMemory: rms.NewReference(typedEntry.ObjectMemory),
					Incoming:     rms.NewReference(typedEntry.Incoming),
					Request:      typedEntry.CallRequest,
				})
		case TranscriptEntryIncomingResult:
			rmsEntry.Set(&rms.VObjectTranscriptReport_TranscriptEntryIncomingResult{
				IncomingResult: rms.NewReference(typedEntry.IncomingResult),
				ObjectState:    rms.NewReference(typedEntry.ObjectMemory),
			})

		case TranscriptEntryOutgoingRequest:
			rmsEntry.Set(
				&rms.VObjectTranscriptReport_TranscriptEntryOutgoingRequest{
					Request: typedEntry.Request,
				})
		case TranscriptEntryOutgoingResult:
			rmsEntry.Set(&rms.VObjectTranscriptReport_TranscriptEntryOutgoingResult{
				OutgoingResult: rms.NewReference(typedEntry.OutgoingResult),
				CallResult: typedEntry.CallResult,
			})
		}

		objectTranscript.Entries = append(objectTranscript.Entries, *rmsEntry)
	}

	return objectTranscript, nil
}
