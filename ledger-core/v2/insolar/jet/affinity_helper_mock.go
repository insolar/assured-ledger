package jet

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// AffinityHelperMock implements AffinityHelper
type AffinityHelperMock struct {
	t minimock.Tester

	funcMe          func() (g1 reference.Global)
	inspectFuncMe   func()
	afterMeCounter  uint64
	beforeMeCounter uint64
	MeMock          mAffinityHelperMockMe

	funcQueryRole          func(ctx context.Context, role node.DynamicRole, obj reference.Local, pulse pulse.Number) (ga1 []reference.Global, err error)
	inspectFuncQueryRole   func(ctx context.Context, role node.DynamicRole, obj reference.Local, pulse pulse.Number)
	afterQueryRoleCounter  uint64
	beforeQueryRoleCounter uint64
	QueryRoleMock          mAffinityHelperMockQueryRole
}

// NewAffinityHelperMock returns a mock for AffinityHelper
func NewAffinityHelperMock(t minimock.Tester) *AffinityHelperMock {
	m := &AffinityHelperMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.MeMock = mAffinityHelperMockMe{mock: m}

	m.QueryRoleMock = mAffinityHelperMockQueryRole{mock: m}
	m.QueryRoleMock.callArgs = []*AffinityHelperMockQueryRoleParams{}

	return m
}

type mAffinityHelperMockMe struct {
	mock               *AffinityHelperMock
	defaultExpectation *AffinityHelperMockMeExpectation
	expectations       []*AffinityHelperMockMeExpectation
}

// AffinityHelperMockMeExpectation specifies expectation struct of the AffinityHelper.Me
type AffinityHelperMockMeExpectation struct {
	mock *AffinityHelperMock

	results *AffinityHelperMockMeResults
	Counter uint64
}

// AffinityHelperMockMeResults contains results of the AffinityHelper.Me
type AffinityHelperMockMeResults struct {
	g1 reference.Global
}

// Expect sets up expected params for AffinityHelper.Me
func (mmMe *mAffinityHelperMockMe) Expect() *mAffinityHelperMockMe {
	if mmMe.mock.funcMe != nil {
		mmMe.mock.t.Fatalf("AffinityHelperMock.Me mock is already set by Set")
	}

	if mmMe.defaultExpectation == nil {
		mmMe.defaultExpectation = &AffinityHelperMockMeExpectation{}
	}

	return mmMe
}

// Inspect accepts an inspector function that has same arguments as the AffinityHelper.Me
func (mmMe *mAffinityHelperMockMe) Inspect(f func()) *mAffinityHelperMockMe {
	if mmMe.mock.inspectFuncMe != nil {
		mmMe.mock.t.Fatalf("Inspect function is already set for AffinityHelperMock.Me")
	}

	mmMe.mock.inspectFuncMe = f

	return mmMe
}

// Return sets up results that will be returned by AffinityHelper.Me
func (mmMe *mAffinityHelperMockMe) Return(g1 reference.Global) *AffinityHelperMock {
	if mmMe.mock.funcMe != nil {
		mmMe.mock.t.Fatalf("AffinityHelperMock.Me mock is already set by Set")
	}

	if mmMe.defaultExpectation == nil {
		mmMe.defaultExpectation = &AffinityHelperMockMeExpectation{mock: mmMe.mock}
	}
	mmMe.defaultExpectation.results = &AffinityHelperMockMeResults{g1}
	return mmMe.mock
}

//Set uses given function f to mock the AffinityHelper.Me method
func (mmMe *mAffinityHelperMockMe) Set(f func() (g1 reference.Global)) *AffinityHelperMock {
	if mmMe.defaultExpectation != nil {
		mmMe.mock.t.Fatalf("Default expectation is already set for the AffinityHelper.Me method")
	}

	if len(mmMe.expectations) > 0 {
		mmMe.mock.t.Fatalf("Some expectations are already set for the AffinityHelper.Me method")
	}

	mmMe.mock.funcMe = f
	return mmMe.mock
}

