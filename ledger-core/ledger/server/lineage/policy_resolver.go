package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type PolicyResolverFunc = func (reference.Holder) (ResolvedDependency, error)

type ResolvedDependency struct {
	RecordType     RecordType
	RootRef        reference.Holder
	RedirectToType RecordType
	RedirectToRef  reference.Holder
}

func (v ResolvedDependency) IsZero() bool {
	return v.RecordType == 0
}

func (v ResolvedDependency) IsNotAvailable() bool {
	return v.RecordType == RecordNotAvailable
}
