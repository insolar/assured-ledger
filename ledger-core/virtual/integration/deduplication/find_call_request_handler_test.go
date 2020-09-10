package deduplication

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

type TestStep func(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T)

// test handling of VFindCallRequest
// three pulses: P1, P2, P3
// always look at P2
// request originated in P1 or P2, influences missing or unknown
type VFindCallRequestHandlingTestInfo struct {
	name string

	events               []TestStep
	requestFromP1        bool
	requestIsConstructor bool

	expectedStatus     rms.VFindCallResponse_CallState
	expectedResult     bool
	expectedDelegation bool
}

func TestDeduplication_VFindCallRequestHandling(t *testing.T) {
	insrail.LogCase(t, "C5115")

	table := []VFindCallRequestHandlingTestInfo{
		{
			name:   "don't know request, missing",
			events: []TestStep{StepIncrementPulseToP3, StepFindMessage},

			expectedStatus: rms.CallStateMissing,
		},
		{
			name:   "don't know request, missing, early msg",
			events: []TestStep{StepFindMessage, StepIncrementPulseToP3},

			expectedStatus: rms.CallStateMissing,
		},
		{
			name:          "don't know request, unknown",
			events:        []TestStep{StepIncrementPulseToP3, StepFindMessage},
			requestFromP1: true,

			expectedStatus: rms.CallStateUnknown,
		},
		{
			name:          "don't know request, unknown, early msg",
			events:        []TestStep{StepFindMessage, StepIncrementPulseToP3},
			requestFromP1: true,

			expectedStatus: rms.CallStateUnknown,
		},

		{
			name:   "found request, method, not pending, result",
			events: []TestStep{StepMethodStartAndFinish, StepIncrementPulseToP3, StepFindMessage},

			expectedStatus: rms.CallStateFound,
			expectedResult: true,
		},
		{
			name:   "found request, method, not pending, result, early msg",
			events: []TestStep{StepFindMessage, StepMethodStartAndFinish, StepIncrementPulseToP3},

			expectedStatus: rms.CallStateFound,
			expectedResult: true,
		},
		{
			name:   "found request, method, pending, no result",
			events: []TestStep{StepMethodStart, StepIncrementPulseToP3, StepFindMessage, StepRequestFinish},

			expectedStatus:     rms.CallStateFound,
			expectedDelegation: true,
		},
		{
			name:   "found request, method, pending, no result, earlyMsg",
			events: []TestStep{StepFindMessage, StepMethodStart, StepIncrementPulseToP3, StepRequestFinish},

			expectedStatus:     rms.CallStateFound,
			expectedDelegation: true,
		},
		{
			name:   "found request, method, pending, result",
			events: []TestStep{StepMethodStart, StepIncrementPulseToP3, StepRequestFinish, StepFindMessage},

			expectedStatus:     rms.CallStateFound,
			expectedDelegation: true,
		},

		{
			name:                 "found request, constructor, not pending, result",
			events:               []TestStep{StepConstructorStartAndFinish, StepIncrementPulseToP3, StepFindMessage},
			requestIsConstructor: true,

			expectedStatus: rms.CallStateFound,
			expectedResult: true,
		},
		// TODO failed
		{
			name:                 "found request, constructor, not pending, result, early msg",
			events:               []TestStep{StepFindMessage, StepConstructorStartAndFinish, StepIncrementPulseToP3},
			requestIsConstructor: true,

			expectedStatus: rms.CallStateFound,
			expectedResult: true,
		},
		{
			name:                 "found request, constructor, pending, no result",
			events:               []TestStep{StepConstructorStart, StepIncrementPulseToP3, StepFindMessage, StepRequestFinish},
			requestIsConstructor: true,

			expectedStatus:     rms.CallStateFound,
			expectedDelegation: true,
		},
		{
			name:                 "found request, constructor, pending, no result, earlyMsg",
			events:               []TestStep{StepFindMessage, StepConstructorStart, StepIncrementPulseToP3, StepRequestFinish},
			requestIsConstructor: true,

			expectedStatus:     rms.CallStateFound,
			expectedDelegation: true,
		},
		{
			name:                 "found request, constructor, pending, result",
			events:               []TestStep{StepConstructorStart, StepIncrementPulseToP3, StepRequestFinish, StepFindMessage},
			requestIsConstructor: true,

			expectedStatus:     rms.CallStateFound,
			expectedDelegation: true,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			suite := &VFindCallRequestHandlingSuite{}

			ctx := suite.initServer(t)
			defer suite.stopServer()

			smVFindCallRequestEnded := suite.server.Journal.WaitStopOf(&handlers.SMVFindCallRequest{}, 1)

			suite.initPulsesP1andP2(ctx)
			suite.generateClass()
			suite.generateCaller()

			outgoingPulse := suite.getP2()
			if test.requestFromP1 {
				outgoingPulse = suite.getP1()
			}
			suite.generateOutgoing(outgoingPulse)

			suite.isConstructor = test.requestIsConstructor

			suite.generateObjectRef()

			suite.setMessageCheckers(ctx, t, test)
			suite.setRunnerMock()

			for _, event := range test.events {
				event(suite, ctx, t)
			}

			// wait for VDelegatedRequestFinished
			if test.expectedDelegation {
				commontestutils.WaitSignalsTimed(t, 10*time.Second, suite.typedChecker.VDelegatedRequestFinished.Wait(ctx, 1))
			}

			// wait for VFindCallResponse
			commontestutils.WaitSignalsTimed(t, 10*time.Second, suite.vFindCallResponseSent)

			commontestutils.WaitSignalsTimed(t, 10*time.Second, smVFindCallRequestEnded)

			suite.finish()
		})
	}
}

