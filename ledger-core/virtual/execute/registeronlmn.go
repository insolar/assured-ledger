// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package execute

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

func mustRecordToAnyRecordLazy(rec rms.BasicRecord) rms.AnyRecordLazy {
	rv := rms.AnyRecordLazy{}
	if err := rv.SetAsLazy(rec); err != nil {
		panic(err)
	}
	return rv
}

func isConstructor(request *rms.VCallRequest) bool {
	return request.CallType == rms.CallTypeConstructor
}

type LMNResultAwaitKey struct {
	Object         reference.Global
	AnticipatedRef rms.Reference
	RequiredFlag   rms.RegistrationFlags
}

type Message struct {
	payload            rmsreg.GoGoSerializable
	registrarSignature rms.Binary
	resultKey          LMNResultAwaitKey
}

func (m *Message) ResultReceived() bool {
	return m.registrarSignature.IsEmpty()
}

func (m *Message) SetResult(signature rms.Binary) {
	m.registrarSignature = signature
}

func (m *Message) Payload() rmsreg.GoGoSerializable {
	return m.payload
}

func (m *Message) CheckKey(key LMNResultAwaitKey) bool {
	return m.resultKey == key
}

type MessagesHolder struct {
	messages  []*Message
	unsentIdx int
}

func NewMessagesHolder() *MessagesHolder {
	return &MessagesHolder{
		messages:  make([]*Message, 0),
		unsentIdx: -1,
	}
}

func (s *MessagesHolder) AppendRecord(record rmsreg.GoGoSerializable, resultKey LMNResultAwaitKey) {
	s.messages = append(s.messages, &Message{
		payload:   record,
		resultKey: resultKey,
	})
}

func (s *MessagesHolder) NextUnsentMessage() *Message {
	s.unsentIdx++
	if s.unsentIdx >= len(s.messages) {
		return nil
	}
	return s.messages[s.unsentIdx]
}

func (s *MessagesHolder) CurrentUnsentMessage() *Message {
	if s.unsentIdx < 0 && s.unsentIdx >= len(s.messages) {
		panic(throw.IllegalState())
	}
	return s.messages[s.unsentIdx]
}

type SMRegisterOnLMN struct {
	// input arguments
	Incoming          *rms.VCallRequest
	Outgoing          *rms.VCallRequest
	OutgoingResult    *rms.VCallResult
	IncomingResult    *execution.Update
	Interference      isolation.InterferenceFlag
	ObjectSharedState object.SharedStateAccessor

	Object          reference.Global
	LastFilamentRef reference.Global
	LastLifelineRef reference.Global

	// internal data
	messages        *MessagesHolder
	bargeInCallback smachine.BargeInHolder
	requiredSafe    uint8

	// output arguments
	NewObjectRef       reference.Global
	NewLastFilamentRef reference.Global
	NewLastLifelineRef reference.Global

	// DI
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
}

var dSMRegisterOnLMNInstance smachine.StateMachineDeclaration = &dSMRegisterOnLMN{}

type dSMRegisterOnLMN struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMRegisterOnLMN) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMRegisterOnLMN)

	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.pulseSlot)
}

func (*dSMRegisterOnLMN) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return nil
}

func (s *SMRegisterOnLMN) GetSubroutineInitState(ctx smachine.SubroutineStartContext) smachine.InitFunc {
	ctx.SetSubroutineCleanupMode(smachine.SubroutineCleanupAliasesAndShares)

	return s.Init
}

func (s *SMRegisterOnLMN) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMRegisterOnLMNInstance
}

func (s *SMRegisterOnLMN) getRecordAnticipatedRef(object reference.Global, _ rms.BasicRecord) reference.Global {
	var (
		currentPulseNumber = s.pulseSlot.CurrentPulseNumber()
		uniqueLocal        = gen.UniqueLocalRefWithPulse(currentPulseNumber)
	)

	if object.IsEmpty() {
		return reference.NewSelf(uniqueLocal)
	} else if !object.IsSelfScope() {
		panic(throw.IllegalState())
	}

	return reference.NewRecordOf(object, uniqueLocal)
}

