// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
)

func Test_API_Create(t *testing.T) {
	t.Log("C4837")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		// verify and decode incoming message
		assert.Len(t, messages, 1)

		server.SendMessage(ctx, messages[0])

		return nil
	}

	code, byteBuffer := server.CallAPICreateWallet(ctx)
	if !assert.Equal(t, 200, code) {
		t.Log(string(byteBuffer))
	} else {
		walletResponse, err := utils.UnmarshalWalletCreateResponse(byteBuffer)
		require.NoError(t, err)
		assert.Empty(t, walletResponse.Err)
		assert.NotEmpty(t, walletResponse.Ref)
		assert.NotEmpty(t, walletResponse.TraceID)
	}
}
