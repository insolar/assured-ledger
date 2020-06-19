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
)

func TestInfo_GetEarliestPulse(t *testing.T) {
	currentPulse := pulse.OfNow()
	prevPulse := pulse.OfNow().Prev(10)

	tolerance := contract.CallTolerable

	for _, tc := range []struct {
		name                  string
		getPendingTable       func() RequestTable
		getKnownRequests      func() RequestTable
		ExpectedEarliestPulse pulse.Number
	}{
		{
			name: "empty",

			ExpectedEarliestPulse: pulse.Unknown,
		},
		{
			name: "only pending",
			getPendingTable: func() RequestTable {
				table := NewRequestTable()
				ref := reference.NewSelf(gen.UniqueIDWithPulse(currentPulse))
				table.GetList(tolerance).Add(ref)
				return table
			},
			ExpectedEarliestPulse: currentPulse,
		},
		{
			name: "only known",
			getKnownRequests: func() RequestTable {
				table := NewRequestTable()
				ref := reference.NewSelf(gen.UniqueIDWithPulse(currentPulse))
				table.GetList(tolerance).Add(ref)
				return table
			},
			ExpectedEarliestPulse: currentPulse,
		},
		{
			name: "both",
			getPendingTable: func() RequestTable {
				table := NewRequestTable()
				ref := reference.NewSelf(gen.UniqueIDWithPulse(currentPulse))
				table.GetList(tolerance).Add(ref)
				return table
			},
			getKnownRequests: func() RequestTable {
				table := NewRequestTable()
				ref := reference.NewSelf(gen.UniqueIDWithPulse(prevPulse))
				table.GetList(tolerance).Add(ref)
				return table
			},
			ExpectedEarliestPulse: prevPulse,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := Info{
				PendingTable:   NewRequestTable(),
				WorkedRequests: NewRequestTable(),
			}
			if tc.getPendingTable != nil {
				info.PendingTable = tc.getPendingTable()
			}
			if tc.getKnownRequests != nil {
				info.WorkedRequests = tc.getKnownRequests()
			}
			assert.Equal(t, tc.ExpectedEarliestPulse, info.GetEarliestPulse(tolerance))
		})
	}
}
