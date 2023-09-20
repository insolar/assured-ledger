package gen_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestGen_IDWithPulse(t *testing.T) {
	// Empty slice for comparison.
	emptySlice := make([]byte, reference.LocalBinaryHashSize)

	for i := 0; i < 100; i++ {
		pulse := gen.PulseNumber()

		idWithPulse := gen.UniqueLocalRefWithPulse(pulse)

		require.Equal(t,
			idWithPulse.Pulse().Bytes(),
			pulse.Bytes(), "pulse bytes should be equal pulse bytes from generated ID")

		pulseFromID := idWithPulse.Pulse()
		require.Equal(t,
			pulse, pulseFromID,
			"pulse should be equal pulse from generated ID")

		idHash := idWithPulse.IdentityHashBytes()
		require.NotEqual(t,
			emptySlice, idHash,
			"ID.IdentityHashBytes() should not be empty")
	}
}
