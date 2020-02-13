// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package synckit

import "sync"

type RWLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

func DummyLocker() RWLocker {
	return &dummyLock
}

var dummyLock = dummyLocker{}

type dummyLocker struct{}

func (*dummyLocker) Lock()    {}
func (*dummyLocker) Unlock()  {}
func (*dummyLocker) RUnlock() {}
func (*dummyLocker) RLock()   {}

func (*dummyLocker) String() string {
	return "dummyLocker"
}
