// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tables"
)

func TestInfo_GetEarliestPulse(t *testing.T) {
	currentPulse := pulse.OfNow()
	prevPulse := pulse.OfNow().Prev(10)

	tolerance := contract.CallTolerable

	for _, tc := range []struct {
		name                  string
		getPendingTable       func() tables.PendingTable
		getKnownRequests      func() tables.WorkingTable
		ExpectedEarliestPulse pulse.Number
	}{
		{
			name: "empty",

			ExpectedEarliestPulse: pulse.Unknown,
		},
		{
			name: "only pending",
			getPendingTable: func() tables.PendingTable {
				table := tables.NewRequestTable()
				ref := reference.NewSelf(gen.UniqueLocalRefWithPulse(currentPulse))
				table.GetList(tolerance).Add(ref)
				return table
			},
			ExpectedEarliestPulse: currentPulse,
		},
		{
			name: "only known",
			getKnownRequests: func() tables.WorkingTable {
				table := tables.NewWorkingTable()
				ref := reference.NewSelf(gen.UniqueLocalRefWithPulse(currentPulse))
				table.GetList(tolerance).Add(ref)
				table.GetList(tolerance).SetActive(ref)
				return table
			},
			ExpectedEarliestPulse: currentPulse,
		},
		{
			name: "both",
			getPendingTable: func() tables.PendingTable {
				table := tables.NewRequestTable()
				ref := reference.NewSelf(gen.UniqueLocalRefWithPulse(currentPulse))
				table.GetList(tolerance).Add(ref)
				return table
			},
			getKnownRequests: func() tables.WorkingTable {
				table := tables.NewWorkingTable()
				ref := reference.NewSelf(gen.UniqueLocalRefWithPulse(prevPulse))
				table.GetList(tolerance).Add(ref)
				table.GetList(tolerance).SetActive(ref)
				return table
			},
			ExpectedEarliestPulse: prevPulse,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := Info{
				PendingTable:  tables.NewRequestTable(),
				KnownRequests: tables.NewWorkingTable(),
			}
			if tc.getPendingTable != nil {
				info.PendingTable = tc.getPendingTable()
			}
			if tc.getKnownRequests != nil {
				info.KnownRequests = tc.getKnownRequests()
			}
			assert.Equal(t, tc.ExpectedEarliestPulse, info.GetEarliestPulse(tolerance))
		})
	}
}
