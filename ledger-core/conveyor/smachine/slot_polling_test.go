package smachine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGrowPollingSlots(t *testing.T) {
	q := PollingQueue{}
	tmStart := time.Now()

	tm := tmStart
	q.PrepareFor(tm)
	n := len(q.polls)
	require.NotZero(t, n)

	for j := 33; j > 0; j-- {
		for i := range q.polls {
			require.NotNil(t, q.polls[i])
		}
		q.AddToLatest(&Slot{})
		tm = tm.Add(1)
		q.PrepareFor(tm)
	}
	require.GreaterOrEqual(t, len(q.polls), 33)

	tmRelease := tmStart
	removeCount := 0
	for j := 11; j > 0; j-- {
		q.FilterOut(tmRelease, func(sq *SlotQueue) {
			require.NotNil(t, sq)
			require.False(t, sq.IsEmpty())
			require.Equal(t, 1, sq.Count())
			sq.RemoveAll()
			require.True(t, sq.IsEmpty())
			removeCount++
		})
		tmRelease = tmRelease.Add(1)
	}
	require.Equal(t, 11, removeCount)

	for j := 33; j > 0; j-- {
		for i := range q.polls {
			require.NotNil(t, q.polls[i])
		}
		q.AddToLatest(&Slot{})

		tm = tm.Add(1)
		q.PrepareFor(tm)
	}
}
