// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	descriptor "github.com/insolar/assured-ledger/ledger-core/v2/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
)

type executionEventSink struct {
	context execution.Context
	output  chan *executionupdate.ContractExecutionStateUpdate
	input   chan []byte
}

func (c *executionEventSink) Error(err error) bool {
	c.output <- &executionupdate.ContractExecutionStateUpdate{
		Type:  executionupdate.TypeError,
		Error: err,
	}

	return true
}
func (c *executionEventSink) ErrorString(text string) bool {
	return c.Error(errors.New(text))
}
func (c *executionEventSink) ErrorWrapped(err error, text string) bool {
	return c.Error(errors.Wrap(err, text))
}

func (c *executionEventSink) ExternalCall(event descriptor.RPC) bool {
	c.output <- &executionupdate.ContractExecutionStateUpdate{
		Type:     executionupdate.TypeOutgoingCall,
		Outgoing: event,
	}
	return true
}
func (c *executionEventSink) Result(result *requestresult.RequestResult) bool {
	c.output <- &executionupdate.ContractExecutionStateUpdate{
		Type:   executionupdate.TypeDone,
		Result: result,
	}
	return true
}

func (c *executionEventSink) Abort() bool {
	c.output <- &executionupdate.ContractExecutionStateUpdate{
		Type: executionupdate.TypeAborted,
	}
	return true
}

func (c *executionEventSink) Stop() {
	close(c.input)
	close(c.output)
}
func newEventSink(execution execution.Context) *executionEventSink {
	return &executionEventSink{
		context: execution,
		output:  make(chan *executionupdate.ContractExecutionStateUpdate, 1),
		input:   make(chan []byte, 1),
	}
}
