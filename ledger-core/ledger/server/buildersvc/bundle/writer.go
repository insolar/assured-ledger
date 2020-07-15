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

func (p *bundleWriter) WaitWriteBundles(done synckit.SignalChannel, fn func(bool)) bool {
	p.mutex.Lock()
	prev, next := p._prepareWait(fn != nil)
	p.mutex.Unlock()

	if next == nil {
		// there is nothing to wait: (1) no writers (2) all done (3) already rolled back
		// and there is no fn to call
		return true
	}

	ok := false
	if prev != nil {
		select {
		case _, ok = <- prev:
		case <- done:
			// make sure that commit/rollback status will be propagated properly
			go propagateReady(prev, next)
			if fn != nil {
				fn(false)
			}
			return false
		}
	}

	defer func() {
		if ok {
			// propagate status properly as someone can be after this wait operation,
			// hence can be affected by rollback
			next <- struct{}{}
		}
		close(next)
	}()

	if fn != nil {
		fn(true)
	}
	return true
}

func (p *bundleWriter) MarkReadOnly() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.snap.MarkReadOnly()
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

	snapshot, err := p.snap.TakeSnapshot()
	if err != nil {
		return err
	}

	defer func() {
		if snapshot != nil {
			snapshot.Rollback(false)
		}
	}()

	if err = bundle.PrepareWrite(snapshot); err != nil {
		return err
	}
	if err = snapshot.Prepared(); err != nil {
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

	locked := false
	chained := false
	rollback := true
	var err error

	waitPrev := func(blocking bool) bool {
		if prev == nil {
			return true
		}

		ok := false
		if blocking {
			_, ok = <-prev
		} else {
			select {
			case _, ok = <-prev:
			default:
				return true
			}
		}

		prev = nil
		if !ok {
			chained = true
			return false
		}
		return true
	}

	defer func() {
		switch {
		case !rollback:
			if locked {
				p.mutex.Unlock()
			}
			return
		case !locked:
			// this makes sure that rollbacks respect sequence.
			// and as chan is always closed, then it is not relevant if we've read it before or not.
			waitPrev(true)

			p.mutex.Lock()
		}
		defer p.mutex.Unlock()

		defer func() {
			_ = recover() // we can't allow panic here
		}()
		snapshot.Rollback(chained)
		if err != nil {
			completedFn(nil, err)
		}
	}()

	err = func() (err error) {
		defer func() {
			err = throw.RW(recover(), err, "applyBundle failed")
		}()

		if !waitPrev(false) {
			return throw.E("chained cancel")
		}

		var assignments []ledger.DirectoryIndex
		assignments, err = bundle.ApplyWrite()

		if err = snapshot.Completed(); err != nil {
			return err
		}

		if !waitPrev(true) {
			return throw.E("chained cancel")
		}

		p.mutex.Lock()
		locked = true

		if !completedFn(assignments, nil) {
			return nil // no error, but rollback
		}
		if err = snapshot.Commit(); err != nil {
			return err
		}
		rollback = false
		return nil
	}()

	if err != nil || rollback {
		return
	}

	if !locked {
		panic(throw.Impossible())
	}

	// have to unlock first, to avoid lock contention and to allow a next writer
	// to get the lock immediately after releasing of the chan
	p.mutex.Unlock()
	locked = false
	next <- struct{}{} // send ok to next
}

