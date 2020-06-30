package deduplication

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

// test handling of VFindCallRequest
// three pulses: P1, P2, P3
// always look at P2
// request originated in P1 or P2, influences missing or unknown
type VFindCallRequestHandlingTestInfo struct {
	name string

	events               []string
	requestFromP1        bool
	requestIsConstructor bool

	expectedStatus payload.VFindCallResponse_CallState
	expectedResult bool
}

func TestDeduplication_VFindCallRequestHandling(t *testing.T) {
	t.Log("C5115")
	t.Skip("https://insolar.atlassian.net/browse/PLAT-389")

	table := []VFindCallRequestHandlingTestInfo{
		{
			name:   "don't know request, missing",
			events: []string{"P2->P3", "findMsg"},

			expectedStatus: payload.MissingCall,
		},
		{
			name:   "don't know request, missing, early msg",
			events: []string{"findMsg", "P2->P3"},

			expectedStatus: payload.MissingCall,
		},
		{
			name:          "don't know request, unknown",
			events:        []string{"P2->P3", "findMsg"},
			requestFromP1: true,

			expectedStatus: payload.UnknownCall,
		},
		{
			name:          "don't know request, unknown, early msg",
			events:        []string{"findMsg", "P2->P3"},
			requestFromP1: true,

			expectedStatus: payload.UnknownCall,
		},


		{
			name:   "found request, method, not pending, result",
			events: []string{"reqMethodStart", "reqFinish", "P2->P3", "findMsg"},

			expectedStatus: payload.FoundCall,
			expectedResult: true,
		},
		{
			name:   "found request, method, not pending, result, early msg",
			events: []string{"findMsg", "reqMethodStart", "reqFinish", "P2->P3"},

			expectedStatus: payload.FoundCall,
			expectedResult: true,
		},
		{
			name:   "found request, method, pending, no result",
			events: []string{"reqMethodStart", "P2->P3", "findMsg"},

			expectedStatus: payload.FoundCall,
		},
		{
			name:   "found request, method, pending, no result, earlyMsg",
			events: []string{"findMsg", "reqMethodStart", "P2->P3"},

			expectedStatus: payload.FoundCall,
		},
		{
			name:   "found request, method, pending, result",
			events: []string{"reqMethodStart", "P2->P3", "reqFinish", "findMsg"},

			expectedStatus: payload.FoundCall,
		},

		{
			name:   "found request, constructor, not pending, result",
			events: []string{"reqConstructorStart", "reqFinish", "P2->P3", "findMsg"},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
			expectedResult: true,
		},
		{
			name:   "found request, constructor, not pending, result, early msg",
			events: []string{"findMsg", "reqConstructorStart", "reqFinish", "P2->P3"},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
			expectedResult: true,
		},
		{
			name:   "found request, constructor, pending, no result",
			events: []string{"reqConstructorStart", "P2->P3", "findMsg"},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
		},
		{
			name:   "found request, constructor, pending, no result, earlyMsg",
			events: []string{"findMsg", "reqConstructorStart", "P2->P3"},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
		},
		{
			name:   "found request, constructor, pending, result",
			events: []string{"reqConstructorStart", "P2->P3", "reqFinish", "findMsg"},
			requestIsConstructor: true,

			expectedStatus: payload.FoundCall,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			suite := &VFindCallRequestHandlingSuite{}

			ctx := suite.initServer(t)
			defer suite.stopServer()

			suite.initPulsesP1andP2(ctx)
			suite.generateClass()
			suite.generateCaller()

			outgoingPulse := suite.getP2()
			if test.requestFromP1 {
				outgoingPulse = suite.getP1()
			}
			suite.generateOutgoing(outgoingPulse)

			if test.requestIsConstructor {
				suite.generateObjectRefFromOutgoing()
			} else {
				suite.generateObjectRef()
			}

			suite.setMessageCheckers(ctx, t, test)
			suite.setRunnerMock()

			for _, event := range test.events {
				switch event {
				case "findMsg":
					findMsg := payload.VFindCallRequest{
						LookAt:   suite.getP1(),
						Callee:   suite.getObject(),
						Outgoing: suite.getOutgoingRef(),
					}
					suite.addPayloadAndWaitIdle(ctx, &findMsg)
				case "reqMethodStart":
					suite.startMethodRequestAndWait(t, ctx)
				case "reqConstructorStart":
					suite.startConstructorRequestAndWait(t, ctx)
				case "reqFinish":
					suite.finishRequestAndWait(t)
				case "P2->P3":
					suite.switchToP3(ctx)
				default:
					panic(throw.IllegalValue())
				}
			}

			suite.waitFindRequestResponse(t)

			suite.finish()
		})
	}
}

