package callregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestWorkingTable(t *testing.T) {
	wt := NewWorkingTable()

	assert.Equal(t, 0, wt.GetList(isolation.CallIntolerable).Count())
	assert.Equal(t, 0, len(wt.GetList(isolation.CallTolerable).requests))

	assert.Equal(t, pulse.Number(0), wt.GetList(isolation.CallIntolerable).earliestActivePulse)
	assert.Equal(t, pulse.Number(0), wt.GetList(isolation.CallTolerable).earliestActivePulse)

	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	ref := gen.UniqueGlobalRefWithPulse(currentPulse)

	intolerableList := wt.GetList(isolation.CallIntolerable)
	assert.True(t, intolerableList.add(ref))

	assert.Equal(t, 1, wt.GetList(isolation.CallIntolerable).Count())
	assert.Equal(t, 0, wt.GetList(isolation.CallTolerable).Count())
	assert.Equal(t, pulse.Unknown, wt.GetList(isolation.CallIntolerable).EarliestPulse())

	assert.True(t, intolerableList.setActive(ref))

	assert.Equal(t, 1, wt.GetList(isolation.CallIntolerable).Count())
	assert.Equal(t, 0, wt.GetList(isolation.CallTolerable).Count())
	assert.Equal(t, currentPulse, wt.GetList(isolation.CallIntolerable).EarliestPulse())

	assert.Equal(t, 0, wt.Len())
	assert.True(t, wt.Add(isolation.CallTolerable, ref))
	assert.False(t, wt.Add(isolation.CallTolerable, ref))
	assert.Equal(t, 1, wt.Len())

	assert.True(t, wt.SetActive(isolation.CallTolerable, ref))
	assert.False(t, wt.SetActive(isolation.CallTolerable, ref))
	assert.False(t, wt.SetActive(isolation.CallTolerable, gen.UniqueGlobalRef()))

	res := &rms.VCallResult{}
	res.Callee.Set(gen.UniqueGlobalRef())

	assert.True(t, wt.Finish(isolation.CallTolerable, ref, res))
	assert.False(t, wt.Finish(isolation.CallTolerable, ref, res))

	assert.True(t, wt.Finish(isolation.CallIntolerable, ref, res))
	assert.False(t, wt.Finish(isolation.CallIntolerable, ref, res))

	results := wt.GetResults()

	summary, ok := results[ref]
	assert.True(t, ok)
	assert.NotNil(t, summary.Result)
	assert.Equal(t, res.Callee, summary.Result.Callee)

	// bad flags
	assert.Panics(t, func() { wt.GetList(isolation.InterferenceFlag(0)) })
}

func TestWorkingList(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	RefOld := gen.UniqueGlobalRefWithPulse(currentPulse)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)
	RefOne := gen.UniqueGlobalRefWithPulse(nextPulseNumber)
	RefTwo := gen.UniqueGlobalRefWithPulse(nextPulseNumber)

	rl := newWorkingList()
	assert.Equal(t, 0, rl.Count())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 0, rl.CountActive())
	assert.Equal(t, pulse.Number(0), rl.EarliestPulse())

	assert.Equal(t, true, rl.add(RefOne))
	assert.Equal(t, RequestStarted, rl.GetState(RefOne))
	assert.Equal(t, true, rl.setActive(RefOne))
	assert.Equal(t, 1, rl.Count())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 1, rl.CountActive())
	assert.Equal(t, nextPulseNumber, rl.EarliestPulse())

	assert.Equal(t, false, rl.add(RefOne))
	assert.Equal(t, false, rl.setActive(RefOne))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOne))
	assert.Equal(t, 1, rl.Count())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 1, rl.CountActive())
	assert.Equal(t, nextPulseNumber, rl.EarliestPulse())

	assert.Equal(t, true, rl.add(RefOld))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOne))
	assert.Equal(t, RequestStarted, rl.GetState(RefOld))
	assert.Equal(t, true, rl.setActive(RefOld))
	assert.Equal(t, 2, rl.Count())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 2, rl.CountActive())
	assert.Equal(t, pd.PulseNumber, rl.EarliestPulse())

	assert.Equal(t, true, rl.add(RefTwo))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOne))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOld))
	assert.Equal(t, RequestStarted, rl.GetState(RefTwo))
	assert.Equal(t, true, rl.setActive(RefTwo))
	assert.Equal(t, 3, rl.Count())
	assert.Equal(t, pd.PulseNumber, rl.EarliestPulse())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 3, rl.CountActive())

	rl.finish(RefOne)
	assert.Equal(t, pd.PulseNumber, rl.EarliestPulse()) // doesn't change
	assert.Equal(t, 1, rl.CountFinish())
	assert.Equal(t, 2, rl.CountActive())

	// try to finish ref that not in list
	successFinish := rl.finish(gen.UniqueGlobalRefWithPulse(currentPulse))
	assert.Equal(t, false, successFinish)
	assert.Equal(t, 1, rl.CountFinish())
	assert.Equal(t, 2, rl.CountActive())
}

func TestWorkingList_Finish(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	RefOne := gen.UniqueGlobalRefWithPulse(currentPulse)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)

	RefTwo := gen.UniqueGlobalRefWithPulse(nextPulseNumber)

	rl := newWorkingList()

	assert.Equal(t, true, rl.add(RefOne))
	assert.Equal(t, 1, rl.Count())
	assert.Equal(t, true, rl.setActive(RefOne))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOne))
	assert.Equal(t, currentPulse, rl.earliestActivePulse)
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 1, rl.CountActive())

	assert.Equal(t, true, rl.add(RefTwo))
	assert.Equal(t, 2, rl.Count())
	assert.Equal(t, true, rl.setActive(RefTwo))
	assert.Equal(t, RequestProcessing, rl.GetState(RefTwo))
	assert.Equal(t, currentPulse, rl.earliestActivePulse)
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 2, rl.CountActive())

	rl.finish(RefOne)

	assert.Equal(t, 1, rl.CountFinish())
	assert.Equal(t, 1, rl.CountActive())
	assert.Equal(t, nextPulseNumber, rl.earliestActivePulse)
}
