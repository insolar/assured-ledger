package network

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	mm_network "github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// NodeKeeperMock implements network.NodeKeeper
type NodeKeeperMock struct {
	t minimock.Tester

	funcGetAccessor          func(n1 pulse.Number) (a1 mm_network.Accessor)
	inspectFuncGetAccessor   func(n1 pulse.Number)
	afterGetAccessorCounter  uint64
	beforeGetAccessorCounter uint64
	GetAccessorMock          mNodeKeeperMockGetAccessor

	funcGetOrigin          func() (n1 node.NetworkNode)
	inspectFuncGetOrigin   func()
	afterGetOriginCounter  uint64
	beforeGetOriginCounter uint64
	GetOriginMock          mNodeKeeperMockGetOrigin

	funcMoveSyncToActive          func(ctx context.Context, n1 pulse.Number)
	inspectFuncMoveSyncToActive   func(ctx context.Context, n1 pulse.Number)
	afterMoveSyncToActiveCounter  uint64
	beforeMoveSyncToActiveCounter uint64
	MoveSyncToActiveMock          mNodeKeeperMockMoveSyncToActive

	funcSetInitialSnapshot          func(nodes []node.NetworkNode)
	inspectFuncSetInitialSnapshot   func(nodes []node.NetworkNode)
	afterSetInitialSnapshotCounter  uint64
	beforeSetInitialSnapshotCounter uint64
	SetInitialSnapshotMock          mNodeKeeperMockSetInitialSnapshot

	funcSync          func(ctx context.Context, n1 pulse.Number, na1 []node.NetworkNode)
	inspectFuncSync   func(ctx context.Context, n1 pulse.Number, na1 []node.NetworkNode)
	afterSyncCounter  uint64
	beforeSyncCounter uint64
	SyncMock          mNodeKeeperMockSync
}

// NewNodeKeeperMock returns a mock for network.NodeKeeper
func NewNodeKeeperMock(t minimock.Tester) *NodeKeeperMock {
	m := &NodeKeeperMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetAccessorMock = mNodeKeeperMockGetAccessor{mock: m}
	m.GetAccessorMock.callArgs = []*NodeKeeperMockGetAccessorParams{}

	m.GetOriginMock = mNodeKeeperMockGetOrigin{mock: m}

	m.MoveSyncToActiveMock = mNodeKeeperMockMoveSyncToActive{mock: m}
	m.MoveSyncToActiveMock.callArgs = []*NodeKeeperMockMoveSyncToActiveParams{}

	m.SetInitialSnapshotMock = mNodeKeeperMockSetInitialSnapshot{mock: m}
	m.SetInitialSnapshotMock.callArgs = []*NodeKeeperMockSetInitialSnapshotParams{}

	m.SyncMock = mNodeKeeperMockSync{mock: m}
	m.SyncMock.callArgs = []*NodeKeeperMockSyncParams{}

	return m
}

type mNodeKeeperMockGetAccessor struct {
	mock               *NodeKeeperMock
	defaultExpectation *NodeKeeperMockGetAccessorExpectation
	expectations       []*NodeKeeperMockGetAccessorExpectation

	callArgs []*NodeKeeperMockGetAccessorParams
	mutex    sync.RWMutex
}

// NodeKeeperMockGetAccessorExpectation specifies expectation struct of the NodeKeeper.GetAccessor
type NodeKeeperMockGetAccessorExpectation struct {
	mock    *NodeKeeperMock
	params  *NodeKeeperMockGetAccessorParams
	results *NodeKeeperMockGetAccessorResults
	Counter uint64
}

// NodeKeeperMockGetAccessorParams contains parameters of the NodeKeeper.GetAccessor
type NodeKeeperMockGetAccessorParams struct {
	n1 pulse.Number
}

// NodeKeeperMockGetAccessorResults contains results of the NodeKeeper.GetAccessor
type NodeKeeperMockGetAccessorResults struct {
	a1 mm_network.Accessor
}

// Expect sets up expected params for NodeKeeper.GetAccessor
func (mmGetAccessor *mNodeKeeperMockGetAccessor) Expect(n1 pulse.Number) *mNodeKeeperMockGetAccessor {
	if mmGetAccessor.mock.funcGetAccessor != nil {
		mmGetAccessor.mock.t.Fatalf("NodeKeeperMock.GetAccessor mock is already set by Set")
	}

	if mmGetAccessor.defaultExpectation == nil {
		mmGetAccessor.defaultExpectation = &NodeKeeperMockGetAccessorExpectation{}
	}

	mmGetAccessor.defaultExpectation.params = &NodeKeeperMockGetAccessorParams{n1}
	for _, e := range mmGetAccessor.expectations {
		if minimock.Equal(e.params, mmGetAccessor.defaultExpectation.params) {
			mmGetAccessor.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGetAccessor.defaultExpectation.params)
		}
	}

	return mmGetAccessor
}

// Inspect accepts an inspector function that has same arguments as the NodeKeeper.GetAccessor
func (mmGetAccessor *mNodeKeeperMockGetAccessor) Inspect(f func(n1 pulse.Number)) *mNodeKeeperMockGetAccessor {
	if mmGetAccessor.mock.inspectFuncGetAccessor != nil {
		mmGetAccessor.mock.t.Fatalf("Inspect function is already set for NodeKeeperMock.GetAccessor")
	}

	mmGetAccessor.mock.inspectFuncGetAccessor = f

	return mmGetAccessor
}

