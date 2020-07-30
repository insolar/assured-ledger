// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"strconv"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executor/common/foundation"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/synchronization"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestVirtual_Method_PulseChanged(t *testing.T) {
	insrail.LogCase(t, "C5211")

	table := []struct {
		name             string
		isolation        contract.MethodIsolation
		withSideEffect   bool
		countChangePulse int
	}{
		{
			name:             "ordered call when the pulse changed",
			isolation:        tolerableFlags(),
			withSideEffect:   false,
			countChangePulse: 1,
		},
		{
			name:             "unordered call when the pulse changed",
			isolation:        intolerableFlags(),
			withSideEffect:   false,
			countChangePulse: 1,
		},
		{
			name:             "Ordered call double pulse change during execution",
			isolation:        tolerableFlags(),
			withSideEffect:   true,
			countChangePulse: 2,
		},
		{
			name:             "Unordered call double pulse change during execution",
			isolation:        intolerableFlags(),
			withSideEffect:   true,
			countChangePulse: 2,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commonTestUtils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			logger := inslogger.FromContext(ctx)

			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})

			var object reference.Global
			{
				server.ReplaceRunner(runnerMock)
				server.Init(ctx)
				server.IncrementPulseAndWaitIdle(ctx)

				object = reference.NewSelf(server.RandomLocalWithPulse())
				prevPulse := server.GetPulse().PulseNumber

				server.IncrementPulse(ctx)

				Method_PrepareObject(ctx, server, payload.Ready, object, prevPulse)
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			var (
				outgoing = server.BuildRandomOutgoingWithPulse()
				p1       = server.GetPulse().PulseNumber

				expectedToken = payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					Callee:            object,
					Outgoing:          outgoing,
					DelegateTo:        server.JetCoordinatorMock.Me(),
				}
				firstTokenValue payload.CallDelegationToken
				isFirstToken    = true
			)

			pl := payload.VCallRequest{
				CallType:       payload.CTMethod,
				CallFlags:      payload.BuildCallFlags(test.isolation.Interference, test.isolation.State),
				Callee:         object,
				CallSiteMethod: "SomeMethod",
				CallOutgoing:   outgoing,
			}

			synchronizeExecution := synchronization.NewPoint(1)

			// add ExecutionMocks to runnerMock
			{
				runnerMock.AddExecutionClassify("SomeMethod", test.isolation, nil)

				requestResult := requestresult.New([]byte("call result"), gen.UniqueGlobalRef())
				if test.withSideEffect {
					newObjDescriptor := descriptor.NewObject(
						reference.Global{}, reference.Local{}, gen.UniqueGlobalRef(), []byte(""), false,
					)
					requestResult.SetAmend(newObjDescriptor, []byte("new memory"))
				}

				objectExecutionMock := runnerMock.AddExecutionMock("SomeMethod")
				objectExecutionMock.AddStart(
					func(_ execution.Context) {
						logger.Debug("ExecutionStart [SomeMethod]")
						synchronizeExecution.Synchronize()
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestResult,
					},
				)
			}

			// add checks to typedChecker
			{
				typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
					// check for pending counts must be in tests: call terminal method case C5104
					assert.Equal(t, object, report.Object)
					assert.Equal(t, payload.Ready, report.Status)
					assert.Zero(t, report.DelegationSpec)
					return false
				})

				typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
					p2 := server.GetPulse().PulseNumber

					assert.Equal(t, object, request.Callee)
					if isFirstToken {
						assert.Zero(t, request.DelegationSpec)
						isFirstToken = false
					} else {
						assert.NotEmpty(t, firstTokenValue)
						assert.Equal(t, firstTokenValue, request.DelegationSpec)
					}

					expectedToken.PulseNumber = p2
					approver := gen.UniqueGlobalRef()
					expectedToken.Approver = approver

					firstTokenValue = payload.CallDelegationToken{
						TokenTypeAndFlags: payload.DelegationTokenTypeCall,
						PulseNumber:       p2,
						Callee:            request.Callee,
						Outgoing:          request.CallOutgoing,
						DelegateTo:        server.JetCoordinatorMock.Me(),
						Approver:          approver,
					}
					msg := payload.VDelegatedCallResponse{
						Callee:                 request.Callee,
						ResponseDelegationSpec: firstTokenValue,
					}

					server.SendPayload(ctx, &msg)
					return false
				})

				typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
					assert.Equal(t, object, finished.Callee)
					assert.Equal(t, expectedToken, finished.DelegationSpec)
					if test.isolation == tolerableFlags() && test.withSideEffect {
						assert.NotEmpty(t, finished.LatestState)
						assert.Equal(t, []byte("new memory"), finished.LatestState.State)
					} else {
						assert.Empty(t, finished.LatestState)
					}
					return false
				})
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					assert.Equal(t, object, res.Callee)
					if test.isolation == intolerableFlags() && test.withSideEffect {
						contractErr, sysErr := foundation.UnmarshalMethodResult(res.ReturnArguments)
						require.NoError(t, sysErr)
						require.Equal(t, "intolerable call trying to change object state", contractErr.Error())
					} else {
						assert.Equal(t, []byte("call result"), res.ReturnArguments)
					}
					assert.Equal(t, p1, res.CallOutgoing.GetLocal().Pulse())
					assert.Equal(t, expectedToken, res.DelegationSpec)
					return false
				})
			}

			server.SendPayload(ctx, &pl)

			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, synchronizeExecution.Wait())
			for i := 0; i < test.countChangePulse; i++ {
				tokenRequestDone := server.Journal.Wait(
					predicate.ChainOf(
						predicate.NewSMTypeFilter(&execute.SMDelegatedTokenRequest{}, predicate.AfterAnyStopOrError),
						predicate.NewSMTypeFilter(&execute.SMExecute{}, predicate.BeforeStep((&execute.SMExecute{}).StepWaitExecutionResult)),
					),
				)

				server.IncrementPulseAndWaitIdle(ctx)

				commonTestUtils.WaitSignalsTimed(t, 10*time.Second, tokenRequestDone)
			}
			synchronizeExecution.WakeUp()

			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			{
				assert.Equal(t, 1, typedChecker.VCallResult.Count())
				assert.Equal(t, 1, typedChecker.VStateReport.Count())
				assert.Equal(t, 1, typedChecker.VDelegatedRequestFinished.Count())

				assert.Equal(t, test.countChangePulse, typedChecker.VDelegatedCallRequest.Count())
			}

			mc.Finish()

		})
	}
}

