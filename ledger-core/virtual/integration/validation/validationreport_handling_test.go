// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package validation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/statemachine"
)

func TestVirtual_ObjectValidationReport_EmptyCache(t *testing.T) {
	defer commonTestUtils.LeakTester(t)

	mc := minimock.NewController(t)

	server, ctx := utils.NewServerWithErrorFilter(nil, t, func(s string) bool {
		return !(strings.Contains(s, "key not found") || strings.Contains(s, "not implemented"))
	})
	defer server.Stop()

	var (
		objectGlobal = server.RandomGlobalWithPulse()
		stateRef     = reference.NewRecordOf(objectGlobal, server.RandomLocalWithPulse())
		class        = server.RandomGlobalWithPulse()
	)

	// send VStateReport
	{
		report := server.StateReportBuilder().Object(objectGlobal).Ready().Class(class).Report()

		server.IncrementPulse(ctx)
		waitReport := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, &report)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitReport)
	}

	// send VObjectValidationReport
	{
		validationReport := &rms.VObjectValidationReport{
			Object:    rms.NewReference(objectGlobal),
			In:        server.GetPulse().PulseNumber,
			Validated: rms.NewReference(stateRef),
		}
		waitValidationReport := server.Journal.WaitStopOf(&handlers.SMVObjectValidationReport{}, 1)
		waitSubroutine := server.Journal.WaitStopOf(&statemachine.SMGetCachedMemory{}, 1)
		server.SendPayload(ctx, validationReport)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitValidationReport)
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitSubroutine)
	}

	mc.Finish()
}

type testSuite struct {
	server       *utils.Server
	runnerMock   *logicless.ServiceMock
	typedChecker *checker.Typed

	objectGlobal reference.Global
	class        reference.Global
	objDesc      descriptor.Object

	report *rms.VStateReport
}

func TestVirtual_ObjectValidationReport(t *testing.T) {
	defer commonTestUtils.LeakTester(t)

	testCases := []struct {
		name                              string
		validatedIsEqualDirty             bool
		vStateReportAfterValidationReport bool
	}{
		{name: "get memory from cache"},
		{name: "use memory from VStateReport", validatedIsEqualDirty: true},
		{name: "VStateReport after VObjectValidationReport", vStateReportAfterValidationReport: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			runnerMock := logicless.NewServiceMock(ctx, mc, nil)

			server.ReplaceRunner(runnerMock)
			server.Init(ctx)

			var (
				objectGlobal = server.RandomGlobalWithPulse()
				class        = server.RandomGlobalWithPulse()
			)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			objDescriptor := descriptor.NewObject(objectGlobal, server.RandomLocalWithPulse(), class, []byte("new state"), false)
			validatedStateRef := objDescriptor.State()

			// prepare cache
			{
				err := server.MemoryCache.Set(ctx, validatedStateRef, objDescriptor)
				require.Nil(t, err)
			}

			server.IncrementPulse(ctx)
			vCallRequestOutgoing := server.BuildRandomOutgoingWithPulse()

			suite := testSuite{
				server:       server,
				runnerMock:   runnerMock,
				typedChecker: typedChecker,
				objectGlobal: objectGlobal,
				objDesc:      objDescriptor,
				class:        class,
			}

			suite.generateStateReport(testCase.validatedIsEqualDirty)
			suite.addCheckers(t, ctx, vCallRequestOutgoing, testCase.vStateReportAfterValidationReport)

			if testCase.vStateReportAfterValidationReport {
				suite.sendVObjectValidationReport(t, ctx, validatedStateRef, testCase.validatedIsEqualDirty)
				suite.sendVStateReport(t, ctx)
			} else {
				suite.sendVStateReport(t, ctx)
				suite.sendVObjectValidationReport(t, ctx, validatedStateRef, testCase.validatedIsEqualDirty)
			}

			suite.sendVCachedMemoryRequest(t, ctx)
			suite.sendVCallRequest(t, ctx, vCallRequestOutgoing)

			mc.Finish()
		})
	}
}