// Return sets up results that will be returned by NodeKeeper.GetAccessor
func (mmGetAccessor *mNodeKeeperMockGetAccessor) Return(a1 mm_network.Accessor) *NodeKeeperMock {
	if mmGetAccessor.mock.funcGetAccessor != nil {
		mmGetAccessor.mock.t.Fatalf("NodeKeeperMock.GetAccessor mock is already set by Set")
	}

	if mmGetAccessor.defaultExpectation == nil {
		mmGetAccessor.defaultExpectation = &NodeKeeperMockGetAccessorExpectation{mock: mmGetAccessor.mock}
	}
	mmGetAccessor.defaultExpectation.results = &NodeKeeperMockGetAccessorResults{a1}
	return mmGetAccessor.mock
}

//Set uses given function f to mock the NodeKeeper.GetAccessor method
func (mmGetAccessor *mNodeKeeperMockGetAccessor) Set(f func(n1 pulse.Number) (a1 mm_network.Accessor)) *NodeKeeperMock {
	if mmGetAccessor.defaultExpectation != nil {
		mmGetAccessor.mock.t.Fatalf("Default expectation is already set for the NodeKeeper.GetAccessor method")
	}

	if len(mmGetAccessor.expectations) > 0 {
		mmGetAccessor.mock.t.Fatalf("Some expectations are already set for the NodeKeeper.GetAccessor method")
	}

	mmGetAccessor.mock.funcGetAccessor = f
	return mmGetAccessor.mock
}

// When sets expectation for the NodeKeeper.GetAccessor which will trigger the result defined by the following
// Then helper
func (mmGetAccessor *mNodeKeeperMockGetAccessor) When(n1 pulse.Number) *NodeKeeperMockGetAccessorExpectation {
	if mmGetAccessor.mock.funcGetAccessor != nil {
		mmGetAccessor.mock.t.Fatalf("NodeKeeperMock.GetAccessor mock is already set by Set")
	}

	expectation := &NodeKeeperMockGetAccessorExpectation{
		mock:   mmGetAccessor.mock,
		params: &NodeKeeperMockGetAccessorParams{n1},
	}
	mmGetAccessor.expectations = append(mmGetAccessor.expectations, expectation)
	return expectation
}

// Then sets up NodeKeeper.GetAccessor return parameters for the expectation previously defined by the When method
func (e *NodeKeeperMockGetAccessorExpectation) Then(a1 mm_network.Accessor) *NodeKeeperMock {
	e.results = &NodeKeeperMockGetAccessorResults{a1}
	return e.mock
}

// GetAccessor implements network.NodeKeeper
func (mmGetAccessor *NodeKeeperMock) GetAccessor(n1 pulse.Number) (a1 mm_network.Accessor) {
	mm_atomic.AddUint64(&mmGetAccessor.beforeGetAccessorCounter, 1)
	defer mm_atomic.AddUint64(&mmGetAccessor.afterGetAccessorCounter, 1)

	if mmGetAccessor.inspectFuncGetAccessor != nil {
		mmGetAccessor.inspectFuncGetAccessor(n1)
	}

	mm_params := &NodeKeeperMockGetAccessorParams{n1}

	// Record call args
	mmGetAccessor.GetAccessorMock.mutex.Lock()
	mmGetAccessor.GetAccessorMock.callArgs = append(mmGetAccessor.GetAccessorMock.callArgs, mm_params)
	mmGetAccessor.GetAccessorMock.mutex.Unlock()

	for _, e := range mmGetAccessor.GetAccessorMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.a1
		}
	}

	if mmGetAccessor.GetAccessorMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetAccessor.GetAccessorMock.defaultExpectation.Counter, 1)
		mm_want := mmGetAccessor.GetAccessorMock.defaultExpectation.params
		mm_got := NodeKeeperMockGetAccessorParams{n1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGetAccessor.t.Errorf("NodeKeeperMock.GetAccessor got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGetAccessor.GetAccessorMock.defaultExpectation.results
		if mm_results == nil {
			mmGetAccessor.t.Fatal("No results are set for the NodeKeeperMock.GetAccessor")
		}
		return (*mm_results).a1
	}
	if mmGetAccessor.funcGetAccessor != nil {
		return mmGetAccessor.funcGetAccessor(n1)
	}
	mmGetAccessor.t.Fatalf("Unexpected call to NodeKeeperMock.GetAccessor. %v", n1)
	return
}

// GetAccessorAfterCounter returns a count of finished NodeKeeperMock.GetAccessor invocations
func (mmGetAccessor *NodeKeeperMock) GetAccessorAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetAccessor.afterGetAccessorCounter)
}

// GetAccessorBeforeCounter returns a count of NodeKeeperMock.GetAccessor invocations
func (mmGetAccessor *NodeKeeperMock) GetAccessorBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetAccessor.beforeGetAccessorCounter)
}

