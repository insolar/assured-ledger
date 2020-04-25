// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handle_test

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/handle"
)

func TestError_BadMsgPayload(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)

	msg := message.NewMessage(watermill.NewUUID(), []byte{1, 2, 3, 4, 5})
	handler := handle.NewError(msg)
	err := handler.Present(ctx, flow.NewFlowMock(t))

	// We get error inside error-handler,
	// but only print log message for this,
	// without error returning.
	require.NoError(t, err)
}

func TestError_HappyPath(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)
	f := flow.NewFlowMock(t)

	meta := payload.Meta{
		Polymorph: uint32(payload.TypeMeta),
		// Good error payload.
		Payload: payload.MustMarshal(&payload.Error{
			Polymorph: uint32(payload.TypeError),
			Code:      payload.CodeUnknown,
			Text:      "something good",
		}),
	}

	p, err := meta.Marshal()
	require.NoError(t, err)

	msg := message.NewMessage(watermill.NewUUID(), p)
	handler := handle.NewError(msg)
	err = handler.Present(ctx, f)

	// We get error inside error-handler,
	// but only print log message for this,
	// without error returning.
	require.NoError(t, err)
}