func StepIncrementPulseToP3(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	s.server.IncrementPulseAndWaitIdle(ctx)

	// wait for VStateReport after pulse change
	if s.vStateReportSent != nil {
		commontestutils.WaitSignalsTimed(t, 10*time.Second, s.vStateReportSent)
	}
}

func StepFindMessage(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	findMsg := rms.VFindCallRequest{
		LookAt:   s.getP2(),
		Callee:   rms.NewReference(s.getObject()),
		Outgoing: rms.NewReference(s.outgoing),
	}
	s.addPayloadAndWaitIdle(ctx, &findMsg)
}

func StepMethodStart(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	if s.executionPoint != nil {
		panic(throw.IllegalState())
	}
	s.executionPoint = synchronization.NewPoint(1)

	report := rms.VStateReport{
		AsOf:   s.getP1(),
		Status: rms.StateStatusReady,
		Object: rms.NewReference(s.getObject()),

		ProvidedContent: &rms.VStateReport_ProvidedContentBody{
			LatestDirtyState: &rms.ObjectState{
				Reference: rms.NewReferenceLocal(gen.UniqueLocalRefWithPulse(s.getP1())),
				Class:     rms.NewReference(s.getClass()),
				State:     rms.NewBytes([]byte("object memory")),
			},
			LatestValidatedState: &rms.ObjectState{
				Reference: rms.NewReferenceLocal(gen.UniqueLocalRefWithPulse(s.getP1())),
				Class:     rms.NewReference(s.getClass()),
				State:     rms.NewBytes([]byte("object memory")),
			},
		},
	}
	s.addPayloadAndWaitIdle(ctx, &report)

	req := utils.GenerateVCallRequestMethod(s.server)
	req.Caller.Set(s.getCaller())
	req.Callee.Set(s.getObject())
	req.CallOutgoing.Set(s.outgoing)

	s.addPayloadAndWaitIdle(ctx, req)
	s.vStateReportSent = make(chan struct{})

	commontestutils.WaitSignalsTimed(t, 10*time.Second, s.executionPoint.Wait())
}

