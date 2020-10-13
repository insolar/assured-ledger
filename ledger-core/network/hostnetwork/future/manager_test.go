// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package future

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

func TestNewManager(t *testing.T) {
	m := NewManager()

	require.IsType(t, m, &futureManager{})
}

func TestFutureManager_Create(t *testing.T) {
	m := NewManager()

	sender := nwapi.NewHost("127.0.0.1:31337")
	receiver := nwapi.NewHost("127.0.0.2:31338")

	p := packet.NewPacket(sender, receiver, types.Unknown, 123)
	future := m.Create(p)

	require.EqualValues(t, future.ID(), p.RequestID)
	require.Equal(t, future.Request(), p)
	require.Equal(t, future.Receiver(), &receiver)
}

func TestFutureManager_Get(t *testing.T) {
	m := NewManager()

	sender := nwapi.NewHost("127.0.0.1:31337")
	receiver := nwapi.NewHost("127.0.0.2:31338")

	p := packet.NewPacket(sender, receiver, types.Unknown, 123)

	require.Nil(t, m.Get(p))

	expectedFuture := m.Create(p)
	actualFuture := m.Get(p)

	require.Equal(t, expectedFuture, actualFuture)
}

func TestFutureManager_Canceler(t *testing.T) {
	m := NewManager()

	sender := nwapi.NewHost("127.0.0.1:31337")
	receiver := nwapi.NewHost("127.0.0.2:31338")

	p := packet.NewPacket(sender, receiver, types.Unknown, 123)

	future := m.Create(p)
	require.NotNil(t, future)

	future.Cancel()

	require.Nil(t, m.Get(p))
}
