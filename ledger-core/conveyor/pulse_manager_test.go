// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func TestPulseDataManager_Init(t *testing.T) {
	t.Run("bad input", func(t *testing.T) {
		assert.PanicsWithValue(t, "illegal value", func() {
			pdm := PulseDataManager{}
			pdm.initCache(0, 0, 0)
		})
		assert.PanicsWithValue(t, "illegal value", func() {
			pdm := PulseDataManager{}
			pdm.initCache(uint32(pulse.MaxTimePulse)+1, 0, 0)
		})
		assert.PanicsWithValue(t, "illegal value", func() {
			pdm := PulseDataManager{}
			pdm.initCache(1, 0, 0)
		})
		assert.PanicsWithValue(t, "illegal value", func() {
			pdm := PulseDataManager{}
			pdm.initCache(uint32(pulse.MaxTimePulse), uint32(pulse.MaxTimePulse)+1, 0)
		})
	})

	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
	})
}

func TestPulseDataManager_GetPresentPulse(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		presentPulse, nearestFuture := pdm.GetPresentPulse()
		assert.Equal(t, pulse.Unknown, presentPulse)
		assert.Equal(t, uninitializedFuture, nearestFuture)
	})

	t.Run("present", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulse.MinTimePulse + 1,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulse.MinTimePulse + 1),
				NextPulseDelta: 2,
			},
		}.AsRange())
		presentPulse, nearestFuture := pdm.GetPresentPulse()
		assert.Equal(t, pulse.Number(pulse.MinTimePulse+1), presentPulse)
		assert.Equal(t, pulse.Number(pulse.MinTimePulse+3), nearestFuture)
	})

}

func TestPulseDataManager_GetPulseData(t *testing.T) {
	t.Run("uninitialized", func(t *testing.T) {
		pdm := PulseDataManager{}
		_, has := pdm.GetPulseData(1)
		assert.False(t, has)
	})
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		_, has := pdm.GetPulseData(1)
		assert.False(t, has)
	})
	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(pulse.MinTimePulse, pulse.MinTimePulse+10, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)
		expected := pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}

		_, has := pdm.GetPulseData(pulseNum)
		assert.False(t, has)
		pdm.putPulseUpdate(BeatData{ Range: expected.AsRange() })
		data, has := pdm.GetPulseData(pulseNum)
		assert.True(t, has)
		assert.Equal(t, expected, data)
	})
}

func TestPulseDataManager_HasPulseData(t *testing.T) {
	t.Run("uninitialized", func(t *testing.T) {
		pdm := PulseDataManager{}
		assert.False(t, pdm.HasPulseData(1))
	})
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		assert.False(t, pdm.HasPulseData(1))
	})
	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(pulse.MinTimePulse, pulse.MinTimePulse+10, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		assert.False(t, pdm.HasPulseData(pulseNum))
		pdm.putPulseUpdate(BeatData{ Range: pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}.AsRange()})
		assert.True(t, pdm.HasPulseData(pulse.MinTimePulse+1))
		assert.False(t, pdm.HasPulseData(1))
	})
}

func TestPulseDataManager_IsAllowedFutureSpan(t *testing.T) {
	t.Run("uninitialized", func(t *testing.T) {
		pdm := PulseDataManager{}
		assert.False(t, pdm.IsAllowedFutureSpan(1))
	})
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 2, 0)
		assert.False(t, pdm.IsAllowedFutureSpan(1))
	})
	t.Run("0 max future", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 2, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}.AsRange())
		assert.True(t, pdm.IsAllowedFutureSpan(pulseNum+2))
	})
	t.Run("1 max future", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 2, 1)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}.AsRange())
		assert.False(t, pdm.IsAllowedFutureSpan(pulseNum+pulse.MinTimePulse+1))
	})
}