// Calls returns a list of arguments used in each call to NodeKeeperMock.GetAccessor.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGetAccessor *mNodeKeeperMockGetAccessor) Calls() []*NodeKeeperMockGetAccessorParams {
	mmGetAccessor.mutex.RLock()

	argCopy := make([]*NodeKeeperMockGetAccessorParams, len(mmGetAccessor.callArgs))
	copy(argCopy, mmGetAccessor.callArgs)

	mmGetAccessor.mutex.RUnlock()

	return argCopy
}

// MinimockGetAccessorDone returns true if the count of the GetAccessor invocations corresponds
// the number of defined expectations
func (m *NodeKeeperMock) MinimockGetAccessorDone() bool {
	for _, e := range m.GetAccessorMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetAccessorMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetAccessorCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetAccessor != nil && mm_atomic.LoadUint64(&m.afterGetAccessorCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetAccessorInspect logs each unmet expectation
func (m *NodeKeeperMock) MinimockGetAccessorInspect() {
	for _, e := range m.GetAccessorMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to NodeKeeperMock.GetAccessor with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetAccessorMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetAccessorCounter) < 1 {
		if m.GetAccessorMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to NodeKeeperMock.GetAccessor")
		} else {
			m.t.Errorf("Expected call to NodeKeeperMock.GetAccessor with params: %#v", *m.GetAccessorMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetAccessor != nil && mm_atomic.LoadUint64(&m.afterGetAccessorCounter) < 1 {
		m.t.Error("Expected call to NodeKeeperMock.GetAccessor")
	}
}

type mNodeKeeperMockGetOrigin struct {
	mock               *NodeKeeperMock
	defaultExpectation *NodeKeeperMockGetOriginExpectation
	expectations       []*NodeKeeperMockGetOriginExpectation
}

// NodeKeeperMockGetOriginExpectation specifies expectation struct of the NodeKeeper.GetOrigin
type NodeKeeperMockGetOriginExpectation struct {
	mock *NodeKeeperMock

	results *NodeKeeperMockGetOriginResults
	Counter uint64
}

// NodeKeeperMockGetOriginResults contains results of the NodeKeeper.GetOrigin
type NodeKeeperMockGetOriginResults struct {
	n1 node.NetworkNode
}

// Expect sets up expected params for NodeKeeper.GetOrigin
func (mmGetOrigin *mNodeKeeperMockGetOrigin) Expect() *mNodeKeeperMockGetOrigin {
	if mmGetOrigin.mock.funcGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("NodeKeeperMock.GetOrigin mock is already set by Set")
	}

	if mmGetOrigin.defaultExpectation == nil {
		mmGetOrigin.defaultExpectation = &NodeKeeperMockGetOriginExpectation{}
	}

	return mmGetOrigin
}

// Inspect accepts an inspector function that has same arguments as the NodeKeeper.GetOrigin
func (mmGetOrigin *mNodeKeeperMockGetOrigin) Inspect(f func()) *mNodeKeeperMockGetOrigin {
	if mmGetOrigin.mock.inspectFuncGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("Inspect function is already set for NodeKeeperMock.GetOrigin")
	}

	mmGetOrigin.mock.inspectFuncGetOrigin = f

	return mmGetOrigin
}

// Return sets up results that will be returned by NodeKeeper.GetOrigin
func (mmGetOrigin *mNodeKeeperMockGetOrigin) Return(n1 node.NetworkNode) *NodeKeeperMock {
	if mmGetOrigin.mock.funcGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("NodeKeeperMock.GetOrigin mock is already set by Set")
	}

	if mmGetOrigin.defaultExpectation == nil {
		mmGetOrigin.defaultExpectation = &NodeKeeperMockGetOriginExpectation{mock: mmGetOrigin.mock}
	}
	mmGetOrigin.defaultExpectation.results = &NodeKeeperMockGetOriginResults{n1}
	return mmGetOrigin.mock
}

//Set uses given function f to mock the NodeKeeper.GetOrigin method
func (mmGetOrigin *mNodeKeeperMockGetOrigin) Set(f func() (n1 node.NetworkNode)) *NodeKeeperMock {
	if mmGetOrigin.defaultExpectation != nil {
		mmGetOrigin.mock.t.Fatalf("Default expectation is already set for the NodeKeeper.GetOrigin method")
	}

	if len(mmGetOrigin.expectations) > 0 {
		mmGetOrigin.mock.t.Fatalf("Some expectations are already set for the NodeKeeper.GetOrigin method")
	}

	mmGetOrigin.mock.funcGetOrigin = f
	return mmGetOrigin.mock
}

// GetOrigin implements network.NodeKeeper
func (mmGetOrigin *NodeKeeperMock) GetOrigin() (n1 node.NetworkNode) {
	mm_atomic.AddUint64(&mmGetOrigin.beforeGetOriginCounter, 1)
	defer mm_atomic.AddUint64(&mmGetOrigin.afterGetOriginCounter, 1)

	if mmGetOrigin.inspectFuncGetOrigin != nil {
		mmGetOrigin.inspectFuncGetOrigin()
	}

	if mmGetOrigin.GetOriginMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetOrigin.GetOriginMock.defaultExpectation.Counter, 1)

		mm_results := mmGetOrigin.GetOriginMock.defaultExpectation.results
		if mm_results == nil {
			mmGetOrigin.t.Fatal("No results are set for the NodeKeeperMock.GetOrigin")
		}
		return (*mm_results).n1
	}
	if mmGetOrigin.funcGetOrigin != nil {
		return mmGetOrigin.funcGetOrigin()
	}
	mmGetOrigin.t.Fatalf("Unexpected call to NodeKeeperMock.GetOrigin.")
	return
}

