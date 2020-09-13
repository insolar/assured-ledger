// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package benchs

import (
	"sync/atomic"
	"time"
)

// nolint:unused
type benchLimiter interface {
	Reset()
	ShouldContinue() bool
}

// -----
// nolint:unused
type countedBenchLimiter struct {
	targetCount  int32
	currentCount int32
}

func (bl *countedBenchLimiter) Reset() {
	atomic.StoreInt32(&bl.currentCount, 0)
}

func (bl *countedBenchLimiter) ShouldContinue() bool {
	return atomic.AddInt32(&bl.currentCount, 1) <= bl.targetCount
}

// -----
// nolint:unused
type timedBenchLimiter struct {
	timeFinished chan struct{}
	pulseTime    int
}

func (bl *timedBenchLimiter) Reset() {
	bl.timeFinished = make(chan struct{})
	go func() {
		timeToRun := getTimeToExecute(bl.pulseTime)
		time.Sleep(time.Duration(timeToRun) * time.Second)
		close(bl.timeFinished)
	}()
}

func (bl *timedBenchLimiter) ShouldContinue() bool {
	select {
	case <-bl.timeFinished:
		return false
	default:
		return true
	}
}

// nolint:unused
func getTimeToExecute(pulseTime int) int {
	var (
		minimumExecTime = 10 // seconds
		numPulses       = 2
	)
	timeToRun := minimumExecTime
	pulseTime /= 1000 // convert to seconds

	pulseTime = numPulses * pulseTime
	if timeToRun < pulseTime {
		timeToRun = pulseTime
	}

	return timeToRun
}
