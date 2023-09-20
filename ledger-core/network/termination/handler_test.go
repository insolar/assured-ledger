package termination

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/suite"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/pulse"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

type CommonTestSuite struct {
	suite.Suite

	mc            *minimock.Controller
	ctx           context.Context
	handler       *Handler
	leaver        *LeaverMock
	pulseAccessor *beat.AppenderMock
}

func TestBasics(t *testing.T) {
	suite.Run(t, new(CommonTestSuite))
}

func (s *CommonTestSuite) BeforeTest(suiteName, testName string) {
	s.mc = minimock.NewController(s.T())
	s.ctx = instestlogger.TestContext(s.T())
	s.leaver = NewLeaverMock(s.T())
	s.pulseAccessor = beat.NewAppenderMock(s.T())
	s.handler = &Handler{Leaver: s.leaver, PulseAccessor: s.pulseAccessor}
}

func (s *CommonTestSuite) AfterTest(suiteName, testName string) {
	s.mc.Wait(time.Second*10)
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
	testPulse := beat.Beat{}
	testPulse.PulseNumber = pulse.Number(2000000000)
	leaveAfter := testPulse.PulseNumber + pulse.Number(5)

	s.pulseAccessor.LatestTimeBeatMock.Return(testPulse, nil)
	s.leaver.LeaveMock.Expect(s.ctx, leaveAfter)
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
	defer mc.Wait(time.Second*10)

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
