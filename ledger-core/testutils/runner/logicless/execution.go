// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logicless

import (
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ExecutionChunk struct {
	tp ExecutionChunkType

	check  interface{}
	update *executionupdate.ContractExecutionStateUpdate
}

type ExecutionChunkType int

const (
	Start ExecutionChunkType = iota
	Continue
	Abort
)

func (t ExecutionChunkType) String() string {
	switch t {
	case Start:
		return "Start"
	case Continue:
		return "Continue"
	case Abort:
		return "Abort"
	default:
		return "Unknown"
	}
}

type ExecutionMock struct {
	key   interface{}
	state *runState

	pos    int
	checks []ExecutionChunk
}

type ExecutionMockStartCheckFunc func(ctx execution.Context)
type ExecutionMockContinueCheckFunc func(result []byte)
type ExecutionMockAbortCheckFunc func()

func (m *ExecutionMock) AddStart(fn ExecutionMockStartCheckFunc, returnValue *executionupdate.ContractExecutionStateUpdate) *ExecutionMock {
	if len(m.checks) > 0 {
		panic(throw.IllegalValue())
	}
	m.checks = append(m.checks, ExecutionChunk{
		tp:     Start,
		check:  fn,
		update: returnValue,
	})
	return m
}

func (m *ExecutionMock) AddContinue(fn ExecutionMockContinueCheckFunc, returnValue *executionupdate.ContractExecutionStateUpdate) *ExecutionMock {
	if len(m.checks) == 0 || m.checks[len(m.checks)-1].tp >= Abort {
		panic(throw.IllegalValue())
	}
	m.checks = append(m.checks, ExecutionChunk{
		tp:     Continue,
		check:  fn,
		update: returnValue,
	})
	return m
}

func (m *ExecutionMock) AddAbort(fn ExecutionMockAbortCheckFunc) *ExecutionMock {
	if len(m.checks) == 0 || m.checks[len(m.checks)-1].tp >= Abort {
		panic(throw.IllegalValue())
	}
	m.checks = append(m.checks, ExecutionChunk{
		tp:     Abort,
		check:  fn,
		update: nil,
	})
	return m
}

func (m *ExecutionMock) next(expectedTp ExecutionChunkType) (*ExecutionChunk, error) {
	var (
		currentCheckPosition = m.pos
	)

	if len(m.checks) < currentCheckPosition {
		return nil, throw.E("unexpected next step 1")
	}

	if m.checks[currentCheckPosition].tp != expectedTp {
		return nil, throw.E("unexpected next step 2")
	}

	m.pos++

	return &m.checks[currentCheckPosition], nil
}

func (m *ExecutionMock) minimockDone() bool {
	var (
		currentCheckPosition = m.pos
	)

	if len(m.checks) > currentCheckPosition {
		panic(throw.IllegalState())
	}

	if len(m.checks) == currentCheckPosition {
		return true
	}

	panic(throw.IllegalState())
}
