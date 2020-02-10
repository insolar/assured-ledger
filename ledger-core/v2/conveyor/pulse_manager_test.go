//
// Copyright 2020 Insolar Network Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package conveyor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

func TestPulseDataManager_Init(t *testing.T) {
	t.Run("bad input", func(t *testing.T) {
		require.PanicsWithValue(t, "illegal value: minCachePulseAge 0", func() {
			pdm := PulseDataManager{}
			pdm.Init(0, 0, 0)
		})
		require.PanicsWithValue(t, fmt.Sprintf("illegal value: minCachePulseAge %d", uint32(pulse.MaxTimePulse)+1),
			func() {
				pdm := PulseDataManager{}
				pdm.Init(uint32(pulse.MaxTimePulse)+1, 0, 0)
			})
		require.PanicsWithValue(t, "illegal value: maxPastPulseAge 0", func() {
			pdm := PulseDataManager{}
			pdm.Init(1, 0, 0)
		})
		require.PanicsWithValue(t, fmt.Sprintf("illegal value: maxPastPulseAge %d", uint32(pulse.MaxTimePulse)+1),
			func() {
				pdm := PulseDataManager{}
				pdm.Init(uint32(pulse.MaxTimePulse), uint32(pulse.MaxTimePulse)+1, 0)
			})
	})

	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 10, 0)
	})
}

func TestPulseDataManager_GetPresentPulse(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 10, 0)
		presentPulse, nearestFuture := pdm.GetPresentPulse()
		assert.Equal(t, pulse.Unknown, presentPulse)
		assert.Equal(t, uninitializedFuture, nearestFuture)
	})

	t.Run("present", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 10, 0)
		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulse.MinTimePulse + 1,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulse.MinTimePulse + 1),
				NextPulseDelta: 2,
			},
		})
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
		pdm.Init(1, 10, 0)
		_, has := pdm.GetPulseData(1)
		assert.False(t, has)
	})
	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(pulse.MinTimePulse, pulse.MinTimePulse+10, 0)

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
		pdm.putPulseData(expected)
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
		pdm.Init(1, 10, 0)
		assert.False(t, pdm.HasPulseData(1))
	})
	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(pulse.MinTimePulse, pulse.MinTimePulse+10, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		assert.False(t, pdm.HasPulseData(pulseNum))
		pdm.putPulseData(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		})
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
		pdm.Init(1, 2, 0)
		assert.False(t, pdm.IsAllowedFutureSpan(1))
	})
	t.Run("0 max future", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 2, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		})
		assert.True(t, pdm.IsAllowedFutureSpan(pulseNum+2))
	})
	t.Run("1 max future", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 2, 1)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		})
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
		pdm.Init(1, 10, 0)
		assert.False(t, pdm.IsAllowedPastSpan(1))
	})
	t.Run("2 past", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 2, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		})
		assert.True(t, pdm.IsAllowedPastSpan(pulseNum-1))
	})
	t.Run("5 past", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 5, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		})
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
		pdm.Init(1, 10, 0)
		assert.False(t, pdm.IsRecentPastRange(1))
	})
	t.Run("2 past", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 2, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulse.MinTimePulse,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulse.MinTimePulse),
				NextPulseDelta: 1,
			},
		})

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		})
		assert.True(t, pdm.IsRecentPastRange(pulseNum-1))
	})
	t.Run("5 past", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 5, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulse.MinTimePulse,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulse.MinTimePulse),
				NextPulseDelta: 1,
			},
		})

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		})

		pdm.setPresentPulse(pulse.Data{
			PulseNumber: pulseNum + 2,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum + 2),
				NextPulseDelta: 2,
			},
		})
		assert.False(t, pdm.IsRecentPastRange(pulse.MinTimePulse))
	})
}

func TestPulseDataManager_PreparePulseDataRequest(t *testing.T) {
	t.Run("bad input", func(t *testing.T) {
		require.PanicsWithValue(t, "illegal value: empty resultFn", func() {
			pdm := PulseDataManager{}
			pdm.Init(1, 10, 0)
			pdm.PreparePulseDataRequest(nil, 1, nil)
		})
	})

	t.Run("ok", func(t *testing.T) {
		t.Skip("PLAT-84: could not be tested for now")
		pdm := PulseDataManager{}
		pdm.Init(1, 10, 0)
		pulseNum := pulse.Number(pulse.MinTimePulse + 1)
		expected := pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		}
		pdm.putPulseData(expected)

		async := pdm.PreparePulseDataRequest(nil, pulseNum, func(isAvailable bool, pd pulse.Data) {
			require.True(t, isAvailable)
			require.Equal(t, expected, pd)
		})

		async.Start()
	})
}

func TestPulseDataManager_TouchPulseData(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 10, 0)
		assert.False(t, pdm.TouchPulseData(1))
	})
	t.Run("double", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(1, 10, 0)
		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		assert.False(t, pdm.TouchPulseData(pulseNum))
		assert.False(t, pdm.TouchPulseData(pulseNum))
	})
	t.Run("ok", func(t *testing.T) {
		pdm := PulseDataManager{}
		pdm.Init(pulse.MinTimePulse, pulse.MinTimePulse+10, 0)

		pulseNum := pulse.Number(pulse.MinTimePulse + 1)

		assert.False(t, pdm.TouchPulseData(pulseNum))
		pdm.putPulseData(pulse.Data{
			PulseNumber: pulseNum,
			DataExt: pulse.DataExt{
				PulseEpoch:     pulse.Epoch(pulseNum),
				NextPulseDelta: 2,
			},
		})
		assert.True(t, pdm.TouchPulseData(pulseNum))
	})
}
