package deduplication

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
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

	expectedStatus payload.VFindCallResponse_CallState
	expectedResult bool
}

func TestDeduplication_VFindCallRequestHandling(t *testing.T) {
	t.Log("C5115")

	table := []VFindCallRequestHandlingTestInfo{
		{
			name:   "don't know request, missing",
			events: []TestStep{StepIncrementPulseToP3, StepFindMessage},

			expectedStatus: payload.MissingCall,
		},
		{
			name:   "don't know request, missing, early msg",
			events: []TestStep{StepFindMessage, StepIncrementPulseToP3},

			expectedStatus: payload.MissingCall,
		},
		{
			name:          "don't know request, unknown",
			events:        []TestStep{StepIncrementPulseToP3, StepFindMessage},
			requestFromP1: true,

			expectedStatus: payload.UnknownCall,
		},
		{
			name:          "don't know request, unknown, early msg",
			events:        []TestStep{StepFindMessage, StepIncrementPulseToP3},
			requestFromP1: true,

			expectedStatus: payload.UnknownCall,
		},

		{
			name:   "found request, method, not pending, result",
			events: []TestStep{StepMethodStartAndFinish, StepIncrementPulseToP3, StepFindMessage},

			expectedStatus: payload.FoundCall,
			expectedResult: true,
		},
		{
			name:   "found request, method, not pending, result, early msg",
			events: []TestStep{StepFindMessage, StepMethodStartAndFinish, StepIncrementPulseToP3},

			expectedStatus: payload.FoundCall,
			expectedResult: true,
		},
		{
			name:   "found request, method, pending, no result",
			events: []TestStep{StepMethodStart, StepIncrementPulseToP3, StepFindMessage, StepRequestFinish},

			expectedStatus: payload.FoundCall,
		},
		{
			name:   "found request, method, pending, no result, earlyMsg",
			events: []TestStep{StepFindMessage, StepMethodStart, StepIncrementPulseToP3, StepRequestFinish},

			expectedStatus: payload.FoundCall,
		},
		{
			name:   "found request, method, pending, result",
			events: []TestStep{StepMethodStart, StepIncrementPulseToP3, StepRequestFinish, StepFindMessage},

			expectedStatus: payload.FoundCall,
		},

		{
			name:                 "found request, constructor, not pending, result",
			events:               []TestStep{StepConstructorStartAndFinish, StepIncrementPulseToP3, StepFindMessage},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
			expectedResult: true,
		},
		{
			name:                 "found request, constructor, not pending, result, early msg",
			events:               []TestStep{StepFindMessage, StepConstructorStartAndFinish, StepIncrementPulseToP3},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
			expectedResult: true,
		},
		{
			name:                 "found request, constructor, pending, no result",
			events:               []TestStep{StepConstructorStart, StepIncrementPulseToP3, StepFindMessage, StepRequestFinish},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
		},
		{
			name:                 "found request, constructor, pending, no result, earlyMsg",
			events:               []TestStep{StepFindMessage, StepConstructorStart, StepIncrementPulseToP3, StepRequestFinish},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
		},
		{
			name:                 "found request, constructor, pending, result",
			events:               []TestStep{StepConstructorStart, StepIncrementPulseToP3, StepRequestFinish, StepFindMessage},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			suite := &VFindCallRequestHandlingSuite{}

			ctx := suite.initServer(t)
			defer suite.stopServer()

			handlerEnded := suite.server.Journal.WaitStopOf(&handlers.SMVFindCallRequest{}, 1)

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

			suite.waitFindRequestResponse(t)

			if suite.finalizedMessageSent != nil {
				testutils.WaitSignalsTimed(t, 10*time.Second, suite.finalizedMessageSent)
			}

			testutils.WaitSignalsTimed(t, 10*time.Second, suite.server.Journal.WaitAllAsyncCallsDone())
			testutils.WaitSignalsTimed(t, 10*time.Second, handlerEnded)

			suite.finish()
		})
	}
}

func StepIncrementPulseToP3(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	s.server.IncrementPulseAndWaitIdle(ctx)
}

func StepFindMessage(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	findMsg := payload.VFindCallRequest{
		LookAt:   s.getP2(),
		Callee:   s.getObject(),
		Outgoing: s.getOutgoingRef(),
	}
	s.addPayloadAndWaitIdle(ctx, &findMsg)
}

func StepMethodStart(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	if s.executionPoint != nil {
		panic(throw.IllegalState())
	}
	s.executionPoint = synchronization.NewPoint(1)

	report := payload.VStateReport{
		AsOf:   s.getP1(),
		Status: payload.Ready,
		Object: s.getObject(),

		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: gen.UniqueLocalRefWithPulse(s.getP1()),
				Class:     s.getClass(),
				State:     []byte("object memory"),
			},
		},
	}
	s.addPayloadAndWaitIdle(ctx, &report)

	req := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Caller:         s.getCaller(),
		Callee:         s.getObject(),
		CallSiteMethod: "SomeMethod",
		CallSequence:   1,
		CallOutgoing:   s.getOutgoingRef(),
	}
	s.addPayloadAndWaitIdle(ctx, &req)
	s.finalizedMessageSent = make(chan struct{})

	testutils.WaitSignalsTimed(t, 10*time.Second, s.executionPoint.Wait())
}

