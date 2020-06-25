// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package synchronization

import (
	"testing"
	"time"
)

type Point struct {
	count int

	input  chan struct{}
	output chan struct{}
}

func (p *Point) Synchronize() {
	// in case of timeout we can try to write to closed channel
	// we know that side effect and will try to ignore it
	defer func() { _ = recover() }()

	p.input <- struct{}{}

	<-p.output
}

// WaitTimeout should be called only from testing goroutine
func (p *Point) WaitTimeout(t *testing.T, tm time.Duration) {
	for i := 0; i < p.count; i++ {
		select {
		case <-p.input:
		case <-p.Wait():
			t.Fatal("timeout: failed to wait until all goroutines are synced")
		}
	}
}

// Wait should be called only from testing goroutine
func (p *Point) Wait() <-chan struct{} {
	doneChan := make(chan struct{})
	go func() {
		for i := 0; i < p.count; i++ {
			_, ok := <-p.input
			if !ok {
				close(doneChan)
			}
		}
		close(doneChan)
	}()

	return doneChan
}

// WakeUp should be called only from testing goroutine
func (p *Point) WakeUp() {
	for i := 0; i < p.count; i++ {
		p.output <- struct{}{}
	}
}

func (p *Point) Done() {
	close(p.input)
	close(p.output)
}

func NewPoint(count int) *Point {
	return &Point{
		count: count,

		input:  make(chan struct{}, count),
		output: make(chan struct{}, 0),
	}
}