func (s *testSuite) generateStateReport(latestDirtyStateIsEqualValidationReport bool) {
	prevPulse := s.server.GetPrevPulse().PulseNumber

	report := &rms.VStateReport{
		AsOf:   prevPulse,
		Status: rms.StateStatusReady,
		Object: rms.NewReference(s.objectGlobal),
		ProvidedContent: &rms.VStateReport_ProvidedContentBody{
			LatestDirtyState: &rms.ObjectState{
				Reference: rms.NewReferenceLocal(gen.UniqueLocalRefWithPulse(prevPulse)),
				Class:     rms.NewReference(s.class),
				Memory:    rms.NewBytes([]byte("dirty state")),
			},
			LatestValidatedState: &rms.ObjectState{
				Reference: rms.NewReferenceLocal(gen.UniqueLocalRefWithPulse(prevPulse)),
				Class:     rms.NewReference(s.class),
				Memory:    rms.NewBytes([]byte("dirty state")),
			},
		},
	}
	if latestDirtyStateIsEqualValidationReport {
		report.ProvidedContent.LatestDirtyState.Reference = rms.NewReference(s.objDesc.State())
		report.ProvidedContent.LatestDirtyState.Memory = rms.NewBytes([]byte("new state"))
	}

	s.report = report
}

func (s *testSuite) addCheckers(t *testing.T, ctx context.Context, vCallRequestOutgoing reference.Global, vStateRequestExpected bool) {
	// add typedChecker
	{
		s.typedChecker.VCachedMemoryResponse.Set(func(response *rms.VCachedMemoryResponse) bool {
			assert.Equal(t, s.objDesc.State(), response.State.Reference.GetValue())
			assert.Equal(t, []byte("new state"), response.State.Memory.GetBytes())
			assert.False(t, response.State.Deactivated)
			return false
		})
		s.typedChecker.VCallResult.Set(func(result *rms.VCallResult) bool {
			return false
		})
		if vStateRequestExpected {
			s.typedChecker.VStateRequest.Set(func(request *rms.VStateRequest) bool {
				assert.Equal(t, s.objectGlobal, request.Object.GetValue())
				s.server.SendPayload(ctx, s.report)
				return false
			})
		}
	}

	// add runnerMock
	{
		s.runnerMock.AddExecutionMock(vCallRequestOutgoing).AddStart(
			func(ctx execution.Context) {
				assert.Equal(t, s.objectGlobal, ctx.Object)
				assert.Equal(t, s.objDesc, ctx.ObjectDescriptor)
			},
			&execution.Update{
				Type:   execution.Done,
				Result: requestresult.New([]byte("result"), s.objectGlobal),
			},
		)
		s.runnerMock.AddExecutionClassify(vCallRequestOutgoing, contract.MethodIsolation{Interference: isolation.CallIntolerable, State: isolation.CallValidated}, nil)
	}
}

func (s *testSuite) sendVStateReport(t *testing.T, ctx context.Context) {
	waitReport := s.server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
	s.server.SendPayload(ctx, s.report)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitReport)
}

func (s *testSuite) sendVObjectValidationReport(t *testing.T, ctx context.Context, validatedStateRef reference.Global, getMemoryFromVStateReport bool) {
	validationReport := &rms.VObjectValidationReport{
		Object:    rms.NewReference(s.objectGlobal),
		In:        s.server.GetPulse().PulseNumber,
		Validated: rms.NewReference(validatedStateRef),
	}
	waitValidationReport := s.server.Journal.WaitStopOf(&handlers.SMVObjectValidationReport{}, 1)
	waitSubroutine := s.server.Journal.WaitStopOf(&statemachine.SMGetCachedMemory{}, 1)
	s.server.SendPayload(ctx, validationReport)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitValidationReport)
	if !getMemoryFromVStateReport {
		commonTestUtils.WaitSignalsTimed(t, 10*time.Second, waitSubroutine)
	}
}

func (s *testSuite) sendVCachedMemoryRequest(t *testing.T, ctx context.Context) {
	pl := &rms.VCachedMemoryRequest{
		Object:  rms.NewReference(s.objectGlobal),
		State: rms.NewReference(s.objDesc.State()),
	}
	executeDone := s.server.Journal.WaitStopOf(&handlers.SMVCachedMemoryRequest{}, 1)
	s.server.SendPayload(ctx, pl)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.typedChecker.VCachedMemoryResponse.Wait(ctx, 1))

	assert.Equal(t, 1, s.typedChecker.VCachedMemoryResponse.Count())
}

func (s *testSuite) sendVCallRequest(t *testing.T, ctx context.Context, vCallRequestOutgoing reference.Global) {
	pl := utils.GenerateVCallRequestMethodImmutable(s.server)
	pl.Callee = rms.NewReference(s.objectGlobal)
	pl.CallOutgoing.Set(vCallRequestOutgoing)

	executeDone := s.server.Journal.WaitStopOf(&execute.SMExecute{}, 1)
	s.server.SendPayload(ctx, pl)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, s.typedChecker.VCallResult.Wait(ctx, 1))

	assert.Equal(t, 1, s.typedChecker.VCallResult.Count())
}