// Me implements AffinityHelper
func (mmMe *AffinityHelperMock) Me() (g1 reference.Global) {
	mm_atomic.AddUint64(&mmMe.beforeMeCounter, 1)
	defer mm_atomic.AddUint64(&mmMe.afterMeCounter, 1)

	if mmMe.inspectFuncMe != nil {
		mmMe.inspectFuncMe()
	}

	if mmMe.MeMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmMe.MeMock.defaultExpectation.Counter, 1)

		mm_results := mmMe.MeMock.defaultExpectation.results
		if mm_results == nil {
			mmMe.t.Fatal("No results are set for the AffinityHelperMock.Me")
		}
		return (*mm_results).g1
	}
	if mmMe.funcMe != nil {
		return mmMe.funcMe()
	}
	mmMe.t.Fatalf("Unexpected call to AffinityHelperMock.Me.")
	return
}

// MeAfterCounter returns a count of finished AffinityHelperMock.Me invocations
func (mmMe *AffinityHelperMock) MeAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMe.afterMeCounter)
}

// MeBeforeCounter returns a count of AffinityHelperMock.Me invocations
func (mmMe *AffinityHelperMock) MeBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmMe.beforeMeCounter)
}

// MinimockMeDone returns true if the count of the Me invocations corresponds
// the number of defined expectations
func (m *AffinityHelperMock) MinimockMeDone() bool {
	for _, e := range m.MeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.MeMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterMeCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMe != nil && mm_atomic.LoadUint64(&m.afterMeCounter) < 1 {
		return false
	}
	return true
}

// MinimockMeInspect logs each unmet expectation
func (m *AffinityHelperMock) MinimockMeInspect() {
	for _, e := range m.MeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to AffinityHelperMock.Me")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.MeMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterMeCounter) < 1 {
		m.t.Error("Expected call to AffinityHelperMock.Me")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcMe != nil && mm_atomic.LoadUint64(&m.afterMeCounter) < 1 {
		m.t.Error("Expected call to AffinityHelperMock.Me")
	}
}

type mAffinityHelperMockQueryRole struct {
	mock               *AffinityHelperMock
	defaultExpectation *AffinityHelperMockQueryRoleExpectation
	expectations       []*AffinityHelperMockQueryRoleExpectation

	callArgs []*AffinityHelperMockQueryRoleParams
	mutex    sync.RWMutex
}

// AffinityHelperMockQueryRoleExpectation specifies expectation struct of the AffinityHelper.QueryRole
type AffinityHelperMockQueryRoleExpectation struct {
	mock    *AffinityHelperMock
	params  *AffinityHelperMockQueryRoleParams
	results *AffinityHelperMockQueryRoleResults
	Counter uint64
}

// AffinityHelperMockQueryRoleParams contains parameters of the AffinityHelper.QueryRole
type AffinityHelperMockQueryRoleParams struct {
	ctx   context.Context
	role  node.DynamicRole
	obj   reference.Local
	pulse pulse.Number
}

// AffinityHelperMockQueryRoleResults contains results of the AffinityHelper.QueryRole
type AffinityHelperMockQueryRoleResults struct {
	ga1 []reference.Global
	err error
}

// Expect sets up expected params for AffinityHelper.QueryRole
func (mmQueryRole *mAffinityHelperMockQueryRole) Expect(ctx context.Context, role node.DynamicRole, obj reference.Local, pulse pulse.Number) *mAffinityHelperMockQueryRole {
	if mmQueryRole.mock.funcQueryRole != nil {
		mmQueryRole.mock.t.Fatalf("AffinityHelperMock.QueryRole mock is already set by Set")
	}

	if mmQueryRole.defaultExpectation == nil {
		mmQueryRole.defaultExpectation = &AffinityHelperMockQueryRoleExpectation{}
	}

	mmQueryRole.defaultExpectation.params = &AffinityHelperMockQueryRoleParams{ctx, role, obj, pulse}
	for _, e := range mmQueryRole.expectations {
		if minimock.Equal(e.params, mmQueryRole.defaultExpectation.params) {
			mmQueryRole.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmQueryRole.defaultExpectation.params)
		}
	}

	return mmQueryRole
}

