// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bundle

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewWriter(snap SnapshotWriter) Writer {
	if snap == nil {
		panic(throw.IllegalValue())
	}
	return &bundleWriter{snap: snap}
}

var _ Writer = &bundleWriter{}

// bundleWriter implements necessary synchronization guarantees for the snapshot writer.
type bundleWriter struct {
	mutex sync.Mutex
	snap  SnapshotWriter

	lastReady synckit.SignalChannel
}

func (p *bundleWriter) WaitWriteBundles(done synckit.SignalChannel) bool {
	p.mutex.Lock()
	prev, next := p._prepareWait(false)
	p.mutex.Unlock()

	if next == nil {
		// there is nothing to wait: (1) no writers (2) all done (3) already rolled back
		return true
	}

	select {
	case _, ok := <- prev:
		if ok {
			// propagate status properly as someone can be after this wait operation, hence can be affected by rollback
			next <- struct{}{}
		}
		close(next)
		return true
	case <- done:
		// make sure that commit/rollback status will be propagated properly
		go propagateReady(prev, next)
		return false
	}
}

func propagateReady(prev synckit.SignalChannel, next synckit.ClosableSignalChannel) {
	_, ok := <- prev
	if ok {
		next <- struct{}{}
	}
	close(next)
}

func (p *bundleWriter) _prepareWait(alwaysSetNext bool) (prev synckit.SignalChannel, next synckit.ClosableSignalChannel) {
	prev = p.lastReady
	p.lastReady = nil

	select {
	case <- prev:
		// ignore prev if can read it - it is either ok, or was rolled back completely
		prev = nil
	default:
		// prev hasn't finished yet
	}

	if alwaysSetNext || prev != nil {
		next = make(chan struct{}, 1)
		p.lastReady = next
	}

	return prev, next
}

func (p *bundleWriter) WriteBundle(bundle Writeable, completedFn ResultFunc) error {
	if completedFn == nil {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	snapshot := p.snap.TakeSnapshot()
	defer func() {
		if snapshot != nil {
			snapshot.Rollback()
		}
	}()

	if err := bundle.PrepareWrite(snapshot); err != nil {
		return err
	}
	if err := snapshot.Prepared(); err != nil {
		return err
	}

	prev, next := p._prepareWait(true)
	go p.applyBundleSafe(snapshot, bundle, prev, next, completedFn)

	snapshot = nil
	return nil
}

func (p *bundleWriter) applyBundleSafe(snapshot Snapshot, bundle Writeable,
	prev synckit.SignalChannel, next synckit.ClosableSignalChannel, completedFn ResultFunc,
) {
	defer close(next) // to be executed as the very last one

	chained := false
	locked  := false
	rollback := true

	defer func() {
		switch {
		case !rollback:
			if locked {
				p.mutex.Unlock()
			}
			return
		case !locked:
			p.mutex.Lock()
		}
		defer p.mutex.Unlock()

		if chained {
			snapshot.ChainedRollback()
		} else {
			snapshot.Rollback()
		}
	}()

	err := func() (err error) {
		defer func() {
			err = throw.RW(recover(), err, "applyBundle failed")
		}()

		select {
		case _, ok := <-prev:
			if !ok {
				chained = true // rollback was made by a previous writer
				return throw.E("chained cancel")
			}
			prev = nil // previous writer is done, no need to wait for it further
		default:
			// don't wait, just check. Also handles prev==nil
		}

		var assignments []ledger.DirectoryIndex
		assignments, err = bundle.ApplyWrite()

		if err = snapshot.Completed(); err != nil {
			return err
		}

		if prev != nil {
			if _, ok := <-prev; !ok {
				chained = true // rollback was made by a previous writer and we have to stop
				return throw.E("chained cancel")
			}
		}

		p.mutex.Lock()
		locked = true

		if !completedFn(assignments, nil) {
			return nil // abort
		}

		rollback = false
		if err = snapshot.Commit(); err != nil {
			return err
		}

		p.mutex.Unlock()
		locked = false

		next <- struct{}{} // send ok to next
		return nil
	}()


	if err != nil {
		if !locked {
			p.mutex.Lock()
			locked = true
		}
		completedFn(nil, err)
	}
}

