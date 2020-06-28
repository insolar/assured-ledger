// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tables

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// store request per object by object reference.
type ObjectsRequestsTable struct {
	knownRequests map[reference.Global]*WorkingTable
}

func NewObjectRequestTable() ObjectsRequestsTable {
	return ObjectsRequestsTable{
		knownRequests: make(map[reference.Global]*WorkingTable),
	}
}

func (ort *ObjectsRequestsTable) GetObjectsKnownRequests(ref reference.Global) (*WorkingTable, bool) {
	workingTable, ok := ort.knownRequests[ref]
	return workingTable, ok
}

func (ort *ObjectsRequestsTable) AddObjectRequests(ref reference.Global, knownRequests WorkingTable) {
	ort.knownRequests[ref] = &knownRequests
}