func (s *SMRegisterOnLMN) createBargeInCallback(ctx smachine.InitializationContext) {
	s.bargeInCallback = ctx.NewBargeInWithParam(func(param interface{}) smachine.BargeInCallbackFunc {
		res, ok := param.(*rms.LRegisterResponse)
		if !ok || res == nil {
			panic(throw.IllegalValue())
		}

		return func(ctx smachine.BargeInContext) smachine.StateUpdate {
			var key = LMNResultAwaitKey{
				Object:         s.Object,
				AnticipatedRef: res.AnticipatedRef,
				RequiredFlag:   res.Flags,
			}

			unsentMsg := s.messages.CurrentUnsentMessage()
			if !unsentMsg.CheckKey(key) {
				panic(throw.E("Message order is broken"))
			}
			unsentMsg.SetResult(res.RegistrarSignature)
			return ctx.WakeUp()
		}
	})
}

func (s *SMRegisterOnLMN) registerMessage(ctx smachine.ExecutionContext, msg *rms.LRegisterRequest) error {
	var (
		bargeInKey = LMNResultAwaitKey{
			Object:         s.Object,
			AnticipatedRef: msg.AnticipatedRef,
			RequiredFlag:   rms.RegistrationFlags_Fast,
		}
	)
	switch msg.Flags {
	case rms.RegistrationFlags_Safe:
		bargeInKey.RequiredFlag = rms.RegistrationFlags_Safe
		if !ctx.PublishGlobalAliasAndBargeIn(bargeInKey, s.bargeInCallback) {
			return throw.E("failed to publish bargeIn callback")
		}
		s.messages.AppendRecord(msg, bargeInKey)
	case rms.RegistrationFlags_FastSafe:
		s.requiredSafe++
		var safeKey = bargeInKey
		safeKey.RequiredFlag = rms.RegistrationFlags_Safe
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &SMLMNSafeHandler{
				ObjectSharedState: s.ObjectSharedState,
				ExpectedKey:       safeKey,
			}
		})
		fallthrough
	case rms.RegistrationFlags_Fast:
		if !ctx.PublishGlobalAliasAndBargeIn(bargeInKey, s.bargeInCallback) {
			return throw.E("failed to publish bargeIn callback")
		}
		s.messages.AppendRecord(msg, bargeInKey)
	default:
		panic(throw.IllegalValue())
	}

	return nil
}

func (s *SMRegisterOnLMN) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	s.messages = NewMessagesHolder()
	s.createBargeInCallback(ctx)
	// possible variants here:
	// all
	// CTConstructor -> Register (RLifelineStart and RInboundRequest) in one message (??)
	// Register (RInboundRequest + ROutboundRequest) in two separate messages
	// Register ROutboundRequest
	// Register ROutboundResponse
	// Register *InboundResult* (all possible variants, different step, one message)
	// Register (RInboundRequest + *InboundResult*) in two separate messages
	switch {
	case s.Incoming != nil && isConstructor(s.Incoming) && s.Object.IsEmpty(): // we need to register lifeline
		return ctx.Jump(s.stepRegisterLifeline)
	case s.Incoming != nil: // we need to register incoming
		return ctx.Jump(s.stepRegisterIncoming)
	case s.OutgoingResult != nil: // we need to register outgoing result
		return ctx.Jump(s.stepRegisterOutgoingResult)
	case s.Outgoing != nil: // we need to register outgoing
		return ctx.Jump(s.stepRegisterOutgoing)
	case s.IncomingResult != nil: // we need to register incoming result
		return ctx.Jump(s.stepRegisterIncomingResult)
	default:
		panic(throw.IllegalState())
	}
}

func (s *SMRegisterOnLMN) stepRegisterLifeline(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !s.Object.IsEmpty() {
		panic(throw.IllegalValue())
	} else if !s.LastFilamentRef.IsEmpty() {
		panic(throw.IllegalValue())
	}

	var (
		record         = &rms.RLifelineStart{}
		anticipatedRef = s.getRecordAnticipatedRef(s.Object, record)
	)

	s.Object = anticipatedRef
	s.LastFilamentRef = anticipatedRef

	if err := s.registerMessage(ctx, &rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // it should be based on
		// TODO: here we should set all overrides, since RLifelineStart contains
		//       ROutboundRequest and it has bad RootRef/PrevRef.
		// OverrideRecordType: rms.RLifelineStart,
		// OverridePrevRef:    NewReference(reference.Global{}), // must be empty
		// OverrideRootRef:    NewReference(reference.Global{}), // must be empty
		// OverrideReasonRef:  NewReference(<reference to outgoing>),
	}); err != nil {
		return ctx.Error(err)
	}

	return ctx.Jump(s.stepSaveSafeCounter)
}

