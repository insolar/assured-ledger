// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package synckit

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewSemaphore(limit int) Semaphore {
	if limit <= 0 {
		panic(throw.IllegalValue())
	}
	return Semaphore{make(chan struct{}, limit)}
}

type Semaphore struct {
	sema chan struct{}
}

func (v Semaphore) Lock() {
	v.sema <- struct{}{}
}

func (v Semaphore) TryLock() bool {
	select {
	case v.sema <- struct{}{}:
		return true
	default:
		return false
	}
}

func (v Semaphore) IsFull() bool {
	return len(v.sema) == cap(v.sema)
}

func (v Semaphore) LockTimeout(d time.Duration) bool {
	return v.LockExt(d, nil)
}

func (v Semaphore) LockExt(d time.Duration, done <-chan struct{}) bool {
	select {
	case v.sema <- struct{}{}:
		return true
	case <-done:
		return false
	default:
		select {
		case v.sema <- struct{}{}:
			return true
		case <-done:
			return false
		case <-time.After(d):
		}
		return false
	}
}

func (v Semaphore) Unlock() {
	<-v.sema
}

func (v Semaphore) Close() {
	close(v.sema)
}
