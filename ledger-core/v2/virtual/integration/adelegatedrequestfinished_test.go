// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

// 1. Send VStateReport with 2 pending ordered
// 2. Send execute
// 3. Send VDelegateRequestFinished
// 4. Check that execute process request after VDelegateRequestFinished
func TestVirtual_SendVStateReport_And_VDelegateRequestFinished(t *testing.T) {

	t.Parallel()

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

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

	callFlags := payload.BuildCallRequestFlags(contract.CallTolerable, contract.CallDirty)

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

	requestIsDone := make(chan bool, 0)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		defer func() { requestIsDone <- true }()

		pl, err := payload.UnmarshalFromMeta(messages[0].Payload)
		require.NoError(t, err)

		switch pl.(type) {
		case *payload.VCallResult:
		default:
			require.Failf(t, "", "bad payload type, expected %s, got %T", "*payload.VCallResult", pl)
		}
		return nil
	}

	vStateReportMsg, err := wrapMsg(pulseNumber, &sr)
	require.NoError(t, err)

	vCallRequestMsg, err := wrapMsg(pulseNumber, &cr)
	require.NoError(t, err)

	vDelegateRequestFinishedMsg, err := wrapMsg(pulseNumber, &df)
	require.NoError(t, err)

	server.SendMessage(ctx, vStateReportMsg)

	server.SendMessage(ctx, vCallRequestMsg)

	select {
	case <-requestIsDone:
		// We should not execute request while we have pending execution.
		require.Failf(t, "", "unexpected execute")
	case <-time.After(5 * time.Second):
	}

	server.SendMessage(ctx, vDelegateRequestFinishedMsg)

	select {
	case <-requestIsDone:
		break
	case <-time.After(5 * time.Second):
		require.Failf(t, "", "timeout")
	}
}

func wrapMsg(pulseNumber pulse.Number, request payload.Marshaler) (*message.Message, error) {
	bytes, err := request.Marshal()
	if err != nil {
		return nil, throw.W(err, "failed to marshal request")
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
		return nil, throw.W(err, "failed to create new message")
	}

	return msg, nil
}