// GetOriginAfterCounter returns a count of finished NodeKeeperMock.GetOrigin invocations
func (mmGetOrigin *NodeKeeperMock) GetOriginAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetOrigin.afterGetOriginCounter)
}

// GetOriginBeforeCounter returns a count of NodeKeeperMock.GetOrigin invocations
func (mmGetOrigin *NodeKeeperMock) GetOriginBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetOrigin.beforeGetOriginCounter)
}

// MinimockGetOriginDone returns true if the count of the GetOrigin invocations corresponds
// the number of defined expectations
func (m *NodeKeeperMock) MinimockGetOriginDone() bool {
	for _, e := range m.GetOriginMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetOriginMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetOrigin != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetOriginInspect logs each unmet expectation
func (m *NodeKeeperMock) MinimockGetOriginInspect() {
	for _, e := range m.GetOriginMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to NodeKeeperMock.GetOrigin")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetOriginMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		m.t.Error("Expected call to NodeKeeperMock.GetOrigin")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetOrigin != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		m.t.Error("Expected call to NodeKeeperMock.GetOrigin")
	}
}

type mNodeKeeperMockMoveSyncToActive struct {
	mock               *NodeKeeperMock
	defaultExpectation *NodeKeeperMockMoveSyncToActiveExpectation
	expectations       []*NodeKeeperMockMoveSyncToActiveExpectation

	callArgs []*NodeKeeperMockMoveSyncToActiveParams
	mutex    sync.RWMutex
}

// NodeKeeperMockMoveSyncToActiveExpectation specifies expectation struct of the NodeKeeper.MoveSyncToActive
type NodeKeeperMockMoveSyncToActiveExpectation struct {
	mock   *NodeKeeperMock
	params *NodeKeeperMockMoveSyncToActiveParams

	Counter uint64
}

// NodeKeeperMockMoveSyncToActiveParams contains parameters of the NodeKeeper.MoveSyncToActive
type NodeKeeperMockMoveSyncToActiveParams struct {
	ctx context.Context
	n1  pulse.Number
}

// Expect sets up expected params for NodeKeeper.MoveSyncToActive
func (mmMoveSyncToActive *mNodeKeeperMockMoveSyncToActive) Expect(ctx context.Context, n1 pulse.Number) *mNodeKeeperMockMoveSyncToActive {
	if mmMoveSyncToActive.mock.funcMoveSyncToActive != nil {
		mmMoveSyncToActive.mock.t.Fatalf("NodeKeeperMock.MoveSyncToActive mock is already set by Set")
	}

	if mmMoveSyncToActive.defaultExpectation == nil {
		mmMoveSyncToActive.defaultExpectation = &NodeKeeperMockMoveSyncToActiveExpectation{}
	}

	mmMoveSyncToActive.defaultExpectation.params = &NodeKeeperMockMoveSyncToActiveParams{ctx, n1}
	for _, e := range mmMoveSyncToActive.expectations {
		if minimock.Equal(e.params, mmMoveSyncToActive.defaultExpectation.params) {
			mmMoveSyncToActive.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmMoveSyncToActive.defaultExpectation.params)
		}
	}

	return mmMoveSyncToActive
}

// Inspect accepts an inspector function that has same arguments as the NodeKeeper.MoveSyncToActive
func (mmMoveSyncToActive *mNodeKeeperMockMoveSyncToActive) Inspect(f func(ctx context.Context, n1 pulse.Number)) *mNodeKeeperMockMoveSyncToActive {
	if mmMoveSyncToActive.mock.inspectFuncMoveSyncToActive != nil {
		mmMoveSyncToActive.mock.t.Fatalf("Inspect function is already set for NodeKeeperMock.MoveSyncToActive")
	}

	mmMoveSyncToActive.mock.inspectFuncMoveSyncToActive = f

	return mmMoveSyncToActive
}

// Return sets up results that will be returned by NodeKeeper.MoveSyncToActive
func (mmMoveSyncToActive *mNodeKeeperMockMoveSyncToActive) Return() *NodeKeeperMock {
	if mmMoveSyncToActive.mock.funcMoveSyncToActive != nil {
		mmMoveSyncToActive.mock.t.Fatalf("NodeKeeperMock.MoveSyncToActive mock is already set by Set")
	}

	if mmMoveSyncToActive.defaultExpectation == nil {
		mmMoveSyncToActive.defaultExpectation = &NodeKeeperMockMoveSyncToActiveExpectation{mock: mmMoveSyncToActive.mock}
	}

	return mmMoveSyncToActive.mock
}

