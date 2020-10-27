// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package vnlmn

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsbox"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SerializableReferencableBasicRecord interface {
	rmsreg.GoGoSerializable
	rms.BasicRecord
	rmsbox.Referencable
}

func mustRecordToAnyRecordLazy(rec SerializableReferencableBasicRecord) rms.AnyRecordLazy {
	if rec == nil {
		panic(throw.IllegalValue())
	}
	rv := rms.AnyRecordLazy{}
	if err := rv.SetAsLazy(rec); err != nil {
		panic(err)
	}
	return rv
}

type RegisterRecordBuilder struct {
	// input arguments
	Incoming          *rms.VCallRequest
	Outgoing          *rms.VCallRequest
	OutgoingRepeat    bool
	OutgoingResult    *rms.VCallResult
	IncomingResult    *execution.Update
	Interference      isolation.InterferenceFlag
	ObjectSharedState object.SharedStateAccessor

	Object      reference.Global
	PulseNumber pulse.Number

	// output arguments
	Context       *RegistrationCtx
	lastBranchRef rmsbox.ReferenceProvider
	lastTrunkRef  rmsbox.ReferenceProvider
	Messages      []SerializableBasicMessage

	// DI
	PulseGetter  func() pulse.Number
	DataDigester cryptkit.DataDigester
}

func (s *RegisterRecordBuilder) getRecordAnticipatedRef(record SerializableReferencableBasicRecord) rmsbox.ReferenceProvider {
	var (
		pulseNumber = s.PulseNumber
	)

	if pulseNumber == pulse.Unknown {
		if s.PulseGetter == nil {
			panic(throw.IllegalState())
		}
		pulseNumber = s.PulseGetter()
	}

	refTemplate := reference.NewRefTemplate(s.Object, pulseNumber)
	rmsbox.InitReferenceFactory(record, s.DataDigester, refTemplate)
	rmsbox.SetReferenceFactoryCanPull(record, true)

	return rmsbox.DefaultLazyReferenceTo(record)
}

//nolint:interfacer
func (s *RegisterRecordBuilder) getObjectAnticipatedRef(record SerializableReferencableBasicRecord) rmsbox.ReferenceProvider {
	var (
		pulseNumber = s.PulseNumber
	)

	if pulseNumber == pulse.Unknown {
		if s.PulseGetter == nil {
			panic(throw.IllegalState())
		}
		pulseNumber = s.PulseGetter()
	}

	refTemplate := reference.NewSelfRefTemplate(pulseNumber, reference.SelfScopeLifeline)
	rmsbox.InitReferenceFactory(record, s.DataDigester, refTemplate)
	rmsbox.SetReferenceFactoryCanPull(record, true)

	return rmsbox.DefaultLazyReferenceTo(record)
}

//nolint:unparam
func (s *RegisterRecordBuilder) registerMessage(msg SerializableBasicMessage) error {
	s.Messages = append(s.Messages, msg)

	return nil
}

func GetLifelineAnticipatedReference(
	digester cryptkit.DataDigester,
	request *rms.VCallRequest,
	_ pulse.Number,
) reference.Global {
	if request.CallOutgoing.IsEmpty() {
		panic(throw.IllegalValue())
	}

	if request.CallType != rms.CallTypeConstructor {
		panic(throw.IllegalValue())
	}

	sm := RegisterRecordBuilder{
		PulseNumber:  request.CallOutgoing.GetPulseOfLocal(),
		Incoming:     request,
		DataDigester: digester,
	}
	return sm.getObjectAnticipatedRef(sm.getLifelineRecord()).TryPullReference()
}

func GetOutgoingAnticipatedReference(
	digester cryptkit.DataDigester,
	request *rms.VCallRequest,
	previousRef reference.Global,
	pn pulse.Number,
) reference.Global {
	sm := RegisterRecordBuilder{
		PulseNumber:  pn,
		Object:       request.Callee.GetValue(),
		Outgoing:     request,
		Context:      NewDummyRegistrationCtx(previousRef),
		DataDigester: digester,
	}
	return sm.getRecordAnticipatedRef(sm.GetOutboundRecord()).TryPullReference()
}

func (s *RegisterRecordBuilder) GetOutboundRecord() *rms.ROutboundRequest {
	s.lastTrunkRef = s.Context.TrunkRef()
	s.lastBranchRef = s.Context.BranchRef()

	return s.getOutboundRecord()
}

func (s *RegisterRecordBuilder) getOutboundRecord() *rms.ROutboundRequest {
	if s.Outgoing == nil {
		panic(throw.IllegalState())
	}

	// first outgoing of incoming should be branched from
	prevRef := s.lastBranchRef
	if prevRef == nil {
		prevRef = s.lastTrunkRef
	}
	if prevRef == nil {
		panic(throw.IllegalState())
	}

	return s.getCommonOutboundRecord(s.Outgoing, prevRef)
}