func TestPulseDataManager_IsAllowedPastSpan(t *testing.T) {
	t.Run("uninitialized", func(t *testing.T) {
		pdm := PulseDataManager{}
		assert.False(t, pdm.IsAllowedPastSpan(1))
	})
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		assert.False(t, pdm.IsAllowedPastSpan(1))
	})
	t.Run("2 past", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 2, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}.AsRange())
		assert.True(t, pdm.IsAllowedPastSpan(pulseNum-1))
	})
	t.Run("5 past", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 5, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}.AsRange())
		assert.False(t, pdm.IsAllowedPastSpan(pulseNum+1))
	})
}

func TestPulseDataManager_IsRecentPastRange(t *testing.T) {
	t.Run("uninitialized", func(t *testing.T) {
		pdm := PulseDataManager{}
		assert.False(t, pdm.IsRecentPastRange(1))
	})
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		assert.False(t, pdm.IsRecentPastRange(1))
	})
	t.Run("2 past", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 2, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulse.MinTimePulse,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulse.MinTimePulse),
				NextPulseDelta: 1,
			},
		}.AsRange())

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}.AsRange())
		assert.True(t, pdm.IsRecentPastRange(pulseNum-1))
	})
	t.Run("5 past", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 5, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulse.MinTimePulse,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulse.MinTimePulse),
				NextPulseDelta: 1,
			},
		}.AsRange())

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}.AsRange())

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum + 2,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum + 2),
				NextPulseDelta: 2,
			},
		}.AsRange())
		assert.False(t, pdm.IsRecentPastRange(pulse.MinTimePulse))
	})
}

type dummyAsync struct{}

func (d dummyAsync) WithCancel(*context.CancelFunc) smachine.AsyncCallRequester {
	return d
}

func (d dummyAsync) WithNested(smachine.CreateFactoryFunc) smachine.AsyncCallRequester {
	return d
}

func (d dummyAsync) WithFlags(flags smachine.AsyncCallFlags) smachine.AsyncCallRequester {
	return d
}

func (d dummyAsync) WithoutAutoWakeUp() smachine.AsyncCallRequester {
	return d
}

func (d dummyAsync) WithLog(isLogging bool) smachine.AsyncCallRequester {
	return d
}

func (dummyAsync) Start() {
	return
}

func (dummyAsync) DelayedStart() smachine.CallConditionalBuilder {
	return nil
}

func TestPulseDataManager_PreparePulseDataRequest(t *testing.T) {
	t.Run("bad input", func(t *testing.T) {
		require.PanicsWithValue(t, "illegal value", func() {
			pdm := PulseDataManager{}
			pdm.initCache(1, 10, 0)
			pdm.pulseDataAdapterFn =
				func(smachine.ExecutionContext, func(context.Context, BeatDataService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
					return dummyAsync{}
				}
			pdm.preparePulseDataRequest(nil, 1, nil)
		})
	})

	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		pdm.pulseDataAdapterFn =
			func(smachine.ExecutionContext, func(context.Context, BeatDataService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
				return dummyAsync{}
			}
		pulseNum := pulse.Number(pulse.MinTimePulse + 1)
		expected := pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}
		pdm.putPulseUpdate(BeatData{expected.AsRange(), nil})

		async := pdm.preparePulseDataRequest(nil, pulseNum, func(bd BeatData) {
			require.NotNil(t, bd.Range)
			require.Equal(t, expected, bd.Range.RightBoundData())
		})

		async.Start()
	})
}

func TestPulseDataManager_TouchPulseData(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		assert.False(t, pdm.TouchPulseData(1))
	})
	t.Run("double", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(1, 10, 0)
		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		assert.False(t, pdm.TouchPulseData(pulseNum))
		assert.False(t, pdm.TouchPulseData(pulseNum))
	})
	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.initCache(pulse.MinTimePulse, pulse.MinTimePulse+10, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		assert.False(t, pdm.TouchPulseData(pulseNum))
		pdm.putPulseUpdate(BeatData{ Range: pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}.AsRange()})
		assert.True(t, pdm.TouchPulseData(pulseNum))
	})
}