//Set uses given function f to mock the NodeKeeper.MoveSyncToActive method
func (mmMoveSyncToActive *mNodeKeeperMockMoveSyncToActive) Set(f func(ctx context.Context, n1 pulse.Number)) *NodeKeeperMock {
	if mmMoveSyncToActive.defaultExpectation != nil {
		mmMoveSyncToActive.mock.t.Fatalf("Default expectation is already set for the NodeKeeper.MoveSyncToActive method")
	}

	if len(mmMoveSyncToActive.expectations) > 0 {
		mmMoveSyncToActive.mock.t.Fatalf("Some expectations are already set for the NodeKeeper.MoveSyncToActive method")
	}

	mmMoveSyncToActive.mock.funcMoveSyncToActive = f
	return mmMoveSyncToActive.mock
}

// MoveSyncToActive implements network.NodeKeeper
func (mmMoveSyncToActive *NodeKeeperMock) MoveSyncToActive(ctx context.Context, n1 pulse.Number) {
	mm_atomic.AddUint64(&mmMoveSyncToActive.beforeMoveSyncToActiveCounter, 1)
	defer mm_atomic.AddUint64(&mmMoveSyncToActive.afterMoveSyncToActiveCounter, 1)

	if mmMoveSyncToActive.inspectFuncMoveSyncToActive != nil {
		mmMoveSyncToActive.inspectFuncMoveSyncToActive(ctx, n1)
	}

	mm_params := &NodeKeeperMockMoveSyncToActiveParams{ctx, n1}

	// Record call args
	mmMoveSyncToActive.MoveSyncToActiveMock.mutex.Lock()
	mmMoveSyncToActive.MoveSyncToActiveMock.callArgs = append(mmMoveSyncToActive.MoveSyncToActiveMock.callArgs, mm_params)
	mmMoveSyncToActive.MoveSyncToActiveMock.mutex.Unlock()

	for _, e := range mmMoveSyncToActive.MoveSyncToActiveMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return
		}
	}

	if mmMoveSyncToActive.MoveSyncToActiveMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmMoveSyncToActive.MoveSyncToActiveMock.defaultExpectation.Counter, 1)
		mm_want := mmMoveSyncToActive.MoveSyncToActiveMock.defaultExpectation.params
		mm_got := NodeKeeperMockMoveSyncToActiveParams{ctx, n1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmMoveSyncToActive.t.Errorf("NodeKeeperMock.MoveSyncToActive got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		return

	}
	if mmMoveSyncToActive.funcMoveSyncToActive != nil {
		mmMoveSyncToActive.funcMoveSyncToActive(ctx, n1)
		return
	}
	mmMoveSyncToActive.t.Fatalf("Unexpected call to NodeKeeperMock.MoveSyncToActive. %v %v", ctx, n1)

}

// MoveSyncToActiveAfterCounter returns a count of finished NodeKeeperMock.MoveSyncToActive invocations
func (mmMoveSyncToActive *NodeKeeperMock) MoveSyncToActiveAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMoveSyncToActive.afterMoveSyncToActiveCounter)
}

// MoveSyncToActiveBeforeCounter returns a count of NodeKeeperMock.MoveSyncToActive invocations
func (mmMoveSyncToActive *NodeKeeperMock) MoveSyncToActiveBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMoveSyncToActive.beforeMoveSyncToActiveCounter)
}

// Calls returns a list of arguments used in each call to NodeKeeperMock.MoveSyncToActive.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmMoveSyncToActive *mNodeKeeperMockMoveSyncToActive) Calls() []*NodeKeeperMockMoveSyncToActiveParams {
	mmMoveSyncToActive.mutex.RLock()

	argCopy := make([]*NodeKeeperMockMoveSyncToActiveParams, len(mmMoveSyncToActive.callArgs))
	copy(argCopy, mmMoveSyncToActive.callArgs)

	mmMoveSyncToActive.mutex.RUnlock()

	return argCopy
}

// MinimockMoveSyncToActiveDone returns true if the count of the MoveSyncToActive invocations corresponds
// the number of defined expectations
func (m *NodeKeeperMock) MinimockMoveSyncToActiveDone() bool {
	for _, e := range m.MoveSyncToActiveMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.MoveSyncToActiveMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterMoveSyncToActiveCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMoveSyncToActive != nil && mm_atomic.LoadUint64(&m.afterMoveSyncToActiveCounter) < 1 {
		return false
	}
	return true
}

