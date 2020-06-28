// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

<<<<<<< HEAD
//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage.lineResolver -s _mock.go -g
//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage.DependencyResolver -s _mock.go -g


type DependencyResolver interface {
	FindOtherDependency(ref reference.Holder) (ResolvedDependency, error)
	FindLineDependency(root reference.Holder, ref reference.LocalHolder) (ResolvedDependency, error)
=======
type DependencyResolver interface {
	FindDependency(ref reference.Holder) ResolvedDependency
	FindLocalDependency(root reference.Holder, ref reference.LocalHolder) ResolvedDependency
>>>>>>> Further work
}

type lineResolver interface {
	getNextRecNo() recordNo
	getNextFilNo() filamentNo
	getLineBase() reference.LocalHolder
	getLocalPN() pulse.Number

<<<<<<< HEAD
	findOtherDependency(ref reference.Holder) (ResolvedDependency, error)
	findLineDependency(root reference.Holder, ref reference.LocalHolder) (ResolvedDependency, error)
	findChainedDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, recNo recordNo, dep ResolvedDependency, recap recordNo)
	findLocalDependency(root reference.LocalHolder, ref reference.LocalHolder) (filamentNo, recordNo, ResolvedDependency)
	findFilament(root reference.LocalHolder) (filamentNo, ResolvedDependency)
	findCollision(local reference.LocalHolder, record *Record) (recordNo, error)

	// TODO findLocalReason(ref reference.LocalHolder) recNo recordNo
=======
	findDependency(ref reference.Holder) ResolvedDependency
	findLineDependencyWithRecap(root reference.Holder, ref reference.LocalHolder) (filNo filamentNo, recap recordNo, isOpen bool, dep ResolvedDependency)
	findLineDependency(root reference.Holder, ref reference.LocalHolder) (recordNo, ResolvedDependency)
	findLocalDependency(root reference.Holder, ref reference.LocalHolder) (filNo filamentNo, recNo recordNo, isOpen bool, dep ResolvedDependency)
	findFilament(ref reference.LocalHolder) (f filamentNo, info ResolvedDependency)
	findCollision(local reference.LocalHolder, record *Record) (recordNo, error)
>>>>>>> Further work
}