func (s *RegisterRecordBuilder) getOutboundRetryableRequest() *rms.ROutboundRetryableRequest {
	return s.getOutboundRecord()
}

func (s *RegisterRecordBuilder) getOutboundRetryRequest() *rms.ROutboundRetryRequest {
	return s.getOutboundRecord()
}

func (s *RegisterRecordBuilder) getLifelineRecord() *rms.RLifelineStart {
	if s.Incoming == nil {
		panic(throw.IllegalState())
	}

	return s.getCommonOutboundRecord(s.Incoming, nil)
}

func (s *RegisterRecordBuilder) getCommonOutboundRecord(msg *rms.VCallRequest, prevRef rmsbox.ReferenceProvider) *rms.ROutboundRequest {
	record := &rms.ROutboundRequest{
		CallType:            msg.CallType,
		CallFlags:           msg.CallFlags,
		CallAsOf:            msg.CallAsOf,
		Caller:              msg.Caller,
		Callee:              msg.Callee,
		CallSiteDeclaration: msg.CallSiteDeclaration,
		CallSiteMethod:      msg.CallSiteMethod,
		CallSequence:        msg.CallSequence,
		CallReason:          msg.CallReason,
		RootTX:              msg.RootTX,
		CallTX:              msg.CallTX,
		ExpenseCenter:       msg.ExpenseCenter,
		ResourceCenter:      msg.ResourceCenter,
		DelegationSpec:      msg.DelegationSpec,
		TXExpiry:            msg.TXExpiry,
		SecurityContext:     msg.SecurityContext,
		TXContext:           msg.TXContext,

		Arguments: msg.Arguments, // TODO: move later to RecordBody
	}

	if prevRef != nil {
		record.RootRef.Set(s.Object)
		record.PrevRef.SetLazy(prevRef)
	}

	return record
}

func (s *RegisterRecordBuilder) getInboundRecord() *rms.RInboundRequest {
	switch {
	case s.Incoming == nil:
		panic(throw.IllegalState())
	case s.lastTrunkRef == nil:
		panic(throw.IllegalState())
	case s.Object.IsEmpty():
		panic(throw.IllegalState())
	}

	return s.getCommonOutboundRecord(s.Incoming, s.lastTrunkRef)
}

func (s *RegisterRecordBuilder) getLineInboundRecord() *rms.RLineInboundRequest {
	switch {
	case s.Incoming == nil:
		panic(throw.IllegalState())
	case s.lastTrunkRef == nil:
		panic(throw.IllegalState())
	case s.Object.IsEmpty():
		panic(throw.IllegalState())
	}

	return s.getCommonOutboundRecord(s.Incoming, s.lastTrunkRef)
}