// 2 ordered and 2 unordered calls
func TestVirtual_Method_CheckPendingsCount(t *testing.T) {
	defer commonTestUtils.LeakTester(t)
	insrail.LogCase(t, "C5104")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServer(nil, t)
	defer server.Stop()

	logger := inslogger.FromContext(ctx)

	executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 4)

	runnerMock := logicless.NewServiceMock(ctx, t, func(execution execution.Context) string {
		return execution.Request.CallSiteMethod
	})
	{
		server.ReplaceRunner(runnerMock)
		server.Init(ctx)
		server.IncrementPulseAndWaitIdle(ctx)
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

	var (
		prevPulse = server.GetPulse().PulseNumber
		object    = gen.UniqueGlobalRefWithPulse(prevPulse)

		approver = gen.UniqueGlobalRef()

		content *payload.VStateReport_ProvidedContentBody

		request = payload.VCallRequest{
			CallType: payload.CTMethod,
			Caller:   server.GlobalCaller(),
			Callee:   object,
			// set flags, outgoing and method name later
		}
	)

	server.IncrementPulse(ctx)

	currPulse := server.GetPulse().PulseNumber

	// create object state
	{
		objectState := payload.ObjectState{
			Reference: gen.UniqueLocalRefWithPulse(prevPulse),
			Class:     testwalletProxy.GetClass(),
			State:     makeRawWalletState(initialBalance),
		}
		content = &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState:     &objectState,
			LatestValidatedState: &objectState,
		}

		vsrPayload := &payload.VStateReport{
			Status:                        payload.Ready,
			Object:                        object,
			AsOf:                          prevPulse,
			UnorderedPendingEarliestPulse: prevPulse,
			ProvidedContent:               content,
		}

		server.WaitIdleConveyor()
		server.SendPayload(ctx, vsrPayload)
		server.WaitActiveThenIdleConveyor()
	}

	synchronizeExecution := synchronization.NewPoint(3)

	// add checks to typedChecker
	{
		typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
			assert.Equal(t, payload.Ready, report.Status)
			assert.Equal(t, currPulse, report.AsOf)
			assert.Equal(t, object, report.Object)
			assert.Zero(t, report.DelegationSpec)

			assert.Equal(t, int32(2), report.UnorderedPendingCount)
			assert.Equal(t, currPulse, report.UnorderedPendingEarliestPulse)

			assert.Equal(t, int32(1), report.OrderedPendingCount)
			assert.Equal(t, currPulse, report.OrderedPendingEarliestPulse)

			assert.Zero(t, report.PreRegisteredQueueCount)
			assert.Empty(t, report.PreRegisteredEarliestPulse)

			assert.Empty(t, report.PriorityCallQueueCount)
			assert.Empty(t, report.LatestValidatedState)
			assert.Empty(t, report.LatestValidatedCode)
			assert.NotEmpty(t, report.LatestDirtyState)
			assert.Empty(t, report.LatestDirtyCode)
			assert.Equal(t, content, report.ProvidedContent)
			return false
		})
		typedChecker.VDelegatedCallRequest.Set(func(request *payload.VDelegatedCallRequest) bool {
			assert.Equal(t, object, request.Callee)
			assert.Zero(t, request.DelegationSpec)

			token := payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				PulseNumber:       currPulse,
				Callee:            request.Callee,
				Outgoing:          request.CallOutgoing,
				DelegateTo:        server.JetCoordinatorMock.Me(),
				Approver:          approver,
			}

			msg := payload.VDelegatedCallResponse{
				Callee:                 request.Callee,
				ResponseDelegationSpec: token,
			}

			server.SendPayload(ctx, &msg)
			return false
		})
		typedChecker.VDelegatedRequestFinished.Set(func(finished *payload.VDelegatedRequestFinished) bool {
			assert.Equal(t, object, finished.Callee)
			assert.NotEmpty(t, finished.DelegationSpec)
			return false
		})
		typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
			assert.Equal(t, object, res.Callee)
			assert.Equal(t, []byte("call result"), res.ReturnArguments)
			assert.Equal(t, currPulse, res.CallOutgoing.GetLocal().Pulse())
			assert.NotEmpty(t, res.DelegationSpec)
			return false
		})
	}

	for i := int64(0); i < 4; i++ {

		var (
			objectExecutionMock *logicless.ExecutionMock
			result              *requestresult.RequestResult

			req              = request
			outgoing         = server.BuildRandomOutgoingWithPulse()
			newObjDescriptor = descriptor.NewObject(
				reference.Global{}, reference.Local{}, gen.UniqueGlobalRef(), []byte(""), false,
			)
		)

		switch i {
		case 0, 1:
			req.CallSiteMethod = "ordered" + strconv.FormatInt(i, 10)
			req.CallFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)

			runnerMock.AddExecutionClassify(req.CallSiteMethod, tolerableFlags(), nil)
			result = requestresult.New([]byte("call result"), object)
			result.SetAmend(newObjDescriptor, []byte("new memory"))
			objectExecutionMock = runnerMock.AddExecutionMock(req.CallSiteMethod)

		case 2, 3:
			req.CallSiteMethod = "unordered" + strconv.FormatInt(i, 10)
			req.CallFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallValidated)

			runnerMock.AddExecutionClassify(req.CallSiteMethod, intolerableFlags(), nil)
			objectExecutionMock = runnerMock.AddExecutionMock(req.CallSiteMethod)
			result = requestresult.New([]byte("call result"), object)
		}

		req.CallOutgoing = outgoing
		objectExecutionMock.AddStart(
			func(_ execution.Context) {
				logger.Debug("ExecutionStart [" + req.CallSiteMethod + "]")
				synchronizeExecution.Synchronize()
			},
			&execution.Update{
				Type:   execution.Done,
				Result: result,
			},
		)
		server.SendPayload(ctx, &req)
	}

	commonTestUtils.WaitSignalsTimed(t, 20*time.Second, synchronizeExecution.Wait())
	server.IncrementPulseAndWaitIdle(ctx)
	synchronizeExecution.WakeUp()

	commonTestUtils.WaitSignalsTimed(t, 30*time.Second, executeDone)
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

	{
		require.Equal(t, 1, typedChecker.VStateReport.Count())
		require.Equal(t, 3, typedChecker.VDelegatedCallRequest.Count())
		require.Equal(t, 3, typedChecker.VDelegatedRequestFinished.Count())
		require.Equal(t, 3, typedChecker.VCallResult.Count())
	}

	mc.Finish()
}

