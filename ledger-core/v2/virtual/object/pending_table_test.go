package object

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

func TestPendingTable(t *testing.T) {
	var (
		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		objectOne = gen.IDWithPulse(pd.PulseNumber)
		objectTwo = gen.IDWithPulse(pd.PulseNumber)
		RefOne    = reference.NewSelf(objectOne)
		RefTwo    = reference.NewSelf(objectTwo)
	)

	pt := newPendingTable()
	require.Equal(t, 0, pt.Count())

	require.Equal(t, true, pt.Add(RefOne))
	require.Equal(t, 1, pt.Count())

	require.Equal(t, false, pt.Add(RefOne))
	require.Equal(t, 1, pt.Count())

	require.Equal(t, true, pt.Add(RefTwo))
	require.Equal(t, 2, pt.Count())
}