// MinimockMoveSyncToActiveInspect logs each unmet expectation
func (m *NodeKeeperMock) MinimockMoveSyncToActiveInspect() {
	for _, e := range m.MoveSyncToActiveMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to NodeKeeperMock.MoveSyncToActive with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.MoveSyncToActiveMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterMoveSyncToActiveCounter) < 1 {
		if m.MoveSyncToActiveMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to NodeKeeperMock.MoveSyncToActive")
		} else {
			m.t.Errorf("Expected call to NodeKeeperMock.MoveSyncToActive with params: %#v", *m.MoveSyncToActiveMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMoveSyncToActive != nil && mm_atomic.LoadUint64(&m.afterMoveSyncToActiveCounter) < 1 {
		m.t.Error("Expected call to NodeKeeperMock.MoveSyncToActive")
	}
}

type mNodeKeeperMockSetInitialSnapshot struct {
	mock               *NodeKeeperMock
	defaultExpectation *NodeKeeperMockSetInitialSnapshotExpectation
	expectations       []*NodeKeeperMockSetInitialSnapshotExpectation

	callArgs []*NodeKeeperMockSetInitialSnapshotParams
	mutex    sync.RWMutex
}

// NodeKeeperMockSetInitialSnapshotExpectation specifies expectation struct of the NodeKeeper.SetInitialSnapshot
type NodeKeeperMockSetInitialSnapshotExpectation struct {
	mock   *NodeKeeperMock
	params *NodeKeeperMockSetInitialSnapshotParams

	Counter uint64
}

// NodeKeeperMockSetInitialSnapshotParams contains parameters of the NodeKeeper.SetInitialSnapshot
type NodeKeeperMockSetInitialSnapshotParams struct {
	nodes []node.NetworkNode
}

// Expect sets up expected params for NodeKeeper.SetInitialSnapshot
func (mmSetInitialSnapshot *mNodeKeeperMockSetInitialSnapshot) Expect(nodes []node.NetworkNode) *mNodeKeeperMockSetInitialSnapshot {
	if mmSetInitialSnapshot.mock.funcSetInitialSnapshot != nil {
		mmSetInitialSnapshot.mock.t.Fatalf("NodeKeeperMock.SetInitialSnapshot mock is already set by Set")
	}

	if mmSetInitialSnapshot.defaultExpectation == nil {
		mmSetInitialSnapshot.defaultExpectation = &NodeKeeperMockSetInitialSnapshotExpectation{}
	}

	mmSetInitialSnapshot.defaultExpectation.params = &NodeKeeperMockSetInitialSnapshotParams{nodes}
	for _, e := range mmSetInitialSnapshot.expectations {
		if minimock.Equal(e.params, mmSetInitialSnapshot.defaultExpectation.params) {
			mmSetInitialSnapshot.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSetInitialSnapshot.defaultExpectation.params)
		}
	}

	return mmSetInitialSnapshot
}

// Inspect accepts an inspector function that has same arguments as the NodeKeeper.SetInitialSnapshot
func (mmSetInitialSnapshot *mNodeKeeperMockSetInitialSnapshot) Inspect(f func(nodes []node.NetworkNode)) *mNodeKeeperMockSetInitialSnapshot {
	if mmSetInitialSnapshot.mock.inspectFuncSetInitialSnapshot != nil {
		mmSetInitialSnapshot.mock.t.Fatalf("Inspect function is already set for NodeKeeperMock.SetInitialSnapshot")
	}

	mmSetInitialSnapshot.mock.inspectFuncSetInitialSnapshot = f

	return mmSetInitialSnapshot
}

// Return sets up results that will be returned by NodeKeeper.SetInitialSnapshot
func (mmSetInitialSnapshot *mNodeKeeperMockSetInitialSnapshot) Return() *NodeKeeperMock {
	if mmSetInitialSnapshot.mock.funcSetInitialSnapshot != nil {
		mmSetInitialSnapshot.mock.t.Fatalf("NodeKeeperMock.SetInitialSnapshot mock is already set by Set")
	}

	if mmSetInitialSnapshot.defaultExpectation == nil {
		mmSetInitialSnapshot.defaultExpectation = &NodeKeeperMockSetInitialSnapshotExpectation{mock: mmSetInitialSnapshot.mock}
	}

	return mmSetInitialSnapshot.mock
}

//Set uses given function f to mock the NodeKeeper.SetInitialSnapshot method
func (mmSetInitialSnapshot *mNodeKeeperMockSetInitialSnapshot) Set(f func(nodes []node.NetworkNode)) *NodeKeeperMock {
	if mmSetInitialSnapshot.defaultExpectation != nil {
		mmSetInitialSnapshot.mock.t.Fatalf("Default expectation is already set for the NodeKeeper.SetInitialSnapshot method")
	}

	if len(mmSetInitialSnapshot.expectations) > 0 {
		mmSetInitialSnapshot.mock.t.Fatalf("Some expectations are already set for the NodeKeeper.SetInitialSnapshot method")
	}

	mmSetInitialSnapshot.mock.funcSetInitialSnapshot = f
	return mmSetInitialSnapshot.mock
}

// SetInitialSnapshot implements network.NodeKeeper
func (mmSetInitialSnapshot *NodeKeeperMock) SetInitialSnapshot(nodes []node.NetworkNode) {
	mm_atomic.AddUint64(&mmSetInitialSnapshot.beforeSetInitialSnapshotCounter, 1)
	defer mm_atomic.AddUint64(&mmSetInitialSnapshot.afterSetInitialSnapshotCounter, 1)

	if mmSetInitialSnapshot.inspectFuncSetInitialSnapshot != nil {
		mmSetInitialSnapshot.inspectFuncSetInitialSnapshot(nodes)
	}

	mm_params := &NodeKeeperMockSetInitialSnapshotParams{nodes}

	// Record call args
	mmSetInitialSnapshot.SetInitialSnapshotMock.mutex.Lock()
	mmSetInitialSnapshot.SetInitialSnapshotMock.callArgs = append(mmSetInitialSnapshot.SetInitialSnapshotMock.callArgs, mm_params)
	mmSetInitialSnapshot.SetInitialSnapshotMock.mutex.Unlock()

	for _, e := range mmSetInitialSnapshot.SetInitialSnapshotMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return
		}
	}

	if mmSetInitialSnapshot.SetInitialSnapshotMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSetInitialSnapshot.SetInitialSnapshotMock.defaultExpectation.Counter, 1)
		mm_want := mmSetInitialSnapshot.SetInitialSnapshotMock.defaultExpectation.params
		mm_got := NodeKeeperMockSetInitialSnapshotParams{nodes}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSetInitialSnapshot.t.Errorf("NodeKeeperMock.SetInitialSnapshot got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		return

	}
	if mmSetInitialSnapshot.funcSetInitialSnapshot != nil {
		mmSetInitialSnapshot.funcSetInitialSnapshot(nodes)
		return
	}
	mmSetInitialSnapshot.t.Fatalf("Unexpected call to NodeKeeperMock.SetInitialSnapshot. %v", nodes)

}

