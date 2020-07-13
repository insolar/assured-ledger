// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewResolvedBundleForTestOnly(records []Record) ResolvedBundle {
	if len(records) == 0 {
		panic(throw.IllegalValue())
	}
	rb := ResolvedBundle{records: make([]resolvedRecord, len(records))}

	for i := range records {
		rb.records[i].Record = records[i]
	}

	return rb
}

type ResolvedBundle struct {
	records    []resolvedRecord
}

func (v ResolvedBundle) IsZero() bool {
	return v.records == nil
}

func (v ResolvedBundle) IsValid() bool {
	return len(v.records) > 0
}

func (v ResolvedBundle) Count() int {
	return len(v.records)
}

func (v ResolvedBundle) Enum(fn func (Record, DustMode) bool) bool {
	for _, r := range v.records {
		if fn(r.Record, 0) {
			return true
		}
	}
	return false
}
