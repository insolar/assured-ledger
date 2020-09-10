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

func (p *bundleWriter) WaitWriteBundlesAsync(cancel synckit.SignalChannel, fn func(cancelled bool)) {
	if fn != nil {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	prev, next := p._prepareWait(true)
	p.mutex.Unlock()

	go waitWriteBundles(cancel, fn, prev, next)
}

func (p *bundleWriter) WaitWriteBundles(cancel synckit.SignalChannel, fn func(cancelled bool)) bool {
	p.mutex.Lock()
	prev, next := p._prepareWait(fn != nil)
	p.mutex.Unlock()

	if next == nil {
		// there is nothing to wait: (1) no writers (2) all done (3) already rolled back
		// and there is no fn call

		return true
	}

	return waitWriteBundles(cancel, fn, prev, next)
}

func waitWriteBundles(cancel synckit.SignalChannel, fn func(cancelled bool), prev synckit.SignalChannel, next synckit.ClosableSignalChannel) bool {
	ok := false
	if prev != nil {
		select {
		case _, ok = <- prev:
		case <- cancel:
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

func (p *bundleWriter) WriteBundle(bundle Writeable, completedFn ResultFunc) (errResult error) {
	switch {
	case completedFn == nil:
		panic(throw.IllegalValue())
	case bundle == nil:
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	snapshot, err := p.snap.TakeSnapshot()
	if err != nil {
		return err
	}

	defer func() {
		if snapshot == nil {
			return
		}
		if err := snapshot.Rollback(false); err != nil {
			errResult = throw.WithDetails(err, errResult)
		}
		bundle.ApplyRollback()
	}()

	if err = bundle.PrepareWrite(snapshot); err != nil {
		return err
	}
	if err = snapshot.Prepared(); err != nil {
		return err
	}

	prev, next := p._prepareWait(true)
	go p.applyBundleSafely(snapshot, bundle, prev, next, completedFn)

	snapshot = nil
	return nil
}

// waitPrevBundle will ONLY return correct (rollbackRequested) for the first call. Consequent calls must ignore result.
func waitPrevBundle(prev synckit.SignalChannel) (rollbackRequested bool) {
	if prev == nil {
		return false
	}
	_, ok := <-prev
	return !ok
}

// checkPrevBundle is a non-wait version of waitPrevBundle. Same limitations apply to (rollbackRequested).
func checkPrevBundle(prev synckit.SignalChannel) (isDone, rollbackRequested bool) {
	if prev == nil {
		return true, false
	}
	select {
	case _, ok := <-prev:
		return true, !ok
	default:
		return false, false
	}
}

func (p *bundleWriter) applyBundleSafely(snapshot Snapshot, bundle Writeable,
	prev synckit.SignalChannel, next synckit.ClosableSignalChannel, completedFn ResultFunc,
) {
	defer close(next) // to be executed as the very last one

	locked := false  // an explicit mark that the lock is active
	chained := false // a mark that rollback was triggered by a previous bundle
	rollback := true // a mark that rollback is required
	var err error

	defer func() {
		// on exit we must check if rollback is needed

		switch {
		case !rollback:
			if locked {
				// this branch is not in use now, yet it is for a precaution for later changes
				p.mutex.Unlock()
			}
			return
		case !locked:
			// this makes sure that rollbacks respect sequence.
			// and as chan is always closed, then it is not relevant if we've read it before or not.
			// WARNING! prev must be set to nil if it was read before, otherwise (rollbackRequested) may be incorrect.
			if rollbackRequested := waitPrevBundle(prev); rollbackRequested {
				// DO NOT set (chained) to false here
				chained = true
			}
			p.mutex.Lock()
		}
		defer p.mutex.Unlock()

		defer func() {
			_ = recover() // we can't allow panic here
		}()

		defer func() {
			if err != nil {
				completedFn(nil, err)
			}
		}()

		if err2 := snapshot.Rollback(chained); err2 != nil {
			err = throw.WithDetails(err2, err)
		}

		bundle.ApplyRollback()
	}()

	err = func() (err error) {
		defer func() {
			err = throw.RW(recover(), err, "applyBundle failed")
		}()

		// a quick check of prev bundle
		switch isDone, rollbackRequested := checkPrevBundle(prev); {
		case rollbackRequested:
			// if it is rolled back, then there is no reason to apply this bundle
			chained = true
			return throw.E("chained cancel")
		case isDone:
			// it was successfully applied
			// so we don't need to wait afterwards
			prev = nil
		}

		var assignments []ledger.DirectoryIndex
		assignments, err = bundle.ApplyWrite()

		if err = snapshot.Completed(); err != nil {
			return err
		}

		// can only proceed after the prev bundle is ready
		if rollbackRequested := waitPrevBundle(prev); rollbackRequested {
			// if it is rolled back, then there is no reason to commit this bundle
			chained = true
			return throw.E("chained cancel")
		}
		// must be unset to avoid false-positive from waitPrevBundle inside defer
		prev = nil

		// have to acquire the lock to ensure proper sequencing on commit and on callback
		p.mutex.Lock()
		locked = true

		// call the callback first as it may fail or request a rollback
		if !completedFn(assignments, nil) {
			// rollback was requested by the callback
			// so return no error, but keep rollback==true
			return nil
		}
		if err = snapshot.Commit(); err != nil {
			// rollback will be applied
			return err
		}
		rollback = false // no need to rollback after successful commit
		return nil
	}()

	if err != nil || rollback {
		return
	}

	if !locked {
		// something was broken - can't get here without a lock
		panic(throw.Impossible())
	}

	// have to unlock first, to avoid lock contention and to allow a next writer
	// to get the lock immediately after releasing of the chan
	p.mutex.Unlock()
	locked = false
	next <- struct{}{} // send ok to next
}

