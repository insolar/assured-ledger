package messagesender

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
)

// ServiceMock implements Service
type ServiceMock struct {
	t minimock.Tester

	funcSendRole          func(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) (err error)
	inspectFuncSendRole   func(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption)
	afterSendRoleCounter  uint64
	beforeSendRoleCounter uint64
	SendRoleMock          mServiceMockSendRole

	funcSendTarget          func(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption) (err error)
	inspectFuncSendTarget   func(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption)
	afterSendTargetCounter  uint64
	beforeSendTargetCounter uint64
	SendTargetMock          mServiceMockSendTarget
}

// NewServiceMock returns a mock for Service
func NewServiceMock(t minimock.Tester) *ServiceMock {
	m := &ServiceMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.SendRoleMock = mServiceMockSendRole{mock: m}
	m.SendRoleMock.callArgs = []*ServiceMockSendRoleParams{}

	m.SendTargetMock = mServiceMockSendTarget{mock: m}
	m.SendTargetMock.callArgs = []*ServiceMockSendTargetParams{}

	return m
}

type mServiceMockSendRole struct {
	mock               *ServiceMock
	defaultExpectation *ServiceMockSendRoleExpectation
	expectations       []*ServiceMockSendRoleExpectation

	callArgs []*ServiceMockSendRoleParams
	mutex    sync.RWMutex
}

// ServiceMockSendRoleExpectation specifies expectation struct of the Service.SendRole
type ServiceMockSendRoleExpectation struct {
	mock    *ServiceMock
	params  *ServiceMockSendRoleParams
	results *ServiceMockSendRoleResults
	Counter uint64
}

// ServiceMockSendRoleParams contains parameters of the Service.SendRole
type ServiceMockSendRoleParams struct {
	ctx    context.Context
	msg    rmsreg.GoGoSerializable
	role   affinity.DynamicRole
	object reference.Global
	pn     pulse.Number
	opts   []SendOption
}

// ServiceMockSendRoleResults contains results of the Service.SendRole
type ServiceMockSendRoleResults struct {
	err error
}

// Expect sets up expected params for Service.SendRole
func (mmSendRole *mServiceMockSendRole) Expect(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) *mServiceMockSendRole {
	if mmSendRole.mock.funcSendRole != nil {
		mmSendRole.mock.t.Fatalf("ServiceMock.SendRole mock is already set by Set")
	}

	if mmSendRole.defaultExpectation == nil {
		mmSendRole.defaultExpectation = &ServiceMockSendRoleExpectation{}
	}

	mmSendRole.defaultExpectation.params = &ServiceMockSendRoleParams{ctx, msg, role, object, pn, opts}
	for _, e := range mmSendRole.expectations {
		if minimock.Equal(e.params, mmSendRole.defaultExpectation.params) {
			mmSendRole.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSendRole.defaultExpectation.params)
		}
	}

	return mmSendRole
}

// Inspect accepts an inspector function that has same arguments as the Service.SendRole
func (mmSendRole *mServiceMockSendRole) Inspect(f func(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption)) *mServiceMockSendRole {
	if mmSendRole.mock.inspectFuncSendRole != nil {
		mmSendRole.mock.t.Fatalf("Inspect function is already set for ServiceMock.SendRole")
	}

	mmSendRole.mock.inspectFuncSendRole = f

	return mmSendRole
}

// Return sets up results that will be returned by Service.SendRole
func (mmSendRole *mServiceMockSendRole) Return(err error) *ServiceMock {
	if mmSendRole.mock.funcSendRole != nil {
		mmSendRole.mock.t.Fatalf("ServiceMock.SendRole mock is already set by Set")
	}

	if mmSendRole.defaultExpectation == nil {
		mmSendRole.defaultExpectation = &ServiceMockSendRoleExpectation{mock: mmSendRole.mock}
	}
	mmSendRole.defaultExpectation.results = &ServiceMockSendRoleResults{err}
	return mmSendRole.mock
}

//Set uses given function f to mock the Service.SendRole method
func (mmSendRole *mServiceMockSendRole) Set(f func(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) (err error)) *ServiceMock {
	if mmSendRole.defaultExpectation != nil {
		mmSendRole.mock.t.Fatalf("Default expectation is already set for the Service.SendRole method")
	}

	if len(mmSendRole.expectations) > 0 {
		mmSendRole.mock.t.Fatalf("Some expectations are already set for the Service.SendRole method")
	}

	mmSendRole.mock.funcSendRole = f
	return mmSendRole.mock
}

