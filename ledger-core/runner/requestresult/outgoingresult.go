// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requestresult

var EmptyOutgoingExecutionResult = OutgoingExecutionResult{}

type OutgoingExecutionResult struct {
	initialized     bool
	ExecutionResult []byte
	Error           error
}

func (r OutgoingExecutionResult) IsEmpty() bool {
	return !r.initialized
}

func NewOutgoingExecutionResult(result []byte, err error) OutgoingExecutionResult {
	return OutgoingExecutionResult{
		initialized:     true,
		ExecutionResult: result,
		Error:           err,
	}
}
