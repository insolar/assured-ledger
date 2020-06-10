// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package authentication

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var deadBeef = [...]byte{0xde, 0xad, 0xbe, 0xef}

type Service interface {
	GetCallDelegationToken(outgoing reference.Global, to reference.Global, pn pulse.Number, object reference.Global) payload.CallDelegationToken
	IsMessageFromVirtualLegitimate(ctx context.Context, payloadObj interface{}, sender reference.Global) error
}

type service struct {
	selfNode     reference.Global
	pulseManager *conveyor.PulseDataManager
	affinity     jet.AffinityHelper
}

func NewService(_ context.Context, selfNode reference.Global, pulseManager *conveyor.PulseDataManager, affinity jet.AffinityHelper) Service {
	return service{selfNode: selfNode, pulseManager: pulseManager, affinity: affinity}
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

func (s service) checkDelegationToken() error {
	// TODO: check signature
	return nil
}

type TokenExtractor interface {
	GetDelegationSpec() payload.CallDelegationToken
}

func (s service) getPrevPulse() (pulse.Number, error) {
	previousPN, prevRange := s.pulseManager.GetPrevPulseRange()
	if previousPN == pulse.Unknown {
		return pulse.Unknown, errors.New("required previous pulse doesn't exists")
	}
	if prevRange == nil {
		// More info https://insolar.atlassian.net/browse/PLAT-355
		panic(errors.NotImplemented())
	}
	return prevRange.RightBoundData().PulseNumber, nil
}

func (s service) IsMessageFromVirtualLegitimate(ctx context.Context, payloadObj interface{}, sender reference.Global) error {
	var (
		token           payload.CallDelegationToken
		objectRef       reference.Global
		currentPulse, _ = s.pulseManager.GetPresentPulse()
		requiredPulse   = currentPulse
		err             error
	)

	tokenExtractor, ok := payloadObj.(TokenExtractor)
	if !ok {
		return errors.New("message must implement DelegationExtractor interface")
	}
	token = tokenExtractor.GetDelegationSpec()
	if !token.IsZero() {
		return s.checkDelegationToken()
	}

	switch obj := payloadObj.(type) {
	case *payload.VCallRequest:
		objectRef = obj.Caller
	case *payload.VCallResult:
		objectRef = obj.Caller
	case *payload.VStateRequest:
		objectRef = obj.Callee
	case *payload.VStateReport:
		objectRef = obj.Callee
		requiredPulse, err = s.getPrevPulse()
		if err != nil {
			return errors.W(err, "can't get prev pulse")
		}
	case *payload.VDelegatedRequestFinished:
		objectRef = obj.Callee
	case *payload.VDelegatedCallResponse:
		objectRef = obj.Callee
	case *payload.VDelegatedCallRequest:
		objectRef = obj.Callee
		requiredPulse, err = s.getPrevPulse()
		if err != nil {
			return errors.W(err, "can't get prev pulse")
		}
	default:
		panic("Unexpected message type")
	}

	if objectRef.Equal(statemachine.APICaller) {
		// it's dirty hack to exclude checking of testAPI requests
		return nil
	}

	expectedVE, err := s.affinity.QueryRole(ctx, node.DynamicRoleVirtualExecutor, objectRef.GetLocal(), requiredPulse)
	if err != nil {
		return errors.W(err, "can't calculate role")
	}

	if len(expectedVE) > 1 {
		panic(errors.NotImplemented())
	}

	if !sender.Equal(expectedVE[0]) {
		return errors.New("unexpected sender", struct {
			Sender     string
			ExpectedVE string
		}{Sender: sender.String(), ExpectedVE: expectedVE[0].String()})
	}

	return nil
}
