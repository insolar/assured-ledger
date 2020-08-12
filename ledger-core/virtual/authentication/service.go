// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package authentication

import (
	"context"

	affinity2 "github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var deadBeef = [...]byte{0xde, 0xad, 0xbe, 0xef}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/virtual/authentication.Service -o ./ -s _mock.go -g
type Service interface {
	GetCallDelegationToken(outgoing reference.Global, to reference.Global, pn pulse.Number, object reference.Global) payload.CallDelegationToken
	CheckMessageFromAuthorizedVirtual(ctx context.Context, payloadObj interface{}, sender reference.Global, pr pulse.Range) (mustReject bool, err error)
	HasToSendToken(token payload.CallDelegationToken) bool
}

type service struct {
	affinity affinity2.Helper
}

func NewService(_ context.Context, affinity affinity2.Helper) Service {
	return service{affinity: affinity}
}

func (s service) GetCallDelegationToken(outgoing reference.Global, to reference.Global, pn pulse.Number, object reference.Global) payload.CallDelegationToken {
	return payload.CallDelegationToken{
		TokenTypeAndFlags: payload.DelegationTokenTypeCall,
		Approver:          s.affinity.Me(),
		DelegateTo:        to,
		PulseNumber:       pn,
		Callee:            object,
		Caller:            to,
		Outgoing:          outgoing,
		ApproverSignature: deadBeef[:],
	}
}

func (s service) HasToSendToken(token payload.CallDelegationToken) bool {
	useToken := true
	if token.Approver == s.affinity.Me() {
		useToken = false
	}
	return useToken
}

func (s service) checkDelegationToken(expectedVE reference.Global, token payload.CallDelegationToken, sender reference.Global) error {
	// TODO: check signature

	if !token.Approver.Equal(expectedVE) {
		return throw.New("token Approver and expectedVE are different",
			struct {
				ExpectedVE string
				Approver   string
			}{ExpectedVE: expectedVE.String(), Approver: token.Approver.String()})
	}

	if !token.DelegateTo.Equal(sender) {
		return throw.WithSeverity(throw.New("token DelegateTo and sender are different",
			struct {
				ExpectedVE string
				Approver   string
			}{ExpectedVE: expectedVE.String(), Approver: token.Approver.String()}), throw.RemoteBreachSeverity)
	}

	if sender.Equal(token.Approver) {
		return throw.WithSeverity(throw.New("sender cannot be approver of the token", struct {
			Sender string
		}{Sender: sender.String()}), throw.FraudSeverity)
	}
	return nil
}

func (s service) getExpectedVE(ctx context.Context, subjectRef reference.Global, verifyForPulse pulse.Number) (reference.Global, error) {
	expectedVE, err := s.affinity.QueryRole(ctx, affinity2.DynamicRoleVirtualExecutor, subjectRef, verifyForPulse)
	if err != nil {
		return reference.Global{}, throw.W(err, "can't calculate role")
	}

	if len(expectedVE) > 1 {
		panic(throw.Impossible())
	}

	return expectedVE[0], nil
}

func (s service) CheckMessageFromAuthorizedVirtual(ctx context.Context, payloadObj interface{}, sender reference.Global, pr pulse.Range) (bool, error) {
	verifyForPulse := pr.RightBoundData().PulseNumber
	subjectRef, mode, ok := payload.GetSenderAuthenticationSubjectAndPulse(payloadObj)
	if !ok {
		return false, throw.New("Unexpected message type")
	}
	switch mode {
	case payload.UsePrevPulse:
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
	case payload.UseAnyPulse:
		return false, nil
	case payload.UseCurrentPulse:
		//
	default:
		panic(throw.IllegalValue())
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
		return false, s.checkDelegationToken(expectedVE, token, sender)
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
