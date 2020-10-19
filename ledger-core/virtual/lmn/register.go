// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package lmn

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

type SerializableBasicRecord interface {
	rmsreg.GoGoSerializable
	rms.BasicRecord
}

func mustRecordToAnyRecordLazy(rec SerializableBasicRecord) rms.AnyRecordLazy {
	fmt.Printf("%#v\n", rec)
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

	Object          reference.Global
	LastFilamentRef reference.Global
	LastLifelineRef reference.Global

	PulseNumber pulse.Number

	// internal data

	// output arguments
	NewObjectRef       reference.Global
	NewLastFilamentRef reference.Global
	NewLastLifelineRef reference.Global
	Messages           []SerializableBasicMessage

	// DI
	PulseSlot        *conveyor.PulseSlot
	ReferenceBuilder RecordReferenceBuilderService
}

func (s *RegisterRecordBuilder) getRecordAnticipatedRef(record SerializableBasicRecord) reference.Global {
	var (
		data        = make([]byte, record.ProtoSize())
		pulseNumber = s.PulseNumber
	)

	if pulseNumber == pulse.Unknown {
		if s.PulseSlot == nil {
			panic(throw.IllegalState())
		}
		pulseNumber = s.PulseSlot.CurrentPulseNumber()
	}

	_, err := record.MarshalTo(data)
	if err != nil {
		panic(throw.W(err, "Fail to serialize record"))
	}
	return s.ReferenceBuilder.AnticipatedRefFromBytes(s.Object, pulseNumber, data)
}

func (s *RegisterRecordBuilder) registerMessage(msg SerializableBasicMessage) error {
	s.Messages = append(s.Messages, msg)

	return nil
}

func GetLifelineAnticipatedReference(
	builder RecordReferenceBuilderService,
	request *rms.VCallRequest,
	pn pulse.Number,
) reference.Global {
	if request.CallOutgoing.IsEmpty() {
		panic(throw.IllegalValue())
	}

	if request.CallType != rms.CallTypeConstructor {
		panic(throw.IllegalValue())
	}

	sm := RegisterRecordBuilder{
		PulseNumber:      request.CallOutgoing.GetPulseOfLocal(),
		Incoming:         request,
		ReferenceBuilder: builder,
	}
	return sm.getRecordAnticipatedRef(sm.getLifelineRecord())
}

func GetOutgoingAnticipatedReference(
	builder RecordReferenceBuilderService,
	request *rms.VCallRequest,
	previousRef reference.Global,
	pn pulse.Number,
) reference.Global {
	sm := RegisterRecordBuilder{
		PulseNumber:      pn,
		Object:           request.Callee.GetValue(),
		Outgoing:         request,
		ReferenceBuilder: builder,
		LastLifelineRef:  previousRef,
	}
	return sm.getRecordAnticipatedRef(sm.getOutboundRecord())
}

func (s *RegisterRecordBuilder) getOutboundRecord() *rms.ROutboundRequest {
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

	return s.getCommonOutboundRecord(s.Incoming, reference.Global{})
}

func (s *RegisterRecordBuilder) getCommonOutboundRecord(msg *rms.VCallRequest, prevRef reference.Global) *rms.ROutboundRequest {
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

	if !prevRef.IsEmpty() {
		record.RootRef.Set(s.Object)
		record.PrevRef.Set(prevRef)
	}

	return record
}

func (s *RegisterRecordBuilder) getInboundRecord() *rms.RInboundRequest {
	switch {
	case s.Incoming == nil:
		panic(throw.IllegalState())
	case s.LastLifelineRef.IsEmpty():
		panic(throw.IllegalState())
	case s.Object.IsEmpty():
		panic(throw.IllegalState())
	}

	return s.getCommonOutboundRecord(s.Incoming, s.LastFilamentRef)
}

func (s *RegisterRecordBuilder) getLineInboundRecord() *rms.RLineInboundRequest {
	switch {
	case s.Incoming == nil:
		panic(throw.IllegalState())
	case s.LastLifelineRef.IsEmpty():
		panic(throw.IllegalState())
	case s.Object.IsEmpty():
		panic(throw.IllegalState())
	}

	return s.getCommonOutboundRecord(s.Incoming, s.LastLifelineRef)
}

