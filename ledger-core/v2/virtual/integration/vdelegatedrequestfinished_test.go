// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"github.com/ThreeDotsLabs/watermill/message"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"
)

// 1. Send VStateReport with 2 pending ordered
// 2. Send execute
// 3. Send VDelegateRequestFinished
// 4. Check that execute process request after VDelegateRequestFinished
func TestVirtual_SendVStateReport_And_VDelegateRequestFinished(t *testing.T) {
	server := utils.NewServer(t)
	ctx := inslogger.TestContext(t)

	//Test steps for checks sequence.
	const (
		stateReportSend           = "stateReportSend"
		callRequestSend           = "callRequestSend"
		delegateRequestFinishSend = "delegateRequestFinishSend"
		callResultReceive         = "callResultReceive"
	)

	testBalance := uint32(100)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.Reference()

	proto := testwalletProxy.GetPrototype()
	pulseNumber := server.GetPulse().PulseNumber
	stateRef := gen.UniqueIDWithPulse(pulseNumber)

	sr := payload.VStateReport{
		Polymorph:           uint32(payload.TypeVStateReport),
		MutablePendingCount: 1,
		Callee:              objectRef,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: stateRef,
				Prototype: proto,
				State:     rawWalletState,
			},
		},
	}

	var callFlags payload.CallRequestFlags

	callFlags.SetTolerance(payload.CallTolerable)

	cr := payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTMethod,
		CallFlags:           callFlags,
		CallAsOf:            0,
		Caller:              server.GlobalCaller(),
		Callee:              objectRef,
		CallSiteDeclaration: proto,
		CallSiteMethod:      "Accept",
		CallRequestFlags:    0,
		CallOutgoing:        server.RandomLocalWithPulse(),
		Arguments:           insolar.MustSerialize([]interface{}{uint32(10)}),
	}

	df := payload.VDelegatedRequestFinished{
		Polymorph: uint32(payload.TypeVDelegatedRequestFinished),
		CallFlags: callFlags,
		Callee:    objectRef,
	}

	requestIsDone := make(chan string, 0)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		defer func() { requestIsDone <- callResultReceive }()

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		require.NoError(t, err)

		switch pl.(type) {
		case *payload.VCallResult:
		default:
			require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
		}
		return nil
	}

	aclSeq := make([]string, 0)

	vStateReportMsg, err := wrapMsg(pulseNumber, &sr)
	require.NoError(t, err)

	vCallRequestMsg, err := wrapMsg(pulseNumber, &cr)
	require.NoError(t, err)

	vDelegateRequestFinishedMsg, err := wrapMsg(pulseNumber, &df)
	require.NoError(t, err)

	server.SendMessage(ctx, vStateReportMsg)
	aclSeq = append(aclSeq, stateReportSend)

	server.SendMessage(ctx, vCallRequestMsg)
	aclSeq = append(aclSeq, callRequestSend)

	server.SendMessage(ctx, vDelegateRequestFinishedMsg)
	aclSeq = append(aclSeq, delegateRequestFinishSend)

	select {
	case res := <-requestIsDone:
		aclSeq = append(aclSeq, res)
		break
	case <-time.After(20 * time.Second):
		require.Failf(t, "", "timeout")
	}

	expSeq := []string{stateReportSend, callRequestSend, delegateRequestFinishSend, callResultReceive}
	require.True(t, reflect.DeepEqual(expSeq, aclSeq))
}

func wrapMsg(pulseNumber insolar.PulseNumber, request payload.Marshaler) (*message.Message, error) {
	bytes, err := request.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	msg, err := payload.NewMessage(&payload.Meta{
		Polymorph:  uint32(payload.TypeMeta),
		Payload:    bytes,
		Sender:     reference.Global{},
		Receiver:   reference.Global{},
		Pulse:      pulseNumber,
		ID:         nil,
		OriginHash: payload.MessageHash{},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new message")
	}

	return msg, nil
}
