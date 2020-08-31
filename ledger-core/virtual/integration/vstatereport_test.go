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

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

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

	class    reference.Global
	object   reference.Global
	requests map[string]*stateReportCheckPendingCountersAndPulsesTestRequestInfo

	newPendingsReleaser chan struct{}
}

func TestVirtual_StateReport_CheckPendingCountersAndPulses(t *testing.T) {
	insrail.LogCase(t, "C5111")

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

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			suite := &stateReportCheckPendingCountersAndPulsesTest{}
			ctx := suite.initServer(t)
			defer suite.finish()

			suite.createEmptyPulse(ctx)
			suite.createPulsesFromP1toP4(ctx)

			suite.generateClass(ctx)
			suite.generateObjectRef(ctx)
			suite.generateRequests(ctx)

			suite.setMessageCheckers(ctx, t, test.checks)

			report := payload.VStateReport{
				AsOf:   suite.getPulse(3),
				Status: payload.StateStatusReady,
				Object: suite.getObject(),

				UnorderedPendingCount:         2,
				UnorderedPendingEarliestPulse: suite.getPulse(1),

				OrderedPendingCount:         1,
				OrderedPendingEarliestPulse: suite.getPulse(1),

				ProvidedContent: &payload.VStateReport_ProvidedContentBody{
					LatestDirtyState: &payload.ObjectState{
						Reference: gen.UniqueLocalRefWithPulse(suite.getPulse(1)),
						Class:     suite.getClass(),
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
			expectedPublished++                      // expect StateReport
			expectedPublished += 2 * len(test.start) // expect GetToken + FindRequest
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

func (s *stateReportCheckPendingCountersAndPulsesTest) generateClass(ctx context.Context) {
	p := s.getPulse(1)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.class = gen.UniqueGlobalRefWithPulse(p)
}

func (s *stateReportCheckPendingCountersAndPulsesTest) generateObjectRef(ctx context.Context) {
	p := s.getPulse(1)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.object = gen.UniqueGlobalRefWithPulse(p)
}

func (s *stateReportCheckPendingCountersAndPulsesTest) getClass() reference.Global {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.class
}

func (s *stateReportCheckPendingCountersAndPulsesTest) getIncomingFromOutgoing(outgoing reference.Global) reference.Global {
	return reference.NewRecordOf(s.getObject(), outgoing.GetLocal())
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
		CallIncoming: reference.NewRecordOf(s.getObject(), reqInfo.ref.GetLocal()),
		CallFlags:    reqInfo.flags,
	}
	s.addPayloadAndWaitIdle(ctx, &pl)
}

func (s *stateReportCheckPendingCountersAndPulsesTest) finishActivePending(
	ctx context.Context, reqName string,
) {
	reqInfo := s.requests[reqName]
	pl := payload.VDelegatedRequestFinished{
		CallType:     payload.CallTypeMethod,
		CallFlags:    reqInfo.flags,
		Callee:       s.getObject(),
		CallOutgoing: reqInfo.ref,
		CallIncoming: s.getObject(),
	}
	s.addPayloadAndWaitIdle(ctx, &pl)
}

func (s *stateReportCheckPendingCountersAndPulsesTest) startNewPending(
	ctx context.Context,
	t *testing.T,
	intFlag contract.InterferenceFlag,
) {
	pulseNumber := s.getPulse(1)
	outgoing := reference.NewRecordOf(s.getObject(), gen.UniqueLocalRefWithPulse(pulseNumber))
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

	pl := utils.GenerateVCallRequestMethod(s.server)
	pl.CallFlags = payload.BuildCallFlags(intFlag, contract.CallDirty)
	pl.Caller = s.getCaller()
	pl.Callee = s.getObject()
	pl.CallOutgoing = outgoing

	s.addPayloadAndWaitIdle(ctx, pl)

	commontestutils.WaitSignalsTimed(t, 10*time.Second, inExecutor)
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
	typedChecker.VObjectTranscriptReport.Set(func(report *rms.VObjectTranscriptReport) bool {
		t.FailNow()
		// TODO add asserts and check counter after https://insolar.atlassian.net/browse/PLAT-753
		return false
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

		token := payload.CallDelegationToken{
			TokenTypeAndFlags: payload.DelegationTokenTypeCall,
			PulseNumber:       s.getPulse(5),
			Callee:            s.getObject(),
			Outgoing:          outgoingRef,
			ApproverSignature: []byte("deadbeef"),
		}

		pl := payload.VDelegatedCallResponse{
			Callee:                 req.Callee,
			CallIncoming:           req.CallIncoming,
			ResponseDelegationSpec: token,
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
	typedChecker.VFindCallRequest.Set(func(req *payload.VFindCallRequest) bool {
		require.Equal(t, s.getPulse(3), req.LookAt)
		require.Equal(t, s.getObject(), req.Callee)

		pl := payload.VFindCallResponse{
			LookedAt: s.getPulse(3),
			Callee:   s.getObject(),
			Outgoing: req.Outgoing,
			Status:   payload.CallStateMissing,
		}
		s.server.SendPayload(ctx, &pl)

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

	return s.server.GlobalCaller()
}

func (s *stateReportCheckPendingCountersAndPulsesTest) waitMessagePublications(
	ctx context.Context,
	t *testing.T,
	expected int,
) {
	t.Helper()
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