// SetInitialSnapshotAfterCounter returns a count of finished NodeKeeperMock.SetInitialSnapshot invocations
func (mmSetInitialSnapshot *NodeKeeperMock) SetInitialSnapshotAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetInitialSnapshot.afterSetInitialSnapshotCounter)
}

// SetInitialSnapshotBeforeCounter returns a count of NodeKeeperMock.SetInitialSnapshot invocations
func (mmSetInitialSnapshot *NodeKeeperMock) SetInitialSnapshotBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSetInitialSnapshot.beforeSetInitialSnapshotCounter)
}

// Calls returns a list of arguments used in each call to NodeKeeperMock.SetInitialSnapshot.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSetInitialSnapshot *mNodeKeeperMockSetInitialSnapshot) Calls() []*NodeKeeperMockSetInitialSnapshotParams {
	mmSetInitialSnapshot.mutex.RLock()

	argCopy := make([]*NodeKeeperMockSetInitialSnapshotParams, len(mmSetInitialSnapshot.callArgs))
	copy(argCopy, mmSetInitialSnapshot.callArgs)

	mmSetInitialSnapshot.mutex.RUnlock()

	return argCopy
}

// MinimockSetInitialSnapshotDone returns true if the count of the SetInitialSnapshot invocations corresponds
// the number of defined expectations
func (m *NodeKeeperMock) MinimockSetInitialSnapshotDone() bool {
	for _, e := range m.SetInitialSnapshotMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetInitialSnapshotMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetInitialSnapshotCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetInitialSnapshot != nil && mm_atomic.LoadUint64(&m.afterSetInitialSnapshotCounter) < 1 {
		return false
	}
	return true
}

// MinimockSetInitialSnapshotInspect logs each unmet expectation
func (m *NodeKeeperMock) MinimockSetInitialSnapshotInspect() {
	for _, e := range m.SetInitialSnapshotMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to NodeKeeperMock.SetInitialSnapshot with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SetInitialSnapshotMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSetInitialSnapshotCounter) < 1 {
		if m.SetInitialSnapshotMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to NodeKeeperMock.SetInitialSnapshot")
		} else {
			m.t.Errorf("Expected call to NodeKeeperMock.SetInitialSnapshot with params: %#v", *m.SetInitialSnapshotMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSetInitialSnapshot != nil && mm_atomic.LoadUint64(&m.afterSetInitialSnapshotCounter) < 1 {
		m.t.Error("Expected call to NodeKeeperMock.SetInitialSnapshot")
	}
}

type mNodeKeeperMockSync struct {
	mock               *NodeKeeperMock
	defaultExpectation *NodeKeeperMockSyncExpectation
	expectations       []*NodeKeeperMockSyncExpectation

	callArgs []*NodeKeeperMockSyncParams
	mutex    sync.RWMutex
}

// NodeKeeperMockSyncExpectation specifies expectation struct of the NodeKeeper.Sync
type NodeKeeperMockSyncExpectation struct {
	mock   *NodeKeeperMock
	params *NodeKeeperMockSyncParams

	Counter uint64
}

// NodeKeeperMockSyncParams contains parameters of the NodeKeeper.Sync
type NodeKeeperMockSyncParams struct {
	ctx context.Context
	n1  pulse.Number
	na1 []node.NetworkNode
}

// Expect sets up expected params for NodeKeeper.Sync
func (mmSync *mNodeKeeperMockSync) Expect(ctx context.Context, n1 pulse.Number, na1 []node.NetworkNode) *mNodeKeeperMockSync {
	if mmSync.mock.funcSync != nil {
		mmSync.mock.t.Fatalf("NodeKeeperMock.Sync mock is already set by Set")
	}

	if mmSync.defaultExpectation == nil {
		mmSync.defaultExpectation = &NodeKeeperMockSyncExpectation{}
	}

	mmSync.defaultExpectation.params = &NodeKeeperMockSyncParams{ctx, n1, na1}
	for _, e := range mmSync.expectations {
		if minimock.Equal(e.params, mmSync.defaultExpectation.params) {
			mmSync.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSync.defaultExpectation.params)
		}
	}

	return mmSync
}