func (s *SMRegisterOnLMN) stepRegisterIncoming(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var record rms.BasicRecord

	switch s.Interference {
	case isolation.CallTolerable:
		if !s.LastLifelineRef.IsEmpty() {
			panic(throw.IllegalState())
		}
		record = &rms.RLineInboundRequest{
			RootRef: rms.NewReference(s.Object),
			PrevRef: rms.NewReference(s.LastLifelineRef),
		}
	case isolation.CallIntolerable:
		if !s.LastFilamentRef.IsEmpty() {
			panic(throw.IllegalState())
		}
		record = &rms.RInboundRequest{
			RootRef: rms.NewReference(s.Object),
			PrevRef: rms.NewReference(s.LastFilamentRef),
		}
	}

	var anticipatedRef = s.getRecordAnticipatedRef(s.Object, record)

	if err := s.registerMessage(ctx, &rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		return ctx.Error(err)
	}

	switch s.Interference {
	case isolation.CallTolerable:
		s.LastLifelineRef = anticipatedRef
	case isolation.CallIntolerable:
		s.LastFilamentRef = anticipatedRef
	}

	switch {
	case s.Outgoing != nil:
		return ctx.Jump(s.stepRegisterOutgoing)
	case s.OutgoingResult != nil:
		return ctx.Jump(s.stepRegisterOutgoingResult)
	case s.IncomingResult != nil:
		return ctx.Jump(s.stepRegisterIncomingResult)
	default:
		panic(throw.Unsupported())
	}
}

func (s *SMRegisterOnLMN) stepRegisterOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	prevRef := s.LastFilamentRef
	if prevRef.IsEmpty() {
		prevRef = s.LastLifelineRef
	}
	if prevRef.IsEmpty() {
		panic(throw.IllegalState())
	}

	record := &rms.ROutboundRequest{
		RootRef: rms.NewReference(s.Object),
		PrevRef: rms.NewReference(prevRef),
	}

	var anticipatedRef = s.getRecordAnticipatedRef(s.Object, record)

	if err := s.registerMessage(ctx, &rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		return ctx.Error(err)
	}

	s.LastFilamentRef = anticipatedRef

	return ctx.Jump(s.stepSaveSafeCounter)
}

func (s *SMRegisterOnLMN) stepRegisterOutgoingResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !s.LastFilamentRef.IsEmpty() {
		panic(throw.IllegalState())
	}

	record := &rms.ROutboundResponse{
		RootRef: rms.NewReference(s.Object),
		PrevRef: rms.NewReference(s.LastFilamentRef),
	}

	var anticipatedRef = s.getRecordAnticipatedRef(s.Object, record)

	if err := s.registerMessage(ctx, &rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		return ctx.Error(err)
	}

	s.LastFilamentRef = anticipatedRef

	switch {
	case s.Outgoing != nil:
		return ctx.Jump(s.stepRegisterOutgoing)
	case s.IncomingResult != nil:
		return ctx.Jump(s.stepRegisterIncomingResult)
	default:
		panic(throw.Unsupported())
	}
}

