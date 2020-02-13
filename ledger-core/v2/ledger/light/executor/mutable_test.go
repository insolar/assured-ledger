// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executor_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/executor"
)

func TestOldestMutable(t *testing.T) {
	t.Run("return nothing if receive nothing", func(t *testing.T) {
		res := executor.OldestMutable(nil)
		require.Nil(t, res)
	})

	t.Run("ignore outgoings and immutable", func(t *testing.T) {
		o := record.OutgoingRequest{}
		outCom := record.CompositeFilamentRecord{
			RecordID: *insolar.NewID(insolar.PulseNumber(8), nil),
			Record: record.Material{
				Polymorph: 0,
				Virtual:   record.Wrap(&o),
			},
		}

		inImmutable := record.IncomingRequest{
			Immutable: true,
		}
		inImCom := record.CompositeFilamentRecord{
			RecordID: *insolar.NewID(insolar.PulseNumber(9), nil),
			Record: record.Material{
				Polymorph: 0,
				Virtual:   record.Wrap(&inImmutable),
			},
		}

		inMutable := record.IncomingRequest{
			Immutable: false,
		}
		expectedID := insolar.NewID(insolar.PulseNumber(10), nil)
		expected := record.CompositeFilamentRecord{
			RecordID: *expectedID,
			Record: record.Material{
				Polymorph: 0,
				Virtual:   record.Wrap(&inMutable),
			},
		}

		res := executor.OldestMutable([]record.CompositeFilamentRecord{
			outCom, inImCom, expected,
		})

		require.NotNil(t, res)
		require.Equal(t, res.RecordID, *expectedID)
	})
}