// Inspect accepts an inspector function that has same arguments as the NodeKeeper.Sync
func (mmSync *mNodeKeeperMockSync) Inspect(f func(ctx context.Context, n1 pulse.Number, na1 []node.NetworkNode)) *mNodeKeeperMockSync {
	if mmSync.mock.inspectFuncSync != nil {
		mmSync.mock.t.Fatalf("Inspect function is already set for NodeKeeperMock.Sync")
	}

	mmSync.mock.inspectFuncSync = f

	return mmSync
}

// Return sets up results that will be returned by NodeKeeper.Sync
func (mmSync *mNodeKeeperMockSync) Return() *NodeKeeperMock {
	if mmSync.mock.funcSync != nil {
		mmSync.mock.t.Fatalf("NodeKeeperMock.Sync mock is already set by Set")
	}

	if mmSync.defaultExpectation == nil {
		mmSync.defaultExpectation = &NodeKeeperMockSyncExpectation{mock: mmSync.mock}
	}

	return mmSync.mock
}

//Set uses given function f to mock the NodeKeeper.Sync method
func (mmSync *mNodeKeeperMockSync) Set(f func(ctx context.Context, n1 pulse.Number, na1 []node.NetworkNode)) *NodeKeeperMock {
	if mmSync.defaultExpectation != nil {
		mmSync.mock.t.Fatalf("Default expectation is already set for the NodeKeeper.Sync method")
	}

	if len(mmSync.expectations) > 0 {
		mmSync.mock.t.Fatalf("Some expectations are already set for the NodeKeeper.Sync method")
	}

	mmSync.mock.funcSync = f
	return mmSync.mock
}

// Sync implements network.NodeKeeper
func (mmSync *NodeKeeperMock) Sync(ctx context.Context, n1 pulse.Number, na1 []node.NetworkNode) {
	mm_atomic.AddUint64(&mmSync.beforeSyncCounter, 1)
	defer mm_atomic.AddUint64(&mmSync.afterSyncCounter, 1)

	if mmSync.inspectFuncSync != nil {
		mmSync.inspectFuncSync(ctx, n1, na1)
	}

	mm_params := &NodeKeeperMockSyncParams{ctx, n1, na1}

	// Record call args
	mmSync.SyncMock.mutex.Lock()
	mmSync.SyncMock.callArgs = append(mmSync.SyncMock.callArgs, mm_params)
	mmSync.SyncMock.mutex.Unlock()

	for _, e := range mmSync.SyncMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return
		}
	}

	if mmSync.SyncMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSync.SyncMock.defaultExpectation.Counter, 1)
		mm_want := mmSync.SyncMock.defaultExpectation.params
		mm_got := NodeKeeperMockSyncParams{ctx, n1, na1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSync.t.Errorf("NodeKeeperMock.Sync got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		return

	}
	if mmSync.funcSync != nil {
		mmSync.funcSync(ctx, n1, na1)
		return
	}
	mmSync.t.Fatalf("Unexpected call to NodeKeeperMock.Sync. %v %v %v", ctx, n1, na1)

}

// SyncAfterCounter returns a count of finished NodeKeeperMock.Sync invocations
func (mmSync *NodeKeeperMock) SyncAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSync.afterSyncCounter)
}

// SyncBeforeCounter returns a count of NodeKeeperMock.Sync invocations
func (mmSync *NodeKeeperMock) SyncBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSync.beforeSyncCounter)
}

// Calls returns a list of arguments used in each call to NodeKeeperMock.Sync.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSync *mNodeKeeperMockSync) Calls() []*NodeKeeperMockSyncParams {
	mmSync.mutex.RLock()

	argCopy := make([]*NodeKeeperMockSyncParams, len(mmSync.callArgs))
	copy(argCopy, mmSync.callArgs)

	mmSync.mutex.RUnlock()

	return argCopy
}

// MinimockSyncDone returns true if the count of the Sync invocations corresponds
// the number of defined expectations
func (m *NodeKeeperMock) MinimockSyncDone() bool {
	for _, e := range m.SyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSyncCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSync != nil && mm_atomic.LoadUint64(&m.afterSyncCounter) < 1 {
		return false
	}
	return true
}

// MinimockSyncInspect logs each unmet expectation
func (m *NodeKeeperMock) MinimockSyncInspect() {
	for _, e := range m.SyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to NodeKeeperMock.Sync with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSyncCounter) < 1 {
		if m.SyncMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to NodeKeeperMock.Sync")
		} else {
			m.t.Errorf("Expected call to NodeKeeperMock.Sync with params: %#v", *m.SyncMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSync != nil && mm_atomic.LoadUint64(&m.afterSyncCounter) < 1 {
		m.t.Error("Expected call to NodeKeeperMock.Sync")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *NodeKeeperMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetAccessorInspect()

		m.MinimockGetOriginInspect()

		m.MinimockMoveSyncToActiveInspect()

		m.MinimockSetInitialSnapshotInspect()

		m.MinimockSyncInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *NodeKeeperMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *NodeKeeperMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetAccessorDone() &&
		m.MinimockGetOriginDone() &&
		m.MinimockMoveSyncToActiveDone() &&
		m.MinimockSetInitialSnapshotDone() &&
		m.MinimockSyncDone()
}
