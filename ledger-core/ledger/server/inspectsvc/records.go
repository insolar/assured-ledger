// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inspectsvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type RegisterRequestSet struct {
	Requests []*rms.LRegisterRequest
	Excerpts []catalog.Excerpt
}

func (v RegisterRequestSet) IsEmpty() bool {
	return len(v.Requests) == 0
}

type InspectedRecordSet struct {
	Records []lineage.Record
}

func (v InspectedRecordSet) IsEmpty() bool {
	return len(v.Records) == 0
}

func (v InspectedRecordSet) Count() int {
	return len(v.Records)
}
