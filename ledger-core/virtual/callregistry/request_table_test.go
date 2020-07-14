// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package callregistry

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func BenchmarkPendingTable(b *testing.B) {
	var x PendingTable
	for i := 0; i < b.N; i++ {
		x = NewRequestTable()
	}
	x = x
}

func BenchmarkWorkingTable(b *testing.B) {
	var x WorkingTable
	for i := 0; i < b.N; i++ {
		x = NewWorkingTable()
	}
	x = x
}

func TestPendingTable(t *testing.T) {
	rt := NewRequestTable()

	require.Equal(t, 0, len(rt.GetList(contract.CallIntolerable).requests))
	require.Equal(t, 0, len(rt.GetList(contract.CallTolerable).requests))

	require.Equal(t, pulse.Number(0), rt.GetList(contract.CallIntolerable).earliestActivePulse)
	require.Equal(t, pulse.Number(0), rt.GetList(contract.CallTolerable).earliestActivePulse)

	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	object := gen.UniqueLocalRefWithPulse(currentPulse)
	ref := reference.NewSelf(object)

	intolerableList := rt.GetList(contract.CallIntolerable)
	intolerableList.Add(ref)

	require.Equal(t, 1, len(rt.GetList(contract.CallIntolerable).requests))
	require.Equal(t, 0, len(rt.GetList(contract.CallTolerable).requests))
	require.Equal(t, currentPulse, rt.GetList(contract.CallIntolerable).EarliestPulse())
}

func TestPendingList(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOld := gen.UniqueLocalRefWithPulse(currentPulse)
	RefOld := reference.NewSelf(objectOld)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)
	objectOne := gen.UniqueLocalRefWithPulse(nextPulseNumber)
	objectTwo := gen.UniqueLocalRefWithPulse(nextPulseNumber)
	RefOne := reference.NewSelf(objectOne)
	RefTwo := reference.NewSelf(objectTwo)

	rl := newRequestList()
	require.Equal(t, 0, rl.Count())
	require.Equal(t, 0, rl.CountFinish())
	require.Equal(t, 0, rl.CountActive())
	require.Equal(t, pulse.Number(0), rl.EarliestPulse())

	require.Equal(t, true, rl.Add(RefOne))
	require.Equal(t, true, rl.Exist(RefOne))
	require.Equal(t, 1, rl.Count())
	require.Equal(t, 0, rl.CountFinish())
	require.Equal(t, 1, rl.CountActive())
	require.Equal(t, nextPulseNumber, rl.EarliestPulse())

	require.Equal(t, false, rl.Add(RefOne))
	require.Equal(t, true, rl.Exist(RefOne))
	require.Equal(t, 1, rl.Count())
	require.Equal(t, 0, rl.CountFinish())
	require.Equal(t, 1, rl.CountActive())
	require.Equal(t, nextPulseNumber, rl.EarliestPulse())

	require.Equal(t, true, rl.Add(RefOld))
	require.Equal(t, true, rl.Exist(RefOne))
	require.Equal(t, true, rl.Exist(RefOld))
	require.Equal(t, 2, rl.Count())
	require.Equal(t, 0, rl.CountFinish())
	require.Equal(t, 2, rl.CountActive())
	require.Equal(t, pd.PulseNumber, rl.EarliestPulse())

	require.Equal(t, true, rl.Add(RefTwo))
	require.Equal(t, true, rl.Exist(RefOne))
	require.Equal(t, true, rl.Exist(RefOld))
	require.Equal(t, true, rl.Exist(RefTwo))
	require.Equal(t, 3, rl.Count())
	require.Equal(t, pd.PulseNumber, rl.EarliestPulse())
	require.Equal(t, 0, rl.CountFinish())
	require.Equal(t, 3, rl.CountActive())

	rl.Finish(RefOne)
	require.Equal(t, pd.PulseNumber, rl.EarliestPulse()) // doesn't change
	require.Equal(t, 1, rl.CountFinish())
	require.Equal(t, 2, rl.CountActive())

	// try to finish ref that not in list
	successFinish := rl.Finish(reference.NewSelf(gen.UniqueLocalRefWithPulse(currentPulse)))
	require.Equal(t, false, successFinish)
	require.Equal(t, 1, rl.CountFinish())
	require.Equal(t, 2, rl.CountActive())
}

func TestPendingList_Finish(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOne := gen.UniqueLocalRefWithPulse(currentPulse)
	RefOne := reference.NewSelf(objectOne)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)

	objectTwo := gen.UniqueLocalRefWithPulse(nextPulseNumber)
	RefTwo := reference.NewSelf(objectTwo)

	rl := newRequestList()

	require.Equal(t, true, rl.Add(RefOne))
	require.Equal(t, 1, rl.Count())
	require.Equal(t, currentPulse, rl.earliestActivePulse)
	require.Equal(t, 0, rl.CountFinish())
	require.Equal(t, 1, rl.CountActive())

	require.Equal(t, true, rl.Add(RefTwo))
	require.Equal(t, 2, rl.Count())
	require.Equal(t, currentPulse, rl.earliestActivePulse)
	require.Equal(t, 0, rl.CountFinish())
	require.Equal(t, 2, rl.CountActive())

	rl.Finish(RefOne)
	require.Equal(t, 1, rl.CountFinish())
	require.Equal(t, 1, rl.CountActive())
	require.Equal(t, nextPulseNumber, rl.earliestActivePulse)
}

func TestPendingList_MustGetIsActive(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOne := gen.UniqueLocalRefWithPulse(currentPulse)
	RefOne := reference.NewSelf(objectOne)

	rl := newRequestList()
	isActive, exist := rl.GetState(RefOne)
	require.Equal(t, false, isActive)
	require.Equal(t, false, exist)

	rl.Add(RefOne)
	isActive, exist = rl.GetState(RefOne)
	require.Equal(t, true, isActive)
	require.Equal(t, true, exist)

	rl.Finish(RefOne)
	isActive, exist = rl.GetState(RefOne)
	require.Equal(t, false, isActive)
	require.Equal(t, true, exist)
}
