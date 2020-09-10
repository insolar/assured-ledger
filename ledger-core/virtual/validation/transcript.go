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

func (t *Transcript) Add(e TranscriptEntry) {
	t.Entries = append(t.Entries, e)
}

func (t *Transcript) GetRMSTranscript() (rms.VObjectTranscriptReport_Transcript, error) {
	objectTranscript := rms.VObjectTranscriptReport_Transcript{}
	for _, entry := range t.Entries {
		rmsEntry := new(rms.Any)

		switch typedEntry := (entry.Custom).(type) {
		case TranscriptEntryIncomingRequest:
			requestMarshaled, err := typedEntry.CallRequest.Marshal()
			if err != nil {
				return objectTranscript, err
			}

			rmsEntry.Set(
				&rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest{
					ObjectMemory: rms.NewReference(typedEntry.ObjectMemory),
					Incoming:     rms.NewReference(typedEntry.Incoming),
					Request:      requestMarshaled, // fixme: fix it after moving all messages to RMS package
				})
		case TranscriptEntryIncomingResult:
			rmsEntry.Set(&rms.VObjectTranscriptReport_TranscriptEntryIncomingResult{
				IncomingResult: rms.NewReference(typedEntry.IncomingResult),
				ObjectState:    rms.NewReference(typedEntry.ObjectMemory),
			})

			// TODO add this later
			// case TranscriptEntryOutgoingRequest:
			// 	rmsEntry.Set(&rms.VObjectTranscriptReport_TranscriptEntryOutgoingRequest{})
			// case TranscriptEntryOutgoingResult:
			// 	rmsEntry.Set(&rms.VObjectTranscriptReport_TranscriptEntryOutgoingResult{})
		default:
			return objectTranscript, throw.New("unknown transcript entry part")
		}

		objectTranscript.Entries = append(objectTranscript.Entries, *rmsEntry)
	}
	// cant marshal it properly
	// TODO remove it
	objectTranscript.Entries = make([]rms.Any, len(objectTranscript.Entries))

	return objectTranscript, nil
}
