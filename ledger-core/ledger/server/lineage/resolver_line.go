// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage.lineResolver -s _mock.go -g


type DependencyResolver interface {
	FindDependency(ref reference.Holder) ResolvedDependency
	FindLocalDependency(root reference.Holder, ref reference.LocalHolder) ResolvedDependency
}

type lineResolver interface {
	getNextRecNo() recordNo
	getNextFilNo() filamentNo
	getLineBase() reference.LocalHolder
	getLocalPN() pulse.Number

	findLineAnyDependency(root reference.Holder, ref reference.LocalHolder) ResolvedDependency
	findLineDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, dep ResolvedDependency, recap recordNo)
	findLocalDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, recNo recordNo, dep ResolvedDependency)
	findFilament(root reference.LocalHolder) (filamentNo, ResolvedDependency)
	findCollision(local reference.LocalHolder, record *Record) (recordNo, error)

	// TODO findLocalReason(ref reference.LocalHolder) recNo recordNo
}