func TestVirtual_MethodCall_IfConstructorIsPending(t *testing.T) {
	insrail.LogCase(t, "C5237")

	table := []struct {
		name      string
		isolation contract.MethodIsolation
	}{
		{
			name:      "ordered call when constructor execution is pending",
			isolation: tolerableFlags(),
		},
		{
			name: "unordered call when constructor execution is pending",
			isolation: contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallDirty, // use dirty state because R0 does not copy dirty to validated state
			},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commonTestUtils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			logger := inslogger.FromContext(ctx)
			executeDone := server.Journal.WaitStopOf(&execute.SMExecute{}, 1)

			runnerMock := logicless.NewServiceMock(ctx, mc, func(execution execution.Context) string {
				return execution.Request.CallSiteMethod
			})
			{
				server.ReplaceRunner(runnerMock)
				server.Init(ctx)
				server.IncrementPulseAndWaitIdle(ctx)
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)

			var (
				class         = gen.UniqueGlobalRef()
				object        = reference.NewSelf(server.RandomLocalWithPulse())
				outgoingP1    = server.BuildRandomOutgoingWithPulse()
				dirtyStateRef = server.RandomLocalWithPulse()
				p1            = server.GetPulse().PulseNumber
				getDelegated  = false
			)

			server.IncrementPulseAndWaitIdle(ctx)
			p2 := server.GetPulse().PulseNumber
			outgoingP2 := server.BuildRandomOutgoingWithPulse()

			// create object state
			{
				vsrPayload := &payload.VStateReport{
					Status:                      payload.Empty,
					Object:                      object,
					AsOf:                        p1,
					OrderedPendingCount:         1,
					OrderedPendingEarliestPulse: p1,
				}

				server.SendPayload(ctx, vsrPayload)
				server.WaitActiveThenIdleConveyor()
			}

			// add ExecutionMock to runnerMock
			{
				runnerMock.AddExecutionClassify("SomeMethod", test.isolation, nil)
				requestResult := requestresult.New([]byte("call result"), gen.UniqueGlobalRef())

				objectExecutionMock := runnerMock.AddExecutionMock("SomeMethod")
				objectExecutionMock.AddStart(
					func(ctx execution.Context) {
						logger.Debug("ExecutionStart [SomeMethod]")
						require.Equal(t, object, ctx.Request.Callee)
						require.Equal(t, []byte("new object memory"), ctx.ObjectDescriptor.Memory())
						require.Equal(t, dirtyStateRef, ctx.ObjectDescriptor.StateID())
						require.True(t, getDelegated)
					},
					&execution.Update{
						Type:   execution.Done,
						Result: requestResult,
					},
				)
			}

			// add checks to typedChecker
			{
				typedChecker.VCallResult.Set(func(res *payload.VCallResult) bool {
					assert.Equal(t, object, res.Callee)
					assert.Equal(t, []byte("call result"), res.ReturnArguments)
					assert.Equal(t, p2, res.CallOutgoing.GetLocal().Pulse())
					assert.Empty(t, res.DelegationSpec)
					return false
				})
				typedChecker.VDelegatedCallResponse.SetResend(false)
			}

			// VCallRequest
			{
				pl := payload.VCallRequest{
					CallType:            payload.CTMethod,
					CallFlags:           payload.BuildCallFlags(test.isolation.Interference, test.isolation.State),
					Caller:              server.GlobalCaller(),
					Callee:              object,
					CallSiteDeclaration: class,
					CallSiteMethod:      "SomeMethod",
					CallOutgoing:        outgoingP2,
				}
				server.SendPayload(ctx, &pl)
			}
			// VDelegatedCallRequest
			{
				delegatedRequest := payload.VDelegatedCallRequest{
					Callee:       object,
					CallOutgoing: outgoingP1,
					CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
				}
				server.SendPayload(ctx, &delegatedRequest)
			}
			// VDelegatedRequestFinished
			{
				finished := payload.VDelegatedRequestFinished{
					Callee:       object,
					CallOutgoing: outgoingP1,
					CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
					LatestState: &payload.ObjectState{
						Reference: dirtyStateRef,
						Class:     class,
						State:     []byte("new object memory"),
					},
				}
				server.SendPayload(ctx, &finished)
				getDelegated = true
			}

			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, executeDone)
			commonTestUtils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			require.Equal(t, 1, typedChecker.VCallResult.Count())

			mc.Finish()
		})
	}
}