func StepConstructorStart(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	if s.executionPoint != nil {
		panic(throw.IllegalState())
	}
	s.executionPoint = synchronization.NewPoint(1)

	if s.getObject().GetLocal().GetPulseNumber() < s.getP2() {
		report := rms.VStateReport{
			AsOf:   s.getP1(),
			Status: rms.StateStatusMissing,
			Object: rms.NewReference(s.getObject()),
		}
		s.addPayloadAndWaitIdle(ctx, &report)
	}

	req := utils.GenerateVCallRequestConstructor(s.server)
	req.Caller.Set(s.getCaller())
	req.Callee.Set(s.getClass())
	req.CallOutgoing.Set(s.outgoing)

	s.addPayloadAndWaitIdle(ctx, req)
	s.vStateReportSent = make(chan struct{})

	commontestutils.WaitSignalsTimed(t, 10*time.Second, s.executionPoint.Wait())
}

func StepRequestFinish(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	s.executionPoint.WakeUp()

	commontestutils.WaitSignalsTimed(t, 20*time.Second, s.executeIsFinished)
	commontestutils.WaitSignalsTimed(t, 10*time.Second, s.typedChecker.VCallResult.Wait(ctx, 1))
}

func StepMethodStartAndFinish(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	StepMethodStart(s, ctx, t)
	StepRequestFinish(s, ctx, t)
}

func StepConstructorStartAndFinish(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	StepConstructorStart(s, ctx, t)
	StepRequestFinish(s, ctx, t)
}

type VFindCallRequestHandlingSuite struct {
	mc           *minimock.Controller
	server       *utils.Server
	runnerMock   *logicless.ServiceMock
	typedChecker *checker.Typed

	p1       pulse.Number
	p2       pulse.Number
	class    reference.Global
	caller   reference.Global
	object   reference.Global
	outgoing reference.Global

	isConstructor bool

	executionPoint        *synchronization.Point
	executeIsFinished     synckit.SignalChannel
	vStateReportSent      chan struct{}
	vFindCallResponseSent chan struct{}
}

func (s *VFindCallRequestHandlingSuite) initServer(t *testing.T) context.Context {

	s.vFindCallResponseSent = make(chan struct{}, 0)

	s.mc = minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	s.server = server

	s.runnerMock = logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(s.runnerMock)

	server.Init(ctx)

	s.typedChecker = s.server.PublisherMock.SetTypedChecker(ctx, s.mc, server)

	s.executeIsFinished = server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

	return ctx
}

func (s *VFindCallRequestHandlingSuite) initPulsesP1andP2(ctx context.Context) {
	s.p1 = s.server.GetPulse().PulseNumber
	s.server.IncrementPulseAndWaitIdle(ctx)
	s.p2 = s.server.GetPulse().PulseNumber
}

func (s *VFindCallRequestHandlingSuite) getP1() pulse.Number {
	return s.p1
}

func (s *VFindCallRequestHandlingSuite) getP2() pulse.Number {
	return s.p2
}

func (s *VFindCallRequestHandlingSuite) switchToP3(ctx context.Context) {
	s.server.IncrementPulseAndWaitIdle(ctx)
}

func (s *VFindCallRequestHandlingSuite) generateCaller() {
	s.caller = s.server.GlobalCaller()
}

func (s *VFindCallRequestHandlingSuite) generateObjectRef() {
	if s.isConstructor {
		s.object = reference.NewSelf(s.outgoing.GetLocal())
		return
	}
	p := s.getP1()
	s.object = gen.UniqueGlobalRefWithPulse(p)
}

func (s *VFindCallRequestHandlingSuite) generateOutgoing(p pulse.Number) {
	s.outgoing = s.server.BuildRandomOutgoingWithGivenPulse(p)
}

func (s *VFindCallRequestHandlingSuite) generateClass() {
	s.class = gen.UniqueGlobalRefWithPulse(s.getP1())
}

func (s *VFindCallRequestHandlingSuite) getObject() reference.Global {
	return s.object
}

func (s *VFindCallRequestHandlingSuite) getCaller() reference.Global {
	return s.caller
}

func (s *VFindCallRequestHandlingSuite) getClass() reference.Global {
	return s.class
}

