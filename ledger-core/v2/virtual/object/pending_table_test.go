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

func TestPendingTable(t *testing.T) {
	pt := NewRequestTable()

	require.Equal(t, 0, len(pt.GetList(contract.CallIntolerable).requests))
	require.Equal(t, 0, len(pt.GetList(contract.CallIntolerable).requests))

	require.Equal(t, pulse.Number(0), pt.GetList(contract.CallIntolerable).earliestPulse)
	require.Equal(t, pulse.Number(0), pt.GetList(contract.CallIntolerable).earliestPulse)

	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	object := gen.UniqueIDWithPulse(currentPulse)
	ref := reference.NewSelf(object)

	intolerableList := pt.GetList(contract.CallIntolerable)
	intolerableList.Add(ref)

	require.Equal(t, 1, len(pt.GetList(contract.CallIntolerable).requests))
}

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

	pl := NewRequestList()
	require.Equal(t, 0, pl.Count())
	require.Equal(t, 0, pl.CountFinish())
	require.Equal(t, pulse.Number(0), pl.EarliestPulse())

	require.Equal(t, true, pl.Add(RefOne))
	require.Equal(t, true, pl.Exist(RefOne))
	require.Equal(t, 1, pl.Count())
	require.Equal(t, nextPulseNumber, pl.EarliestPulse())

	require.Equal(t, false, pl.Add(RefOne))
	require.Equal(t, true, pl.Exist(RefOne))
	require.Equal(t, 1, pl.Count())
	require.Equal(t, nextPulseNumber, pl.EarliestPulse())

	require.Equal(t, true, pl.Add(RefOld))
	require.Equal(t, true, pl.Exist(RefOne))
	require.Equal(t, true, pl.Exist(RefOld))
	require.Equal(t, 2, pl.Count())
	require.Equal(t, pd.PulseNumber, pl.EarliestPulse())

	require.Equal(t, true, pl.Add(RefTwo))
	require.Equal(t, true, pl.Exist(RefOne))
	require.Equal(t, true, pl.Exist(RefOld))
	require.Equal(t, true, pl.Exist(RefTwo))
	require.Equal(t, 3, pl.Count())
	require.Equal(t, pd.PulseNumber, pl.EarliestPulse())
	require.Equal(t, 0, pl.CountFinish())

	pl.Finish(RefOne)
	require.Equal(t, pd.PulseNumber, pl.EarliestPulse()) // doesn't change
	require.Equal(t, 1, pl.CountFinish())
	require.Equal(t, 2, pl.CountActive())


	tryTooFinishRefThatNotInList := pl.Finish(reference.NewSelf(gen.UniqueIDWithPulse(currentPulse)))
	require.Equal(t, false, tryTooFinishRefThatNotInList)
}

func TestPendingList_Finish(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOne := gen.UniqueIDWithPulse(currentPulse)
	RefOne := reference.NewSelf(objectOne)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)

	objectTwo := gen.UniqueIDWithPulse(nextPulseNumber)
	RefTwo := reference.NewSelf(objectTwo)

	pt := NewRequestList()

	require.Equal(t, true, pt.Add(RefOne))
	require.Equal(t, 1, pt.Count())
	require.Equal(t, currentPulse, pt.earliestPulse)

	require.Equal(t, true, pt.Add(RefTwo))
	require.Equal(t, 2, pt.Count())
	require.Equal(t, currentPulse, pt.earliestPulse)
	require.Equal(t, 0, pt.CountFinish())

	pt.Finish(RefOne)
	require.Equal(t, 1, pt.CountFinish())
	require.Equal(t, nextPulseNumber, pt.earliestPulse)
}
