//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

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