func (s *SMRegisterOnLMN) stepRegisterIncomingResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.IncomingResult.Type != execution.Done {
		panic(throw.IllegalState())
	}

	var (
		haveFilament  = true
		isConstructor = s.IncomingResult.Result.Type() == requestresult.SideEffectActivate
		isDestructor  = s.IncomingResult.Result.Type() == requestresult.SideEffectDeactivate
		isNone        = s.IncomingResult.Result.Type() == requestresult.SideEffectNone
	)

	{ // result of execution
		prevRef := s.LastFilamentRef
		if prevRef.IsEmpty() {
			haveFilament = false
			prevRef = s.LastLifelineRef
		}
		if prevRef.IsEmpty() {
			panic(throw.IllegalState())
		}

		record := &rms.RInboundResponse{
			RootRef: rms.NewReference(s.Object),
			PrevRef: rms.NewReference(prevRef),
		}

		var anticipatedRef = s.getRecordAnticipatedRef(s.Object, record)

		if err := s.registerMessage(ctx, &rms.LRegisterRequest{
			AnticipatedRef: rms.NewReference(anticipatedRef),
			Flags:          rms.RegistrationFlags_Safe,
			AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
		}); err != nil {
			return ctx.Error(err)
		}

		s.LastFilamentRef = anticipatedRef
	}

	// TODO: RejoinRef to LastFilamentRef
	{ // new memory (if needed)
		if s.LastLifelineRef.IsEmpty() {
			panic(throw.IllegalState())
		}

		var record rms.BasicRecord

		switch {
		case !haveFilament && isConstructor:
			record = &rms.RLineMemoryInit{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReference(s.LastLifelineRef),
			}
		case isDestructor:
			record = nil
		case isNone:
			// TODO: we should post here a link to previous memory
			record = &rms.RLineMemoryReuse{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReference(s.LastLifelineRef),
			}
		default:
			record = &rms.RLineMemory{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReference(s.LastLifelineRef),
			}
		}

		if record != nil {
			var anticipatedRef = s.getRecordAnticipatedRef(s.Object, record)

			if err := s.registerMessage(ctx, &rms.LRegisterRequest{
				AnticipatedRef: rms.NewReference(anticipatedRef),
				Flags:          rms.RegistrationFlags_Safe,
				AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
			}); err != nil {
				return ctx.Error(err)
			}

			s.LastLifelineRef = anticipatedRef
		}
	}

	// TODO: RejoinRef to LastFilamentRef
	{
		var record rms.BasicRecord

		switch {
		case isConstructor:
			record = &rms.RLineActivate{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReference(s.LastLifelineRef),
			}
		case isDestructor:
			record = &rms.RLineDeactivate{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReference(s.LastLifelineRef),
			}
		default:
			record = nil
		}

		if record != nil {
			var anticipatedRef = s.getRecordAnticipatedRef(s.Object, record)

			if err := s.registerMessage(ctx, &rms.LRegisterRequest{
				AnticipatedRef: rms.NewReference(anticipatedRef),
				Flags:          rms.RegistrationFlags_Safe,
				AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
			}); err != nil {
				return ctx.Error(err)
			}

			s.LastLifelineRef = anticipatedRef
		}
	}

	return ctx.Jump(s.stepSaveSafeCounter)
}

func (s *SMRegisterOnLMN) stepSaveSafeCounter(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.requiredSafe > 0 {
		action := func(state *object.SharedState) {
			state.RequiredSafeCounter = s.requiredSafe
		}

		switch s.ObjectSharedState.Prepare(action).TryUse(ctx).GetDecision() {
		case smachine.NotPassed:
			return ctx.WaitShared(s.ObjectSharedState.SharedDataLink).ThenRepeat()
		case smachine.Passed:
		default:
			panic(throw.Impossible())
		}
	}
	return ctx.Jump(s.stepSendMessages)
}

func (s *SMRegisterOnLMN) stepSendMessages(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		obj          = s.Object
		currentPulse = s.pulseSlot.CurrentPulseNumber()
		msg          = s.messages.NextUnsentMessage()
	)

	if msg == nil {
		return ctx.Stop()
	}

	return s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, msg.Payload(), affinity.DynamicRoleLightExecutor, obj, currentPulse)
		if err != nil {
			return func(ctx smachine.AsyncResultContext) {
				ctx.Log().Error("failed to send message", err)
			}
		}

		return nil
	}).DelayedStart().ThenJump(s.stepWaitResponses)
}

func (s *SMRegisterOnLMN) stepWaitResponses(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.NewLastLifelineRef = s.LastLifelineRef
	s.NewLastFilamentRef = s.LastFilamentRef
	s.NewObjectRef = s.Object

	if !s.messages.CurrentUnsentMessage().ResultReceived() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepSaveSafeCounter)
}