// Inspect accepts an inspector function that has same arguments as the AffinityHelper.QueryRole
func (mmQueryRole *mAffinityHelperMockQueryRole) Inspect(f func(ctx context.Context, role node.DynamicRole, obj reference.Local, pulse pulse.Number)) *mAffinityHelperMockQueryRole {
	if mmQueryRole.mock.inspectFuncQueryRole != nil {
		mmQueryRole.mock.t.Fatalf("Inspect function is already set for AffinityHelperMock.QueryRole")
	}

	mmQueryRole.mock.inspectFuncQueryRole = f

	return mmQueryRole
}

// Return sets up results that will be returned by AffinityHelper.QueryRole
func (mmQueryRole *mAffinityHelperMockQueryRole) Return(ga1 []reference.Global, err error) *AffinityHelperMock {
	if mmQueryRole.mock.funcQueryRole != nil {
		mmQueryRole.mock.t.Fatalf("AffinityHelperMock.QueryRole mock is already set by Set")
	}

	if mmQueryRole.defaultExpectation == nil {
		mmQueryRole.defaultExpectation = &AffinityHelperMockQueryRoleExpectation{mock: mmQueryRole.mock}
	}
	mmQueryRole.defaultExpectation.results = &AffinityHelperMockQueryRoleResults{ga1, err}
	return mmQueryRole.mock
}

//Set uses given function f to mock the AffinityHelper.QueryRole method
func (mmQueryRole *mAffinityHelperMockQueryRole) Set(f func(ctx context.Context, role node.DynamicRole, obj reference.Local, pulse pulse.Number) (ga1 []reference.Global, err error)) *AffinityHelperMock {
	if mmQueryRole.defaultExpectation != nil {
		mmQueryRole.mock.t.Fatalf("Default expectation is already set for the AffinityHelper.QueryRole method")
	}

	if len(mmQueryRole.expectations) > 0 {
		mmQueryRole.mock.t.Fatalf("Some expectations are already set for the AffinityHelper.QueryRole method")
	}

	mmQueryRole.mock.funcQueryRole = f
	return mmQueryRole.mock
}

// When sets expectation for the AffinityHelper.QueryRole which will trigger the result defined by the following
// Then helper
func (mmQueryRole *mAffinityHelperMockQueryRole) When(ctx context.Context, role node.DynamicRole, obj reference.Local, pulse pulse.Number) *AffinityHelperMockQueryRoleExpectation {
	if mmQueryRole.mock.funcQueryRole != nil {
		mmQueryRole.mock.t.Fatalf("AffinityHelperMock.QueryRole mock is already set by Set")
	}

	expectation := &AffinityHelperMockQueryRoleExpectation{
		mock:   mmQueryRole.mock,
		params: &AffinityHelperMockQueryRoleParams{ctx, role, obj, pulse},
	}
	mmQueryRole.expectations = append(mmQueryRole.expectations, expectation)
	return expectation
}

// Then sets up AffinityHelper.QueryRole return parameters for the expectation previously defined by the When method
func (e *AffinityHelperMockQueryRoleExpectation) Then(ga1 []reference.Global, err error) *AffinityHelperMock {
	e.results = &AffinityHelperMockQueryRoleResults{ga1, err}
	return e.mock
}