func (s *RegisterRecordBuilder) BuildLifeline() error {
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

	if err := s.registerMessage(&rms.LRegisterRequest{
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
		panic(throw.W(err, "failed to register message"))
	}

	s.NewLastLifelineRef = s.LastLifelineRef
	s.NewLastFilamentRef = s.LastFilamentRef
	s.NewObjectRef = s.Object

	return nil
}

func (s *RegisterRecordBuilder) BuildRegisterIncomingRequest() error {
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

	if err := s.registerMessage(&rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(anticipatedRef),
		Flags:          flags,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		panic(throw.W(err, "failed to register message"))
	}

	switch s.Interference {
	case isolation.CallTolerable:
		s.LastLifelineRef = anticipatedRef
	case isolation.CallIntolerable:
		s.LastFilamentRef = anticipatedRef
	}

	// switch {
	// case s.Outgoing != nil:
	// 	return s.BuildRegisterOutgoingRequest()
	// case s.IncomingResult != nil:
	// 	return s.BuildRegisterIncomingResult()
	// default:
	// 	panic(throw.Unsupported())
	// }
	return nil
}

func (s *RegisterRecordBuilder) BuildRegisterOutgoingRequest() error {
	if s.Incoming != nil {
		if err := s.BuildRegisterIncomingRequest(); err != nil {
			return err
		}
	}

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

	if err := s.registerMessage(&rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		panic(throw.W(err, "failed to register message"))
	}

	s.LastFilamentRef = anticipatedRef

	s.NewLastLifelineRef = s.LastLifelineRef
	s.NewLastFilamentRef = s.LastFilamentRef
	s.NewObjectRef = s.Object

	return nil
}

func (s *RegisterRecordBuilder) BuildRegisterOutgoingResult() error {

	if s.LastFilamentRef.IsEmpty() {
		panic(throw.IllegalState())
	}

	record := &rms.ROutboundResponse{
		RootRef: rms.NewReference(s.Object),
		PrevRef: rms.NewReference(s.LastFilamentRef),
	}

	var anticipatedRef = s.getRecordAnticipatedRef(record)

	if err := s.registerMessage(&rms.LRegisterRequest{
		AnticipatedRef: rms.NewReference(anticipatedRef),
		Flags:          rms.RegistrationFlags_FastSafe,
		AnyRecordLazy:  mustRecordToAnyRecordLazy(record), // TODO: here we should provide record from incoming
	}); err != nil {
		panic(throw.W(err, "failed to register message"))
	}

	s.LastFilamentRef = anticipatedRef

	s.NewLastLifelineRef = s.LastLifelineRef
	s.NewLastFilamentRef = s.LastFilamentRef
	s.NewObjectRef = s.Object

	return nil
}

func (s *RegisterRecordBuilder) BuildRegisterIncomingResult() error {
	if s.Incoming != nil {
		if err := s.BuildRegisterIncomingRequest(); err != nil {
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

		if err := s.registerMessage(&rms.LRegisterRequest{
			AnticipatedRef: rms.NewReference(anticipatedRef),
			Flags:          rms.RegistrationFlags_Safe,
			AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
		}); err != nil {
			panic(throw.W(err, "failed to register message"))
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

			if err := s.registerMessage(&rms.LRegisterRequest{
				AnticipatedRef: rms.NewReference(anticipatedRef),
				Flags:          rms.RegistrationFlags_Safe,
				AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
			}); err != nil {
				panic(throw.W(err, "failed to register message"))
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

			if err := s.registerMessage(&rms.LRegisterRequest{
				AnticipatedRef: rms.NewReference(anticipatedRef),
				Flags:          rms.RegistrationFlags_Safe,
				AnyRecordLazy:  mustRecordToAnyRecordLazy(record),
			}); err != nil {
				panic(throw.W(err, "failed to register message"))
			}

			s.LastLifelineRef = anticipatedRef
		}
	}

	s.NewLastLifelineRef = s.LastLifelineRef
	s.NewLastFilamentRef = s.LastFilamentRef
	s.NewObjectRef = s.Object

	return nil
}
