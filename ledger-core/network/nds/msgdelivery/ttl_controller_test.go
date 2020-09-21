// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

func TestShipToWithTTL(t *testing.T) {
	payloadLen := 64

	head := TestString{string(rndBytes(payloadLen))}

	sh := Shipment{
		Head: &head,
	}

	var _, srv2 Service

	ch1 := make(chan string, 1)
	recv1 := func(a ReturnAddress, done nwapi.PayloadCompleteness, v interface{}) error {
		vo := v.(fmt.Stringer).String()
		t.Log(a.String(), fmt.Sprintf("ctrl-1:"), len(vo))

		require.True(t, bool(done))

		ch1 <- vo

		return nil
	}

	_, srv2, stop := startUniprotoServers(t, recv1, noopReceiver)
	defer stop()

	err := srv2.ShipTo(NewDirectAddress(1), sh)
	require.NoError(t, err)

	expPayload := head.S
	actlPayload := <-ch1

	require.Equal(t, expPayload, actlPayload)
}

type TestOneWayTransport struct {
	l1.OneWayTransport
	ch chan string
}

func (t TestOneWayTransport) Send(payload io.WriterTo) error {
	<-t.ch
	return t.OneWayTransport.Send(payload)
}