// When sets expectation for the Service.SendRole which will trigger the result defined by the following
// Then helper
func (mmSendRole *mServiceMockSendRole) When(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) *ServiceMockSendRoleExpectation {
	if mmSendRole.mock.funcSendRole != nil {
		mmSendRole.mock.t.Fatalf("ServiceMock.SendRole mock is already set by Set")
	}

	expectation := &ServiceMockSendRoleExpectation{
		mock:   mmSendRole.mock,
		params: &ServiceMockSendRoleParams{ctx, msg, role, object, pn, opts},
	}
	mmSendRole.expectations = append(mmSendRole.expectations, expectation)
	return expectation
}

// Then sets up Service.SendRole return parameters for the expectation previously defined by the When method
func (e *ServiceMockSendRoleExpectation) Then(err error) *ServiceMock {
	e.results = &ServiceMockSendRoleResults{err}
	return e.mock
}

// SendRole implements Service
func (mmSendRole *ServiceMock) SendRole(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) (err error) {
	mm_atomic.AddUint64(&mmSendRole.beforeSendRoleCounter, 1)
	defer mm_atomic.AddUint64(&mmSendRole.afterSendRoleCounter, 1)

	if mmSendRole.inspectFuncSendRole != nil {
		mmSendRole.inspectFuncSendRole(ctx, msg, role, object, pn, opts...)
	}

	mm_params := &ServiceMockSendRoleParams{ctx, msg, role, object, pn, opts}

	// Record call args
	mmSendRole.SendRoleMock.mutex.Lock()
	mmSendRole.SendRoleMock.callArgs = append(mmSendRole.SendRoleMock.callArgs, mm_params)
	mmSendRole.SendRoleMock.mutex.Unlock()

	for _, e := range mmSendRole.SendRoleMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmSendRole.SendRoleMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSendRole.SendRoleMock.defaultExpectation.Counter, 1)
		mm_want := mmSendRole.SendRoleMock.defaultExpectation.params
		mm_got := ServiceMockSendRoleParams{ctx, msg, role, object, pn, opts}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSendRole.t.Errorf("ServiceMock.SendRole got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSendRole.SendRoleMock.defaultExpectation.results
		if mm_results == nil {
			mmSendRole.t.Fatal("No results are set for the ServiceMock.SendRole")
		}
		return (*mm_results).err
	}
	if mmSendRole.funcSendRole != nil {
		return mmSendRole.funcSendRole(ctx, msg, role, object, pn, opts...)
	}
	mmSendRole.t.Fatalf("Unexpected call to ServiceMock.SendRole. %v %v %v %v %v %v", ctx, msg, role, object, pn, opts)
	return
}

// SendRoleAfterCounter returns a count of finished ServiceMock.SendRole invocations
func (mmSendRole *ServiceMock) SendRoleAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSendRole.afterSendRoleCounter)
}

// SendRoleBeforeCounter returns a count of ServiceMock.SendRole invocations
func (mmSendRole *ServiceMock) SendRoleBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSendRole.beforeSendRoleCounter)
}

// Calls returns a list of arguments used in each call to ServiceMock.SendRole.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSendRole *mServiceMockSendRole) Calls() []*ServiceMockSendRoleParams {
	mmSendRole.mutex.RLock()

	argCopy := make([]*ServiceMockSendRoleParams, len(mmSendRole.callArgs))
	copy(argCopy, mmSendRole.callArgs)

	mmSendRole.mutex.RUnlock()

	return argCopy
}

// MinimockSendRoleDone returns true if the count of the SendRole invocations corresponds
// the number of defined expectations
func (m *ServiceMock) MinimockSendRoleDone() bool {
	for _, e := range m.SendRoleMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SendRoleMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSendRoleCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSendRole != nil && mm_atomic.LoadUint64(&m.afterSendRoleCounter) < 1 {
		return false
	}
	return true
}

