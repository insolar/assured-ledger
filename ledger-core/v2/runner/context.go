// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package runner

import (
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	descriptor "github.com/insolar/assured-ledger/ledger-core/v2/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
)

type executionContext struct {
	execution execution.Execution
	output    chan *executionupdate.ContractExecutionStateUpdate
	input     chan interface{}
}

func (c *executionContext) Error(err error) bool {
	c.output <- &executionupdate.ContractExecutionStateUpdate{
		Type:  executionupdate.ContractError,
		Error: err,
	}

	return true
}
func (c *executionContext) ErrorString(text string) bool {
	return c.Error(errors.New(text))
}
func (c *executionContext) ErrorWrapped(err error, text string) bool {
	return c.Error(errors.Wrap(err, text))
}

func (c *executionContext) ExternalCall(event descriptor.RPCEvent) bool {
	c.output <- &executionupdate.ContractExecutionStateUpdate{
		Type:     executionupdate.ContractOutgoingCall,
		Outgoing: event,
	}
	return true
}
func (c *executionContext) Result(result *requestresult.RequestResult) bool {
	c.output <- &executionupdate.ContractExecutionStateUpdate{
		Type:   executionupdate.ContractDone,
		Result: result,
	}
	return true
}

func (c *executionContext) Abort() bool {
	c.output <- &executionupdate.ContractExecutionStateUpdate{
		Type: executionupdate.ContractAborted,
	}
	return true
}

func (c *executionContext) Stop() {
	close(c.input)
	close(c.output)
}
func newExecutionContext(execution execution.Execution) *executionContext {
	return &executionContext{
		execution: execution,
		output:    make(chan *executionupdate.ContractExecutionStateUpdate, 1),
		input:     make(chan interface{}, 1),
	}
}
