package validation

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Transcript struct {
	Entries []TranscriptEntry
}

func NewTranscript() Transcript {
	return Transcript{
		Entries: make([]TranscriptEntry, 0),
	}
}

func (t *Transcript) Add(e... TranscriptEntry) {
	t.Entries = append(t.Entries, e...)
}

type FilterFn func (e TranscriptEntry) bool

func (t *Transcript) GetRMSTranscript(fn FilterFn) rms.Transcript {
	objectTranscript := rms.Transcript{}
	for _, entry := range t.Entries {
		if fn != nil && !fn(entry) {
			continue
		}

		var rmsEntry rms.Any

		switch typedEntry := interface{}(entry.Custom).(type) {
		case TranscriptEntryIncomingRequest:
			rmsEntry = rms.NewAny(&rms.Transcript_TranscriptEntryIncomingRequest{
					ObjectMemory: rms.NewReference(typedEntry.ObjectMemory),
					Incoming:     rms.NewReference(typedEntry.Incoming),
					Request:      typedEntry.CallRequest,
				})
		case TranscriptEntryIncomingResult:
			rmsEntry = rms.NewAny(&rms.Transcript_TranscriptEntryIncomingResult{
				IncomingResult: rms.NewReference(typedEntry.IncomingResult),
				ObjectState:    rms.NewReference(typedEntry.ObjectMemory),
				Reason:         rms.NewReference(entry.Reason),
			})

		case TranscriptEntryOutgoingRequest:
			rmsEntry = rms.NewAny(&rms.Transcript_TranscriptEntryOutgoingRequest{
					Request: rms.NewReference(typedEntry.Request),
					Reason:  rms.NewReference(entry.Reason),
				})
		case TranscriptEntryOutgoingResult:
			rmsEntry = rms.NewAny(&rms.Transcript_TranscriptEntryOutgoingResult{
				OutgoingResult: rms.NewReference(typedEntry.OutgoingResult),
				CallResult:     typedEntry.CallResult,
				Reason:         rms.NewReference(entry.Reason),
			})
		default:
			panic(throw.IllegalValue())
		}

		objectTranscript.Entries = append(objectTranscript.Entries, rmsEntry)
	}

	return objectTranscript
}
