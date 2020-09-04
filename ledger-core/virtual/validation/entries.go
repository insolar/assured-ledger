package validation

import "github.com/insolar/assured-ledger/ledger-core/reference"

type TranscriptEntry struct {
	// some common fields are expect, but a bit later

	Custom CustomTranscriptEntryPart
}

type CustomTranscriptEntryPart interface {
	TranscriptEntryMarker()
}

var _ CustomTranscriptEntryPart = TranscriptEntryIncomingRequest{}

type TranscriptEntryIncomingRequest struct {
}

func (TranscriptEntryIncomingRequest) TranscriptEntryMarker() {
}

var _ CustomTranscriptEntryPart = TranscriptEntryIncomingResult{}

type TranscriptEntryIncomingResult struct {
	IncomingResult reference.Global
	ObjectMemory reference.Global
}

func (TranscriptEntryIncomingResult) TranscriptEntryMarker() {
}

var _ CustomTranscriptEntryPart = TranscriptEntryOutgoingRequest{}

type TranscriptEntryOutgoingRequest struct {
}

func (TranscriptEntryOutgoingRequest) TranscriptEntryMarker() {
}

var _ CustomTranscriptEntryPart = TranscriptEntryOutgoingResult{}

type TranscriptEntryOutgoingResult struct {
}

func (TranscriptEntryOutgoingResult) TranscriptEntryMarker() {
}
