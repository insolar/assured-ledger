package requests

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ dataextractor.Output = &outputToLReadResponse{}
type outputToLReadResponse struct {
	wrapSize int
	entries  []rms.LReadResponse_Entry
	draft    rms.LReadResponse_Entry
}

func (p *outputToLReadResponse) BeginEntry(ce *catalog.Entry, fullExcerpt bool) error {
	// TODO verify RootRef and ReasonRef
	switch {
	case ce.EntryData.RecordRef.IsZero():
		panic(throw.IllegalValue())
	case p.hasDraft():
		panic(throw.IllegalState())
	case fullExcerpt:
		p.draft = rms.LReadResponse_Entry{ EntryData: ce.EntryData }
		return nil
	}

	p.draft = rms.LReadResponse_Entry{}
	p.draft.EntryData.RecordRef = ce.EntryData.RecordRef
	p.draft.EntryData.RegistrarSignature = ce.EntryData.RegistrarSignature
	p.draft.EntryData.RegisteredBy = ce.EntryData.RegisteredBy

	return nil
}

func (p *outputToLReadResponse) AddBody(slice readbundle.Slice) error {
	if !p.hasDraft() {
		panic(throw.IllegalState())
	}

	p.draft.RecordBinary.Set(longbits.CopyFixed(slice))
	return nil
}

func (p *outputToLReadResponse) AddPayload(slice readbundle.Slice) error {
	if !p.hasDraft() {
		panic(throw.IllegalState())
	}

	payload := longbits.AsBytes(slice)
	p.draft.Payloads = append(p.draft.Payloads, payload)
	return nil
}

func (p *outputToLReadResponse) AddExtension(_ ledger.ExtensionID, slice readbundle.Slice) error {
	return p.AddPayload(slice)
}

func (p *outputToLReadResponse) EndEntry() (consumedSize int, _ error) {
	if !p.hasDraft() {
		panic(throw.IllegalState())
	}

	consumedSize = p.draft.ProtoSize() + p.wrapSize
	p.entries = append(p.entries, p.draft)
	p.draft.EntryData.RecordRef.SetExact(nil)
	return consumedSize, nil
}

func (p *outputToLReadResponse) hasDraft() bool {
	return !p.draft.EntryData.RecordRef.IsZero()
}
