package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewRegRecord(excerpt catalog.Excerpt, req *rms.LRegisterRequest) Record {
	return Record{
		Excerpt:            excerpt,
		RecRef:             req.AnticipatedRef.Get(),
		ProducedBy: 		req.ProducedBy.Get(),
		ProducerSignature:  req.ProducerSignature,
		regReq:             req,
	}
}

func NewRecapRecord(excerpt catalog.Excerpt, recRef reference.Holder, recap *rms.RLineRecap) Record {
	switch {
	case recRef == nil:
		panic(throw.IllegalValue())
	case recap == nil:
		panic(throw.IllegalValue())
	}

	return Record{
		Excerpt:            excerpt,
		RecRef:             recRef,
		// ProducedBy: 		recap.ProducedBy.Get(), // TODO
		recapRec:           recap,
	}
}

type Record struct {
	Excerpt            catalog.Excerpt // TODO these fields can be removed on cleanup as well
	RecRef             reference.Holder
	RecapRef           reference.Holder
	ProducedBy         reference.Holder
	ProducerSignature  rms.Binary
	RegisteredBy	   reference.Holder
	RegistrarSignature cryptkit.SignedDigest

	regReq     *rms.LRegisterRequest
	recapRec   *rms.RLineRecap
}

func (v Record) cleanup()  {
	v.regReq = nil
}

func (v Record) EqualForRecordIdempotency(record Record) bool {
	return v.Excerpt.Equal(&record.Excerpt)
}

func (v Record) GetRecordRef() reference.Holder {
	return v.RecRef
}

func (v Record) IsValid() bool {
	switch {
	case reference.IsEmpty(v.GetRecordRef()):
		return false
	case v.regReq != nil:
		return !v.regReq.AnyRecordLazy.IsZero()
	case v.recapRec != nil:
		return true
	default:
		panic(throw.IllegalState())
	}
}

func (v Record) asBasicRecord() rms.BasicRecord {
	switch {
	case v.regReq != nil:
		return &v.regReq.AnyRecordLazy
	case v.recapRec != nil:
		return v.recapRec
	default:
		panic(throw.IllegalState())
	}
}

/***********************************/

type RecordExtension struct {
	Body     rms.BasicRecord
	FilHead  ledger.DirectoryIndex
	Flags    ledger.DirectoryEntryFlags
	Dust     DustMode
}


/***********************************/
type ReadRecord struct {
	Record
	StorageIndex ledger.DirectoryIndex
}