// MinimockSendRoleInspect logs each unmet expectation
func (m *ServiceMock) MinimockSendRoleInspect() {
	for _, e := range m.SendRoleMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ServiceMock.SendRole with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SendRoleMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSendRoleCounter) < 1 {
		if m.SendRoleMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ServiceMock.SendRole")
		} else {
			m.t.Errorf("Expected call to ServiceMock.SendRole with params: %#v", *m.SendRoleMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSendRole != nil && mm_atomic.LoadUint64(&m.afterSendRoleCounter) < 1 {
		m.t.Error("Expected call to ServiceMock.SendRole")
	}
}

type mServiceMockSendTarget struct {
	mock               *ServiceMock
	defaultExpectation *ServiceMockSendTargetExpectation
	expectations       []*ServiceMockSendTargetExpectation

	callArgs []*ServiceMockSendTargetParams
	mutex    sync.RWMutex
}

// ServiceMockSendTargetExpectation specifies expectation struct of the Service.SendTarget
type ServiceMockSendTargetExpectation struct {
	mock    *ServiceMock
	params  *ServiceMockSendTargetParams
	results *ServiceMockSendTargetResults
	Counter uint64
}

// ServiceMockSendTargetParams contains parameters of the Service.SendTarget
type ServiceMockSendTargetParams struct {
	ctx    context.Context
	msg    rmsreg.GoGoSerializable
	target reference.Global
	opts   []SendOption
}

// ServiceMockSendTargetResults contains results of the Service.SendTarget
type ServiceMockSendTargetResults struct {
	err error
}

// Expect sets up expected params for Service.SendTarget
func (mmSendTarget *mServiceMockSendTarget) Expect(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption) *mServiceMockSendTarget {
	if mmSendTarget.mock.funcSendTarget != nil {
		mmSendTarget.mock.t.Fatalf("ServiceMock.SendTarget mock is already set by Set")
	}

	if mmSendTarget.defaultExpectation == nil {
		mmSendTarget.defaultExpectation = &ServiceMockSendTargetExpectation{}
	}

	mmSendTarget.defaultExpectation.params = &ServiceMockSendTargetParams{ctx, msg, target, opts}
	for _, e := range mmSendTarget.expectations {
		if minimock.Equal(e.params, mmSendTarget.defaultExpectation.params) {
			mmSendTarget.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmSendTarget.defaultExpectation.params)
		}
	}

	return mmSendTarget
}

// Inspect accepts an inspector function that has same arguments as the Service.SendTarget
func (mmSendTarget *mServiceMockSendTarget) Inspect(f func(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption)) *mServiceMockSendTarget {
	if mmSendTarget.mock.inspectFuncSendTarget != nil {
		mmSendTarget.mock.t.Fatalf("Inspect function is already set for ServiceMock.SendTarget")
	}

	mmSendTarget.mock.inspectFuncSendTarget = f

	return mmSendTarget
}

// Return sets up results that will be returned by Service.SendTarget
func (mmSendTarget *mServiceMockSendTarget) Return(err error) *ServiceMock {
	if mmSendTarget.mock.funcSendTarget != nil {
		mmSendTarget.mock.t.Fatalf("ServiceMock.SendTarget mock is already set by Set")
	}

	if mmSendTarget.defaultExpectation == nil {
		mmSendTarget.defaultExpectation = &ServiceMockSendTargetExpectation{mock: mmSendTarget.mock}
	}
	mmSendTarget.defaultExpectation.results = &ServiceMockSendTargetResults{err}
	return mmSendTarget.mock
}

//Set uses given function f to mock the Service.SendTarget method
func (mmSendTarget *mServiceMockSendTarget) Set(f func(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption) (err error)) *ServiceMock {
	if mmSendTarget.defaultExpectation != nil {
		mmSendTarget.mock.t.Fatalf("Default expectation is already set for the Service.SendTarget method")
	}

	if len(mmSendTarget.expectations) > 0 {
		mmSendTarget.mock.t.Fatalf("Some expectations are already set for the Service.SendTarget method")
	}

	mmSendTarget.mock.funcSendTarget = f
	return mmSendTarget.mock
}

// When sets expectation for the Service.SendTarget which will trigger the result defined by the following
// Then helper
func (mmSendTarget *mServiceMockSendTarget) When(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption) *ServiceMockSendTargetExpectation {
	if mmSendTarget.mock.funcSendTarget != nil {
		mmSendTarget.mock.t.Fatalf("ServiceMock.SendTarget mock is already set by Set")
	}

	expectation := &ServiceMockSendTargetExpectation{
		mock:   mmSendTarget.mock,
		params: &ServiceMockSendTargetParams{ctx, msg, target, opts},
	}
	mmSendTarget.expectations = append(mmSendTarget.expectations, expectation)
	return expectation
}

