// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

var _ lineResolver = &LineResolver{}

type LineResolver struct {
	// local negative cache
	// local positive cache
}

func (p *LineResolver) getNextRecNo() recordNo {
	panic("implement me")
}

func (p *LineResolver) getNextFilNo() filamentNo {
	panic("implement me")
}

func (p *LineResolver) getLineBase() reference.LocalHolder {
	panic("implement me")
}

func (p *LineResolver) getLocalPN() pulse.Number {
	panic("implement me")
}

func (p *LineResolver) findDependency(ref reference.Holder) ResolvedDependency {
	panic("implement me")
}

func (p *LineResolver) findLineDependencyWithRecap(root reference.Holder, ref reference.LocalHolder) (filNo filamentNo, recap recordNo, isOpen bool, dep ResolvedDependency) {
	panic("implement me")
}

func (p *LineResolver) findLineDependency(root reference.Holder, ref reference.LocalHolder) (recordNo, ResolvedDependency) {
	panic("implement me")
}

func (p *LineResolver) findLocalDependency(root reference.Holder, ref reference.LocalHolder) (filNo filamentNo, recNo recordNo, isOpen bool, dep ResolvedDependency) {
	panic("implement me")
}

func (p *LineResolver) findFilament(ref reference.LocalHolder) (f filamentNo, info ResolvedDependency) {
	panic("implement me")
}

func (p *LineResolver) findCollision(local reference.LocalHolder, record *Record) (recordNo, error) {
	panic("implement me")
}
