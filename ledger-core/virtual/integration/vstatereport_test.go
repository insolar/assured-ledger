// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

// 1. Send CallRequest
// 2. Change pulse in mocked executor
// 4. Since we changed pulse during execution, we expect that VStateReport will be sent
// 5. Check that in VStateReport new object state is stored
func TestVirtual_SendVStateReport_IfPulseChanged(t *testing.T) {
	t.Log("C4934")
	t.Skip("https://insolar.atlassian.net/browse/PLAT-314")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	testBalance := uint32(555)
	additionalBalance := uint(133)
	objectRef := gen.UniqueGlobalRef()
	stateID := gen.UniqueLocalRefWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet

		rawWalletState := makeRawWalletState(testBalance)
		pl := makeVStateReportEvent(objectRef, stateID, rawWalletState)
		server.SendPayload(ctx, pl)
	}

	// generate new state since it will be changed by CallAPIAddAmount
	newRawWalletState := makeRawWalletState(testBalance + uint32(additionalBalance))

	callMethod := func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error) {
		// we want to change pulse during execution
		server.IncrementPulseAndWaitIdle(ctx)

		emptyResult := makeEmptyResult()
		return newRawWalletState, emptyResult, nil
	}

	mockExecutor(t, server, callMethod, nil)

	var (
		countVStateReport int
	)
	gotVStateReport := make(chan *payload.VStateReport, 0)
	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		require.Len(t, messages, 1)

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		if err != nil {
			return nil
		}

		switch payLoadData := pl.(type) {
		case *payload.VStateReport:
			countVStateReport++
			gotVStateReport <- payLoadData
		case *payload.VCallResult:
		default:
			t.Logf("Going message: %T", payLoadData)
		}

		server.SendMessage(ctx, messages[0])
		return nil
	})

	code, _ := server.CallAPIAddAmount(ctx, objectRef, additionalBalance)
	require.Equal(t, 200, code)

	select {
	case _ = <-gotVStateReport:
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}

	require.Equal(t, 1, countVStateReport)
}

type stateReportCheckPendingCountersAndPulsesTestChecks struct {
	UnorderedPendingCount         int32
	UnorderedPendingEarliestPulse int // pulse 1-5, 0 for not set
	OrderedPendingCount           int32
	OrderedPendingEarliestPulse   int // pulse 1-5, 0 for not set
}

type stateReportCheckPendingCountersAndPulsesTestRequestInfo struct {
	name  string
	ref   reference.Global
	flags payload.CallFlags
	token payload.CallDelegationToken
}

type stateReportCheckPendingCountersAndPulsesTest struct {
	mu sync.RWMutex

	mc         *minimock.Controller
	server     *utils.Server
	runnerMock *logicless.ServiceMock

	currentPulseIndex int
	pulses            [6]pulse.Number

	caller   reference.Global
	object   reference.Global
	requests map[string]*stateReportCheckPendingCountersAndPulsesTestRequestInfo

	newPendingsReleaser chan struct{}
}

