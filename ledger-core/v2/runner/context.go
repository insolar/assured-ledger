// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	runner2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/requestresult"
)

type executionContext struct {
	id        uuid.UUID
	execution runner2.Execution
	output    chan *runner2.ContractExecutionStateUpdate
	input     chan interface{}
}

func (c *executionContext) Error(err error) bool {
	c.output <- &runner2.ContractExecutionStateUpdate{
		Type:  runner2.ContractError,
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

func (c *executionContext) ExternalCall(event runner2.RPCEvent) bool {
	c.output <- &runner2.ContractExecutionStateUpdate{
		Type:     runner2.ContractOutgoingCall,
		Outgoing: event,
	}
	return true
}
func (c *executionContext) Result(result *requestresult.RequestResult) bool {
	c.output <- &runner2.ContractExecutionStateUpdate{
		Type:   runner2.ContractDone,
		Result: result,
	}
	return true
}

func (c *executionContext) Abort() bool {
	c.output <- &runner2.ContractExecutionStateUpdate{
		Type: runner2.ContractAborted,
	}
	return true
}

func (c *executionContext) Stop() {
	close(c.input)
	close(c.output)
}
func newExecutionContext(execution runner2.Execution) *executionContext {
	return &executionContext{
		execution: execution,
		output:    make(chan *runner2.ContractExecutionStateUpdate, 1),
		input:     make(chan interface{}, 1),
	}
}
