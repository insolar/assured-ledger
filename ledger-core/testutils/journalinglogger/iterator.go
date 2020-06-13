// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journalinglogger

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type JournalIterator struct {
	id       int
	await    chan struct{}
	position int
	logger   *JournalingLogger

	openedAsyncCalls int
}

func (it *JournalIterator) WaitEvent(predicate func(debuglogger.UpdateEvent) bool) <-chan struct{} {
	output := make(chan struct{})

	go func() {
		for {
			event, ok := it.logger.journal.Get(it.position)
			if ok {
				it.position++

				if predicate(event) {
					output <- struct{}{}
				}
				continue
			}

			_, closed := <-it.await
			if !closed {
				close(output)
				return
			}
		}
	}()

	return output
}

func smStopped(event debuglogger.UpdateEvent) bool {
	updateType := event.Update.UpdateType
	return updateType == "stop" || updateType == "panic" || updateType == "error"
}

func (it *JournalIterator) asyncProcessing(event debuglogger.UpdateEvent) {
	if event.AdapterID == "" {
		return
	}
	switch event.Data.Flags.AdapterFlags() {
	case smachine.StepLoggerAdapterAsyncCancel, smachine.StepLoggerAdapterAsyncExpiredCancel:
		fallthrough
	case smachine.StepLoggerAdapterAsyncResult, smachine.StepLoggerAdapterAsyncExpiredResult:
		it.openedAsyncCalls--
	case smachine.StepLoggerAdapterAsyncCall:
		it.openedAsyncCalls++
	default:
		panic(throw.IllegalValue())
	}
}

func (it *JournalIterator) WaitStop(sm smachine.StateMachine, count int) <-chan struct{} {
	var declarationType = reflect.TypeOf(sm)

	return it.WaitEvent(func(event debuglogger.UpdateEvent) bool {
		it.asyncProcessing(event)

		if reflect.TypeOf(event.SM) == declarationType {
			if smStopped(event) {
				count--
				return count == 0
			}
		}
		return false
	})
}

func (it *JournalIterator) WaitAllAsyncCallsFinished() <-chan struct{} {
	return it.WaitEvent(func(event debuglogger.UpdateEvent) bool {
		it.asyncProcessing(event)

		return it.openedAsyncCalls == 0
	})
}

func (it *JournalIterator) WaitAnyStop(count int) <-chan struct{} {
	return it.WaitEvent(func(event debuglogger.UpdateEvent) bool {
		it.asyncProcessing(event)

		if smStopped(event) {
			count--
			return count == 0
		}
		return false
	})
}

func (it *JournalIterator) Stop() {
	it.logger.iterators.Finish(it)
	close(it.await)
}
