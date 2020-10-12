// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package retries

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testStrategy struct {
	count   RetryID
	m       map[RetryID]RetryState
	retries [][]RetryID
}

func (p *testStrategy) Retry(ids []RetryID, repeatFn func(RetryID), _ func([]RetryID)) {
	p.retries = append(p.retries, ids)
	for _, id := range ids {
		repeatFn(id)
	}
}

func (p *testStrategy) CheckState(id RetryID) RetryState {
	if st, ok := p.m[id]; ok {
		return st
	}
	return RemoveCompletely
}

func (p *testStrategy) Remove(ids []RetryID) {
	for _, id := range ids {
		delete(p.m, id)
	}
}

func (p *testStrategy) Next() RetryID {
	p.count++
	p.m[p.count] = KeepRetrying
	return p.count
}

func Ln(start, end RetryID) []RetryID {
	r := make([]RetryID, end-start+1)
	for i := range r {
		r[i] = start + RetryID(i)
	}
	return r
}

func TestStagedController(t *testing.T) {
	ts := &testStrategy{m: make(map[RetryID]RetryState)}
	sc := StagedController{}
	sc.InitStages(1000, [...]int{10, 50, 200})

	for i := 10; i > 0; i-- {
		require.Zero(t, len(ts.retries), i)
		sc.Add(ts.Next(), 100, ts)
	}

	for k, n := range []int{10, 40, 150, 200, 200, 200, 200} {
		sc.NextCycle(ts)
		require.Equal(t, 1, len(ts.retries), "%d(%d)", k, n)
		require.Equal(t, Ln(1, 10), ts.retries[0])
		ts.retries = nil

		for i := n - 1; i > 0; i-- {
			sc.NextCycle(ts)
			require.Zero(t, len(ts.retries), "%d/%d", i, n)
		}
	}
}
