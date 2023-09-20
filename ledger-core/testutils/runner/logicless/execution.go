package logicless

import (
	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ExecutionChunk struct {
	eventType ExecutionChunkType

	check  interface{}
	update *execution.Update
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
	t     minimock.Tester
	key   interface{}
	state *runState

	pos    int
	checks []ExecutionChunk
}

type ExecutionMockStartCheckFunc func(ctx execution.Context)
type ExecutionMockContinueCheckFunc func(result []byte)
type ExecutionMockAbortCheckFunc func()

func (m *ExecutionMock) AddStart(fn ExecutionMockStartCheckFunc, returnValue *execution.Update) *ExecutionMock {
	if len(m.checks) > 0 {
		panic(throw.IllegalValue())
	}
	m.checks = append(m.checks, ExecutionChunk{
		eventType: Start,
		check:     fn,
		update:    returnValue,
	})
	return m
}

func (m *ExecutionMock) AddContinue(fn ExecutionMockContinueCheckFunc, returnValue *execution.Update) *ExecutionMock {
	if len(m.checks) == 0 || m.checks[len(m.checks)-1].eventType >= Abort {
		panic(throw.IllegalValue())
	}
	m.checks = append(m.checks, ExecutionChunk{
		eventType: Continue,
		check:     fn,
		update:    returnValue,
	})
	return m
}

func (m *ExecutionMock) AddAbort(fn ExecutionMockAbortCheckFunc) *ExecutionMock {
	if len(m.checks) == 0 || m.checks[len(m.checks)-1].eventType >= Abort {
		panic(throw.IllegalValue())
	}
	m.checks = append(m.checks, ExecutionChunk{
		eventType: Abort,
		check:     fn,
		update:    nil,
	})
	return m
}

func (m *ExecutionMock) next(expectedTp ExecutionChunkType) (*ExecutionChunk, error) {
	var (
		currentCheckPosition = m.pos
	)

	if len(m.checks) <= currentCheckPosition {
		return nil, throw.E("unexpected next step")
	}

	if m.checks[currentCheckPosition].eventType != expectedTp {
		return nil, throw.E("unexpected next step type")
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
