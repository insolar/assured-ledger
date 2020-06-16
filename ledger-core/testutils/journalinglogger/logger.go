// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journalinglogger

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

var _ smachine.SlotMachineLogger = &JournalingLogger{}

type JournalingLogger struct {
	child *debuglogger.DebugMachineLogger

	started  atomickit.Int
	stopLock sync.Mutex
	stopped  bool

	journal   *SyncJournal
	iterators JournalIteratorList
}

func (l *JournalingLogger) CreateStepLogger(ctx context.Context, machine smachine.StateMachine, id smachine.TracerID) smachine.StepLogger {
	return l.child.CreateStepLogger(ctx, machine, id)
}

func (l *JournalingLogger) LogMachineInternal(data smachine.SlotMachineData, msg string) {
	l.child.LogMachineInternal(data, msg)
}

func (l *JournalingLogger) LogMachineCritical(data smachine.SlotMachineData, msg string) {
	l.child.LogMachineCritical(data, msg)
}

func NewJournalingLogger(underlying smachine.SlotMachineLogger) *JournalingLogger {
	return &JournalingLogger{
		child:   debuglogger.NewDebugMachineLogger(underlying),
		journal: &SyncJournal{},
		iterators: JournalIteratorList{
			list: make(map[int]*JournalIterator),
		},
	}
}

func (l *JournalingLogger) Start() bool {
	if !l.started.CompareAndSwap(0, 1) {
		panic("double start")
	}

	go func() {
		for {
			rv := l.child.GetEvent()
			if rv.IsEmpty() {
				return
			}

			l.stopLock.Lock()
			if l.stopped {
				return
			}
			l.child.Continue()
			l.stopLock.Unlock()

			l.journal.Append(rv)
			l.iterators.Notify()
		}
	}()

	return true
}

func (l *JournalingLogger) GetJournalIterator() *JournalIterator {
	return l.iterators.createIterator(l)
}

func (l *JournalingLogger) Stop() {
	l.stopLock.Lock()
	defer l.stopLock.Unlock()

	l.iterators.Stop()
	l.child.Stop()
	l.stopped = true
}