type VFindCallRequestHandlingSuite struct {
	mc           *minimock.Controller
	server       *utils.Server
	runnerMock   *logicless.ServiceMock
	typedChecker *mock.TypePublishChecker

	p1       pulse.Number
	p2       pulse.Number
	class    reference.Global
	caller   reference.Global
	object   reference.Global
	outgoing reference.Local

	startedSignal          chan struct{}
	releaseSignal          chan struct{}
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
	p := s.getP1()
	local := gen.UniqueLocalRefWithPulse(p)
	s.object = reference.NewSelf(local)
}

func (s *VFindCallRequestHandlingSuite) generateObjectRefFromOutgoing() {
	s.object = reference.NewSelf(s.outgoing)
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
	callee := s.getObject() // TODO: FIXME: PLAT-558: should be caller

	return reference.NewRecordOf(callee, s.outgoing)
}

func (s *VFindCallRequestHandlingSuite) getOutgoingLocal() reference.Local {
	return s.outgoing
}

func (s *VFindCallRequestHandlingSuite) startMethodRequest(ctx context.Context) {
	s.startedSignal = make(chan struct{}, 0)
	s.releaseSignal = make(chan struct{}, 0)

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
		CallOutgoing:   s.getOutgoingLocal(),
	}
	s.addPayloadAndWaitIdle(ctx, &req)
}

func (s *VFindCallRequestHandlingSuite) startMethodRequestAndWait(t *testing.T, ctx context.Context) {
	t.Helper()
	s.startMethodRequest(ctx)
	testutils.WaitSignalsTimed(t, 10*time.Second, s.startedSignal)
}

func (s *VFindCallRequestHandlingSuite) startConstructorRequest(ctx context.Context) {
	s.startedSignal = make(chan struct{}, 0)
	s.releaseSignal = make(chan struct{}, 0)

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
		CallOutgoing:   s.getOutgoingLocal(),
	}
	s.addPayloadAndWaitIdle(ctx, &req)
}

func (s *VFindCallRequestHandlingSuite) startConstructorRequestAndWait(t *testing.T, ctx context.Context) {
	t.Helper()
	s.startConstructorRequest(ctx)
	testutils.WaitSignalsTimed(t, 10*time.Second, s.startedSignal)
}

func (s *VFindCallRequestHandlingSuite) finishRequest() {
	close(s.releaseSignal)
}

func (s *VFindCallRequestHandlingSuite) finishRequestAndWait(t *testing.T) {
	t.Helper()

	s.finishRequest()
	testutils.WaitSignalsTimed(t, 10*time.Second, s.executeIsFinished)
}

func (s *VFindCallRequestHandlingSuite) setMessageCheckers(
	ctx context.Context,
	t *testing.T,
	testInfo VFindCallRequestHandlingTestInfo,
) {

	s.typedChecker.VFindCallResponse.Set(func(res *payload.VFindCallResponse) bool {
		require.Equal(t, s.getObject(), res.Callee)
		require.Equal(t, s.getOutgoingRef(), res.Outgoing)
		require.Equal(t, testInfo.expectedStatus, res.Status)

		if testInfo.expectedResult {
			require.NotNil(t, res.CallResult)
			require.Equal(t, res.CallResult.CallOutgoing, s.getOutgoingLocal())
		}

		close(s.haveFindResponseSignal)

		return false
	})

	s.typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		require.Equal(t, s.getP2(), report.AsOf)
		require.Equal(t, s.getObject(), report.Object)
		return false
	})

	s.typedChecker.VCallResult.SetResend(false)

	s.typedChecker.VDelegatedCallRequest.Set(func(req *payload.VDelegatedCallRequest) bool {
		pl := payload.VDelegatedCallResponse{
			ResponseDelegationSpec: payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				PulseNumber:       s.server.GetPulse().PulseNumber,
				Callee:            s.getObject(),
				Outgoing:          s.getOutgoingRef(),
				ApproverSignature: []byte("deadbeef"),
			},
		}
		s.server.SendPayload(ctx, &pl)
		return false
	})
	s.typedChecker.VDelegatedRequestFinished.SetResend(false)

	s.typedChecker.VFindCallRequest.Set(func(req *payload.VFindCallRequest) bool {
		require.Equal(t, s.getP1(), req.LookAt)
		require.Equal(t, s.getObject(), req.Callee)
		require.Equal(t, s.getOutgoingRef(), req.Outgoing)

		pl := payload.VFindCallResponse{
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

	requestResult := requestresult.New([]byte("execution"), gen.UniqueGlobalRef())
	requestResult.SetAmend(newObjDescriptor, []byte("new memory"))

	executionMock := s.runnerMock.AddExecutionMock("SomeMethod")
	executionMock.AddStart(func(ctx execution.Context) {
		close(s.startedSignal)

		<-s.releaseSignal
	}, &execution.Update{
		Type:   execution.Done,
		Result: requestResult,
	})
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
	s.server.Stop()
}
