// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotmachine

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type watchdog struct {
	timerEnd        time.Time
	defaultDuration time.Duration
	timer           *time.Timer
}

func (w *watchdog) setDuration(duration time.Duration) bool {
	w.timerEnd = time.Now().Add(w.defaultDuration)
	return w.timer.Reset(duration)
}

func (w *watchdog) Heartbeat() {
	if w.timer == nil {
		panic(throw.IllegalState())
	}
	if !w.setDuration(w.defaultDuration) {
		panic(throw.FailHere("timer already stopped"))
	}
}

func (w *watchdog) HeartbeatAsChannel() chan<- struct{} {
	if w.timer == nil {
		panic(throw.IllegalState())
	}
	ch := make(chan struct{})
	go func() {
		_, ok := <-ch
		if !ok {
			return
		}
		w.Heartbeat()
	}()
	return ch
}

func (w *watchdog) Extend(duration time.Duration) {
	if w.timer == nil {
		panic(throw.IllegalState())
	}

	resultDuration := time.Until(w.timerEnd) + duration

	if !w.setDuration(resultDuration) {
		panic(throw.FailHere("timer already stopped"))
	}
}

func (w *watchdog) Stop() bool {
	if timer := w.timer; timer == nil {
		panic(throw.IllegalState())
	} else {
		w.timer = nil
		return timer.Stop()
	}
}

func (watchdog) afterFunc() {
	global.Fatalm(throw.FailHere("timeout"))
}

func newWatchdog(duration time.Duration) *watchdog {
	w := watchdog{
		defaultDuration: duration,
	}
	w.timer = time.AfterFunc(duration, w.afterFunc)
	w.timerEnd = time.Now().Add(duration)

	return &w
}
