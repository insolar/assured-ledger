// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bundle

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
)

func TestWriter(t *testing.T) {
	sw := NewSnapshotWriterMock(t)

	snap := NewSnapshotMock(t)
	snap.PreparedMock.Return(nil)
	snap.CompletedMock.Return(nil)
	snap.CommitMock.Return(nil)

	sw.TakeSnapshotMock.Return(snap, nil)

	w := NewWriter(sw)

	start := sync.WaitGroup{}
	start.Add(1)
	mid := sync.WaitGroup{}
	mid.Add(1)
	applied := sync.WaitGroup{}
	applied.Add(3)

	wb1 := NewWriteableMock(t)
	wb1.PrepareWriteMock.Return(nil)
	wb1.ApplyWriteMock.Set(func() ([]ledger.DirectoryIndex, error) {
		start.Wait()
		applied.Done()
		return []ledger.DirectoryIndex{1}, nil
	})

	wb2 := NewWriteableMock(t)
	wb2.PrepareWriteMock.Return(nil)
	wb2.ApplyWriteMock.Set(func() ([]ledger.DirectoryIndex, error) {
		mid.Wait()
		return []ledger.DirectoryIndex{1}, nil
	})

	check := make(chan int, 5)

	w.WaitWriteBundles(nil, nil) // nothing to wait

	writeBundle(t, w, wb1, func() { check <- 1 })
	writeBundle(t, w, wb1, func() { check <- 2 })
	writeBundle(t, w, wb2, func() { check <- 3 })
	writeBundle(t, w, wb1, func() { check <- 4 })

	go func() {
		w.WaitWriteBundles(nil, nil)
		check <- 5
	}()

	select {
	case <- check:
		require.FailNow(t, "must not be done yet")
	default:
	}

	start.Done()
	require.Equal(t, 1, <- check)
	require.Equal(t, 2, <- check)

	applied.Wait()

	select {
	case <- check:
		require.FailNow(t, "must not be done yet")
	default:
	}

	mid.Done()
	require.Equal(t, 3, <- check)
	require.Equal(t, 4, <- check)
	require.Equal(t, 5, <- check)

	w.WaitWriteBundles(nil, nil) // nothing to wait
}

func writeBundle(t *testing.T, w Writer, wb Writeable, fn func()) {
	err := w.WriteBundle(wb, func(d []ledger.DirectoryIndex, err error) bool {
		require.Equal(t, []ledger.DirectoryIndex{1}, d)
		require.NoError(t, err)
		fn()
		return true
	})
	require.NoError(t, err)
}
