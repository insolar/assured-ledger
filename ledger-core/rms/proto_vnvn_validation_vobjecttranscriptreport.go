// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func (m *VObjectTranscriptReport) Validate(currentPulse PulseNumber) error {
	object := m.Object.GetValue()
	_, err := validSelfScopedGlobalWithPulseBeforeOrEq(object, currentPulse, "Object")
	if err != nil {
		return err
	}

	if !isTimePulseBefore(m.GetAsOf(), currentPulse) {
		return throw.New("AsOf should be time pulse and less than current pulse")
	}

	transcript := m.GetObjectTranscript()
	if len(transcript.GetEntries()) == 0 {
		return throw.New("Transcript mustn't be empty")
	}

	transcriptErr := validateEntries(transcript.GetEntries())
	if transcriptErr != nil {
		return throw.W(transcriptErr, "ObjectTranscript validation failed")
	}

	pendings := m.GetPendingTranscripts()
	if len(pendings) != 0 {
		pendingTranscriptsErr := validatePendingTranscripts(pendings)
		if pendingTranscriptsErr != nil {
			return pendingTranscriptsErr
		}
	}

	return nil
}

func validateEntries(entries []Any) error {
	for _, entry := range entries {
		entryValue := entry.Get()
		switch entryValue.(type) {
		case *Transcript_TranscriptEntryIncomingRequest:
		case *Transcript_TranscriptEntryIncomingResult:
		case *Transcript_TranscriptEntryOutgoingRequest:
		case *Transcript_TranscriptEntryOutgoingResult:
		default:
			return throw.New("unexpected type: %T", entryValue)
		}
	}
	return nil
}

func validatePendingTranscripts(transcripts []Transcript) error {
	for _, tr := range transcripts {
		entries := tr.GetEntries()
		if len(entries) == 0 {
			return throw.New("PendingTranscript entries mustn't be empty")
		}
		transcriptErr := validateEntries(entries)
		if transcriptErr != nil {
			return throw.W(transcriptErr, "PendingTranscript validation failed")
		}
	}
	return nil
}
