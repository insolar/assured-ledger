// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logicless

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/adapter"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type runState struct {
	id     call.ID
	result *executionupdate.ContractExecutionStateUpdate
}

func (r runState) GetResult() *executionupdate.ContractExecutionStateUpdate {
	return r.result
}

func (r runState) GetID() call.ID {
	return r.id
}

type executionMapping struct {
	byKey map[string]*ExecutionMock
	byID  map[call.ID]*ExecutionMock
}

func (m *executionMapping) add(key string, val *ExecutionMock) {
	if _, ok := m.byKey[key]; ok {
		panic("already exists by key")
	}
	if _, ok := m.byID[val.state.GetID()]; ok {
		panic("already exists by value")
	}

	m.byKey[key] = val
	m.byID[val.state.GetID()] = val
}

func (m *executionMapping) getByKey(key string) (*ExecutionMock, bool) {
	val, ok := m.byKey[key]
	return val, ok
}

func (m *executionMapping) getByID(id call.ID) (*ExecutionMock, bool) {
	val, ok := m.byID[id]
	return val, ok
}

func (m *executionMapping) minimockDone() bool {
	for _, executionMock := range m.byID {
		if !executionMock.minimockDone() {
			return false
		}
	}

	return true
}

type ServiceMock struct {
	ctx            context.Context
	t              minimock.Tester
	lastID         call.ID
	keyConstructor func(execution execution.Context) string

	executionMapping executionMapping
	classifyMapping  ExecutionClassifyMock
}

func defaultKeyConstructor(execution execution.Context) string {
	return execution.Outgoing.String()
}

func NewServiceMock(ctx context.Context, t minimock.Tester, keyConstructor func(execution execution.Context) string) *ServiceMock {
	if keyConstructor == nil {
		keyConstructor = defaultKeyConstructor
	}
	m := &ServiceMock{
		ctx:            ctx,
		t:              t,
		lastID:         0,
		keyConstructor: keyConstructor,

		executionMapping: executionMapping{
			byKey: make(map[string]*ExecutionMock),
			byID:  make(map[call.ID]*ExecutionMock),
		},
		classifyMapping: ExecutionClassifyMock{
			t:      t,
			mapper: make(map[string]*ExecutionClassifyMockInstance),
		},
	}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	return m
}

func (s *ServiceMock) CreateAdapter(ctx context.Context) runner.ServiceAdapter {
	return adapter.NewImposter(ctx, s, 16)
}

func (s *ServiceMock) AddExecutionMock(key string) *ExecutionMock {
	s.lastID++

	executionMock := &ExecutionMock{
		key:   key,
		state: &runState{id: s.lastID},
	}

	s.executionMapping.add(key, executionMock)

	return executionMock
}

func (s ServiceMock) ExecutionStart(execution execution.Context) runner.RunState {
	executionMock, ok := s.executionMapping.getByKey(s.keyConstructor(execution))
	if !ok {
		panic(throw.NotImplemented())
	}

	executionMock.state.result = nil

	executionChunk, err := executionMock.next(Start)
	if err != nil {
		s.t.Fatal(err.Error())

		return nil
	}

	if fn := executionChunk.check; fn != nil {
		checkFunc, ok := fn.(ExecutionMockStartCheckFunc)
		if !ok {
			panic(throw.IllegalState())
		} else if checkFunc != nil {
			checkFunc(execution)
		}
	}

	executionMock.state.result = executionChunk.update

	return executionMock.state
}

func (s ServiceMock) ExecutionContinue(run runner.RunState, outgoingResult []byte) {
	executionMock, ok := s.executionMapping.getByID(run.GetID())
	if !ok {
		panic(throw.NotImplemented())
	}

	executionMock.state.result = nil

	executionChunk, err := executionMock.next(Continue)
	if err != nil {
		s.t.Fatal(err.Error())

		return
	}

	if fn := executionChunk.check; fn != nil {
		checkFunc, ok := fn.(ExecutionMockContinueCheckFunc)
		if !ok {
			panic(throw.IllegalState())
		} else if checkFunc != nil {
			checkFunc(outgoingResult)
		}
	}

	executionMock.state.result = executionChunk.update
}

func (s ServiceMock) ExecutionAbort(run runner.RunState) {
	executionMock, ok := s.executionMapping.getByID(run.GetID())
	if !ok {
		panic(throw.NotImplemented())
	}

	executionMock.state.result = nil

	executionChunk, err := executionMock.next(Abort)
	if err != nil {
		s.t.Fatal(err.Error())

		return
	}

	if fn := executionChunk.check; fn != nil {
		checkFunc, ok := fn.(ExecutionMockAbortCheckFunc)
		if !ok {
			panic(throw.IllegalState())
		} else if checkFunc != nil {
			checkFunc()
		}
	}

	executionMock.state.result = nil
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (s ServiceMock) MinimockFinish() {
	if !s.minimockDone() {
		s.t.Fatal("failed to check")
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (s ServiceMock) MinimockWait(timeout time.Duration) {
	timeoutCh := time.After(timeout)
	for {
		if s.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			s.MinimockFinish()
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (s ServiceMock) minimockDone() bool {
	return s.classifyMapping.minimockDone() &&
		s.executionMapping.minimockDone()
}

type ExecutionClassifyMockInstance struct {
	count uint32

	v1 contract.MethodIsolation
	v2 error
}

type ExecutionClassifyMock struct {
	t      minimock.Tester
	mapper map[string]*ExecutionClassifyMockInstance
}

func (m *ExecutionClassifyMock) Set(key string, v1 contract.MethodIsolation, v2 error) *ExecutionClassifyMock {
	if _, ok := m.mapper[key]; ok {
		panic(throw.IllegalValue())
	}

	m.mapper[key] = &ExecutionClassifyMockInstance{
		v1: v1,
		v2: v2,
	}

	return m
}

func (s ServiceMock) ExecutionClassify(execution execution.Context) (contract.MethodIsolation, error) {
	key := s.keyConstructor(execution)
	if chunk, ok := s.classifyMapping.mapper[key]; ok {
		chunk.count++
		return chunk.v1, chunk.v2
	}

	s.t.Fatalf("failed to registered value for key '%s'", key)
	return contract.MethodIsolation{}, nil
}

func (s *ServiceMock) AddExecutionClassify(key string, v1 contract.MethodIsolation, v2 error) {
	s.classifyMapping.Set(key, v1, v2)
}

func (m *ExecutionClassifyMock) minimockDone() bool {
	for _, v := range m.mapper {
		if atomic.LoadUint32(&v.count) == 0 {
			return false
		}
	}

	return true
}
