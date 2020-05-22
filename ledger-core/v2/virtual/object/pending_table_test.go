package object

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

func TestPendingList(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOld := gen.UniqueIDWithPulse(currentPulse)
	RefOld := reference.NewSelf(objectOld)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)
	objectOne := gen.UniqueIDWithPulse(nextPulseNumber)
	objectTwo := gen.UniqueIDWithPulse(nextPulseNumber)
	RefOne := reference.NewSelf(objectOne)
	RefTwo := reference.NewSelf(objectTwo)

	pt := NewPendingList()
	require.Equal(t, 0, pt.Count())
	require.Equal(t, 0, pt.CountFinish())
	require.Equal(t, pulse.Number(0), pt.oldestPulse)

	require.Equal(t, true, pt.Add(RefOne))
	require.Equal(t, 1, pt.Count())
	require.Equal(t, nextPulseNumber, pt.oldestPulse)

	require.Equal(t, false, pt.Add(RefOne))
	require.Equal(t, 1, pt.Count())
	require.Equal(t, nextPulseNumber, pt.oldestPulse)

	require.Equal(t, true, pt.Add(RefOld))
	require.Equal(t, 2, pt.Count())
	require.Equal(t, pd.PulseNumber, pt.oldestPulse)

	require.Equal(t, true, pt.Add(RefTwo))
	require.Equal(t, 3, pt.Count())
	require.Equal(t, pd.PulseNumber, pt.oldestPulse)
	require.Equal(t, 0, pt.CountFinish())

	pt.Finish(RefOld)
	require.Equal(t, 1, pt.CountFinish())
}
