// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package lmn

import (
	"context"
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute/shared"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SerializableBasicRecord interface {
	rmsreg.GoGoSerializable
	rms.BasicRecord
}

func mustRecordToAnyRecordLazy(rec SerializableBasicRecord) rms.AnyRecordLazy {
	if rec == nil {
		panic(throw.IllegalValue())
	}
	rv := rms.AnyRecordLazy{}
	if err := rv.SetAsLazy(rec); err != nil {
		panic(err)
	}
	return rv
}

func isConstructor(request *rms.VCallRequest) bool {
	return request.CallType == rms.CallTypeConstructor
}

type Message struct {
	payload            rmsreg.GoGoSerializable
	registrarSignature rms.Binary
	resultKey          ResultAwaitKey
}

func (m Message) ResultReceived() bool {
	return !m.registrarSignature.IsEmpty()
}

func (m *Message) SetResult(signature rms.Binary) {
	m.registrarSignature = signature
}

func (m Message) Payload() rmsreg.GoGoSerializable {
	return m.payload
}

func (m *Message) CheckKey(key ResultAwaitKey) bool {
	return m.resultKey == key
}

type MessagesHolder struct {
	messages     []*Message
	sentPosition int
}

func (s *MessagesHolder) AppendMessage(record rmsreg.GoGoSerializable, resultKey ResultAwaitKey) {
	s.messages = append(s.messages, &Message{
		payload:   record,
		resultKey: resultKey,
	})
}

func (s *MessagesHolder) NextUnsentMessage() *Message {
	s.sentPosition++
	if s.sentPosition >= len(s.messages) {
		return nil
	}
	return s.messages[s.sentPosition]
}

func (s *MessagesHolder) CurrentSentMessage() *Message {
	if s.sentPosition < 0 && s.sentPosition >= len(s.messages) {
		panic(throw.IllegalState())
	}
	return s.messages[s.sentPosition]
}

type SubSMRegister struct {
	// input arguments
	Incoming          *rms.VCallRequest
	Outgoing          *rms.VCallRequest
	OutgoingRepeat    bool
	OutgoingResult    *rms.VCallResult
	IncomingResult    *execution.Update
	Interference      isolation.InterferenceFlag
	ObjectSharedState object.SharedStateAccessor

	Object          reference.Global
	LastFilamentRef reference.Global
	LastLifelineRef reference.Global

	SafeResponseCounter smachine.SharedDataLink
	PulseNumber         pulse.Number

	// internal data
	messages     MessagesHolder
	requiredSafe int
	sendError    error

	// output arguments
	NewObjectRef       reference.Global
	NewLastFilamentRef reference.Global
	NewLastLifelineRef reference.Global

	// DI
	messageSender messageSenderAdapter.MessageSender
	pulseSlot     *conveyor.PulseSlot
	refBuilder    RecordReferenceBuilderService
}

var dSubSMRegisterInstance smachine.StateMachineDeclaration = &dSubSMRegister{}

type dSubSMRegister struct {
	smachine.StateMachineDeclTemplate
}

func (*dSubSMRegister) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SubSMRegister)

	injector.MustInject(&s.messageSender)
	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.refBuilder)
}

func (*dSubSMRegister) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return nil
}

func (s *SubSMRegister) GetSubroutineInitState(ctx smachine.SubroutineStartContext) smachine.InitFunc {
	ctx.SetSubroutineCleanupMode(smachine.SubroutineCleanupAliasesAndShares)

	return s.Init
}

func (s *SubSMRegister) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSubSMRegisterInstance
}

func (s *SubSMRegister) getRecordAnticipatedRef(record SerializableBasicRecord) reference.Global {
	var (
		data        = make([]byte, record.ProtoSize())
		pulseNumber = s.PulseNumber
	)

	if pulseNumber == pulse.Unknown {
		if s.pulseSlot == nil {
			panic(throw.IllegalState())
		}
		pulseNumber = s.pulseSlot.CurrentPulseNumber()
	}

	_, err := record.MarshalTo(data)
	if err != nil {
		panic(throw.W(err, "Fail to serialize record"))
	}
	return s.refBuilder.AnticipatedRefFromBytes(s.Object, pulseNumber, data)
}