func TestVirtual_StateReport_CheckPendingCountersAndPulses(t *testing.T) {
	t.Log("C5111")

	// setup
	// three pendings: 1 ordered (RO1), 2 unordered (RU1, RU2)
	// five pulses: p1, p2, p3, p4, p5
	// RO1 - P2, RU1 - P2, RU2 - P3
	// incoming StateReport comes AsOf P3, switch P3 -> P4
	// outgoing StateReport is checked AsOf P4 when switch P4 -> P5
	// checking counters and earliest pulses in the outgoing state report

	table := []struct {
		name    string
		confirm []string
		finish  []string
		start   []contract.InterferenceFlag

		checks stateReportCheckPendingCountersAndPulsesTestChecks
	}{
		{
			name: "no confirmations, all pending nulled at the end",
			checks: stateReportCheckPendingCountersAndPulsesTestChecks{
				OrderedPendingCount:           0,
				OrderedPendingEarliestPulse:   0,
				UnorderedPendingCount:         0,
				UnorderedPendingEarliestPulse: 0,
			},
		},
		{
			name:    "partial confirmation, some pendings nulled at the end",
			confirm: []string{"Ordered1", "Unordered2"},
			checks: stateReportCheckPendingCountersAndPulsesTestChecks{
				OrderedPendingCount:           1,
				OrderedPendingEarliestPulse:   2,
				UnorderedPendingCount:         1,
				UnorderedPendingEarliestPulse: 3,
			},
		},
		{
			name:    "all confirmed, no ends",
			confirm: []string{"Ordered1", "Unordered1", "Unordered2"},
			checks: stateReportCheckPendingCountersAndPulsesTestChecks{
				OrderedPendingCount:           1,
				OrderedPendingEarliestPulse:   2,
				UnorderedPendingCount:         2,
				UnorderedPendingEarliestPulse: 2,
			},
		},
		{
			name:    "all confirmed, some end",
			confirm: []string{"Ordered1", "Unordered1", "Unordered2"},
			finish:  []string{"Ordered1", "Unordered1"},

			checks: stateReportCheckPendingCountersAndPulsesTestChecks{
				OrderedPendingCount:           0,
				OrderedPendingEarliestPulse:   0,
				UnorderedPendingCount:         1,
				UnorderedPendingEarliestPulse: 3,
			},
		},
		{
			name:    "all confirmed, all end",
			confirm: []string{"Ordered1", "Unordered1", "Unordered2"},
			finish:  []string{"Ordered1", "Unordered1", "Unordered2"},

			checks: stateReportCheckPendingCountersAndPulsesTestChecks{
				OrderedPendingCount:           0,
				OrderedPendingEarliestPulse:   0,
				UnorderedPendingCount:         0,
				UnorderedPendingEarliestPulse: 0,
			},
		},
		{
			name:    "all confirmed, some end, start new pendings",
			confirm: []string{"Ordered1", "Unordered1", "Unordered2"},
			finish:  []string{"Ordered1", "Unordered2"},

			start: []contract.InterferenceFlag{
				contract.CallIntolerable,
				contract.CallTolerable,
			},

			checks: stateReportCheckPendingCountersAndPulsesTestChecks{
				OrderedPendingCount:           1,
				OrderedPendingEarliestPulse:   1,
				UnorderedPendingCount:         2,
				UnorderedPendingEarliestPulse: 1,
			},
		},
	}

	class := gen.UniqueGlobalRef()

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {

			suite := &stateReportCheckPendingCountersAndPulsesTest{}
			ctx := suite.initServer(t)
			defer suite.finish()

			suite.createEmptyPulse(ctx)
			suite.createPulsesFromP1toP4(ctx)

			suite.generateCaller(ctx)
			suite.generateObjectRef(ctx)
			suite.generateRequests(ctx)

			suite.setMessageCheckers(ctx, t, test.checks)

			report := payload.VStateReport{
				AsOf:   suite.getPulse(3),
				Status: payload.Ready,
				Object: suite.getObject(),

				UnorderedPendingCount:         2,
				UnorderedPendingEarliestPulse: suite.getPulse(1),

				OrderedPendingCount:         1,
				OrderedPendingEarliestPulse: suite.getPulse(1),

				ProvidedContent: &payload.VStateReport_ProvidedContentBody{
					LatestDirtyState: &payload.ObjectState{
						Reference: gen.UniqueLocalRefWithPulse(suite.getPulse(1)),
						Class:     class,
						State:     []byte("object memory"),
					},
				},
			}
			suite.addPayloadAndWaitIdle(ctx, &report)

			expectedPublished := 0

			for _, reqName := range test.confirm {
				suite.confirmPending(ctx, reqName)
				expectedPublished++ // token response
			}
			suite.waitMessagePublications(ctx, t, expectedPublished)

			for _, reqName := range test.finish {
				suite.finishActivePending(ctx, reqName)
			}

			for _, tolerance := range test.start {
				suite.startNewPending(ctx, t, tolerance)
			}

			suite.createPulseP5(ctx)
			expectedPublished++                  // expect StateReport
			expectedPublished += len(test.start) // expect request to get token for each pending
			suite.waitMessagePublications(ctx, t, expectedPublished)

			suite.releaseNewlyCreatedPendings()
			expectedPublished += len(test.start) * 2 // pending finished + result
			suite.waitMessagePublications(ctx, t, expectedPublished)

			// request state again
			reportRequest := payload.VStateRequest{
				AsOf:   suite.getPulse(4),
				Object: suite.getObject(),
			}
			suite.addPayloadAndWaitIdle(ctx, &reportRequest)

			expectedPublished++
			suite.waitMessagePublications(ctx, t, expectedPublished)
		})
	}
}

func (s *stateReportCheckPendingCountersAndPulsesTest) initServer(t *testing.T) context.Context {

	s.mc = minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	s.server = server

	s.runnerMock = logicless.NewServiceMock(ctx, t, nil)
	server.ReplaceRunner(s.runnerMock)

	server.Init(ctx)

	return ctx
}

