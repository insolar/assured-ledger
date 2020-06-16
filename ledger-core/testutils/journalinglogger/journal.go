// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journalinglogger

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const maxJournalLength = 15000

type SyncJournal struct {
	lock   sync.RWMutex
	events []debuglogger.UpdateEvent
}

func (j *SyncJournal) Append(event debuglogger.UpdateEvent) {
	j.lock.Lock()
	defer j.lock.Unlock()

	if len(j.events) > maxJournalLength {
		panic(throw.New("too long journal"))
	}

	j.events = append(j.events, event)
}

func (j *SyncJournal) Get(position int) (debuglogger.UpdateEvent, bool) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	if len(j.events) <= position {
		return debuglogger.UpdateEvent{}, false
	}

	return j.events[position], true
}
