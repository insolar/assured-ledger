// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestWorkingTable(t *testing.T) {
	rt := NewWorkingTable()

	assert.Equal(t, 0, rt.GetList(contract.CallIntolerable).Count())
	assert.Equal(t, 0, len(rt.GetList(contract.CallTolerable).requests))

	assert.Equal(t, pulse.Number(0), rt.GetList(contract.CallIntolerable).earliestActivePulse)
	assert.Equal(t, pulse.Number(0), rt.GetList(contract.CallTolerable).earliestActivePulse)

	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	object := gen.UniqueLocalRefWithPulse(currentPulse)
	ref := reference.NewSelf(object)

	intolerableList := rt.GetList(contract.CallIntolerable)
	assert.True(t, intolerableList.Add(ref))

	assert.Equal(t, 1, rt.GetList(contract.CallIntolerable).Count())
	assert.Equal(t, 0, rt.GetList(contract.CallTolerable).Count())
	assert.Equal(t, pulse.Unknown, rt.GetList(contract.CallIntolerable).EarliestPulse())

	assert.True(t, intolerableList.SetActive(ref))

	assert.Equal(t, 1, rt.GetList(contract.CallIntolerable).Count())
	assert.Equal(t, 0, rt.GetList(contract.CallTolerable).Count())
	assert.Equal(t, currentPulse, rt.GetList(contract.CallIntolerable).EarliestPulse())
}

func TestWorkingList(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOld := gen.UniqueLocalRefWithPulse(currentPulse)
	RefOld := reference.NewSelf(objectOld)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)
	objectOne := gen.UniqueLocalRefWithPulse(nextPulseNumber)
	objectTwo := gen.UniqueLocalRefWithPulse(nextPulseNumber)
	RefOne := reference.NewSelf(objectOne)
	RefTwo := reference.NewSelf(objectTwo)

	rl := NewWorkingList()
	assert.Equal(t, 0, rl.Count())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 0, rl.CountActive())
	assert.Equal(t, pulse.Number(0), rl.EarliestPulse())

	assert.Equal(t, true, rl.Add(RefOne))
	assert.Equal(t, RequestStarted, rl.GetState(RefOne))
	assert.Equal(t, true, rl.SetActive(RefOne))
	assert.Equal(t, 1, rl.Count())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 1, rl.CountActive())
	assert.Equal(t, nextPulseNumber, rl.EarliestPulse())

	assert.Equal(t, false, rl.Add(RefOne))
	assert.Equal(t, false, rl.SetActive(RefOne))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOne))
	assert.Equal(t, 1, rl.Count())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 1, rl.CountActive())
	assert.Equal(t, nextPulseNumber, rl.EarliestPulse())

	assert.Equal(t, true, rl.Add(RefOld))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOne))
	assert.Equal(t, RequestStarted, rl.GetState(RefOld))
	assert.Equal(t, true, rl.SetActive(RefOld))
	assert.Equal(t, 2, rl.Count())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 2, rl.CountActive())
	assert.Equal(t, pd.PulseNumber, rl.EarliestPulse())

	assert.Equal(t, true, rl.Add(RefTwo))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOne))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOld))
	assert.Equal(t, RequestStarted, rl.GetState(RefTwo))
	assert.Equal(t, true, rl.SetActive(RefTwo))
	assert.Equal(t, 3, rl.Count())
	assert.Equal(t, pd.PulseNumber, rl.EarliestPulse())
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 3, rl.CountActive())

	expResult := payload.VCallResult{
		Caller: gen.UniqueGlobalRef(),
		Callee: gen.UniqueGlobalRef(),
	}

	rl.Finish(RefOne, &expResult)
	assert.Equal(t, pd.PulseNumber, rl.EarliestPulse()) // doesn't change
	assert.Equal(t, 1, rl.CountFinish())
	assert.Equal(t, 2, rl.CountActive())
	actResult, ok := rl.GetResult(RefOne)
	assert.True(t, ok)
	assert.Equal(t, &expResult, actResult)

	// try to finish ref that not in list
	successFinish := rl.Finish(reference.NewSelf(gen.UniqueLocalRefWithPulse(currentPulse)), nil)
	assert.Equal(t, false, successFinish)
	assert.Equal(t, 1, rl.CountFinish())
	assert.Equal(t, 2, rl.CountActive())
}

func TestWorkingList_Finish(t *testing.T) {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	currentPulse := pd.PulseNumber

	objectOne := gen.UniqueLocalRefWithPulse(currentPulse)
	RefOne := reference.NewSelf(objectOne)

	nextPulseNumber := currentPulse + pulse.Number(pd.NextPulseDelta)

	objectTwo := gen.UniqueLocalRefWithPulse(nextPulseNumber)
	RefTwo := reference.NewSelf(objectTwo)

	rl := NewWorkingList()

	assert.Equal(t, true, rl.Add(RefOne))
	assert.Equal(t, 1, rl.Count())
	assert.Equal(t, true, rl.SetActive(RefOne))
	assert.Equal(t, RequestProcessing, rl.GetState(RefOne))
	assert.Equal(t, currentPulse, rl.earliestActivePulse)
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 1, rl.CountActive())

	assert.Equal(t, true, rl.Add(RefTwo))
	assert.Equal(t, 2, rl.Count())
	assert.Equal(t, true, rl.SetActive(RefTwo))
	assert.Equal(t, RequestProcessing, rl.GetState(RefTwo))
	assert.Equal(t, currentPulse, rl.earliestActivePulse)
	assert.Equal(t, 0, rl.CountFinish())
	assert.Equal(t, 2, rl.CountActive())

	result := payload.VCallResult{
		Caller: gen.UniqueGlobalRef(),
		Callee: gen.UniqueGlobalRef(),
	}

	rl.Finish(RefOne, &result)

	assert.Equal(t, 1, rl.CountFinish())
	assert.Equal(t, 1, rl.CountActive())
	assert.Equal(t, nextPulseNumber, rl.earliestActivePulse)
	actResult, ok := rl.GetResult(RefOne)
	assert.True(t, ok)
	assert.Equal(t, &result, actResult)
}
