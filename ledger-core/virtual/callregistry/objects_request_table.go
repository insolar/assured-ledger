// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package callregistry

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// ObjectsResultCallRegistry is used for store results for all processed requests per object by object reference.
type ObjectsResultCallRegistry struct {
	objects map[reference.Global]*ObjectCallResults
}

type ObjectCallResults struct {
	CallResults map[reference.Global]CallSummary
}

func NewObjectRequestTable() ObjectsResultCallRegistry {
	return ObjectsResultCallRegistry{
		objects: make(map[reference.Global]*ObjectCallResults),
	}
}

func (ort *ObjectsResultCallRegistry) GetObjectCallResults(ref reference.Global) (*ObjectCallResults, bool) {
	workingTable, ok := ort.objects[ref]
	return workingTable, ok
}

func (ort *ObjectsResultCallRegistry) AddObjectCallResults(ref reference.Global, callResults ObjectCallResults) {
	ort.objects[ref] = &callResults
}
