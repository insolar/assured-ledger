// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package retries

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func L(args ...int) []RetryID {
	r := make([]RetryID, len(args))
	for i, a := range args {
		r[i] = RetryID(a)
	}
	return r
}

func isDone(v RetryID) RetryState {
	switch {
	case v == 0:
		return StopRetrying
	case int(v) < 0:
		return RemoveCompletely
	default:
		return KeepRetrying
	}
}

func R(a, b int) []int {
	return []int{a, b}
}

func TestSegregate_Simple(t *testing.T) {
	list := L()
	require.Equal(t, R(0, 0), R(Segregate(list, isDone)))

	list = L(1, 2, 3, 4)
	require.Equal(t, R(4, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(1, 2, 3, 4), list)

	list = L(-1, -2, -3, -4)
	require.Equal(t, R(0, 0), R(Segregate(list, isDone)))
	require.Equal(t, L(-1, -2, -3, -4), list)

	list = L(0, 0, 0, 0)
	require.Equal(t, R(0, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(0, 0, 0, 0), list)

	list = L(1, 2, 0, 0, -5, -6)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(1, 2, 0, 0, -5, -6), list)
}

func TestSegregate_KeepStop(t *testing.T) {
	list := L(1, 2, 3, 0)
	require.Equal(t, R(3, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(1, 2, 3, 0), list)

	list = L(1, 2, 0, 4)
	require.Equal(t, R(3, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(1, 2, 4), list[:3])

	list = L(1, 0, 3, 4)
	require.Equal(t, R(3, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(1, 3, 4), list[:3])

	list = L(0, 2, 3, 4)
	require.Equal(t, R(3, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(2, 3, 4), list[:3])

	list = L(0, 2, 3, 0)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(2, 3), list[:2])

	list = L(0, 2, 0, 4)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(2, 4), list[:2])

	list = L(0, 0, 3, 4)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(3, 4), list[:2])
}

func TestSegregate_RemoveStop(t *testing.T) {
	list := L(-1, -2, -3, 0)
	require.Equal(t, R(0, 1), R(Segregate(list, isDone)))
	require.Equal(t, L(-1, -2, -3), list[1:])

	list = L(-1, -2, 0, -4)
	require.Equal(t, R(0, 1), R(Segregate(list, isDone)))
	require.Equal(t, L(-1, -2, -4), list[1:])

	list = L(-1, 0, -3, -4)
	require.Equal(t, R(0, 1), R(Segregate(list, isDone)))
	require.Equal(t, L(-1, -3, -4), list[1:])

	list = L(0, -2, -3, -4)
	require.Equal(t, R(0, 1), R(Segregate(list, isDone)))
	require.Equal(t, L(-2, -3, -4), list[1:])

	list = L(0, -2, -3, 0)
	require.Equal(t, R(0, 2), R(Segregate(list, isDone)))
	require.Equal(t, L(-2, -3), list[2:])

	list = L(0, -2, 0, -4)
	require.Equal(t, R(0, 2), R(Segregate(list, isDone)))
	require.Equal(t, L(-2, -4), list[2:])

	list = L(-1, 0, -3, 0)
	require.Equal(t, R(0, 2), R(Segregate(list, isDone)))
	require.Equal(t, L(-1, -3), list[2:])

	list = L(-1, 0, 0, -4)
	require.Equal(t, R(0, 2), R(Segregate(list, isDone)))
	require.Equal(t, L(-1, -4), list[2:])
}

func TestSegregate_Mixed(t *testing.T) {
	list := L(-1, -2, 0, 0, 5, 6)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(6, 5), list[:2])
	require.Equal(t, L(-2, -1), list[4:])

	list = L(-1, 2, 0, 0, -5, 6)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(6, 2), list[:2])
	require.Equal(t, L(-5, -1), list[4:])

	list = L(1, -2, 0, 0, -5, 6)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(1, 6), list[:2])
	require.Equal(t, L(-5, -2), list[4:])

	list = L(-1, 2, 0, 0, 5, -6)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(5, 2), list[:2])
	require.Equal(t, L(-1, -6), list[4:])

	list = L(0, -1, 2, 0, 5, -6)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(5, 2), list[:2])
	require.Equal(t, L(-1, -6), list[4:])

	list = L(-1, 2, 0, 5, -6, 0)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(5, 2), list[:2])
	require.Equal(t, L(-1, -6), list[4:])

	list = L(0, -1, 2, 5, -6, 0)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(5, 2), list[:2])
	require.Equal(t, L(-1, -6), list[4:])

	list = L(0, 1, -2, -5, 6, 0)
	require.Equal(t, R(2, 4), R(Segregate(list, isDone)))
	require.Equal(t, L(1, 6), list[:2])
	require.Equal(t, L(-5, -2), list[4:])
}