func (s *SubSMRegister) bargeInHandler(param interface{}) smachine.BargeInCallbackFunc {
	res, ok := param.(*rms.LRegisterResponse)
	if !ok || res == nil {
		panic(throw.IllegalValue())
	}

	return func(ctx smachine.BargeInContext) smachine.StateUpdate {
		var key = NewResultAwaitKey(res.AnticipatedRef, res.Flags)

		unsentMsg := s.messages.CurrentSentMessage()
		if !unsentMsg.CheckKey(key) {
			panic(throw.E("Message order is broken"))
		}

		unsentMsg.SetResult(res.RegistrarSignature)

		return ctx.WakeUp()
	}
}

type logRegisterBargeIn struct {
	*log.Msg       `txt:"publishing bargeIn callback"`
	AnticipatedRef reference.Global
	Flag           rms.RegistrationFlags
	Type           reflect.Type
}

func (s *SubSMRegister) registerMessage(ctx smachine.ExecutionContext, msg *rms.LRegisterRequest) error {
	waitFlag := msg.Flags

	switch msg.Flags {
	case rms.RegistrationFlags_FastSafe:
		s.requiredSafe++

		// TODO: add destructor to finish that SM's if not all answers were
		ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
			return &SMWaitSafeResponse{
				ObjectSharedState:   s.ObjectSharedState,
				ExpectedKey:         NewResultAwaitKey(msg.AnticipatedRef, rms.RegistrationFlags_Safe),
				SafeResponseCounter: s.SafeResponseCounter,
			}
		})

		waitFlag = rms.RegistrationFlags_Fast

		fallthrough
	case rms.RegistrationFlags_Fast, rms.RegistrationFlags_Safe:
		var (
			bargeInKey = NewResultAwaitKey(msg.AnticipatedRef, waitFlag)
			bargeIn    = ctx.NewBargeInWithParam(s.bargeInHandler)
		)

		if !ctx.PublishGlobalAliasAndBargeIn(bargeInKey, bargeIn) {
			return throw.E("failed to publish bargeIn callback")
		}
		s.messages.AppendMessage(msg, bargeInKey)
	default:
		panic(throw.IllegalValue())
	}

	return nil
}

func (s *SubSMRegister) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	// do not do anything on migration, let parent SM to decide what needs to be done
	ctx.SetDefaultMigration(func(ctx smachine.MigrationContext) smachine.StateUpdate { return ctx.Stop() })

	// initialize message handler
	s.messages.sentPosition = -1

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

func GetLifelineAnticipatedReference(
	builder RecordReferenceBuilderService,
	request *rms.VCallRequest,
	pn pulse.Number,
) reference.Global {
	if request.CallOutgoing.IsEmpty() {
		panic(throw.IllegalState())
	}

	sm := SubSMRegister{
		PulseNumber: request.CallOutgoing.GetPulseOfLocal(),
		Incoming:    request,
		refBuilder:  builder,
	}
	return sm.getRecordAnticipatedRef(sm.getLifelineRecord())
}

func GetOutgoingAnticipatedReference(
	builder RecordReferenceBuilderService,
	request *rms.VCallRequest,
	previousRef reference.Global,
	pn pulse.Number,
) reference.Global {
	sm := SubSMRegister{
		PulseNumber:     pn,
		Object:          request.Callee.GetValue(),
		Outgoing:        request,
		refBuilder:      builder,
		LastLifelineRef: previousRef,
	}
	return sm.getRecordAnticipatedRef(sm.getOutboundRecord())
}

