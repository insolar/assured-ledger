// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gen_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

func TestGen_IDWithPulse(t *testing.T) {
	// Empty slice for comparison.
	emptySlice := make([]byte, insolar.RecordHashSize)

	for i := 0; i < 100; i++ {
		pulse := gen.PulseNumber()

		idWithPulse := gen.IDWithPulse(pulse)

		require.Equal(t,
			idWithPulse.Pulse().Bytes(),
			pulse.Bytes(), "pulse bytes should be equal pulse bytes from generated ID")

		pulseFromID := idWithPulse.Pulse()
		require.Equal(t,
			pulse, pulseFromID,
			"pulse should be equal pulse from generated ID")

		idHash := idWithPulse.Hash()
		require.NotEqual(t,
			emptySlice, idHash,
			"ID.Hash() should not be empty")
	}
}
