// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execution

import (
	"runtime"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	SinkInitial int = iota
	SinkAborted
	SinkStopped
)

type EventSink struct {
	lock  sync.Mutex
	id    call.ID
	state int

	context     Context
	output      *Update
	input       chan requestresult.OutgoingExecutionResult
	inputLocked requestresult.OutgoingExecutionResult
}

func (c *EventSink) GetEvent() *Update {
	c.lock.Lock()
	defer c.lock.Unlock()

	output := c.output
	if output != nil && output.Type != Abort {
		c.output = nil
	}
	return output
}

func (c *EventSink) provideEvent(event *Update) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.state >= SinkAborted {
		return true
	}

	if c.output != nil {
		// no double writes should be
		return false
	}
	c.output = event
	return true
}

func (c *EventSink) Error(err error) bool {
	return c.provideEvent(&Update{
		Type:  Error,
		Error: err,
	})
}

func (c *EventSink) ExternalCall(event RPC) bool {
	return c.provideEvent(&Update{
		Type:     OutgoingCall,
		Outgoing: event,
	})
}
func (c *EventSink) Result(result *requestresult.RequestResult) bool {
	return c.provideEvent(&Update{
		Type:   Done,
		Result: result,
	})
}

func (c *EventSink) IsAborted() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.state == SinkAborted
}

// should be called from Interceptor (??)
func (c *EventSink) InternalAbort() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.state != SinkInitial {
		// we are trying to abort already aborted or stopped task
		// do nothing
		return
	}

	c.state = SinkAborted
	close(c.input)
	c.output = &Update{Type: Abort}
}

// should be called only from contract execution goroutine
func (c *EventSink) InternalStop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.state == SinkStopped {
		panic(throw.IllegalState()) // double stop
	} else if c.state == SinkInitial {
		close(c.input)
	}

	c.state = SinkStopped
}

// could be called from anywhere
func (c *EventSink) ID() call.ID {
	return c.id
}

// could be called from anywhere
func (c *EventSink) Context() Context {
	return c.context
}

// should be called only from interceptor worker
func (c *EventSink) InputProvide(input requestresult.OutgoingExecutionResult) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.inputLocked.IsEmpty() {
		panic(throw.IllegalState())
	}

	c.inputLocked = input
}

func (c *EventSink) InputFlush() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.state >= SinkAborted {
		panic(throw.IllegalState())
	}

	if c.inputLocked.IsEmpty() {
		panic(throw.IllegalState())
	}

	input := c.inputLocked
	c.inputLocked = requestresult.EmptyOutgoingExecutionResult

	// we should be able to make write here
	// otherwise code is buggy and will lead to deadlock
	select {
	case c.input <- input:
	default:
		panic(throw.IllegalState())
	}
}

// should be called only from execution goroutine
// safe to be called not under lock
func (c *EventSink) WaitInput() requestresult.OutgoingExecutionResult {
	rv, ok := <-c.input
	if !ok {
		runtime.Goexit()
	}
	return rv
}

func NewEventSink(id call.ID, execution Context) *EventSink {
	return &EventSink{
		id:      id,
		context: execution,
		input:   make(chan requestresult.OutgoingExecutionResult, 1),
	}
}
