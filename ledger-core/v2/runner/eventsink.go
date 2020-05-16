// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	descriptor "github.com/insolar/assured-ledger/ledger-core/v2/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
)

type executionEventSink struct {
	id      call.ID
	context execution.Context
	output  *executionupdate.ContractExecutionStateUpdate
	input   chan []byte
}

func (c *executionEventSink) GetResult() *executionupdate.ContractExecutionStateUpdate {
	output := c.output
	c.output = nil
	return output
}

func (c *executionEventSink) Error(err error) bool {
	c.output = &executionupdate.ContractExecutionStateUpdate{
		Type:  executionupdate.Error,
		Error: err,
	}

	return true
}

func (c *executionEventSink) ExternalCall(event descriptor.RPC) bool {
	c.output = &executionupdate.ContractExecutionStateUpdate{
		Type:     executionupdate.OutgoingCall,
		Outgoing: event,
	}
	return true
}
func (c *executionEventSink) Result(result *requestresult.RequestResult) bool {
	c.output = &executionupdate.ContractExecutionStateUpdate{
		Type:   executionupdate.Done,
		Result: result,
	}
	return true
}

func (c *executionEventSink) Abort() bool {
	c.output = &executionupdate.ContractExecutionStateUpdate{
		Type: executionupdate.Aborted,
	}
	return true
}

func (c *executionEventSink) Stop() {
	close(c.input)
	c.output = nil
}

func newEventSink(execution execution.Context) *executionEventSink {
	return &executionEventSink{
		context: execution,
		output:  nil,
		input:   make(chan []byte, 1),
	}
}
