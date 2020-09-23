// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPulseGenerator(t *testing.T) {
	delta := uint16(13)
	pg := NewPulseGenerator(delta)

	_, err := pg.Last()
	require.Error(t, err)

	packet := pg.Generate()
	lastPacket, err := pg.Last()
	require.NoError(t, err)
	require.Equal(t, packet, lastPacket)

	{
		lastPacket = pg.Generate()
		require.NotEqual(t, lastPacket, packet)
		require.Equal(t, delta, uint16(lastPacket.PulseNumber-packet.PulseNumber))

		lp, err := pg.Last()
		require.NoError(t, err)
		require.Equal(t, lp, lastPacket)
	}

	{
		back0Packet, err := pg.CountBack(0)
		require.NoError(t, err)
		require.Equal(t, lastPacket, back0Packet)

		back1Packet, err := pg.CountBack(1)
		require.NoError(t, err)
		require.Equal(t, back1Packet, packet)

		_, err = pg.CountBack(2)
		require.Error(t, err)

		_, err = pg.CountBack(2000)
		require.Error(t, err)
	}
}