func (s *SubSMRegister) getOutboundRecord() *rms.ROutboundRequest {
	if s.Outgoing == nil {
		panic(throw.IllegalState())
	}

	// first outgoing of incoming should be branched from
	prevRef := s.LastFilamentRef
	if prevRef.IsEmpty() {
		prevRef = s.LastLifelineRef
	}
	if prevRef.IsEmpty() {
		panic(throw.IllegalState())
	}

	return &rms.ROutboundRequest{
		CallType:                s.Outgoing.CallType,
		CallFlags:               s.Outgoing.CallFlags,
		CallAsOf:                s.Outgoing.CallAsOf,
		Caller:                  s.Outgoing.Caller,
		Callee:                  s.Outgoing.Callee,
		CallSiteDeclaration:     s.Outgoing.CallSiteDeclaration,
		CallSiteMethod:          s.Outgoing.CallSiteMethod,
		CallSequence:            s.Outgoing.CallSequence,
		CallReason:              s.Outgoing.CallReason,
		RootTX:                  s.Outgoing.RootTX,
		CallTX:                  s.Outgoing.CallTX,
		ExpenseCenter:           s.Outgoing.ExpenseCenter,
		ResourceCenter:          s.Outgoing.ResourceCenter,
		DelegationSpec:          s.Outgoing.DelegationSpec,
		ProducerSignature:       s.Outgoing.ProducerSignature,
		RegistrarSignature:      s.Outgoing.RegistrarSignature,
		RegistrarDelegationSpec: s.Outgoing.RegistrarDelegationSpec,
		CallRequestFlags:        s.Outgoing.CallRequestFlags,
		KnownCalleeIncoming:     s.Outgoing.KnownCalleeIncoming,
		TXExpiry:                s.Outgoing.TXExpiry,
		SecurityContext:         s.Outgoing.SecurityContext,
		TXContext:               s.Outgoing.TXContext,
		Arguments:               s.Outgoing.Arguments, // TODO: move later to RecordBody

		RootRef: rms.NewReference(s.Object),
		PrevRef: rms.NewReference(prevRef),
	}
}

func (s *SubSMRegister) getOutboundRetryableRequest() *rms.ROutboundRetryableRequest {
	retryableOutbound := rms.ROutboundRetryableRequest(*s.getOutboundRecord())

	return &retryableOutbound
}

func (s *SubSMRegister) getOutboundRetryRequest() *rms.ROutboundRetryRequest {
	retryOutbound := rms.ROutboundRetryRequest(*s.getOutboundRecord())

	return &retryOutbound
}

func (s *SubSMRegister) getLifelineRecord() *rms.RLifelineStart {
	if s.Incoming == nil {
		panic(throw.IllegalState())
	}

	return &rms.RLifelineStart{
		CallType:                s.Incoming.CallType,
		CallFlags:               s.Incoming.CallFlags,
		CallAsOf:                s.Incoming.CallAsOf,
		Caller:                  s.Incoming.Caller,
		Callee:                  s.Incoming.Callee,
		CallSiteDeclaration:     s.Incoming.CallSiteDeclaration,
		CallSiteMethod:          s.Incoming.CallSiteMethod,
		CallSequence:            s.Incoming.CallSequence,
		CallReason:              s.Incoming.CallReason,
		RootTX:                  s.Incoming.RootTX,
		CallTX:                  s.Incoming.CallTX,
		ExpenseCenter:           s.Incoming.ExpenseCenter,
		ResourceCenter:          s.Incoming.ResourceCenter,
		DelegationSpec:          s.Incoming.DelegationSpec,
		ProducerSignature:       s.Incoming.ProducerSignature,
		RegistrarSignature:      s.Incoming.RegistrarSignature,
		RegistrarDelegationSpec: s.Incoming.RegistrarDelegationSpec,
		CallRequestFlags:        s.Incoming.CallRequestFlags,
		KnownCalleeIncoming:     s.Incoming.KnownCalleeIncoming,
		TXExpiry:                s.Incoming.TXExpiry,
		SecurityContext:         s.Incoming.SecurityContext,
		TXContext:               s.Incoming.TXContext,
		Arguments:               s.Incoming.Arguments, // TODO: move later to RecordBody
	}
}

