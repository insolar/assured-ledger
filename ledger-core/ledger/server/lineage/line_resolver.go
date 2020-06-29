// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

var _ lineResolver = &basicLineResolver{}

type basicLineResolver struct {
	// local negative cache
	// local positive cache
	base    reference.Local
	localPN pulse.Number

	records lineRecords
	recMap  map[reference.LocalHash]recordNo

}

func (p *basicLineResolver) findLineAnyDependency(root reference.Holder, ref reference.LocalHolder) ResolvedDependency {
	panic("implement me")
}

func (p *basicLineResolver) findLineDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, dep ResolvedDependency, recap recordNo) {
	panic("implement me")
}

func (p *basicLineResolver) findLocalDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, recNo recordNo, dep ResolvedDependency) {
	panic("implement me")
}

func (p *basicLineResolver) findFilament(root reference.LocalHolder) (filamentNo, ResolvedDependency) {
	panic("implement me")
}

func (p *basicLineResolver) findCollision(local reference.LocalHolder, record *Record) (recordNo, error) {
	panic("implement me")
}

func (p *basicLineResolver) getNextRecNo() recordNo {
	return p.records.getNextRecordNo()
}

func (p *basicLineResolver) getNextFilNo() filamentNo {
	panic("implement me")
}

func (p *basicLineResolver) getLineBase() reference.LocalHolder {
	return p.base
}

func (p *basicLineResolver) getLocalPN() pulse.Number {
	return p.localPN
}

