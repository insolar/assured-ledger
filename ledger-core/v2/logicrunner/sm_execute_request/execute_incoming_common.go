// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_execute_request

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/reply"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/artifacts"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_artifact"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_contract_runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/s_sender"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/sm_object"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type SharedRequestState struct {
	Nonce       uint64
	RequestInfo *common.ParsedRequestInfo
}

type ExecuteIncomingCommon struct {
	SharedRequestState

	objectCatalog  *sm_object.LocalObjectCatalog
	pulseSlot      *conveyor.PulseSlot
	ArtifactClient *s_artifact.ArtifactClientServiceAdapter
	Sender         *s_sender.SenderServiceAdapter
	ContractRunner *s_contract_runner.ContractRunnerServiceAdapter

	objectInfo sm_object.ObjectInfo

	sharedStateLink sm_object.SharedObjectStateAccessor

	externalError error

	// input
	MessageMeta *payload.Meta

	contractTranscript  *common.Transcript
	executionResult     artifacts.RequestResult
	newObjectDescriptor artifacts.ObjectDescriptor
}

func (s *ExecuteIncomingCommon) InjectDependencies(sm smachine.StateMachine, slotLink smachine.SlotLink, injector *injector.DependencyInjector) {
	injector.MustInject(&s.ArtifactClient)
	injector.MustInject(&s.Sender)
	injector.MustInject(&s.ContractRunner)

	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.objectCatalog)
}

func (s *ExecuteIncomingCommon) useSharedObjectInfo(ctx smachine.ExecutionContext, cb func(state *sm_object.SharedObjectState)) smachine.StateUpdate {
	goCtx := ctx.GetContext()

	if s.sharedStateLink.IsZero() {
		if s.RequestInfo.Request.IsCreationRequest() {
			ctx.Log().Warn("creation request")
			s.sharedStateLink = s.objectCatalog.Create(ctx, s.RequestInfo.RequestObjectReference)
		} else {
			ctx.Log().Warn("another request")
			s.sharedStateLink = s.objectCatalog.GetOrCreate(ctx, s.RequestInfo.RequestObjectReference)
		}
	}

	switch s.sharedStateLink.Prepare(cb).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		inslogger.FromContext(goCtx).Error("NotPassed")
		return ctx.WaitShared(s.sharedStateLink.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		inslogger.FromContext(goCtx).Error("Impossible")
		// the holder of the sharedState is stopped
		return ctx.Stop()
	case smachine.Passed:
		inslogger.FromContext(goCtx).Error("Passed")
	default:
		panic("unknown state from TryUse")
	}
	return smachine.StateUpdate{}
}

func (s *ExecuteIncomingCommon) internalStepRegisterResult(ctx smachine.ExecutionContext, fetchNew bool) smachine.ConditionalBuilder {
	var (
		goCtx       = ctx.GetContext()
		asyncLogger = ctx.LogAsync()

		objectReference  = s.RequestInfo.RequestObjectReference
		requestReference = s.RequestInfo.RequestReference
		executionResult  = s.executionResult
	)

	return s.ArtifactClient.PrepareAsync(ctx, func(svc s_artifact.ArtifactClientService) smachine.AsyncResultFunc {
		defer common.LogAsyncTime(asyncLogger, time.Now(), "RegisterIncomingResult")

		var objectDescriptor artifacts.ObjectDescriptor

		err := svc.RegisterResult(goCtx, requestReference, executionResult)
		if err == nil && fetchNew {
			objectDescriptor, err = svc.GetObject(goCtx, objectReference, nil)
			inslogger.FromContext(goCtx).Debugf("NewObject fetched %s", objectReference.String())
		}

		return func(ctx smachine.AsyncResultContext) {
			s.externalError = err
			if objectDescriptor != nil {
				s.newObjectDescriptor = objectDescriptor
			}
		}
	}).DelayedStart().Sleep()
}

// it'll panic or execute
func (s *ExecuteIncomingCommon) internalSendResult(ctx smachine.ExecutionContext) {
	var (
		goCtx = ctx.GetContext()

		executionBytes []byte
		executionError string
	)

	switch {
	case s.externalError != nil: // execution error
		executionError = s.externalError.Error()
		ctx.Log().Trace("return: external error")
	case s.RequestInfo.Result != nil: // result of deduplication
		if s.executionResult != nil {
			panic("we got deduplicated result and execution result, unreachable")
		}

		executionBytes = s.RequestInfo.GetResultBytes()
		ctx.Log().Trace("return: duplicated results")
	case s.executionResult != nil: // result of execution
		executionBytes = reply.ToBytes(&reply.CallMethod{
			Object: &s.objectInfo.ObjectReference,
			Result: s.executionResult.Result(),
		})
		ctx.Log().Trace("return: execution results")
	default:
		// we have no result and no error (??)
		panic("unreachable")
	}

	pl := &payload.ReturnResults{
		RequestRef: s.RequestInfo.RequestReference,
		Reply:      executionBytes,
		Error:      executionError,
	}

	var (
		incoming   = s.RequestInfo.Request.(*record.IncomingRequest)
		APIRequest = s.RequestInfo.Request.IsAPIRequest()

		target insolar.Reference
	)
	if !APIRequest {
		target = incoming.Caller

		pl.Target = incoming.Caller
		pl.Reason = incoming.Reason
	} else {
		target = incoming.APINode
	}

	msg, err := payload.NewResultMessage(pl)
	if err != nil {
		panic("couldn't serialize message: " + err.Error())
	}

	s.Sender.PrepareNotify(ctx, func(svc s_sender.SenderService) {
		// TODO[bigbes]: there should be retry sender
		// retrySender := bus.NewWaitOKWithRetrySender(svc, svc, 1)

		var done func()
		if APIRequest {
			_, done = svc.SendTarget(goCtx, msg, target)
		} else {
			_, done = svc.SendRole(goCtx, msg, insolar.DynamicRoleVirtualExecutor, target)
		}
		done()
	}).Send()
}

func (s *ExecuteIncomingCommon) stepStop(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.externalError != nil {
		return ctx.Jump(s.stepError)
	}
	return ctx.Stop()
}

func (s *ExecuteIncomingCommon) stepError(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.internalSendResult(ctx)
	return ctx.Error(s.externalError)
}
