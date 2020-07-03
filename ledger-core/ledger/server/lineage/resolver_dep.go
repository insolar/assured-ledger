// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type UnresolvedDependency struct {
	// RecordRef is nil for a dependency on filament root, defined by RecapRootRef
	RecordRef    reference.Holder
	// RecapRootRef is not nil, it denotes a required continuation/recap for the given root of the RecordRef
	RecapRootRef reference.Holder
}

func (v UnresolvedDependency) Key() reference.Holder {
	if v.RecordRef != nil {
		return v.RecordRef
	}
	return v.RecapRootRef
}

func (v UnresolvedDependency) Equal(o UnresolvedDependency) bool {
	return reference.Equal(v.RecordRef, o.RecordRef) && reference.Equal(v.RecapRootRef, o.RecapRootRef)
}