// QueryRole implements AffinityHelper
func (mmQueryRole *AffinityHelperMock) QueryRole(ctx context.Context, role node.DynamicRole, obj reference.Local, pulse pulse.Number) (ga1 []reference.Global, err error) {
	mm_atomic.AddUint64(&mmQueryRole.beforeQueryRoleCounter, 1)
	defer mm_atomic.AddUint64(&mmQueryRole.afterQueryRoleCounter, 1)

	if mmQueryRole.inspectFuncQueryRole != nil {
		mmQueryRole.inspectFuncQueryRole(ctx, role, obj, pulse)
	}

	mm_params := &AffinityHelperMockQueryRoleParams{ctx, role, obj, pulse}

	// Record call args
	mmQueryRole.QueryRoleMock.mutex.Lock()
	mmQueryRole.QueryRoleMock.callArgs = append(mmQueryRole.QueryRoleMock.callArgs, mm_params)
	mmQueryRole.QueryRoleMock.mutex.Unlock()

	for _, e := range mmQueryRole.QueryRoleMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.ga1, e.results.err
		}
	}

	if mmQueryRole.QueryRoleMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmQueryRole.QueryRoleMock.defaultExpectation.Counter, 1)
		mm_want := mmQueryRole.QueryRoleMock.defaultExpectation.params
		mm_got := AffinityHelperMockQueryRoleParams{ctx, role, obj, pulse}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmQueryRole.t.Errorf("AffinityHelperMock.QueryRole got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmQueryRole.QueryRoleMock.defaultExpectation.results
		if mm_results == nil {
			mmQueryRole.t.Fatal("No results are set for the AffinityHelperMock.QueryRole")
		}
		return (*mm_results).ga1, (*mm_results).err
	}
	if mmQueryRole.funcQueryRole != nil {
		return mmQueryRole.funcQueryRole(ctx, role, obj, pulse)
	}
	mmQueryRole.t.Fatalf("Unexpected call to AffinityHelperMock.QueryRole. %v %v %v %v", ctx, role, obj, pulse)
	return
}

// QueryRoleAfterCounter returns a count of finished AffinityHelperMock.QueryRole invocations
func (mmQueryRole *AffinityHelperMock) QueryRoleAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmQueryRole.afterQueryRoleCounter)
}

// QueryRoleBeforeCounter returns a count of AffinityHelperMock.QueryRole invocations
func (mmQueryRole *AffinityHelperMock) QueryRoleBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmQueryRole.beforeQueryRoleCounter)
}

// Calls returns a list of arguments used in each call to AffinityHelperMock.QueryRole.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmQueryRole *mAffinityHelperMockQueryRole) Calls() []*AffinityHelperMockQueryRoleParams {
	mmQueryRole.mutex.RLock()

	argCopy := make([]*AffinityHelperMockQueryRoleParams, len(mmQueryRole.callArgs))
	copy(argCopy, mmQueryRole.callArgs)

	mmQueryRole.mutex.RUnlock()

	return argCopy
}

// MinimockQueryRoleDone returns true if the count of the QueryRole invocations corresponds
// the number of defined expectations
func (m *AffinityHelperMock) MinimockQueryRoleDone() bool {
	for _, e := range m.QueryRoleMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.QueryRoleMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterQueryRoleCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcQueryRole != nil && mm_atomic.LoadUint64(&m.afterQueryRoleCounter) < 1 {
		return false
	}
	return true
}

// MinimockQueryRoleInspect logs each unmet expectation
func (m *AffinityHelperMock) MinimockQueryRoleInspect() {
	for _, e := range m.QueryRoleMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to AffinityHelperMock.QueryRole with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.QueryRoleMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterQueryRoleCounter) < 1 {
		if m.QueryRoleMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to AffinityHelperMock.QueryRole")
		} else {
			m.t.Errorf("Expected call to AffinityHelperMock.QueryRole with params: %#v", *m.QueryRoleMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcQueryRole != nil && mm_atomic.LoadUint64(&m.afterQueryRoleCounter) < 1 {
		m.t.Error("Expected call to AffinityHelperMock.QueryRole")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *AffinityHelperMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockMeInspect()

		m.MinimockQueryRoleInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *AffinityHelperMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *AffinityHelperMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockMeDone() &&
		m.MinimockQueryRoleDone()
}
