// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type DependencyResolver interface {
	FindDependency(ref reference.Holder) ResolvedDependency
	FindLocalDependency(root reference.Holder, ref reference.LocalHolder) ResolvedDependency
}

type lineResolver interface {
	getNextRecNo() recordNo
	getNextFilNo() filamentNo
	getLineBase() reference.LocalHolder
	getLocalPN() pulse.Number

	findDependency(ref reference.Holder) ResolvedDependency
	findLineDependencyWithRecap(root reference.Holder, ref reference.LocalHolder) (filNo filamentNo, recap recordNo, isOpen bool, dep ResolvedDependency)
	findLineDependency(root reference.Holder, ref reference.LocalHolder) (recordNo, ResolvedDependency)
	findLocalDependency(root reference.Holder, ref reference.LocalHolder) (filNo filamentNo, recNo recordNo, isOpen bool, dep ResolvedDependency)
	findFilament(ref reference.LocalHolder) (f filamentNo, info ResolvedDependency)
	findCollision(local reference.LocalHolder, record *Record) (recordNo, error)
}
