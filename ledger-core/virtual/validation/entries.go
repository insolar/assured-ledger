package validation

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
