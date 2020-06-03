// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

func TestRequestTable(t *testing.T) {
	rt := NewRequestTable()

	require.Equal(t, 0, len(rt.GetList(contract.CallIntolerable).requests))
	require.Equal(t, 0, len(rt.GetList(contract.CallIntolerable).requests))

	require.Equal(t, pulse.Number(0), rt.GetList(contract.CallIntolerable).earliestPulse)
	require.Equal(t, pulse.Number(0), rt.GetList(contract.CallIntolerable).earliestPulse)

	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	object := gen.UniqueIDWithPulse(currentPulse)
	ref := reference.NewSelf(object)

	intolerableList := rt.GetList(contract.CallIntolerable)
	intolerableList.Add(ref)

	require.Equal(t, 1, len(rt.GetList(contract.CallIntolerable).requests))
}

func TestRequestList(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOld := gen.UniqueIDWithPulse(currentPulse)
	RefOld := reference.NewSelf(objectOld)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)
	objectOne := gen.UniqueIDWithPulse(nextPulseNumber)
	objectTwo := gen.UniqueIDWithPulse(nextPulseNumber)
	RefOne := reference.NewSelf(objectOne)
	RefTwo := reference.NewSelf(objectTwo)

	rl := NewRequestList()
	require.Equal(t, 0, rl.Count())
	require.Equal(t, 0, rl.CountFinish())
	require.Equal(t, pulse.Number(0), rl.EarliestPulse())

	require.Equal(t, true, rl.Add(RefOne))
	require.Equal(t, true, rl.Exist(RefOne))
	require.Equal(t, 1, rl.Count())
	require.Equal(t, nextPulseNumber, rl.EarliestPulse())

	require.Equal(t, false, rl.Add(RefOne))
	require.Equal(t, true, rl.Exist(RefOne))
	require.Equal(t, 1, rl.Count())
	require.Equal(t, nextPulseNumber, rl.EarliestPulse())

	require.Equal(t, true, rl.Add(RefOld))
	require.Equal(t, true, rl.Exist(RefOne))
	require.Equal(t, true, rl.Exist(RefOld))
	require.Equal(t, 2, rl.Count())
	require.Equal(t, pd.PulseNumber, rl.EarliestPulse())

	require.Equal(t, true, rl.Add(RefTwo))
	require.Equal(t, true, rl.Exist(RefOne))
	require.Equal(t, true, rl.Exist(RefOld))
	require.Equal(t, true, rl.Exist(RefTwo))
	require.Equal(t, 3, rl.Count())
	require.Equal(t, pd.PulseNumber, rl.EarliestPulse())
	require.Equal(t, 0, rl.CountFinish())

	rl.Finish(RefOne)
	require.Equal(t, pd.PulseNumber, rl.EarliestPulse()) // doesn't change
	require.Equal(t, 1, rl.CountFinish())
	require.Equal(t, 2, rl.CountActive())


	// try to finish ref that not in list
	successFinish := rl.Finish(reference.NewSelf(gen.UniqueIDWithPulse(currentPulse)))
	require.Equal(t, false, successFinish)
}

func TestRequestList_Finish(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOne := gen.UniqueIDWithPulse(currentPulse)
	RefOne := reference.NewSelf(objectOne)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)

	objectTwo := gen.UniqueIDWithPulse(nextPulseNumber)
	RefTwo := reference.NewSelf(objectTwo)

	rl := NewRequestList()

	require.Equal(t, true, rl.Add(RefOne))
	require.Equal(t, 1, rl.Count())
	require.Equal(t, currentPulse, rl.earliestPulse)

	require.Equal(t, true, rl.Add(RefTwo))
	require.Equal(t, 2, rl.Count())
	require.Equal(t, currentPulse, rl.earliestPulse)
	require.Equal(t, 0, rl.CountFinish())

	rl.Finish(RefOne)
	require.Equal(t, 1, rl.CountFinish())
	require.Equal(t, nextPulseNumber, rl.earliestPulse)
}