func (s *SubSMRegister) getInboundRecord() *rms.RInboundRequest {
	switch {
	case s.Incoming == nil:
		panic(throw.IllegalState())
	case s.LastFilamentRef.IsEmpty():
		panic(throw.IllegalState())
	case s.Object.IsEmpty():
		panic(throw.IllegalState())
	}

	return &rms.RInboundRequest{
		RootRef: rms.NewReference(s.Object),
		PrevRef: rms.NewReference(s.LastFilamentRef),

		CallType:                s.Incoming.CallType,
		CallFlags:               s.Incoming.CallFlags,
		CallAsOf:                s.Incoming.CallAsOf,
		Caller:                  s.Incoming.Caller,
		Callee:                  s.Incoming.Callee,
		CallSiteDeclaration:     s.Incoming.CallSiteDeclaration,
		CallSiteMethod:          s.Incoming.CallSiteMethod,
		CallSequence:            s.Incoming.CallSequence,
		CallReason:              s.Incoming.CallReason,
		RootTX:                  s.Incoming.RootTX,
		CallTX:                  s.Incoming.CallTX,
		ExpenseCenter:           s.Incoming.ExpenseCenter,
		ResourceCenter:          s.Incoming.ResourceCenter,
		DelegationSpec:          s.Incoming.DelegationSpec,
		ProducerSignature:       s.Incoming.ProducerSignature,
		RegistrarSignature:      s.Incoming.RegistrarSignature,
		RegistrarDelegationSpec: s.Incoming.RegistrarDelegationSpec,
		CallRequestFlags:        s.Incoming.CallRequestFlags,
		KnownCalleeIncoming:     s.Incoming.KnownCalleeIncoming,
		CallOutgoing:            s.Incoming.CallOutgoing,
		TXExpiry:                s.Incoming.TXExpiry,
		SecurityContext:         s.Incoming.SecurityContext,
		TXContext:               s.Incoming.TXContext,
		Arguments:               s.Incoming.Arguments, // TODO: move later to RecordBody
	}
}

func (s *SubSMRegister) getLineInboundRecord() *rms.RLineInboundRequest {
	switch {
	case s.Incoming == nil:
		panic(throw.IllegalState())
	case s.LastLifelineRef.IsEmpty():
		panic(throw.IllegalState())
	case s.Object.IsEmpty():
		panic(throw.IllegalState())
	}

	return &rms.RLineInboundRequest{
		RootRef: rms.NewReference(s.Object),
		PrevRef: rms.NewReference(s.LastLifelineRef),

		CallType:                s.Incoming.CallType,
		CallFlags:               s.Incoming.CallFlags,
		CallAsOf:                s.Incoming.CallAsOf,
		Caller:                  s.Incoming.Caller,
		Callee:                  s.Incoming.Callee,
		CallSiteDeclaration:     s.Incoming.CallSiteDeclaration,
		CallSiteMethod:          s.Incoming.CallSiteMethod,
		CallSequence:            s.Incoming.CallSequence,
		CallReason:              s.Incoming.CallReason,
		RootTX:                  s.Incoming.RootTX,
		CallTX:                  s.Incoming.CallTX,
		ExpenseCenter:           s.Incoming.ExpenseCenter,
		ResourceCenter:          s.Incoming.ResourceCenter,
		DelegationSpec:          s.Incoming.DelegationSpec,
		ProducerSignature:       s.Incoming.ProducerSignature,
		RegistrarSignature:      s.Incoming.RegistrarSignature,
		RegistrarDelegationSpec: s.Incoming.RegistrarDelegationSpec,
		CallRequestFlags:        s.Incoming.CallRequestFlags,
		KnownCalleeIncoming:     s.Incoming.KnownCalleeIncoming,
		CallOutgoing:            s.Incoming.CallOutgoing,
		TXExpiry:                s.Incoming.TXExpiry,
		SecurityContext:         s.Incoming.SecurityContext,
		TXContext:               s.Incoming.TXContext,
		Arguments:               s.Incoming.Arguments, // TODO: move later to RecordBody
	}
}

