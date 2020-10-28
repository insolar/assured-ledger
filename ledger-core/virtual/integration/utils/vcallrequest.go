// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/vnlmn"
)

type VCallRequestConstructorHandler struct {
	request rms.VCallRequest
	pn      pulse.Number
	scheme  crypto.PlatformScheme

	object      reference.Global
	previousRef reference.Global

	outgoing          reference.Global
	outgoingRewritten bool
}

func (h VCallRequestConstructorHandler) GetObject() reference.Global {
	return h.object
}

func (h VCallRequestConstructorHandler) GetOutgoing() reference.Global {
	return h.outgoing
}

func (h VCallRequestConstructorHandler) GetIncoming() reference.Global {
	return reference.NewRecordOf(h.request.Callee.GetValue(), h.outgoing.GetLocal())
}

func (h VCallRequestConstructorHandler) Get() rms.VCallRequest {
	return h.request
}

func (h *VCallRequestConstructorHandler) regenerate() {
	digester := h.scheme.RecordScheme().ReferenceDigester()

	if !h.outgoingRewritten {
		duplicateRequest := h.request
		duplicateRequest.CallOutgoing = rms.NewReference(reference.Global{})

		h.outgoing = vnlmn.GetOutgoingAnticipatedReference(digester, &duplicateRequest, h.previousRef, h.pn)
	}

	h.request.CallOutgoing = rms.NewReference(h.outgoing)

	h.object = vnlmn.GetLifelineAnticipatedReference(digester, &h.request, h.pn)

}

//nolint:interfacer
func (h *VCallRequestConstructorHandler) SetClass(ref reference.Global) {
	h.request.Callee.Set(ref)
	h.regenerate()
}

//nolint:interfacer
func (h *VCallRequestConstructorHandler) SetCaller(ref reference.Global) {
	h.request.Caller.Set(ref)
	h.regenerate()
}

func (h *VCallRequestConstructorHandler) SetCallFlags(flags rms.CallFlags) {
	h.request.CallFlags = flags
	h.regenerate()
}

func (h *VCallRequestConstructorHandler) SetCallFlagsFromMethodIsolation(mi contract.MethodIsolation) {
	h.request.CallFlags = rms.BuildCallFlags(mi.Interference, mi.State)
	h.regenerate()
}

func (h *VCallRequestConstructorHandler) SetCallSequence(sequence uint32) {
	h.request.CallSequence = sequence
	h.regenerate()
}

func (h *VCallRequestConstructorHandler) SetArguments(args []byte) {
	h.request.Arguments = rms.NewBytes(args)
	h.regenerate()
}

func (h *VCallRequestConstructorHandler) SetMethod(method string) {
	h.request.CallSiteMethod = method
	h.regenerate()
}

// deprecated
func (h *VCallRequestConstructorHandler) SetCallOutgoing(callOutgoing reference.Global) {
	h.outgoing = callOutgoing
	h.outgoingRewritten = true
	h.regenerate()
}

func generateVCallRequestConstructorForPulse(server *Server, pn pulse.Number) VCallRequestConstructorHandler {
	var (
		cIsolation = contract.ConstructorIsolation()
		callFlags  = rms.BuildCallFlags(cIsolation.Interference, cIsolation.State)
		arguments  = insolar.MustSerialize([]interface{}{})
		callee     = gen.UniqueGlobalRefWithPulse(pn)
	)

	hdl := VCallRequestConstructorHandler{
		request: rms.VCallRequest{
			CallType:       rms.CallTypeConstructor,
			CallFlags:      callFlags,
			Caller:         rms.NewReference(server.GlobalCaller()),
			Callee:         rms.NewReference(callee),
			CallSiteMethod: "New",
			CallSequence:   1,
			Arguments:      rms.NewBytes(arguments),
		},
		scheme:      server.platformScheme,
		pn:          pn,
		previousRef: gen.UniqueGlobalRefWithPulse(pn),
	}

	hdl.regenerate()

	return hdl
}

func GenerateVCallRequestConstructorForPulse(server *Server, pn pulse.Number) VCallRequestConstructorHandler {
	return generateVCallRequestConstructorForPulse(server, pn)
}

func GenerateVCallRequestConstructor(server *Server) VCallRequestConstructorHandler {
	return generateVCallRequestConstructorForPulse(server, server.GetPulseNumber())
}

// GenerateVCallRequestMethod returns CallTypeMethod VCallRequest for tolerable/dirty request by default
func GenerateVCallRequestMethod(server *Server) *rms.VCallRequest {
	return &rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		CallFlags:      rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
		Caller:         rms.NewReference(server.GlobalCaller()),
		Callee:         rms.NewReference(server.RandomGlobalWithPulse()),
		CallSiteMethod: "Method",
		CallSequence:   1,
		CallOutgoing:   rms.NewReference(server.BuildRandomOutgoingWithPulse()),
		CallReason:     rms.NewReference(server.BuildRandomOutgoingWithPulse()),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}
}

func GenerateVCallRequestMethodImmutable(server *Server) *rms.VCallRequest {
	pl := GenerateVCallRequestMethod(server)
	pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated)

	return pl
}
