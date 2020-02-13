// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package termination

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	mock "github.com/insolar/assured-ledger/ledger-core/v2/testutils/network"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/suite"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

type CommonTestSuite struct {
	suite.Suite

	mc            *minimock.Controller
	ctx           context.Context
	handler       *Handler
	leaver        *testutils.LeaverMock
	pulseAccessor *mock.PulseAccessorMock
}

func TestBasics(t *testing.T) {
	suite.Run(t, new(CommonTestSuite))
}

func (s *CommonTestSuite) BeforeTest(suiteName, testName string) {
	s.mc = minimock.NewController(s.T())
	s.ctx = inslogger.TestContext(s.T())
	s.leaver = testutils.NewLeaverMock(s.T())
	s.pulseAccessor = mock.NewPulseAccessorMock(s.T())
	s.handler = &Handler{Leaver: s.leaver, PulseAccessor: s.pulseAccessor}

}

func (s *CommonTestSuite) AfterTest(suiteName, testName string) {
	s.mc.Wait(time.Minute)
	s.mc.Finish()
}

func (s *CommonTestSuite) TestHandlerInitialState() {
	s.Equal(0, cap(s.handler.done))
	s.Equal(false, s.handler.terminating)
}

func (s *CommonTestSuite) HandlerIsTerminating() {
	s.Equal(true, s.handler.terminating)
	s.Equal(1, cap(s.handler.done))
}

func TestLeave(t *testing.T) {
	suite.Run(t, new(LeaveTestSuite))
}

type LeaveTestSuite struct {
	CommonTestSuite
}

func (s *LeaveTestSuite) TestLeaveNow() {
	s.leaver.LeaveMock.Expect(s.ctx, 0)
	s.handler.leave(s.ctx, 0)

	s.HandlerIsTerminating()
}

func (s *LeaveTestSuite) TestLeaveEta() {
	mockPulseNumber := insolar.PulseNumber(2000000000)
	testPulse := &insolar.Pulse{PulseNumber: mockPulseNumber}
	pulseDelta := testPulse.NextPulseNumber - testPulse.PulseNumber
	leaveAfter := insolar.PulseNumber(5)

	s.pulseAccessor.GetLatestPulseMock.Return(*testPulse, nil)
	s.leaver.LeaveMock.Expect(s.ctx, mockPulseNumber+leaveAfter*pulseDelta)
	s.handler.leave(s.ctx, leaveAfter)

	s.HandlerIsTerminating()
}

func TestOnLeaveApproved(t *testing.T) {
	suite.Run(t, new(OnLeaveApprovedTestSuite))
}

type OnLeaveApprovedTestSuite struct {
	CommonTestSuite
}

func (s *OnLeaveApprovedTestSuite) TestBasicUsage() {
	s.handler.terminating = true
	s.handler.done = make(chan struct{}, 1)

	s.handler.OnLeaveApproved(s.ctx)

	select {
	case <-s.handler.done:
		s.Equal(false, s.handler.terminating)
	case <-time.After(time.Second):
		s.Fail("done chanel doesn't close")
	}
}

func TestAbort(t *testing.T) {
	mc := minimock.NewController(t)
	defer mc.Finish()
	defer mc.Wait(time.Minute)

	ctx := context.Background()
	handler := NewHandler(nil)
	require.NotNil(t, handler)

	intercept := interceptLoggerOutput{make(chan string)}
	ctx = inslogger.UpdateLogger(context.Background(), func(l log.Logger) (log.Logger, error) {
		return l.Copy().WithOutput(&intercept).Build()
	})

	go func() {
		// As About() calls Fatal, then the call must hang on write to avoid os.Exit()
		handler.Abort(ctx, "abort")
		require.FailNow(t, "must hang")
	}()

	select {
	case msg := <-intercept.out:
		require.Contains(t, msg, "abort")
	case <-time.After(10 * time.Second):
		require.FailNow(t, "timeout")
	}
}

type interceptLoggerOutput struct {
	out chan string
}

func (p interceptLoggerOutput) Write(b []byte) (int, error) {
	p.out <- string(b)
	select {} // hang up
}
