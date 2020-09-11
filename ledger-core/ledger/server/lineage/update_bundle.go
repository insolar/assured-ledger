// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewUpdateBundleForTestOnly(records []Record) UpdateBundle {
	if len(records) == 0 {
		panic(throw.IllegalValue())
	}
	rb := UpdateBundle{records: make([]resolvedRecord, len(records))}

	for i := range records {
		rb.records[i].Record = records[i]
	}

	return rb
}

type UpdateBundle struct {
	records    []resolvedRecord
}

func (v UpdateBundle) IsZero() bool {
	return v.records == nil
}

func (v UpdateBundle) IsValid() bool {
	return len(v.records) > 0
}

func (v UpdateBundle) Count() int {
	return len(v.records)
}

func (v UpdateBundle) Enum(fn func (Record, RecordExtension) bool) bool {
	for _, r := range v.records {
		if fn(r.Record, RecordExtension{
			Body:    r.asBasicRecord(),
			FilHead: r.filamentStartIndex.DirectoryIndex(),
			Flags:   r.filamentStartIndex.Flags(),
			// Dust:    0,
		}) {
			return true
		}
	}
	return false
}
