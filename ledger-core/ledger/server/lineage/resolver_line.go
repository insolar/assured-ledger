package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage.lineResolver -s _mock.go -g
//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage.DependencyResolver -s _mock.go -g


type DependencyResolver interface {
	FindOtherDependency(ref reference.Holder) (ResolvedDependency, error)
	FindLineDependency(root reference.Holder, ref reference.LocalHolder) (ResolvedDependency, error)
}

type lineResolver interface {
	getNextRecNo() recordNo
	getNextFilNo() filamentNo
	getLineBase() reference.LocalHolder
	getLocalPN() pulse.Number

	findOtherDependency(ref reference.Holder) (ResolvedDependency, error)
	findLineDependency(root reference.Holder, ref reference.LocalHolder) (ResolvedDependency, error)
	findChainedDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, recNo recordNo, dep ResolvedDependency, recap recordNo)
	findLocalDependency(root reference.LocalHolder, ref reference.LocalHolder) (filamentNo, recordNo, ResolvedDependency)
	findFilament(root reference.LocalHolder) (filamentNo, ResolvedDependency)
	findCollision(local reference.LocalHolder, record *Record) (recordNo, error)

	// TODO findLocalReason(ref reference.LocalHolder) recNo recordNo
}