func (s *stateReportCheckPendingCountersAndPulsesTest) newPulse(ctx context.Context, pulseIndex int) {
	s.server.IncrementPulseAndWaitIdle(ctx)
	s.pulses[pulseIndex] = s.server.GetPulse().PulseNumber
}

func (s *stateReportCheckPendingCountersAndPulsesTest) createEmptyPulse(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pulses[0] = pulse.Number(0)
}

func (s *stateReportCheckPendingCountersAndPulsesTest) createPulsesFromP1toP4(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.newPulse(ctx, 1)
	s.newPulse(ctx, 2)
	s.newPulse(ctx, 3)
	s.newPulse(ctx, 4)
	s.currentPulseIndex = 4
}

func (s *stateReportCheckPendingCountersAndPulsesTest) createPulseP5(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.newPulse(ctx, 5)
	s.currentPulseIndex = 5
}

func (s *stateReportCheckPendingCountersAndPulsesTest) generateCaller(ctx context.Context) {
	p := s.getPulse(1)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.caller = reference.NewSelf(gen.UniqueLocalRefWithPulse(p))
}

func (s *stateReportCheckPendingCountersAndPulsesTest) generateObjectRef(ctx context.Context) {
	p := s.getPulse(1)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.object = reference.NewSelf(gen.UniqueLocalRefWithPulse(p))
}

func (s *stateReportCheckPendingCountersAndPulsesTest) generateRequests(ctx context.Context) {
	s.requests = map[string]*stateReportCheckPendingCountersAndPulsesTestRequestInfo{
		"Ordered1": {
			ref:   reference.NewRecordOf(s.getCaller(), gen.UniqueLocalRefWithPulse(s.getPulse(2))),
			flags: payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		},
		"Unordered1": {
			ref:   reference.NewRecordOf(s.getCaller(), gen.UniqueLocalRefWithPulse(s.getPulse(2))),
			flags: payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
		},
		"Unordered2": {
			ref:   reference.NewRecordOf(s.getCaller(), gen.UniqueLocalRefWithPulse(s.getPulse(3))),
			flags: payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
		},
	}
}

func (s *stateReportCheckPendingCountersAndPulsesTest) confirmPending(
	ctx context.Context, reqName string,
) {
	reqInfo := s.requests[reqName]
	pl := payload.VDelegatedCallRequest{
		Callee:       s.getObject(),
		CallOutgoing: reqInfo.ref,
		CallFlags:    reqInfo.flags,
	}
	s.addPayloadAndWaitIdle(ctx, &pl)
}

func (s *stateReportCheckPendingCountersAndPulsesTest) finishActivePending(
	ctx context.Context, reqName string,
) {
	reqInfo := s.requests[reqName]
	pl := payload.VDelegatedRequestFinished{
		Callee:       s.getObject(),
		CallOutgoing: reqInfo.ref,
		CallFlags:    reqInfo.flags,
	}
	s.addPayloadAndWaitIdle(ctx, &pl)
}

func (s *stateReportCheckPendingCountersAndPulsesTest) startNewPending(
	ctx context.Context,
	t *testing.T,
	intFlag contract.InterferenceFlag,
) {
	pulseNumber := s.getPulse(1)
	outgoingLocal := gen.UniqueLocalRefWithPulse(pulseNumber)
	outgoing := reference.NewRecordOf(s.getObject(), outgoingLocal)
	key := outgoing.String()

	if s.newPendingsReleaser == nil {
		s.newPendingsReleaser = make(chan struct{}, 0)
	}
	releaser := s.newPendingsReleaser
	inExecutor := make(chan struct{}, 0)

	blockOnReleaser := func(_ execution.Context) {
		close(inExecutor)
		<-releaser
	}

	s.runnerMock.AddExecutionMock(key).
		AddStart(
			blockOnReleaser,
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("result"), s.getObject()),
			},
		)
	s.runnerMock.AddExecutionClassify(
		key,
		contract.MethodIsolation{
			Interference: intFlag,
			State:        contract.CallDirty,
		},
		nil,
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTMethod,
		CallFlags:      payload.BuildCallFlags(intFlag, contract.CallDirty),
		Caller:         s.getCaller(),
		Callee:         s.getObject(),
		CallSiteMethod: "Some",
		CallSequence:   1,
		CallOutgoing:   outgoingLocal,
	}
	s.addPayloadAndWaitIdle(ctx, &pl)

	testutils.WaitSignalsTimed(t, 10*time.Second, inExecutor)
}

