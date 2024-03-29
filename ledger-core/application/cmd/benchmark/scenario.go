package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/application/api/sdk"
)

type benchmark struct {
	scenario       scenario
	name           string
	concurrent     int
	repetitions    int
	out            io.Writer
	totalTime      int64
	goroutineTimes []time.Duration
	successes      uint32
	errors         uint32
	timeouts       uint32
}

type scenario interface {
	canBeStarted() error
	prepare(int)
	start(int, int) (string, error)
	getBalanceCheckMembers() []sdk.Member
}

func (b *benchmark) getOperationsNumber() int {
	return b.concurrent * b.repetitions
}

func (b *benchmark) getAverageOperationDuration() time.Duration {
	return time.Duration(b.totalTime / int64(b.getOperationsNumber()))
}

func (b *benchmark) getOperationPerSecond() float64 {
	if len(b.goroutineTimes) == 0 {
		return 0
	}

	max := b.goroutineTimes[0]
	for _, t := range b.goroutineTimes {
		if max < t {
			max = t
		}
	}
	elapsedInSeconds := float64(max) / float64(time.Second)
	return float64(b.getOperationsNumber()-int(b.timeouts)) / elapsedInSeconds
}

func (b *benchmark) getName() string {
	return b.name
}

func (b *benchmark) getOut() io.Writer {
	return b.out
}

func (b *benchmark) start(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < b.concurrent; i++ {
		wg.Add(1)
		go b.startScenario(ctx, i, &wg)
	}
	wg.Wait()
}

func (b *benchmark) printResult() {
	writeToOutput(b.out, fmt.Sprintf("Scenario result:\n\tSuccesses: %d\n\tErrors: %d\n\tTimeouts: %d\n", b.successes, b.errors, b.timeouts))
}

func (b *benchmark) startScenario(ctx context.Context, concurrentIndex int, wg *sync.WaitGroup) {
	defer wg.Done()
	goroutineTime := time.Duration(0)
	for j := 0; j < b.repetitions; j++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var start time.Time
		var stop time.Duration
		var traceID string
		var err error

		start = time.Now()
		traceID, err = b.scenario.start(concurrentIndex, j)
		stop = time.Since(start)

		var netErr net.Error

		switch {
		case err == nil:
			atomic.AddUint32(&b.successes, 1)
			atomic.AddInt64(&b.totalTime, int64(stop))
			goroutineTime += stop
		case errors.As(err, &netErr) && netErr.Timeout():
			atomic.AddUint32(&b.timeouts, 1)
			writeToOutput(b.out, fmt.Sprintf("[Member №%d] Scenario error. Timeout. Error: %s \n", concurrentIndex, err.Error()))
		default:
			atomic.AddUint32(&b.errors, 1)
			atomic.AddInt64(&b.totalTime, int64(stop))
			goroutineTime += stop

			if strings.Contains(err.Error(), "invalid state record") {
				writeToOutput(b.out, fmt.Sprintf("[ OK ] Invalid state record.    Trace: %s.\n", traceID))
			} else {
				writeToOutput(b.out, fmt.Sprintf("[Member №%d] Scenario error with traceID: %s. Response: %s.\n", concurrentIndex, traceID, err.Error()))
			}
		}
	}
	b.goroutineTimes = append(b.goroutineTimes, goroutineTime)
}