func (s *VFindCallRequestHandlingSuite) setMessageCheckers(
	ctx context.Context,
	t *testing.T,
	testInfo VFindCallRequestHandlingTestInfo,
) {

	s.typedChecker.VFindCallResponse.Set(func(res *rms.VFindCallResponse) bool {
		defer func() {
			close(s.vFindCallResponseSent)
		}()
		assert.Equal(t, s.getP2(), res.LookedAt)
		assert.Equal(t, s.getObject(), res.Callee)
		assert.Equal(t, s.outgoing, res.Outgoing)
		assert.Equal(t, testInfo.expectedStatus, res.Status)

		if testInfo.expectedResult {
			require.NotNil(t, res.CallResult)
			require.Equal(t, s.outgoing, res.CallResult.CallOutgoing)
		}

		return false
	})

	s.typedChecker.VStateReport.Set(func(report *rms.VStateReport) bool {
		assert.Equal(t, s.getP2(), report.AsOf)
		assert.Equal(t, s.getObject(), report.Object)
		if s.vStateReportSent != nil {
			close(s.vStateReportSent)
		}
		return false
	})

	s.typedChecker.VCallResult.SetResend(false)

	s.typedChecker.VDelegatedCallRequest.Set(func(req *rms.VDelegatedCallRequest) bool {
		delegationToken := s.server.DelegationToken(req.CallOutgoing.GetValue(), s.getCaller(), req.Callee.GetValue())

		s.server.SendPayload(ctx, &rms.VDelegatedCallResponse{
			Callee:                 req.Callee,
			CallIncoming:           req.CallIncoming,
			ResponseDelegationSpec: delegationToken,
		})

		return false
	})
	s.typedChecker.VDelegatedRequestFinished.SetResend(false)

	s.typedChecker.VFindCallRequest.Set(func(req *rms.VFindCallRequest) bool {
		assert.Equal(t, s.getP2(), req.LookAt)
		assert.Equal(t, s.getObject(), req.Callee)
		assert.Equal(t, s.outgoing, req.Outgoing)

		pl := rms.VFindCallResponse{
			LookedAt: s.getP2(),
			Callee:   rms.NewReference(s.getObject()),
			Outgoing: rms.NewReference(s.outgoing),
			Status:   rms.CallStateMissing,
		}
		s.server.SendPayload(ctx, &pl)
		return false
	})
}

func (s *VFindCallRequestHandlingSuite) setRunnerMock() {
	isolation := contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty}
	s.runnerMock.AddExecutionClassify(s.outgoing, isolation, nil)

	newObjDescriptor := descriptor.NewObject(
		reference.Global{}, reference.Local{}, s.getClass(), []byte(""), false,
	)

	{
		methodResult := requestresult.New([]byte("execution"), s.server.RandomGlobalWithPulse())
		methodResult.SetAmend(newObjDescriptor, []byte("new memory"))

		executionMock := s.runnerMock.AddExecutionMock(s.outgoing)
		executionMock.AddStart(func(ctx execution.Context) {
			s.executionPoint.Synchronize()
		}, &execution.Update{
			Type:   execution.Done,
			Result: methodResult,
		})
	}

	{
		constructorResult := requestresult.New([]byte("exection"), s.getObject())
		constructorResult.SetActivate(reference.Global{}, s.getClass(), []byte("new memory"))

		executionMock := s.runnerMock.AddExecutionMock("New")
		executionMock.AddStart(func(ctx execution.Context) {
			s.executionPoint.Synchronize()
		}, &execution.Update{
			Type:   execution.Done,
			Result: constructorResult,
		})
	}
}

func (s *VFindCallRequestHandlingSuite) addPayloadAndWaitIdle(ctx context.Context, pl rms.GoGoSerializable) {
	s.server.SuspendConveyorAndWaitThenResetActive()
	s.server.SendPayload(ctx, pl)
	s.server.WaitActiveThenIdleConveyor()
}

func (s *VFindCallRequestHandlingSuite) finish() {
	s.mc.Finish()
}

func (s *VFindCallRequestHandlingSuite) stopServer() {
	if s.executionPoint != nil {
		s.executionPoint.Done()
	}
	s.server.Stop()
}