func (s *SubSMRegister) stepRegisterLifeline(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !s.Object.IsEmpty() {
		panic(throw.IllegalValue())
	} else if !s.LastFilamentRef.IsEmpty() {
		panic(throw.IllegalValue())
	}

	s.PulseNumber = s.Incoming.CallOutgoing.GetPulseOfLocal()

	var (
		record         = s.getLifelineRecord()
		anticipatedRef = s.getRecordAnticipatedRef(record)
	)

	s.PulseNumber = pulse.Unknown

	s.Object = anticipatedRef
	s.LastLifelineRef = anticipatedRef

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

func (s *SubSMRegister) stepRegisterIncoming(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var record SerializableBasicRecord

	switch s.Interference {
	case isolation.CallTolerable:
		record = s.getLineInboundRecord()
	case isolation.CallIntolerable:
		record = s.getInboundRecord()
	default:
		panic(throw.IllegalValue())
	}

	var anticipatedRef = s.getRecordAnticipatedRef(record)

	flags := rms.RegistrationFlags_FastSafe
	if s.Incoming != nil {
		flags = rms.RegistrationFlags_Safe
	}

	if err := s.registerMessage(ctx, &rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(anticipatedRef),
		Flags:          flags,
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
	case s.IncomingResult != nil:
		return ctx.Jump(s.stepRegisterIncomingResult)
	default:
		panic(throw.Unsupported())
	}
}

func (s *SubSMRegister) stepRegisterOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var record SerializableBasicRecord
	switch {
	case s.Outgoing.CallType == rms.CallTypeConstructor && s.OutgoingRepeat:
		record = s.getOutboundRetryRequest()
	case s.Outgoing.CallType == rms.CallTypeConstructor && !s.OutgoingRepeat:
		record = s.getOutboundRetryableRequest()
	case s.Outgoing.CallType == rms.CallTypeMethod:
		record = s.getOutboundRecord()
	}

	var anticipatedRef = s.getRecordAnticipatedRef(record)

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

func (s *SubSMRegister) stepRegisterOutgoingResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.LastFilamentRef.IsEmpty() {
		panic(throw.IllegalState())
	}

	record := &rms.ROutboundResponse{
		RootRef: rms.NewReference(s.Object),
		PrevRef: rms.NewReference(s.LastFilamentRef),
	}

	var anticipatedRef = s.getRecordAnticipatedRef(record)

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

func (s *SubSMRegister) stepRegisterIncomingResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.IncomingResult.Type != execution.Done && s.IncomingResult.Type != execution.Error {
		panic(throw.IllegalState())
	}

	var (
		haveFilament  = true
		isIntolerable = s.Interference == isolation.CallIntolerable
		isConstructor = s.IncomingResult.Result.Type() == requestresult.SideEffectActivate
		isDestructor  = s.IncomingResult.Result.Type() == requestresult.SideEffectDeactivate
		isNone        = s.IncomingResult.Result.Type() == requestresult.SideEffectNone
		isError       = s.IncomingResult.Error != nil
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

		var anticipatedRef = s.getRecordAnticipatedRef(record)

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

		var record SerializableBasicRecord

		switch {
		case isIntolerable:
			record = nil
		case !haveFilament && isConstructor:
			record = &rms.RLineMemoryInit{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReference(s.LastLifelineRef),
			}
		case isDestructor:
			record = nil
		case isError:
			// TODO: ???
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
			var anticipatedRef = s.getRecordAnticipatedRef(record)

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
		var record SerializableBasicRecord

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
			var anticipatedRef = s.getRecordAnticipatedRef(record)

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

func (s *SubSMRegister) stepSaveSafeCounter(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// save new intermediate state
	s.NewLastLifelineRef = s.LastLifelineRef
	s.NewLastFilamentRef = s.LastFilamentRef
	s.NewObjectRef = s.Object

	if s.requiredSafe < 0 {
		panic(throw.IllegalState())
	}

	stateUpdate := shared.CounterIncrement(ctx, s.SafeResponseCounter, s.requiredSafe)
	if !stateUpdate.IsEmpty() {
		return stateUpdate
	}

	return ctx.Jump(s.stepSendMessage)
}

func (s *SubSMRegister) stepSendMessage(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.pulseSlot == nil {
		panic(throw.IllegalState())
	}

	var (
		obj          = s.Object
		msg          = s.messages.NextUnsentMessage()
		currentPulse = s.pulseSlot.CurrentPulseNumber()
	)

	if msg == nil {
		return ctx.Stop()
	}

	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, msg.Payload(), affinity.DynamicRoleLightExecutor, obj, currentPulse)
		return func(ctx smachine.AsyncResultContext) {
			s.sendError = throw.W(err, "failed to send LRegisterRequest message")
		}
	}).Start()

	return ctx.Jump(s.stepWaitResponse)
}

func (s *SubSMRegister) stepWaitResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.sendError != nil {
		return ctx.Error(s.sendError)
	}

	if !s.messages.CurrentSentMessage().ResultReceived() {
		return ctx.Sleep().ThenRepeat()
	}

	return ctx.Jump(s.stepSendMessage)
}
