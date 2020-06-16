// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package authentication

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var deadBeef = [...]byte{0xde, 0xad, 0xbe, 0xef}

type Service interface {
	GetCallDelegationToken(outgoing reference.Global, to reference.Global, pn pulse.Number, object reference.Global) payload.CallDelegationToken
	IsMessageFromVirtualLegitimate(ctx context.Context, payloadObj interface{}, sender reference.Global, pr pulse.Range) (mustReject bool, err error)
}

type service struct {
	selfNode reference.Global
	affinity jet.AffinityHelper
}

func NewService(_ context.Context, selfNode reference.Global, affinity jet.AffinityHelper) Service {
	return service{selfNode: selfNode, affinity: affinity}
}

func (s service) GetCallDelegationToken(outgoing reference.Global, to reference.Global, pn pulse.Number, object reference.Global) payload.CallDelegationToken {
	return payload.CallDelegationToken{
		TokenTypeAndFlags: payload.DelegationTokenTypeCall,
		Approver:          s.selfNode,
		DelegateTo:        to,
		PulseNumber:       pn,
		Callee:            object,
		Caller:            to,
		Outgoing:          outgoing,
		ApproverSignature: deadBeef[:],
	}
}

func (s service) checkDelegationToken(expectedVE reference.Global, token payload.CallDelegationToken) error {
	// TODO: check signature

	if !token.Approver.Equal(expectedVE) {
		return throw.New("token Approver and expectedVE are different",
			struct {
				ExpectedVE string
				Approver   string
			}{ExpectedVE: expectedVE.String(), Approver: token.Approver.String()})
	}

	if token.Approver.Equal(s.selfNode) {
		return throw.New("selfNode cannot be equal to token Approver",
			struct {
				SelfNode string
				Approver string
			}{SelfNode: s.selfNode.String(), Approver: token.Approver.String()})
	}
	return nil
}

func (s service) getExpectedVE(ctx context.Context, subjectRef reference.Global, verifyForPulse pulse.Number) (reference.Global, error) {
	expectedVE, err := s.affinity.QueryRole(ctx, node.DynamicRoleVirtualExecutor, subjectRef.GetLocal(), verifyForPulse)
	if err != nil {
		return reference.Global{}, throw.W(err, "can't calculate role")
	}

	if len(expectedVE) > 1 {
		panic(throw.Impossible())
	}

	return expectedVE[0], nil
}

func (s service) IsMessageFromVirtualLegitimate(ctx context.Context, payloadObj interface{}, sender reference.Global, pr pulse.Range) (bool, error) {
	verifyForPulse := pr.RightBoundData().PulseNumber
	subjectRef, usePrev, ok := payload.GetSenderAuthenticationSubjectAndPulse(payloadObj)
	switch {
	case !ok:
		panic("Unexpected message type")
	case usePrev:
		if !pr.IsArticulated() {
			if prevDelta := pr.LeftPrevDelta(); prevDelta > 0 {
				if prevPN, ok := pr.LeftBoundNumber().TryPrev(prevDelta); ok {
					verifyForPulse = prevPN
					break
				}
			}
		}
		// this is either a first pulse or the node is just started. In both cases we should allow the message to run
		// but have to indicate that it has to be rejected
		verifyForPulse = pulse.Unknown
	}

	if subjectRef.Equal(statemachine.APICaller) {
		// it's dirty hack to exclude checking of testAPI requests
		return false, nil
	}

	if verifyForPulse == pulse.Unknown {
		return true, nil
	}

	expectedVE, err := s.getExpectedVE(ctx, subjectRef, verifyForPulse)
	if err != nil {
		return false, throw.W(err, "can't get expected VE")
	}

	if token, ok := payload.GetSenderDelegationToken(payloadObj); ok && !token.IsZero() {
		return false, s.checkDelegationToken(expectedVE, token)
	}

	if !sender.Equal(expectedVE) {
		return false, throw.New("unexpected sender", struct {
			Sender     string
			ExpectedVE string
			Pulse      string
		}{Sender: sender.String(), ExpectedVE: expectedVE.String(), Pulse: verifyForPulse.String()})
	}

	return false, nil
}
