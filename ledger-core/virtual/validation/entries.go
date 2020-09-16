package validation

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type TranscriptEntry struct {
	// some common fields are expect, but a bit later

	Custom CustomTranscriptEntryPart
}

type CustomTranscriptEntryPart interface {
	TranscriptEntryMarker()
}

var _ CustomTranscriptEntryPart = TranscriptEntryIncomingRequest{}

type TranscriptEntryIncomingRequest struct {
	ObjectMemory reference.Global
	Incoming     reference.Global
	CallRequest  rms.VCallRequest
}

func (TranscriptEntryIncomingRequest) TranscriptEntryMarker() {
}

var _ CustomTranscriptEntryPart = TranscriptEntryIncomingResult{}

type TranscriptEntryIncomingResult struct {
	IncomingResult reference.Global
	ObjectMemory   reference.Global
}

func (TranscriptEntryIncomingResult) TranscriptEntryMarker() {
}

var _ CustomTranscriptEntryPart = TranscriptEntryOutgoingRequest{}

type TranscriptEntryOutgoingRequest struct {
	Request rms.VCallRequest
}

func (TranscriptEntryOutgoingRequest) TranscriptEntryMarker() {
}

var _ CustomTranscriptEntryPart = TranscriptEntryOutgoingResult{}

type TranscriptEntryOutgoingResult struct {
	OutgoingResult reference.Global
	CallResult []byte
}

func (TranscriptEntryOutgoingResult) TranscriptEntryMarker() {
}
