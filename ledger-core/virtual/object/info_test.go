// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
)

func TestInfo_GetEarliestPulse(t *testing.T) {
	var (
		previousPulse = pulse.OfNow().Prev(10)
		currentPulse  = pulse.OfNow()
		nextPulse     = pulse.OfNow().Next(10)
	)

	tolerance := isolation.CallTolerable

	for _, tc := range []struct {
		name                  string
		getPendingTable       func() callregistry.PendingTable
		getKnownRequests      func() callregistry.WorkingTable
		ExpectedEarliestPulse pulse.Number
	}{
		{
			name: "empty",

			ExpectedEarliestPulse: pulse.Unknown,
		},
		{
			name: "only pending",
			getPendingTable: func() callregistry.PendingTable {
				table := callregistry.NewRequestTable()
				ref := gen.UniqueGlobalRefWithPulse(currentPulse)
				table.GetList(tolerance).Add(ref)
				return table
			},
			ExpectedEarliestPulse: currentPulse,
		},
		{
			name: "only known",
			getKnownRequests: func() callregistry.WorkingTable {
				table := callregistry.NewWorkingTable()
				ref := gen.UniqueGlobalRefWithPulse(currentPulse)
				table.Add(tolerance, ref)
				table.SetActive(tolerance, ref)
				return table
			},
			ExpectedEarliestPulse: currentPulse,
		},
		{
			name: "both (should be first)",
			getPendingTable: func() callregistry.PendingTable {
				table := callregistry.NewRequestTable()
				ref := gen.UniqueGlobalRefWithPulse(currentPulse)
				table.GetList(tolerance).Add(ref)
				return table
			},
			getKnownRequests: func() callregistry.WorkingTable {
				table := callregistry.NewWorkingTable()
				ref := gen.UniqueGlobalRefWithPulse(previousPulse)
				table.Add(tolerance, ref)
				table.SetActive(tolerance, ref)
				return table
			},
			ExpectedEarliestPulse: previousPulse,
		},
		{
			name: "both (should be second)",
			getPendingTable: func() callregistry.PendingTable {
				table := callregistry.NewRequestTable()
				ref := gen.UniqueGlobalRefWithPulse(currentPulse)
				table.GetList(tolerance).Add(ref)
				return table
			},
			getKnownRequests: func() callregistry.WorkingTable {
				table := callregistry.NewWorkingTable()
				ref := gen.UniqueGlobalRefWithPulse(nextPulse)
				table.Add(tolerance, ref)
				table.SetActive(tolerance, ref)
				return table
			},
			ExpectedEarliestPulse: currentPulse,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			info := Info{
				PendingTable:  callregistry.NewRequestTable(),
				KnownRequests: callregistry.NewWorkingTable(),
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
