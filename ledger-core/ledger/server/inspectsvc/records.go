package inspectsvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewRegisterRequestSet(reqs ...*rms.LRegisterRequest) (RegisterRequestSet, error) {
	var err error
	if len(reqs) > 0 {
		if lv := reqs[0].AnyRecordLazy.TryGetLazy(); !lv.IsZero() {
			ex, err := catalog.ReadExcerptFromLazy(lv)
			if err == nil {
				return RegisterRequestSet{Requests: reqs, Excerpt: ex}, nil
			}
		}
	}
	return RegisterRequestSet{}, throw.WithDetails(err, "invalid record set")
}

type RegisterRequestSet struct {
	Requests []*rms.LRegisterRequest
	Excerpt  catalog.Excerpt
}

type VerifyRequestSet RegisterRequestSet

func (v RegisterRequestSet) IsEmpty() bool {
	return len(v.Requests) == 0
}

func (v RegisterRequestSet) Validate() {
	for _, r := range v.Requests {
		switch {
		case r == nil:
			panic(throw.IllegalValue())
		case r.AnticipatedRef.IsEmpty():
			panic(throw.IllegalState())
		}
	}
	if v.Excerpt.RecordType == 0 {
		panic(throw.IllegalValue())
	}
}

func (v RegisterRequestSet) GetRootRef() reference.Holder {
	switch {
	case !v.Excerpt.RootRef.IsEmpty():
		return v.Excerpt.RootRef.Get()
	case v.Requests[0].AnticipatedRef.IsEmpty():
		panic(throw.IllegalValue())
	default:
		return v.Requests[0].AnticipatedRef.Get()
	}
}

func (v RegisterRequestSet) GetFlags() rms.RegistrationFlags {
	return v.Requests[0].Flags
}

/*******************************************************/

type InspectedRecordSet struct {
	Records []lineage.Record
}

func (v InspectedRecordSet) IsEmpty() bool {
	return len(v.Records) == 0
}

func (v InspectedRecordSet) Count() int {
	return len(v.Records)
}