func (s *RegisterRecordBuilder) BuildLifeline() error {
	s.lastTrunkRef = s.Context.TrunkRef()
	s.lastBranchRef = s.Context.BranchRef()

	switch {
	case !s.Object.IsEmpty():
		panic(throw.IllegalValue())
	case s.lastTrunkRef != nil:
		panic(throw.IllegalValue())
	case s.lastBranchRef != nil:
		panic(throw.IllegalValue())
	}

	s.PulseNumber = s.Incoming.CallOutgoing.GetPulseOfLocal()

	var (
		record         = s.getLifelineRecord()
		anticipatedRef = s.getObjectAnticipatedRef(record)
	)

	s.PulseNumber = pulse.Unknown

	if err := s.registerMessage(&rms.LRegisterRequest{
		AnticipatedRef: rms.NewReferenceLazy(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // it should be based on
		// TODO: here we should set all overrides, since RLifelineStart contains
		//       ROutboundRequest and it has bad RootRef/PrevRef.
		// OverrideRecordType: rms.RLifelineStart,
		// OverridePrevRef:    NewReference(reference.Global{}), // must be empty
		// OverrideRootRef:    NewReference(reference.Global{}), // must be empty
		// OverrideReasonRef:  NewReference(<reference to outgoing>),
	}); err != nil {
		panic(throw.W(err, "failed to register message"))
	}

	s.lastTrunkRef = anticipatedRef
	return s.Finalize()
}

func (s *RegisterRecordBuilder) buildRegisterIncomingRequest() error {
	var record SerializableReferencableBasicRecord

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

	if err := s.registerMessage(&rms.LRegisterRequest{
		AnticipatedRef: rms.NewReferenceLazy(anticipatedRef),
		Flags:          flags,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		panic(throw.W(err, "failed to register message"))
	}

	switch s.Interference {
	case isolation.CallTolerable:
		s.lastTrunkRef = anticipatedRef
	case isolation.CallIntolerable:
		s.lastBranchRef = anticipatedRef
	}

	return nil
}

func (s *RegisterRecordBuilder) BuildRegisterOutgoingRequest() error {
	s.lastTrunkRef = s.Context.TrunkRef()
	s.lastBranchRef = s.Context.BranchRef()

	if s.Incoming != nil {
		if err := s.buildRegisterIncomingRequest(); err != nil {
			return err
		}
	}

	var record SerializableReferencableBasicRecord
	switch {
	case s.Outgoing.CallType == rms.CallTypeConstructor && s.OutgoingRepeat:
		record = s.getOutboundRetryRequest()
	case s.Outgoing.CallType == rms.CallTypeConstructor && !s.OutgoingRepeat:
		record = s.getOutboundRetryableRequest()
	case s.Outgoing.CallType == rms.CallTypeMethod:
		record = s.getOutboundRecord()
	}

	var anticipatedRef = s.getRecordAnticipatedRef(record)

	if err := s.registerMessage(&rms.LRegisterRequest{
		AnticipatedRef: rms.NewReferenceLazy(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		panic(throw.W(err, "failed to register message"))
	}

	s.lastBranchRef = anticipatedRef
	return s.Finalize()
}

func (s *RegisterRecordBuilder) BuildRegisterOutgoingResult() error {
	s.lastTrunkRef = s.Context.TrunkRef()
	s.lastBranchRef = s.Context.BranchRef()

	if s.lastBranchRef == nil {
		panic(throw.IllegalState())
	}

	record := &rms.ROutboundResponse{
		RootRef: rms.NewReference(s.Object),
		PrevRef: rms.NewReferenceLazy(s.lastBranchRef),
	}

	var anticipatedRef = s.getRecordAnticipatedRef(record)

	if err := s.registerMessage(&rms.LRegisterRequest{
		AnticipatedRef: rms.NewReferenceLazy(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		panic(throw.W(err, "failed to register message"))
	}

	s.lastBranchRef = anticipatedRef
	return s.Finalize()
}

func (s *RegisterRecordBuilder) BuildRegisterIncomingResult() error {
	s.lastTrunkRef = s.Context.TrunkRef()
	s.lastBranchRef = s.Context.BranchRef()

	if s.Incoming != nil {
		if err := s.buildRegisterIncomingRequest(); err != nil {
			return err
		}
	}

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
		prevRef := s.lastBranchRef
		if prevRef == nil {
			haveFilament = false
			prevRef = s.lastTrunkRef
		}
		if prevRef == nil {
			panic(throw.IllegalState())
		}

		record := &rms.RInboundResponse{
			RootRef: rms.NewReference(s.Object),
			PrevRef: rms.NewReferenceLazy(prevRef),
		}

		var anticipatedRef = s.getRecordAnticipatedRef(record)

		if err := s.registerMessage(&rms.LRegisterRequest{
			AnticipatedRef: rms.NewReferenceLazy(anticipatedRef),
			Flags:          rms.RegistrationFlags_Safe,
			AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
		}); err != nil {
			panic(throw.W(err, "failed to register message"))
		}

		s.lastBranchRef = anticipatedRef
	}

	// TODO: RejoinRef to LastFilamentRef
	{ // new memory (if needed)
		if s.lastTrunkRef == nil {
			panic(throw.IllegalState())
		}

		var record SerializableReferencableBasicRecord

		switch {
		case isIntolerable:
			record = nil
		case !haveFilament && isConstructor:
			record = &rms.RLineMemoryInit{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReferenceLazy(s.lastTrunkRef),
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
				PrevRef: rms.NewReferenceLazy(s.lastTrunkRef),
			}
		default:
			record = &rms.RLineMemory{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReferenceLazy(s.lastTrunkRef),
			}
		}

		if record != nil {
			var anticipatedRef = s.getRecordAnticipatedRef(record)

			if err := s.registerMessage(&rms.LRegisterRequest{
				AnticipatedRef: rms.NewReferenceLazy(anticipatedRef),
				Flags:          rms.RegistrationFlags_Safe,
				AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
			}); err != nil {
				panic(throw.W(err, "failed to register message"))
			}

			s.lastTrunkRef = anticipatedRef
		}
	}

	// TODO: RejoinRef to LastFilamentRef
	{
		var record SerializableReferencableBasicRecord

		switch {
		case isConstructor:
			record = &rms.RLineActivate{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReferenceLazy(s.lastTrunkRef),
			}
		case isDestructor:
			record = &rms.RLineDeactivate{
				RootRef: rms.NewReference(s.Object),
				PrevRef: rms.NewReferenceLazy(s.lastTrunkRef),
			}
		default:
			record = nil
		}

		if record != nil {
			var anticipatedRef = s.getRecordAnticipatedRef(record)

			if err := s.registerMessage(&rms.LRegisterRequest{
				AnticipatedRef: rms.NewReferenceLazy(anticipatedRef),
				Flags:          rms.RegistrationFlags_Safe,
				AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
			}); err != nil {
				panic(throw.W(err, "failed to register message"))
			}

			s.lastTrunkRef = anticipatedRef
		}
	}

	return s.Finalize()
}

func (s *RegisterRecordBuilder) Finalize() error {
	s.Context.SetNewReferences(s.lastTrunkRef, s.lastBranchRef)

	return nil
}