// Then sets up Service.SendTarget return parameters for the expectation previously defined by the When method
func (e *ServiceMockSendTargetExpectation) Then(err error) *ServiceMock {
	e.results = &ServiceMockSendTargetResults{err}
	return e.mock
}

// SendTarget implements Service
func (mmSendTarget *ServiceMock) SendTarget(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption) (err error) {
	mm_atomic.AddUint64(&mmSendTarget.beforeSendTargetCounter, 1)
	defer mm_atomic.AddUint64(&mmSendTarget.afterSendTargetCounter, 1)

	if mmSendTarget.inspectFuncSendTarget != nil {
		mmSendTarget.inspectFuncSendTarget(ctx, msg, target, opts...)
	}

	mm_params := &ServiceMockSendTargetParams{ctx, msg, target, opts}

	// Record call args
	mmSendTarget.SendTargetMock.mutex.Lock()
	mmSendTarget.SendTargetMock.callArgs = append(mmSendTarget.SendTargetMock.callArgs, mm_params)
	mmSendTarget.SendTargetMock.mutex.Unlock()

	for _, e := range mmSendTarget.SendTargetMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmSendTarget.SendTargetMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmSendTarget.SendTargetMock.defaultExpectation.Counter, 1)
		mm_want := mmSendTarget.SendTargetMock.defaultExpectation.params
		mm_got := ServiceMockSendTargetParams{ctx, msg, target, opts}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmSendTarget.t.Errorf("ServiceMock.SendTarget got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmSendTarget.SendTargetMock.defaultExpectation.results
		if mm_results == nil {
			mmSendTarget.t.Fatal("No results are set for the ServiceMock.SendTarget")
		}
		return (*mm_results).err
	}
	if mmSendTarget.funcSendTarget != nil {
		return mmSendTarget.funcSendTarget(ctx, msg, target, opts...)
	}
	mmSendTarget.t.Fatalf("Unexpected call to ServiceMock.SendTarget. %v %v %v %v", ctx, msg, target, opts)
	return
}

// SendTargetAfterCounter returns a count of finished ServiceMock.SendTarget invocations
func (mmSendTarget *ServiceMock) SendTargetAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSendTarget.afterSendTargetCounter)
}

// SendTargetBeforeCounter returns a count of ServiceMock.SendTarget invocations
func (mmSendTarget *ServiceMock) SendTargetBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmSendTarget.beforeSendTargetCounter)
}

// Calls returns a list of arguments used in each call to ServiceMock.SendTarget.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmSendTarget *mServiceMockSendTarget) Calls() []*ServiceMockSendTargetParams {
	mmSendTarget.mutex.RLock()

	argCopy := make([]*ServiceMockSendTargetParams, len(mmSendTarget.callArgs))
	copy(argCopy, mmSendTarget.callArgs)

	mmSendTarget.mutex.RUnlock()

	return argCopy
}

// MinimockSendTargetDone returns true if the count of the SendTarget invocations corresponds
// the number of defined expectations
func (m *ServiceMock) MinimockSendTargetDone() bool {
	for _, e := range m.SendTargetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SendTargetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSendTargetCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSendTarget != nil && mm_atomic.LoadUint64(&m.afterSendTargetCounter) < 1 {
		return false
	}
	return true
}

// MinimockSendTargetInspect logs each unmet expectation
func (m *ServiceMock) MinimockSendTargetInspect() {
	for _, e := range m.SendTargetMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ServiceMock.SendTarget with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.SendTargetMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterSendTargetCounter) < 1 {
		if m.SendTargetMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ServiceMock.SendTarget")
		} else {
			m.t.Errorf("Expected call to ServiceMock.SendTarget with params: %#v", *m.SendTargetMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcSendTarget != nil && mm_atomic.LoadUint64(&m.afterSendTargetCounter) < 1 {
		m.t.Error("Expected call to ServiceMock.SendTarget")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *ServiceMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockSendRoleInspect()

		m.MinimockSendTargetInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *ServiceMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *ServiceMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockSendRoleDone() &&
		m.MinimockSendTargetDone()
}
