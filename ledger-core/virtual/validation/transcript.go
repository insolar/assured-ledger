package validation

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