func StepConstructorStart(s *VFindCallRequestHandlingSuite, ctx context.Context, t *testing.T) {
	if s.executionPoint != nil {
		panic(throw.IllegalState())
	}
	s.executionPoint = synchronization.NewPoint(1)

	report := payload.VStateReport{
		AsOf:   s.getP1(),
		Status: payload.Missing,
		Object: s.getObject(),
	}
	s.addPayloadAndWaitIdle(ctx, &report)

	req := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		Caller:         s.getCaller(),
		Callee:         s.getClass(),
		CallSiteMethod: "New",
		CallSequence:   1,
		CallOutgoing:   s.getOutgoingRef(),
	}
	s.addPayloadAndWaitIdle(ctx, &req)
	s.finalizedMessageSent = make(chan struct{})

	testutils.WaitSignalsTimed(t, 10*time.Second, s.executionPoint.Wait())
}

func StepRequestFinish(s *VFindCallRequestHandlingSuite, _ context.Context, t *testing.T) {
	s.executionPoint.WakeUp()

	testutils.WaitSignalsTimed(t, 10*time.Second, s.executeIsFinished)
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
	outgoing reference.Local

	isConstructor        bool
	finalizedMessageSent chan struct{}

	executionPoint         *synchronization.Point
	executeIsFinished      synckit.SignalChannel
	haveFindResponseSignal chan struct{}
}

func (s *VFindCallRequestHandlingSuite) initServer(t *testing.T) context.Context {

	s.haveFindResponseSignal = make(chan struct{}, 0)

	s.mc = minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	s.server = server

	s.runnerMock = logicless.NewServiceMock(ctx, t, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
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
	p := s.getP1()
	local := gen.UniqueLocalRefWithPulse(p)
	s.caller = reference.NewSelf(local)
}

func (s *VFindCallRequestHandlingSuite) generateObjectRef() {
	if s.isConstructor {
		s.object = reference.NewSelf(s.outgoing)
		return
	}
	p := s.getP1()
	local := gen.UniqueLocalRefWithPulse(p)
	s.object = reference.NewSelf(local)
}

func (s *VFindCallRequestHandlingSuite) generateOutgoing(p pulse.Number) {
	s.outgoing = gen.UniqueLocalRefWithPulse(p)
}

func (s *VFindCallRequestHandlingSuite) generateClass() {
	s.class = gen.UniqueGlobalRef()
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

func (s *VFindCallRequestHandlingSuite) getOutgoingRef() reference.Global {
	return reference.NewRecordOf(s.caller, s.outgoing)
}

func (s *VFindCallRequestHandlingSuite) setMessageCheckers(
	ctx context.Context,
	t *testing.T,
	testInfo VFindCallRequestHandlingTestInfo,
) {

	s.typedChecker.VFindCallResponse.Set(func(res *payload.VFindCallResponse) bool {
		defer func() {
			close(s.haveFindResponseSignal)
		}()
		assert.Equal(t, s.getP2(), res.LookedAt)
		assert.Equal(t, s.getObject(), res.Callee)
		assert.Equal(t, s.getOutgoingRef(), res.Outgoing)
		assert.Equal(t, testInfo.expectedStatus, res.Status)

		if testInfo.expectedResult {
			require.NotNil(t, res.CallResult)
			require.Equal(t, s.getOutgoingRef(), res.CallResult.CallOutgoing)
		}

		return false
	})

	s.typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, s.getP2(), report.AsOf)
		assert.Equal(t, s.getObject(), report.Object)
		if s.finalizedMessageSent != nil {
			close(s.finalizedMessageSent)
		}
		return false
	})

	s.typedChecker.VCallResult.SetResend(false)

	s.typedChecker.VDelegatedCallRequest.Set(func(req *payload.VDelegatedCallRequest) bool {
		delegationToken := s.server.DelegationToken(req.CallOutgoing, s.getCaller(), req.Callee)

		s.server.SendPayload(ctx, &payload.VDelegatedCallResponse{
			Callee:                 req.Callee,
			CallIncoming:           req.CallIncoming,
			ResponseDelegationSpec: delegationToken,
		})

		return false
	})
	s.typedChecker.VDelegatedRequestFinished.SetResend(false)

	s.typedChecker.VFindCallRequest.Set(func(req *payload.VFindCallRequest) bool {
		assert.Equal(t, s.getP2(), req.LookAt)
		assert.Equal(t, s.getObject(), req.Callee)
		assert.Equal(t, s.getOutgoingRef(), req.Outgoing)

		pl := payload.VFindCallResponse{
			LookedAt: s.getP2(),
			Callee:   s.getObject(),
			Outgoing: s.getOutgoingRef(),
			Status:   payload.MissingCall,
		}
		s.server.SendPayload(ctx, &pl)
		return false
	})
}

func (s *VFindCallRequestHandlingSuite) setRunnerMock() {
	isolation := contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty}
	s.runnerMock.AddExecutionClassify("SomeMethod", isolation, nil)

	newObjDescriptor := descriptor.NewObject(
		reference.Global{}, reference.Local{}, s.getClass(), []byte(""), reference.Global{},
	)

	{
		methodResult := requestresult.New([]byte("execution"), gen.UniqueGlobalRef())
		methodResult.SetAmend(newObjDescriptor, []byte("new memory"))

		executionMock := s.runnerMock.AddExecutionMock("SomeMethod")
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

func (s *VFindCallRequestHandlingSuite) waitFindRequestResponse(
	t *testing.T,
) {
	t.Helper()
	testutils.WaitSignalsTimed(t, 10*time.Second, s.haveFindResponseSignal)
}

func (s *VFindCallRequestHandlingSuite) addPayloadAndWaitIdle(
	ctx context.Context, pl payload.Marshaler,
) {
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