func (s *stateReportCheckPendingCountersAndPulsesTest) releaseNewlyCreatedPendings() {
	if s.newPendingsReleaser != nil {
		close(s.newPendingsReleaser)
		s.newPendingsReleaser = nil
		s.server.WaitActiveThenIdleConveyor()
	}
}

func (s *stateReportCheckPendingCountersAndPulsesTest) setMessageCheckers(
	ctx context.Context,
	t *testing.T,
	checks stateReportCheckPendingCountersAndPulsesTestChecks,
) {

	typedChecker := s.server.PublisherMock.SetTypedChecker(ctx, s.mc, s.server)
	typedChecker.VStateReport.Set(func(rep *payload.VStateReport) bool {
		require.Equal(t, s.getPulse(4), rep.AsOf)
		require.Equal(t, s.getObject(), rep.Object)

		require.Equal(t, checks.UnorderedPendingCount, rep.UnorderedPendingCount)
		require.Equal(
			t,
			s.getPulse(checks.UnorderedPendingEarliestPulse),
			rep.UnorderedPendingEarliestPulse,
		)

		require.Equal(t, checks.OrderedPendingCount, rep.OrderedPendingCount)
		require.Equal(
			t,
			s.getPulse(checks.OrderedPendingEarliestPulse),
			rep.OrderedPendingEarliestPulse,
		)

		return false // no resend msg
	})
	typedChecker.VDelegatedCallResponse.Set(func(del *payload.VDelegatedCallResponse) bool {
		outgoingRef := del.ResponseDelegationSpec.Outgoing
		require.False(t, outgoingRef.IsZero())
		require.False(t, outgoingRef.IsEmpty())

		found := false
		for _, reqInfo := range s.requests {
			if outgoingRef.Equal(reqInfo.ref) {
				found = true
				reqInfo.token = del.ResponseDelegationSpec
				break
			}
		}
		require.True(t, found)
		return false
	})
	typedChecker.VDelegatedCallRequest.Set(func(req *payload.VDelegatedCallRequest) bool {
		outgoingRef := req.CallOutgoing
		require.False(t, outgoingRef.IsZero())
		require.False(t, outgoingRef.IsEmpty())

		require.Equal(t, s.getObject(), req.Callee)

		pl := payload.VDelegatedCallResponse{
			ResponseDelegationSpec: payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				PulseNumber:       s.getPulse(5),
				Callee:            s.getObject(),
				Outgoing:          outgoingRef,
				ApproverSignature: []byte("deadbeef"),
			},
		}
		s.server.SendPayload(ctx, &pl)
		return false
	})
	typedChecker.VDelegatedRequestFinished.Set(func(res *payload.VDelegatedRequestFinished) bool {
		outgoingRef := res.CallOutgoing
		require.False(t, outgoingRef.IsZero())
		require.False(t, outgoingRef.IsEmpty())

		require.Equal(t, s.getObject(), res.Callee)

		return false
	})
	typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
		outgoingRef := res.CallOutgoing
		require.False(t, outgoingRef.IsZero())
		require.False(t, outgoingRef.IsEmpty())

		require.Equal(t, s.getObject(), res.Callee)

		return false
	})
}

func (s *stateReportCheckPendingCountersAndPulsesTest) getPulse(
	pulseIndex int,
) pulse.Number {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.pulses[pulseIndex]
}

func (s *stateReportCheckPendingCountersAndPulsesTest) getCurrentPulse() pulse.Number {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.pulses[s.currentPulseIndex]
}

func (s *stateReportCheckPendingCountersAndPulsesTest) getObject() reference.Global {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.object
}

func (s *stateReportCheckPendingCountersAndPulsesTest) getCaller() reference.Global {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.caller
}

func (s *stateReportCheckPendingCountersAndPulsesTest) waitMessagePublications(
	ctx context.Context,
	t *testing.T,
	expected int,
) {
	if !s.server.PublisherMock.WaitCount(expected, 10*time.Second) {
		panic("timeout waiting for messages on publisher")
	}
	require.Equal(t, expected, s.server.PublisherMock.GetCount())
}

func (s *stateReportCheckPendingCountersAndPulsesTest) addPayloadAndWaitIdle(
	ctx context.Context, pl payload.Marshaler,
) {
	s.server.SuspendConveyorAndWaitThenResetActive()
	s.server.SendPayload(ctx, pl)
	s.server.WaitActiveThenIdleConveyor()
}

func (s *stateReportCheckPendingCountersAndPulsesTest) finish() {
	s.server.Stop()
	s.mc.Finish()
}
